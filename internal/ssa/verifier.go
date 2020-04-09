package ssa

import "github.com/vs-ude/fyrlang/internal/errlog"

type verifierToken struct {
	constraint GroupingConstraint
	// The constrained grouping that gave rise to the token.
	// This is used to generate meaningful error messages.
	origin        *Grouping
	visitedScopes []*ssaScope
}

type verifierData struct {
	tokens []*verifierToken
}

type groupingVerifier struct {
	s             *ssaTransformer
	log           *errlog.ErrorLog
	markerCounter int
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
		println("DOUBLE token", t.constraint.NamedGroup, t2.constraint.NamedGroup)
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
	return &groupingVerifier{s: s, log: log}
}

func (ver *groupingVerifier) verify() {
	for _, grp := range ver.s.parameterGroupings {
		ver.floodConstraint(grp)
	}
	for _, grp := range ver.s.scopedGroupings {
		// Shortcut, because this is true most of the time
		if len(grp.Input) == 0 && len(grp.Output) == 0 {
			continue
		}
		ver.floodConstraint(grp)
	}

	for _, grp := range ver.s.parameterGroupings {
		ver.check(grp)
	}
	for _, grp := range ver.s.scopedGroupings {
		// Shortcut, because this is true most of the time
		if len(grp.Input) == 0 && len(grp.Output) == 0 {
			continue
		}
		ver.check(grp)
	}
}

func (ver *groupingVerifier) floodConstraint(grp *Grouping) {
	println("flood constraint from", grp.GroupingName())
	t := &verifierToken{origin: grp, constraint: grp.Constraint}
	grp.data.tokens = append(grp.data.tokens, t)
	ver.floodToken(grp, t)
}

func (ver *groupingVerifier) floodToken(grp *Grouping, t *verifierToken) {
	println("flood from", grp.GroupingName(), len(t.visitedScopes), "in", len(grp.Input), "out", len(grp.Output))
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
		println("  pass loop-phi", len(t.visitedScopes))
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
					println("    remove scope")
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
		println("  not accepted by", grp.GroupingName())
		return
	}
	for _, s := range t.visitedScopes {
		if ver.doesInhibitToken(grp.scope, s) {
			return
		}
	}
	grp.data.tokens = append(grp.data.tokens, t)
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
	println("check", grp.GroupingName(), len(grp.data.tokens), len(merged), len(open))
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
			println("ERR: Cannot merge constraints")
			// Find all merges that could contribute to this error
			var locs []errlog.LocationRange
			for _, g2 := range ver.groupingMerges(grp) {
				println("   via " + g2.Name)
				locs = append(locs, g2.Location)
			}
			// Raise an error
			ver.log.AddGroupingError(constraintToErrorString(grp.Constraint), grp.Location, constraintToErrorString(ot.constraint), locs...)
		}
	}
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
