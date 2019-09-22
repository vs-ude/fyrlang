package ssa

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
)

// GroupResult ...
type GroupResult struct {
	Scope           *ircode.CommandScope
	NamedGroup      string
	OverConstrained bool
	Error           bool
}

func (s *ssaTransformer) computeGroupBlock(c *ircode.Command) {
	if c.Op != ircode.OpBlock && c.Op != ircode.OpIf && c.Op != ircode.OpLoop {
		panic("Not a block")
	}
	for _, c2 := range c.Block {
		s.computeGroupCommand(c2)
	}
}

func (s *ssaTransformer) computeGroupCommand(c *ircode.Command) {
	switch c.Op {
	case ircode.OpBlock:
		s.computeGroupBlock(c)
	case ircode.OpIf:
		// computeGroup the condition
		s.computeGroupArguments(c)
		// computeGroup the if-clause
		s.computeGroupBlock(c)
		// computeGroup the else-clause
		if c.Else != nil {
			s.computeGroupBlock(c.Else)
		}
	case ircode.OpLoop:
		s.computeGroupBlock(c)
	case ircode.OpBreak, ircode.OpContinue, ircode.OpDefVariable:
		// Do nothing by intention
	case ircode.OpSetVariable, ircode.OpSet, ircode.OpGet, ircode.OpStruct, ircode.OpArray:
		s.computeGroupArguments(c)
		s.computeGroupDests(c)
		s.computeGammas(c)
	case ircode.OpAdd,
		ircode.OpPrintln,
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
		ircode.OpBitwiseComplement:

		// Do nothing by intention
	default:
		panic("Ooop")
	}
}

func (s *ssaTransformer) computeGroupArguments(c *ircode.Command) {
	// Evaluate arguments right to left
	for i := len(c.Args) - 1; i >= 0; i-- {
		if c.Args[i].Cmd != nil {
			panic("Ooops, should not happen")
		}
		if c.Args[i].Var != nil {
			// println("COMPUTE ARG", c.Args[i].Var.ToString())
			s.computeVariable(c.Args[i].Var, c)
		}
	}
}

func (s *ssaTransformer) computeGroupDests(c *ircode.Command) {
	for i := range c.Dest {
		// println("COMPUTE DEST GROUP", c.Dest[i].ToString())
		s.computeVariable(c.Dest[i], c)
	}
}

func (s *ssaTransformer) computeGammas(c *ircode.Command) {
	for _, gv := range c.Gammas {
		// println("COMPUTE CMD GAMMA", gv.ToString())
		result, _ := s.computeGroupVariable(gv, nil, c)
		gv.ComputedGroup = result
		gv.HasComputedGroup = true
	}
}

func (s *ssaTransformer) computeVariable(v *ircode.Variable, c *ircode.Command) {
	if v.GroupVariable == nil {
		return
	}
	result, _ := s.computeGroupVariable(v.GroupVariable, v, c)
	v.GroupVariable.ComputedGroup = result
	v.GroupVariable.HasComputedGroup = true
}

// `v` and `c` are only used for generating meaningful error messages.
// `v` is nil if the group cannot be matched to a variable.
func (s *ssaTransformer) computeGroupVariable(gv *ircode.Variable, v *ircode.Variable, c *ircode.Command) (result GroupResult, hasLoops bool) {
	if gv.Marked {
		// println("MARKED", gv.ToString())
		hasLoops = true
		return
	}
	if gv.HasComputedGroup {
		return gv.ComputedGroup.(GroupResult), false
	}
	// println("Marking", gv.ToString())
	gv.Marked = true
	if len(gv.Phi) != 0 {
		str := "COMPUTE PHI " + gv.ToString() + " = phi("
		for _, p := range gv.Phi {
			str += ", " + p.ToString()
		}
		str += ")"
		println(str)
		for i, p := range gv.Phi {
			r, l := s.computeGroupVariable(p, v, c)
			hasLoops = hasLoops || l
			if i == 0 {
				result = r
			} else {
				result = s.computePhi(result, r, v, c)
			}
		}
	} else if len(gv.Gamma) != 0 {
		str := "COMPUTE GAMMA " + gv.ToString() + " = gamma("
		for _, p := range gv.Gamma {
			str += ", " + p.ToString()
		}
		str += ")"
		println(str)
		for i, p := range gv.Gamma {
			r, l := s.computeGroupVariable(p, v, c)
			hasLoops = hasLoops || l
			if i == 0 {
				result = r
			} else {
				result = s.computeGamma(result, r, v, c)
			}
		}
	} else {
		switch gv.Kind {
		case ircode.VarGroup:
			// Do nothin by intention
		case ircode.VarNamedGroup:
			result.NamedGroup = gv.Name[3:]
		case ircode.VarScopeGroup:
			result.Scope = gv.Scope
		case ircode.VarIsolatedGroup:
			// Do nothin by intention
		}
	}
	if !hasLoops {
		gv.ComputedGroup = result
		gv.HasComputedGroup = true
	}
	gv.Marked = false
	return
}

func (s *ssaTransformer) computePhi(a, b GroupResult, context *ircode.Variable, c *ircode.Command) (result GroupResult) {
	if a.Error || b.Error {
		result.Error = true
		return
	}
	result = s.computeGamma(a, b, context, c)
	if result.Error {
		result.Error = false
		result.OverConstrained = true
	}
	return
}

func (s *ssaTransformer) computeGamma(a, b GroupResult, v *ircode.Variable, c *ircode.Command) (result GroupResult) {
	if a.Error || b.Error {
		result.Error = true
		return
	}
	var errorKind string
	var errorArgs []string
	if a.OverConstrained || b.OverConstrained {
		result.OverConstrained = true
	}
	if a.OverConstrained && (b.NamedGroup != "" || b.Scope != nil || b.OverConstrained) {
		errorKind = "overconstrained"
		result.Error = true
	} else if b.OverConstrained && (a.NamedGroup != "" || a.Scope != nil || a.OverConstrained) {
		errorKind = "overconstrained"
		result.Error = true
	}
	if a.Scope != nil && b.Scope != nil && a == b {
		result.Scope = a.Scope
	} else if a.Scope != nil && b.Scope != nil && a.Scope.HasParent(b.Scope) {
		result.Scope = a.Scope
	} else if a.Scope != nil && b.Scope != nil && b.Scope.HasParent(a.Scope) {
		result.Scope = b.Scope
	} else if a.Scope != nil && b.Scope == nil {
		result.Scope = a.Scope
	} else if a.Scope == nil && b.Scope != nil {
		result.Scope = b.Scope
	} else if a.Scope == nil && b.Scope == nil {
		result.Scope = nil
	} else {
		errorKind = "both_scoped"
		result.Error = true
	}
	if a.NamedGroup != "" && b.NamedGroup != "" && a.NamedGroup == b.NamedGroup {
		result.NamedGroup = a.NamedGroup
	} else if a.NamedGroup != "" && b.NamedGroup == "" {
		result.NamedGroup = a.NamedGroup
	} else if a.NamedGroup == "" && b.NamedGroup != "" {
		// println("B IS NAMED")
		result.NamedGroup = b.NamedGroup
	} else if a.NamedGroup == "" && b.NamedGroup == "" {
		result.NamedGroup = ""
	} else {
		errorKind = "both_named"
		errorArgs = []string{a.NamedGroup, b.NamedGroup}
		result.Error = true
	}
	if result.NamedGroup != "" && result.Scope != nil {
		errorKind = "scoped_and_named"
		errorArgs = []string{result.NamedGroup}
		result.Error = true
	}
	if result.Error {
		// println("GROUP ERROR for", v.ToString())
		if v != nil && v.Kind != ircode.VarTemporary {
			errorArgs = append(errorArgs, v.Original.Name)
		}
		errorArgs = append([]string{errorKind}, errorArgs...)
		s.log.AddError(errlog.ErrorGroupsCannotBeMerged, c.Location, errorArgs...)
	}
	return
}
