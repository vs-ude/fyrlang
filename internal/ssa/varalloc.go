package ssa

import (
	"sort"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
)

/*
type EquivalenceClassKind int

const (
	ECNormal EquivalenceClassKind = 0
	ECPhi = 1
	ECNamed = 2
	ECScoped = 3
)
*/

type visitorScope struct {
	parent  *visitorScope
	groups  map[*ircode.Variable]*EquivalenceClass
	classes map[*EquivalenceClass]bool
}

// EquivalenceClass ...
type EquivalenceClass struct {
	GroupVariables map[*ircode.Variable]bool
	// Scope          *ircode.CommandScope
	Closed       bool
	IsPhiClass   bool
	Constraints  GroupResult
	uniqueString string
	groupNames   []string
}

func newVisitorScope() *visitorScope {
	return &visitorScope{groups: make(map[*ircode.Variable]*EquivalenceClass), classes: make(map[*EquivalenceClass]bool)}
}

func (vs *visitorScope) lookup(gv *ircode.Variable) *EquivalenceClass {
	for p := vs; p != nil; p = p.parent {
		if ec, ok := p.groups[gv]; ok {
			return ec
		}
	}
	return nil
}

func newEquivalenceClass() *EquivalenceClass {
	return &EquivalenceClass{GroupVariables: make(map[*ircode.Variable]bool)}
}

// UniqueString ...
func (ec *EquivalenceClass) UniqueString() string {
	if ec.uniqueString != "" {
		return ec.uniqueString
	}
	sort.Strings(ec.groupNames)
	for _, str := range ec.groupNames {
		ec.uniqueString += str + ","
	}
	return ec.uniqueString
}

// JoinGroup ...
func (ec *EquivalenceClass) JoinGroup(gv *ircode.Variable, v *ircode.Variable, c *ircode.Command, log *errlog.ErrorLog) {
	if ec.Closed {
		panic("Oooops")
	}
	if _, ok := ec.GroupVariables[gv]; ok {
		return
	}
	ec.Constraints = computeGammaGroupResult(ec.Constraints, computeGroupResult(gv), v, c, log)
	ec.uniqueString = ""
	ec.GroupVariables[gv] = true
	ec.groupNames = append(ec.groupNames, gv.ToString())
}

// JoinClass ...
func (ec *EquivalenceClass) JoinClass(ec2 *EquivalenceClass, v *ircode.Variable, c *ircode.Command, log *errlog.ErrorLog) {
	if ec.Closed {
		panic("Oooops")
	}
	if ec2.Closed {
		panic("Oooops")
	}
	ec.uniqueString = ""
	ec.Constraints = computeGammaGroupResult(ec.Constraints, ec2.Constraints, v, c, log)
	for gv := range ec.GroupVariables {
		ec.GroupVariables[gv] = true
		ec.groupNames = append(ec.groupNames, gv.ToString())
	}
}

// Close ...
func (ec *EquivalenceClass) Close() {
	ec.Closed = true
}

func (s *ssaTransformer) visitBlock(c *ircode.Command, vs *visitorScope) {
	if c.Op != ircode.OpBlock && c.Op != ircode.OpIf && c.Op != ircode.OpLoop {
		panic("Not a block")
	}
	for _, c2 := range c.Block {
		s.visitCommand(c2, vs)
	}
}

func (s *ssaTransformer) visitCommand(c *ircode.Command, vs *visitorScope) {
	switch c.Op {
	case ircode.OpBlock:
		s.visitBlock(c, vs)
	case ircode.OpIf:
		// visit the condition
		s.visitArguments(c, vs)
		// visit the if-clause
		s.visitBlock(c, vs)
		// visit the else-clause
		if c.Else != nil {
			s.visitBlock(c.Else, vs)
		}
	case ircode.OpLoop:
		s.visitBlock(c, vs)
		s.visitGammas(c, vs)
	case ircode.OpBreak, ircode.OpContinue, ircode.OpDefVariable:
		// Do nothing by intention
	case ircode.OpSetVariable, ircode.OpSet, ircode.OpGet, ircode.OpStruct, ircode.OpArray:
		s.visitArguments(c, vs)
		s.visitDests(c, vs)
		s.visitGammas(c, vs)
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

func (s *ssaTransformer) visitArguments(c *ircode.Command, vs *visitorScope) {
	// Evaluate arguments right to left
	for i := len(c.Args) - 1; i >= 0; i-- {
		if c.Args[i].Cmd != nil {
			panic("Ooops, should not happen")
		}
		if c.Args[i].Var != nil {
			// println("COMPUTE ARG", c.Args[i].Var.ToString())
			s.visitVariable(c.Args[i].Var, c, vs)
		}
	}
}

func (s *ssaTransformer) visitDests(c *ircode.Command, vs *visitorScope) {
	for i := range c.Dest {
		// println("COMPUTE DEST GROUP", c.Dest[i].ToString())
		s.visitVariable(c.Dest[i], c, vs)
	}
}

func (s *ssaTransformer) visitGammas(c *ircode.Command, vs *visitorScope) {
	for _, gv := range c.Gammas {
		// println("COMPUTE CMD GAMMA", gv.ToString())
		s.visitGroupVariable(gv, nil, c, vs)
	}
}

func (s *ssaTransformer) visitVariable(v *ircode.Variable, c *ircode.Command, vs *visitorScope) {
	if v.GroupVariable == nil {
		return
	}
	s.visitGroupVariable(v.GroupVariable, v, c, vs)
}

func (s *ssaTransformer) visitGroupVariable(gv *ircode.Variable, v *ircode.Variable, c *ircode.Command, vs *visitorScope) *EquivalenceClass {
	// The group-variable has already been processed?
	if ec := vs.lookup(gv); ec != nil {
		return ec
	}
	if len(gv.Gamma) != 0 {
		ec := newEquivalenceClass()
		ec.JoinGroup(gv, v, c, s.log)
		vs.groups[gv] = ec
		vs.classes[ec] = true
		for _, gv2 := range gv.Gamma {
			ec2 := s.visitGroupVariable(gv2, v, c, vs)
			if ec2.IsPhiClass {
				continue
			}
			vs.classes[ec2] = false
			ec.JoinClass(ec2, v, c, s.log)
			for gv3 := range ec2.GroupVariables {
				vs.groups[gv3] = ec
			}
		}
		return ec
	} else if len(gv.Phi) == 1 {
		ec := newEquivalenceClass()
		ec.JoinGroup(gv, v, c, s.log)
		vs.groups[gv] = ec
		vs.classes[ec] = true
		gv2 := gv.Phi[0]
		ec2 := s.visitGroupVariable(gv2, v, c, vs)
		if !ec2.IsPhiClass {
			vs.classes[ec2] = false
			ec.JoinClass(ec2, v, c, s.log)
			for gv3 := range ec2.GroupVariables {
				vs.groups[gv3] = ec
			}
		}
		return ec
	} else if len(gv.Phi) > 1 {
		ec := newEquivalenceClass()
		ec.JoinGroup(gv, v, c, s.log)
		ec.IsPhiClass = true
		vs.groups[gv] = ec
		vs.classes[ec] = true
		for _, gv2 := range gv.Gamma {
			ec2 := s.visitGroupVariable(gv2, v, c, vs)
			if !ec2.IsPhiClass {
				continue
			}
			vs.classes[ec2] = false
			ec.JoinClass(ec2, v, c, s.log)
			for gv3 := range ec2.GroupVariables {
				vs.groups[gv3] = ec
			}
		}
		return ec
	}
	ec := newEquivalenceClass()
	ec.JoinGroup(gv, v, c, s.log)
	vs.groups[gv] = ec
	vs.classes[ec] = true
	return ec
}
