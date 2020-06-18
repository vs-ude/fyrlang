package ssa

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
)

type verifierToken struct {
	origin   *Grouping
	original *verifierToken
	// The same token as used in the parent scope
	parent               *verifierToken
	possiblyMergedTokens []*verifierToken
}

type verifierData struct {
	outputTokens []*verifierToken
}

type verifierScope struct {
	command *ircode.Command
	parent  *verifierScope
	// All scopes that break out of this loop
	breaks    []*verifierScope
	continues []*verifierScope
	tokens    []*verifierToken
}

type groupingVerifier struct {
	s   *ssaTransformer
	log *errlog.ErrorLog
	// Used to remember the locations at which errors have been reported.
	// Thus way, the compile will not emit the same error multiple times for the same line
	// errorLocations map[errlog.LocationRange]bool
	markerCounter int
}

func (t *verifierToken) addPossiblyMergedToken(m *verifierToken) {
	if t == m {
		return
	}
	for _, t2 := range t.possiblyMergedTokens {
		if t2 == m {
			return
		}
	}
	// println("!!!!! addPossiblyMerged", m.origin.Name, m, "to", t.origin.Name, t)
	t.possiblyMergedTokens = append(t.possiblyMergedTokens, m)
}

func (t *verifierToken) hasPossiblyMerged(m *verifierToken) bool {
	for _, t2 := range t.possiblyMergedTokens {
		if t2 == m {
			return true
		}
	}
	return false
}

func (t *verifierToken) isEquivalent(t2 *verifierToken) bool {
	if t == t2 {
		return true
	}
	if len(t.possiblyMergedTokens) != len(t2.possiblyMergedTokens) {
		return false
	}
	if !t.hasPossiblyMerged(t2) || !t2.hasPossiblyMerged(t) {
		return false
	}
	for _, m := range t.possiblyMergedTokens {
		if m != t2 && !t2.hasPossiblyMerged(m) {
			return false
		}
	}
	return true
}

func newVerifierScope(c *ircode.Command, parent *verifierScope) *verifierScope {
	return &verifierScope{parent: parent, command: c}
}

func (s *verifierScope) lookupToken(t *verifierToken) *verifierToken {
	for _, t2 := range s.tokens {
		if t2.original == t.original {
			return t2
		}
	}
	tparent := s.parent.lookupToken(t)
	t2 := &verifierToken{origin: tparent.origin, original: tparent.original, parent: tparent}
	// println(".... lookup new proxy", t2.origin.Name, t2, "instead of", t)
	s.tokens = append(s.tokens, t2)
	if len(tparent.possiblyMergedTokens) > 0 {
		t2.possiblyMergedTokens = make([]*verifierToken, len(tparent.possiblyMergedTokens))
		for i, m := range tparent.possiblyMergedTokens {
			t2.possiblyMergedTokens[i] = s.lookupToken(m)
		}
	}
	return t2
}

func (s *verifierScope) searchToken(t *verifierToken) *verifierToken {
	for _, t2 := range s.tokens {
		if t2.original == t.original {
			return t2
		}
	}
	if s.parent == nil {
		return nil
	}
	return s.parent.searchToken(t)
}

func newGroupingVerifier(s *ssaTransformer, log *errlog.ErrorLog) *groupingVerifier {
	return &groupingVerifier{s: s, log: log /*, errorLocations: make(map[errlog.LocationRange]bool)*/}
}

func (ver *groupingVerifier) verify() {
	s := newVerifierScope(&ver.s.f.Body, nil)
	ver.verifyGroupings(&ver.s.f.Body, s)
	ver.verifyBlock(&ver.s.f.Body, s)
}

/*
func (ver *groupingVerifier) acceptToken(grp *Grouping, from *Grouping, t *verifierToken) {
	if grp.data.effectiveConstraint.Error {
		// Do not continue here. It cannot become worse than an error anyway
		return
	}
	tc := t.effectiveConstraint()

	firstToken := false
	if len(grp.data.inputTokens) == 0 {
		grp.data.inputTokens = make([]*verifierToken, len(grp.Input))
		grp.data.effectiveConstraint = tc
		firstToken = true
	}
	inputIndex := -1
	duplicate := false
	for i, input := range grp.Input {
		if from.Original == input.Original {
			if grp.data.inputTokens[i] == t {
				// println("    already got the token at", grp.Name)
				return
			}
			grp.data.inputTokens[i] = t
			inputIndex = i
		} else if grp.data.inputTokens[i] == t {
			duplicate = true
		}
	}
	if inputIndex == -1 {
		// println(len(grp.Input))
		panic("Oooops " + grp.Name)
	}
	if duplicate {
		// println("    already got the token from another input at", grp.Name)
		return
	}
	// No token came along this way so far?
	if firstToken {
		// println(t.origin.Constraint.NamedGroup+": accepted by", grp.GroupingName())
		grp.data.outputToken = t
		ver.floodToken(grp, t)
		return
	}

	switch grp.Kind {
	case LoopPhiGrouping:
	case BranchPhiGrouping:
		// At least one token has already passed this grouping via one branch.
		// Now there is another token coming along another branch. Try to merge (in case of scoped constraints).
		// Otherwise, the grouping becomes Overconstrained or even Error (in case of non-mergeable scopes).
		// println(t.origin.Constraint.NamedGroup+": phi-merged by", grp.GroupingName())
		c := mergePhiGroupingConstraint(tc, grp.data.effectiveConstraint, nil, nil)
		if c.Equals(&grp.data.effectiveConstraint) {
			// The merge did not bring any new information.
			// This can happen if `grp.data.token` was already Overcontrained or Error, or an earlier token delivered the same constraint.
			// No need to flood this token any further.
			t.addDominatingToken(grp.data.outputToken)
			// println("   already got constraint", grp.GroupingName())
			return
		}
		grp.data.effectiveConstraint = c
		// println(t.origin.Constraint.NamedGroup+": accepted by", grp.GroupingName())
		if !c.Equals(&tc) {
			// The new constraint `c` is different than `t` or `grp.data.token`.
			// This can happen if `c` is Overconstrained or Error.
			// Create a new token and flood it.
			t = &verifierToken{origin: grp}
		}
		for _, it := range grp.data.inputTokens {
			if it != nil && it != t {
				it.replaceDominatingToken(t, grp.data.outputToken)
			}
		}
		grp.data.outputToken = t
		ver.floodToken(grp, t)
	case StaticMergeGrouping:
	case DynamicMergeGrouping:
		// println(t.origin.Constraint.NamedGroup+": merged by", grp.GroupingName())
		c := mergeGroupingConstraint(tc, grp.data.effectiveConstraint, nil, nil, nil)
		if c.Error {
			// println(t.origin.Constraint.NamedGroup+": error by", grp.GroupingName())
			var locs []errlog.LocationRange
			for _, g2 := range ver.groupingMerges(grp) {
				locs = append(locs, g2.Location)
			}
			// Raise an error
			ver.log.AddGroupingError(constraintToErrorString(grp.data.effectiveConstraint), grp.Location, constraintToErrorString(tc), locs...)
			grp.data.effectiveConstraint = c
			return
		}
		if c.Equals(&grp.data.effectiveConstraint) {
			t.addDominatingToken(grp.data.outputToken)
			// println("   already got constraint via merge", grp.GroupingName())
			return
		}
		grp.data.effectiveConstraint = c
		// println(t.origin.Constraint.NamedGroup+": accepted by", grp.GroupingName())
		if !c.Equals(&tc) {
			// The new constraint `c` is different than `t` or `grp.data.token`.
			// This can happen if `c` is Overconstrained or Error.
			// Create a new token and flood it.
			t = &verifierToken{origin: grp}
		}
		for _, it := range grp.data.inputTokens {
			if it != nil && it != t {
				it.replaceDominatingToken(t, grp.data.outputToken)
			}
		}
		grp.data.outputToken = t
		ver.floodToken(grp, t)
	default:
		panic("Oooops")
	}
}
*/

func (ver *groupingVerifier) verifyBlock(c *ircode.Command, s *verifierScope) {
	if c.Op != ircode.OpBlock && c.Op != ircode.OpIf && c.Op != ircode.OpLoop {
		panic("Not a block")
	}
	for _, c2 := range c.Block {
		ver.verifyCommand(c2, s)
	}
	c.Scope.Marker = 1
}

func (ver *groupingVerifier) verifyPreBlock(c *ircode.Command, s *verifierScope) {
	for _, c2 := range c.PreBlock {
		ver.verifyCommand(c2, s)
	}
}

func (ver *groupingVerifier) verifyIterBlock(c *ircode.Command, s *verifierScope) {
	for _, c2 := range c.IterBlock {
		ver.verifyCommand(c2, s)
	}
}

func (ver *groupingVerifier) verifyCommand(c *ircode.Command, s *verifierScope) {
	if c.PreBlock != nil {
		ver.verifyPreBlock(c, s)
	}
	switch c.Op {
	case ircode.OpBlock:
		s2 := newVerifierScope(c, s)
		ver.verifyGroupings(c, s)
		ver.verifyBlock(c, s2)
		ver.mergeScope(s, s2)
	case ircode.OpIf:
		// visit the condition
		ver.verifyArguments(c)
		sif := newVerifierScope(c, s)
		ver.verifyBlock(c, sif)
		// visit the else-clause
		if c.Else != nil {
			selse := newVerifierScope(c, s)
			ver.verifyBlock(c.Else, selse)
			if !c.DoesNotComplete && !c.Else.DoesNotComplete {
				ver.mergeAlternateScopes(s, sif, selse)
			} else if !c.Else.DoesNotComplete {
				ver.mergeOptionalScope(s, selse)
			} else if !c.DoesNotComplete {
				ver.mergeOptionalScope(s, sif)
			}
		} else if !c.DoesNotComplete {
			ver.mergeOptionalScope(s, sif)
		}
		// The groupings caused by an `if` must be BranchPhiGroupings.
		ver.verifyGroupings(c, s)
	case ircode.OpLoop:
		sloop := newVerifierScope(c, s)
		// The groupings checked here are LoopPhiGroupings.
		ver.verifyLoopStartGroupings(c, s, sloop)
		ver.verifyIterBlock(c, sloop)
		ver.verifyBlock(c, sloop)
		// The groupings checked here are BranchPhiGroupings.
		// LoopPhiGroupings are treated a second time, now that
		// the loop body has been verified.
		ver.verifyLoopEndGroupings(c, s, sloop)
	case ircode.OpBreak:
		// Find the loop-scope
		loopDepth := int(c.Args[0].Const.ExprType.IntegerValue.Uint64()) + 1
		var loopScope *verifierScope = s
		for ; loopScope != nil; loopScope = loopScope.parent {
			if loopScope.command.Op == ircode.OpLoop {
				loopDepth--
				if loopDepth == 0 {
					break
				}
			}
		}
		if loopScope == nil {
			panic("Oooops")
		}
		loopScope.breaks = append(loopScope.breaks, s)
	case ircode.OpContinue:
		// Find the loop-scope
		loopDepth := int(c.Args[0].Const.ExprType.IntegerValue.Uint64()) + 1
		var loopScope *verifierScope = s
		for ; loopScope != nil; loopScope = loopScope.parent {
			if loopScope.command.Op == ircode.OpLoop {
				loopDepth--
				if loopDepth == 0 {
					break
				}
			}
		}
		if loopScope == nil {
			panic("Oooops")
		}
		loopScope.continues = append(loopScope.continues, s)
	case ircode.OpDefVariable:
		// Do nothing by intention
	case ircode.OpSet,
		ircode.OpSetAndAdd,
		ircode.OpSetAndSub,
		ircode.OpSetAndMul,
		ircode.OpSetAndDiv,
		ircode.OpSetAndRemainder,
		ircode.OpSetAndBinaryAnd,
		ircode.OpSetAndBinaryOr,
		ircode.OpSetAndBinaryXor,
		ircode.OpSetAndBitClear,
		ircode.OpSetAndShiftLeft,
		ircode.OpSetAndShiftRight:

		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
	case ircode.OpGet,
		ircode.OpGetForeignGroup:

		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
	case ircode.OpTake:
		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
	case ircode.OpSetVariable:
		ver.verifyGroupings(c, s)
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
		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
	case ircode.OpPanic,
		ircode.OpPrintln:
		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
	case ircode.OpGroupOf:
		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
		ver.verifyGroupArgs(c)
	case ircode.OpArray,
		ircode.OpStruct:
		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
	case ircode.OpMalloc,
		ircode.OpMallocSlice,
		ircode.OpStringConcat:
		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
	case ircode.OpAppend:
		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
	case ircode.OpOpenScope,
		ircode.OpCloseScope:
		// Do nothing by intention
	case ircode.OpReturn,
		ircode.OpDelete:
		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
	case ircode.OpCall:
		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
		ver.verifyGroupArgs(c)
	case ircode.OpAssert:
		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
	case ircode.OpSetGroupVariable:
		ver.verifyGroupings(c, s)
		ver.verifyGroupArgs(c)
	case ircode.OpFree:
		ver.verifyGroupings(c, s)
		ver.verifyArguments(c)
	case ircode.OpIncRef:
		ver.verifyGroupings(c, s)
		ver.verifyGroupArgs(c)
	case ircode.OpMerge:
		// Do nothing by intention
	default:
		// println(c.Op)
		panic("Ooops")
	}
}

func (ver *groupingVerifier) mergeScope(scope *verifierScope, child *verifierScope) {
	// println("MERGE SCOPES")
	for _, ct := range child.tokens {
		parent := scope.searchToken(ct)
		if parent == nil {
			scope.tokens = append(scope.tokens, ct)
		} else {
			for _, m := range ct.possiblyMergedTokens {
				if parentM := scope.searchToken(m); parentM != nil {
					parent.addPossiblyMergedToken(parentM)
					parentM.addPossiblyMergedToken(parent)
				} else {
					parent.addPossiblyMergedToken(m)
					m.addPossiblyMergedToken(parent)
				}
			}
		}
	}
}

func (ver *groupingVerifier) mergeOptionalScope(scope *verifierScope, ifScope *verifierScope) {
	ver.mergeScope(scope, ifScope)
}

func (ver *groupingVerifier) mergeAlternateScopes(scope *verifierScope, ifScope, elseScope *verifierScope) {
	ver.mergeScope(scope, ifScope)
	ver.mergeScope(scope, elseScope)
}

func (ver *groupingVerifier) verifyGroupings(c *ircode.Command, s *verifierScope) {
	for _, igrp := range c.Groupings {
		grp := igrp.(*Grouping)
		ver.verifyGrouping(c, grp, s)
	}
}

func (ver *groupingVerifier) verifyLoopStartGroupings(c *ircode.Command, s *verifierScope, loopScope *verifierScope) {
	// println(">>>>>>>>>>>>>>>>>>> LOOP START")
	for _, igrp := range c.Groupings {
		grp := igrp.(*Grouping)
		if grp.Kind == BranchPhiGrouping {
			// Do nothing by intention
		} else if grp.Kind == LoopPhiGrouping {
			// The probeToken collects all possible merges that an input to this
			// loop-phi-grouping might encounter while passing through the loop body.
			// All tokens that pass this loop-phi-grouping (except those of the first input)
			// are merged with the probeToken at the end of the loop.
			probeToken := &verifierToken{origin: grp}
			probeToken.original = probeToken
			loopScope.tokens = append(loopScope.tokens, probeToken)
			grp.data.outputTokens = append(grp.data.outputTokens, probeToken)
			// println("LOOP PHI pre-loop", grp.Name, probeToken, "input:", len(grp.Input))
		} else {
			panic("Oooops")
		}
	}
}

func (ver *groupingVerifier) verifyLoopEndGroupings(c *ircode.Command, s *verifierScope, loopScope *verifierScope) {
	// println(">>>>>>>>>>>>>>>>>>> LOOP END")
	var err *errlog.Error
	for _, igrp := range c.Groupings {
		grp := igrp.(*Grouping)
		if grp.Kind == BranchPhiGrouping {
			// Do nothing by intention
		} else if grp.Kind == LoopPhiGrouping {
			// println("LOOP PHI post-loop", grp.Name, len(loopScope.continues))
			probeToken := grp.data.outputTokens[0]
			grp.data.outputTokens[0] = grp.Input[0].data.outputTokens[0]
			var merges []*verifierToken
			if !c.BlockDoesNotComplete && err != nil {
				merges, err = ver.verifyProbeToken(grp, probeToken, merges, c, loopScope)
				if err != nil {
					continue
				}
			}
			for _, continueScope := range loopScope.continues {
				pt := continueScope.lookupToken(probeToken)
				if probeToken != pt && err != nil {
					merges, err = ver.verifyProbeToken(grp, pt, merges, c, loopScope)
					if err != nil {
						continue
					}
				}
			}
			// println(".... merge loop")
			for i := 0; i < len(merges); i += 2 {
				ver.mergeTokensTransitive(c, merges[i], merges[i+1], loopScope)
			}
			// println("LOOP PHI post-break", grp.Name)
			for _, breakScope := range loopScope.breaks {
				pt := breakScope.lookupToken(probeToken)
				if probeToken != pt && err != nil {
					var breakMerges []*verifierToken
					breakMerges, err = ver.verifyProbeToken(grp, pt, breakMerges, c, breakScope)
					if err != nil {
						continue
					}
					// println(".... merge break")
					for i := 0; i < len(breakMerges); i += 2 {
						ver.mergeTokensTransitive(c, breakMerges[i], breakMerges[i+1], breakScope)
					}
				}
			}
		} else {
			panic("Oooops")
		}
	}

	ver.mergeOptionalScope(s, loopScope)
	for _, breakScope := range loopScope.breaks {
		ver.mergeOptionalScope(s, breakScope)
	}

	for _, igrp := range c.Groupings {
		grp := igrp.(*Grouping)
		if grp.Kind == BranchPhiGrouping {
			ver.verifyGrouping(c, grp, s)
		}
	}
}

func (ver *groupingVerifier) verifyProbeToken(grp *Grouping, probeToken *verifierToken, merges []*verifierToken, c *ircode.Command, loopScope *verifierScope) ([]*verifierToken, *errlog.Error) {
	// println("Verify probe token", grp.Name, len(probeToken.possiblyMergedTokens), probeToken)
	for _, inp1 := range grp.Input {
		// println("    input", inp1.Name)
		for _, t1 := range inp1.data.outputTokens {
			// println("     token", t1.origin.Name, t1)
			needsMerge, err := ver.verifyTokenMergeTransitive(c, grp, t1, probeToken, loopScope)
			if err != nil {
				return merges, err
			}
			if needsMerge {
				merges = append(merges, t1, probeToken)
			}
		}
	}
	return merges, nil
}

func (ver *groupingVerifier) verifyGrouping(c *ircode.Command, grp *Grouping, s *verifierScope) {
	switch grp.Kind {
	case DefaultGrouping,
		ConstantGrouping,
		ParameterGrouping,
		ForeignGrouping,
		ScopedGrouping:
		// Create a new token
		t := &verifierToken{origin: grp}
		t.original = t
		grp.data.outputTokens = []*verifierToken{t}
		s.tokens = append(s.tokens, t)
		if grp.Constraint.Scope != nil {
			// println("CREATED SCOPED", grp.Name, grp, s)
		} else {
			// println("CREATED", grp.Name, grp, s)
		}
	case BranchPhiGrouping:
		// println("BRANCH PHI", grp.Name)
		// Create a new token
		t := &verifierToken{origin: grp}
		t.original = t
		grp.data.outputTokens = []*verifierToken{t}
		s.tokens = append(s.tokens, t)
		// Merge `t` with all inputs transitively. However, do not merge the inputs among each other.
		for _, inp1 := range grp.Input {
			// println("    input", inp1.Name)
			for _, t1 := range inp1.data.outputTokens {
				// println("     token", t1.origin.Name, t1)
				t1 = s.lookupToken(t1)
				t.addPossiblyMergedToken(t1)
				t1.addPossiblyMergedToken(t)
				for _, m := range t1.possiblyMergedTokens {
					// println("      token", m.origin.Name, m)
					m = s.lookupToken(m)
					t.addPossiblyMergedToken(m)
					m.addPossiblyMergedToken(t)
				}
			}
		}
	case StaticMergeGrouping,
		DynamicMergeGrouping:
		var merges []*verifierToken
		// println("MERGE", len(grp.Input), s, "into", grp.Name)
		grp.data.outputTokens = grp.Input[0].data.outputTokens
		// All tokens of all inputs must merge with all tokens of all inputs
		for _, inp1 := range grp.Input {
			inp1 = inp1.Original
			// println("    input1", inp1.Name, inp1)
			for _, t1 := range inp1.data.outputTokens {
				// println("     token1 ", t1.origin.Name, t1)
				for _, inp2 := range grp.Input {
					inp2 = inp2.Original
					if inp1 == inp2 {
						continue
					}
					// println("      input2", inp2.Name, inp2)
					for _, t2 := range inp2.data.outputTokens {
						// println("       token2 ", t2.origin.Name)
						if t1.original == t2.original {
							continue
						}
						// println("       verify", t1.origin.Name, t2.origin.Name)
						needsMerge, err := ver.verifyTokenMergeTransitive(c, grp, t1, t2, s)
						if err != nil {
							return
						}
						if needsMerge {
							merges = append(merges, t1, t2)
						}
					}
				}
			}
		}
		// println(".... merge it")
		for i := 0; i < len(merges); i += 2 {
			ver.mergeTokensTransitive(c, merges[i], merges[i+1], s)
		}
		// println("<<< MERGE")
	default:
		// println(grp.Kind)
		panic("Oooops")
	}
}

func (ver *groupingVerifier) verifyTokenMergeTransitive(c *ircode.Command, grp *Grouping, t1 *verifierToken, t2 *verifierToken, s *verifierScope) (bool, *errlog.Error) {
	t1 = s.lookupToken(t1)
	t2 = s.lookupToken(t2)
	if needsMerge, err := ver.verifyTokenMerge(c, grp, t1, t2); !needsMerge || err != nil {
		return false, err
	}
	for _, m1 := range t1.possiblyMergedTokens {
		m1 = s.lookupToken(m1)
		if _, err := ver.verifyTokenMerge(c, grp, m1, t2); err != nil {
			return true, err
		}
		for _, m2 := range t2.possiblyMergedTokens {
			m2 = s.lookupToken(m2)
			if _, err := ver.verifyTokenMerge(c, grp, m1, m2); err != nil {
				return true, err
			}
		}
	}
	for _, m2 := range t2.possiblyMergedTokens {
		m2 = s.lookupToken(m2)
		if _, err := ver.verifyTokenMerge(c, grp, m2, t1); err != nil {
			return true, err
		}
	}
	return true, nil
}

func (ver *groupingVerifier) verifyTokenMerge(c *ircode.Command, grp *Grouping, t1 *verifierToken, t2 *verifierToken) (needsMerge bool, err *errlog.Error) {
	// We test `hasPossiblyMerged` twice, because a branch_phi token merges its input, but the input does not merge the branch_phi token.
	if t1 == t2 || t1.isEquivalent(t2) {
		return false, nil
	}
	constraint := mergeGroupingConstraint(t1.origin.Constraint, t2.origin.Constraint, nil, nil, nil)
	if constraint.Error {
		var locs []errlog.LocationRange
		for _, g2 := range ver.groupingMerges(grp) {
			locs = append(locs, g2.Location)
		}
		// Raise an error
		err := ver.log.AddGroupingError(constraintToErrorString(t1.origin.Constraint), grp.Location, constraintToErrorString(t2.origin.Constraint), locs...)
		return false, err
		// ver.log.AddError(errlog.ErrorGroupingConstraints, c.Location)
		// panic("RAISE an error " + t1.origin.Name + " " + t2.origin.Name)
	}
	return true, nil
}

func (ver *groupingVerifier) mergeTokensTransitive(c *ircode.Command, t1 *verifierToken, t2 *verifierToken, s *verifierScope) {
	t1 = s.lookupToken(t1)
	t2 = s.lookupToken(t2)
	ver.mergeTokens(c, t1, t2)
	for _, m1 := range t1.possiblyMergedTokens {
		m1 = s.lookupToken(m1)
		ver.mergeTokens(c, m1, t2)
		for _, m2 := range t2.possiblyMergedTokens {
			m2 = s.lookupToken(m2)
			ver.mergeTokens(c, m1, m2)
		}
	}
	for _, m2 := range t2.possiblyMergedTokens {
		m2 = s.lookupToken(m2)
		ver.mergeTokens(c, m2, t1)
	}
}

func (ver *groupingVerifier) mergeTokens(c *ircode.Command, t1 *verifierToken, t2 *verifierToken) {
	if t1 == t2 {
		return
	}
	t1.addPossiblyMergedToken(t2)
	t2.addPossiblyMergedToken(t1)
}

func (ver *groupingVerifier) verifyArguments(c *ircode.Command) {
	for i := len(c.Args) - 1; i >= 0; i-- {
		if c.Args[i].Cmd != nil {
			panic("Oooops")
		} else if c.Args[i].Var != nil {
			grp, ok := c.Args[i].Var.Grouping.(*Grouping)
			if ok && grp != nil {
				if !ver.verifyGroupArg(grp, c) {
					return
				}
			}
		}
	}
	for i := 0; i < len(c.Dest); i++ {
		if c.Dest[i] == nil || c.Dest[i].Grouping == nil {
			continue
		}
		grp, ok := c.Dest[i].Grouping.(*Grouping)
		if ok && grp != nil {
			if !ver.verifyGroupArg(grp, c) {
				return
			}
		}
	}
}

func (ver *groupingVerifier) verifyGroupArgs(c *ircode.Command) {
	for _, garg := range c.GroupArgs {
		if !ver.verifyGroupArg(garg.(*Grouping), c) {
			return
		}
	}
}

// Checks whether the grouping `grp` can we be used as argument to command `c`.
func (ver *groupingVerifier) verifyGroupArg(grp *Grouping, c *ircode.Command) bool {
	for _, t := range grp.data.outputTokens {
		if t.origin.Constraint.Scope != nil && t.origin.Constraint.Scope.Marker == 1 {
			ver.s.log.AddError(errlog.ErrorGroupingOutOfScope, c.Location)
			return false
		}
		for _, m := range t.possiblyMergedTokens {
			if m.origin.Constraint.Scope != nil && m.origin.Constraint.Scope.Marker == 1 {
				ver.s.log.AddError(errlog.ErrorGroupingOutOfScope, c.Location)
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
	} else if c.Error {
		return "illegal group `" + c.Display + "`"
	} else if c.OverConstrained {
		return "overconstrained group `" + c.Display + "`"
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
