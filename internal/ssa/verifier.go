package ssa

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
)

type verifierToken struct {
	constraint GroupingConstraint
	// The constrained grouping that gave rise to the token.
	// This is used to generate meaningful error messages.
	origin        *Grouping
	visitedScopes []*ssaScope
}

type verifierData struct {
	tokens          []*verifierToken
	scopeConstraint *ircode.CommandScope
}

type groupingVerifier struct {
	s   *ssaTransformer
	log *errlog.ErrorLog
	// Used to remember the locations at which errors have been reported.
	// Thus way, the compile will not emit the same error multiple times for the same line
	errorLocations map[errlog.LocationRange]bool
	markerCounter  int
}

func (d *verifierData) hasToken(t *verifierToken) bool {
	for _, t2 := range d.tokens {
		if t2 == t {
			return true
		}
		if !t2.constraint.Equals(&t.constraint) {
			continue
		}
		ok := true
		for _, s2 := range t2.visitedScopes {
			found := false
			for _, s := range t.visitedScopes {
				if s == s2 {
					found = true
					break
				}
			}
			if !found {
				ok = false
				break
			}
		}
		// println("DOUBLE token", t.constraint.NamedGroup, t2.constraint.NamedGroup)
		if ok {
			return true
		}
	}
	return false
}

func (t *verifierToken) clone() *verifierToken {
	t2 := &verifierToken{constraint: t.constraint, origin: t.origin, visitedScopes: make([]*ssaScope, len(t.visitedScopes))}
	copy(t2.visitedScopes, t.visitedScopes)
	return t
}

func newGroupingVerifier(s *ssaTransformer, log *errlog.ErrorLog) *groupingVerifier {
	return &groupingVerifier{s: s, log: log, errorLocations: make(map[errlog.LocationRange]bool)}
}

func (ver *groupingVerifier) verify() {
	//
	// Verify that the constraints of merged groupings
	// do not cause conflicts, e.g. when two different parameter
	// groupings are being merged, or when two scoped groupings
	// are merged even though the scopes are not nested or equal.
	//
	for _, grp := range ver.s.parameterGroupings {
		ver.floodConstraint(grp)
	}
	for _, grp := range ver.s.valueGroupings {
		grp.data.scopeConstraint = grp.Constraint.Scope
		// Shortcut, because this is true most of the time
		if len(grp.Input) == 0 && len(grp.Output) == 0 {
			continue
		}
		ver.floodConstraint(grp)
	}
	for _, grp := range ver.s.parameterGroupings {
		ver.check(grp)
	}
	for _, grp := range ver.s.valueGroupings {
		// Shortcut, because this is true most of the time
		if len(grp.Input) == 0 && len(grp.Output) == 0 {
			continue
		}
		ver.check(grp)
	}

	// Verify that groupings are available when they are being used
	ver.verifyBlock(&ver.s.f.Body)
}

func (ver *groupingVerifier) floodConstraint(grp *Grouping) {
	// println("flood constraint from", grp.GroupingName())
	t := &verifierToken{origin: grp, constraint: grp.Constraint}
	grp.data.tokens = append(grp.data.tokens, t)
	ver.floodToken(grp, t)
}

func (ver *groupingVerifier) floodToken(grp *Grouping, t *verifierToken) {
	// println("flood from", grp.GroupingName(), len(t.visitedScopes), "in", len(grp.Input), "out", len(grp.Output))
	for _, dest := range grp.Input {
		ver.forwardToken(grp, dest.Original, t)
	}
	for _, dest := range grp.Output {
		ver.forwardToken(grp, dest.Original, t)
	}
}

func (ver *groupingVerifier) forwardToken(grp *Grouping, dest *Grouping, t *verifierToken) {
	if grp.Original.scope != dest.Original.scope {
		t = t.clone()
		t.visitedScopes = append(t.visitedScopes, grp.scope)
	}
	if dest.Kind == LoopPhiGrouping {
		// println("  pass loop-phi", len(t.visitedScopes))
		// Did the token come across a scope that breaks or returns out of the destination loop?
		// If so, the token must not pass the loop-phi-grouping, because control flow does not take the loop.
		breaksTheLoop := false
		for _, scope := range t.visitedScopes {
			for _, exit := range scope.exitScopes {
				if exit == dest.loopScope {
					breaksTheLoop = true
					break
				}
			}
			if breaksTheLoop {
				break
			}
		}
		// If control flow could possibly take the loop, remove all visited scopes inside the loop from the token,
		// because control flow can visit them again in the next loop iteration.
		if !breaksTheLoop {
			for i := 0; i < len(t.visitedScopes); {
				if t.visitedScopes[i].hasParent(dest.loopScope) {
					// println("    remove scope")
					t.visitedScopes = append(t.visitedScopes[:i], t.visitedScopes[i+1:]...)
				} else {
					i++
				}
			}
		}
	}
	ver.acceptToken(dest, t)
}

func (ver *groupingVerifier) acceptToken(grp *Grouping, t *verifierToken) {
	if grp.data.hasToken(t) {
		// println("  not accepted by", grp.GroupingName())
		return
	}
	for _, s := range t.visitedScopes {
		if ver.doesInhibitToken(grp.scope, s) {
			return
		}
	}
	grp.data.tokens = append(grp.data.tokens, t)
	grp.data.scopeConstraint = mergeScopeConstraint(grp.data.scopeConstraint, t.constraint.Scope)
	ver.floodToken(grp, t)
}

func (ver *groupingVerifier) doesInhibitToken(s *ssaScope, tokenScope *ssaScope) bool {
	ver.markerCounter++
	marker := ver.markerCounter
	for s2 := tokenScope; s2 != nil && s2 != s && s2 != s.parent; s2 = s2.parent {
		s2.marker = marker
	}
	for s2 := s; s2 != nil; s2 = s2.parent {
		for _, alt := range s2.alternativeScopes {
			if alt.marker == marker {
				return true
			}
		}
	}
	return false
}

func (ver *groupingVerifier) check(grp *Grouping) {
	// Try to merge the constraints of all tokens.
	// Remember all tokens that could not be merged (see `open`).
	// These either represent an error, or they are alternatives, which means the grouping is over-constrained
	// (meaning there is a constraint but we can no longer say which). Over-constrained groupings because of alternative
	// program flow (e.g. if-else) are not an error.
	var open []*verifierToken
	var merged []*verifierToken
	constraint := grp.Constraint
	for _, t := range grp.data.tokens {
		c := mergeGroupingConstraint(t.constraint, constraint, nil, nil, nil)
		if c.Error {
			open = append(open, t)
		} else {
			merged = append(merged, t)
			constraint = c
		}
	}
	// println("check", grp.GroupingName(), len(grp.data.tokens), len(merged), len(open))
	// Iterate over the tokens that could not be merged.
	// Determine whether these are alternatives resulting from alternative program flows (no error).
	// Otherwise, raise an error.
	for _, ot := range open {
		done := false
		for _, mt := range merged {
			if ver.doesInhibitToken(ot.origin.scope, mt.origin.scope) {
				constraint.OverConstrained = true
				done = true
			}
		}
		if !done {
			// println("ERR: Cannot merge constraints")
			// Find all merges that could contribute to this error
			var locs []errlog.LocationRange
			for _, g2 := range ver.groupingMerges(grp) {
				// println("   via " + g2.Name)
				locs = append(locs, g2.Location)
			}
			// Raise an error
			ver.log.AddGroupingError(constraintToErrorString(grp.Constraint), grp.Location, constraintToErrorString(ot.constraint), locs...)
		}
	}
	grp.Constraint = constraint
}

func (ver *groupingVerifier) verifyBlock(c *ircode.Command) {
	if c.Op != ircode.OpBlock && c.Op != ircode.OpIf && c.Op != ircode.OpLoop {
		panic("Not a block")
	}
	for _, c2 := range c.Block {
		ver.verifyCommand(c2)
	}
	c.Scope.Marker = 1
}

func (ver *groupingVerifier) verifyPreBlock(c *ircode.Command) {
	for _, c2 := range c.PreBlock {
		ver.verifyCommand(c2)
	}
}

func (ver *groupingVerifier) verifyIterBlock(c *ircode.Command) {
	for _, c2 := range c.IterBlock {
		ver.verifyCommand(c2)
	}
}

func (ver *groupingVerifier) verifyCommand(c *ircode.Command) {
	if c.PreBlock != nil {
		ver.verifyPreBlock(c)
	}
	switch c.Op {
	case ircode.OpBlock:
		ver.verifyBlock(c)
	case ircode.OpIf:
		// visit the condition
		ver.verifyArguments(c)
		ver.verifyBlock(c)
		// visit the else-clause
		if c.Else != nil {
			ver.verifyBlock(c.Else)
		}
	case ircode.OpLoop:
		ver.verifyBlock(c)
		ver.verifyIterBlock(c)
	case ircode.OpBreak:
		// Do nothing by intention
	case ircode.OpContinue:
		// Do nothing by intention
	case ircode.OpDefVariable:
		// Do nothing by intention
	case ircode.OpSet:
		ver.verifyArguments(c)
	case ircode.OpGet:
		ver.verifyArguments(c)
	case ircode.OpTake:
		ver.verifyArguments(c)
	case ircode.OpSetVariable:
		ver.verifyArguments(c)
	case ircode.OpAdd,
		ircode.OpSub,
		ircode.OpMul,
		ircode.OpDiv,
		ircode.OpRemainder,
		ircode.OpBinaryXor,
		ircode.OpBinaryOr,
		ircode.OpBinaryAnd,
		ircode.OpShiftLeft,
		ircode.OpShiftRight,
		ircode.OpBitClear,
		ircode.OpLogicalOr,
		ircode.OpLogicalAnd,
		ircode.OpEqual,
		ircode.OpNotEqual,
		ircode.OpLess,
		ircode.OpGreater,
		ircode.OpLessOrEqual,
		ircode.OpGreaterOrEqual,
		ircode.OpNot,
		ircode.OpMinusSign,
		ircode.OpBitwiseComplement,
		ircode.OpSizeOf,
		ircode.OpLen,
		ircode.OpCap:

		ver.verifyArguments(c)
	case ircode.OpPanic,
		ircode.OpPrintln:
		ver.verifyArguments(c)
	case ircode.OpGroupOf:
		ver.verifyArguments(c)
		ver.verifyGroupArgs(c)
	case ircode.OpArray,
		ircode.OpStruct:

		ver.verifyArguments(c)
	case ircode.OpMalloc,
		ircode.OpMallocSlice,
		ircode.OpStringConcat:

		ver.verifyArguments(c)
	case ircode.OpAppend:
		ver.verifyArguments(c)
	case ircode.OpOpenScope,
		ircode.OpCloseScope:
		// Do nothing by intention
	case ircode.OpReturn:
		ver.verifyArguments(c)
	case ircode.OpCall:
		ver.verifyArguments(c)
		ver.verifyGroupArgs(c)
	case ircode.OpAssert:
		ver.verifyArguments(c)
	case ircode.OpSetGroupVariable:
		ver.verifyGroupArgs(c)
	case ircode.OpFree:
		ver.verifyArguments(c)
	case ircode.OpMerge:
		ver.verifyGroupArgs(c)
	default:
		println(c.Op)
		panic("Ooops")
	}
}

func (ver *groupingVerifier) verifyArguments(c *ircode.Command) {
	for i := len(c.Args) - 1; i >= 0; i-- {
		if c.Args[i].Cmd != nil {
			ver.verifyCommand(c.Args[i].Cmd)
		} else if c.Args[i].Var != nil {
			grp, ok := c.Args[i].Var.Grouping.(*Grouping)
			if ok && grp != nil {
				if !ver.verifyGrouping(grp, c, c.Location) {
					return
				}
			}
		}
	}
}

func (ver *groupingVerifier) verifyGroupArgs(c *ircode.Command) {
	for _, garg := range c.GroupArgs {
		if !ver.verifyGrouping(garg.(*Grouping), c, c.Location) {
			return
		}
	}
}

func (ver *groupingVerifier) verifyGrouping(grp *Grouping, c *ircode.Command, loc errlog.LocationRange) bool {
	scope := c.Scope
	if grp.data.scopeConstraint != nil {
		if grp.data.scopeConstraint != scope && !scope.HasParent(grp.data.scopeConstraint) {
			if !grp.data.scopeConstraint.HasParent(scope) || grp.data.scopeConstraint.Marker != 0 {
				loc2 := errlog.StripPosition(loc)
				if _, ok := ver.errorLocations[loc2]; ok {
					return false
				}
				ver.errorLocations[loc2] = true
				ver.s.log.AddError(errlog.ErrGroupingOutOfScope, loc)
				return false
			}
		}
	}
	return true
}

func constraintToErrorString(c GroupingConstraint) string {
	if c.NamedGroup != "" {
		return "group `" + c.NamedGroup + "`"
	} else if c.Scope != nil {
		return "scoped group"
	}
	panic("Oooops")
}

// Returns a list of all `StaticMergeGrouping` and `DynamicMergeGrouping` groupings that are reachable from `grp`
func (ver *groupingVerifier) groupingMerges(grp *Grouping) []*Grouping {
	ver.markerCounter++
	marker := ver.markerCounter
	grp.markerNumber = marker
	var result []*Grouping
	for _, g2 := range grp.Input {
		result = groupingMergesIntern(g2.Original, result, marker)
	}
	for _, g2 := range grp.Output {
		result = groupingMergesIntern(g2.Original, result, marker)
	}
	return result
}

func groupingMergesIntern(grp *Grouping, result []*Grouping, marker int) []*Grouping {
	if grp.markerNumber == marker {
		return result
	}
	grp.markerNumber = marker
	if grp.Kind == StaticMergeGrouping || grp.Kind == DynamicMergeGrouping || grp.Kind == ParameterGrouping || grp.Kind == ScopedGrouping {
		result = append(result, grp)
	}
	for _, g2 := range grp.Input {
		result = groupingMergesIntern(g2.Original, result, marker)
	}
	for _, g2 := range grp.Output {
		result = groupingMergesIntern(g2.Original, result, marker)
	}
	return result
}
