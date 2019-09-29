package ssa

import (
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
	parent        *visitorScope
	groups        map[*ircode.Variable]*EquivalenceClass
	classes       map[*EquivalenceClass]bool
	continueCount int
	breakCount    int
	isLoop        bool
}

// EquivalenceClass ...
type EquivalenceClass struct {
	In             map[*EquivalenceClass]bool
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

func (vs *visitorScope) lookup(gv *ircode.Variable) (*visitorScope, *EquivalenceClass) {
	for p := vs; p != nil; p = p.parent {
		if ec, ok := p.groups[gv]; ok {
			return p, ec
		}
	}
	return nil, nil
}

func newEquivalenceClass() *EquivalenceClass {
	return &EquivalenceClass{GroupVariables: make(map[*ircode.Variable]bool), In: make(map[*EquivalenceClass]bool)}
}

/*
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
*/

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
	for gv := range ec2.GroupVariables {
		ec.GroupVariables[gv] = true
		ec.groupNames = append(ec.groupNames, gv.ToString())
	}
	for x, v := range ec2.In {
		ec.In[x] = v
	}
}

// Close ...
func (ec *EquivalenceClass) Close() {
	ec.Closed = true
}

// AddInput ...
func (ec *EquivalenceClass) AddInput(input *EquivalenceClass) {
	if _, ok := ec.In[input]; ok {
		return
	}
	ec.In[input] = true
}

func (s *ssaTransformer) visitBlock(c *ircode.Command, vs *visitorScope) bool {
	if c.Op != ircode.OpBlock && c.Op != ircode.OpIf && c.Op != ircode.OpLoop {
		panic("Not a block")
	}
	for i, c2 := range c.Block {
		if !s.visitCommand(c2, vs) {
			if i+1 < len(c.Block) {
				s.log.AddError(errlog.ErrorUnreachable, c.Block[i+1].Location)
			}
			return false
		}
	}
	return true
}

func (s *ssaTransformer) visitCommand(c *ircode.Command, vs *visitorScope) bool {
	switch c.Op {
	case ircode.OpBlock:
		s.visitBlock(c, vs)
	case ircode.OpIf:
		// visit the condition
		s.visitArguments(c, vs)
		// visit the if-clause
		ifScope := newVisitorScope()
		ifScope.parent = vs
		ifCompletes := s.visitBlock(c, ifScope)
		// visit the else-clause
		if c.Else != nil {
			elseScope := newVisitorScope()
			elseScope.parent = vs
			elseCompletes := s.visitBlock(c.Else, elseScope)
			if ifCompletes && elseCompletes {
				// Control flow flows through the if-clause or else-clause and continues afterwards
			} else if ifCompletes {
				// Control flow can either continue through the if-clause, or it does not reach past the end of the else-clause
			} else if elseCompletes {
				// Control flow can either continue through the else-clause, or it does not reach past the end of the if-clause
			}
			return ifCompletes || elseCompletes
		} else if ifCompletes {
			// No else, but control flow continues after the if
			return true
		}
		// No else, and control flow does not come past the if.
		// Return true, because the else case can continue the control flow
		return true
	case ircode.OpLoop:
		loopScope := newVisitorScope()
		loopScope.parent = vs
		loopScope.isLoop = true
		doesLoop := s.visitBlock(c, loopScope)
		s.visitGammas(c, vs)
		if doesLoop || loopScope.continueCount > 0 {
			// The loop can run more than once
		}
		// How many breaks are breaking exactly at this loop?
		// Breaks targeting an outer loop are not considered.
		return loopScope.breakCount > 0
	case ircode.OpBreak:
		loopDepth := int(c.Args[0].Const.ExprType.IntegerValue.Uint64()) + 1
		loopScope := vs
		for loopDepth > 0 {
			if loopScope.isLoop {
				loopDepth--
				if loopDepth == 0 {
					break
				}
			}
			loopScope = loopScope.parent
			if s == nil {
				panic("Ooooops")
			}
		}
		loopScope.breakCount++
		return false
	case ircode.OpContinue:
		loopDepth := int(c.Args[0].Const.ExprType.IntegerValue.Uint64()) + 1
		loopScope := vs
		for loopDepth > 0 {
			if loopScope.isLoop {
				loopDepth--
				if loopDepth == 0 {
					break
				}
			}
			loopScope = loopScope.parent
			if s == nil {
				panic("Ooooops")
			}
		}
		loopScope.continueCount++
		return false
	case ircode.OpDefVariable:
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
	return true
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

func (s *ssaTransformer) visitGroupVariable(gv *ircode.Variable, v *ircode.Variable, c *ircode.Command, vs *visitorScope) (*visitorScope, *EquivalenceClass) {
	// The group-variable has already been processed?
	if vs2, ec := vs.lookup(gv); ec != nil {
		return vs2, ec
	}
	if len(gv.Gamma) != 0 {
		ec := newEquivalenceClass()
		ec.JoinGroup(gv, v, c, s.log)
		vs.groups[gv] = ec
		vs.classes[ec] = true
		for _, gv2 := range gv.Gamma {
			vs2, ec2 := s.visitGroupVariable(gv2, v, c, vs)
			if ec == ec2 {
				continue
			}
			if vs2 != vs || ec2.Closed {
				ec2.Closed = true
				ec.AddInput(ec)
				continue
			}
			vs.classes[ec2] = false
			ec.JoinClass(ec2, v, c, s.log)
			for gv3 := range ec2.GroupVariables {
				vs.groups[gv3] = ec
			}
		}
		return nil, ec
	} else if len(gv.Phi) != 0 {
		ec := newEquivalenceClass()
		ec.JoinGroup(gv, v, c, s.log)
		ec.IsPhiClass = true
		vs.groups[gv] = ec
		vs.classes[ec] = true
		for _, gv2 := range gv.Phi {
			vs2, ec2 := s.visitGroupVariable(gv2, v, c, vs)
			if ec == ec2 {
				continue
			}
			if vs2 != vs || ec2.Closed {
				ec2.Closed = true
				ec.AddInput(ec)
				continue
			}
			if !ec2.IsPhiClass {
				panic("Ooooops")
			}
			vs.classes[ec2] = false
			ec.JoinClass(ec2, v, c, s.log)
		}
		return nil, ec
	}
	ec := newEquivalenceClass()
	ec.JoinGroup(gv, v, c, s.log)
	vs.groups[gv] = ec
	vs.classes[ec] = true
	return nil, ec
}
