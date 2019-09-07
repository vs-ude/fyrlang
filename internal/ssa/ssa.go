package ssa

import (
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
)

type ssaTransformer struct {
	f     *ircode.Function
	stack []*ssaVariableInfoScope
	log   *errlog.ErrorLog
}

type ssaVariableInfo struct {
	v *ircode.Variable
	// At analysis time, a variable can be determined to be equivalent to a constant.
	// Thic constant is stored here.
	value       *ircode.Constant
	initialized bool
}

type ssaVariableInfoScope struct {
	vars         map[*ircode.Variable]ssaVariableInfo
	loopBreak    bool
	loopContinue bool
	targetCount  int
}

func newVariableInfoScope() *ssaVariableInfoScope {
	return &ssaVariableInfoScope{vars: make(map[*ircode.Variable]ssaVariableInfo)}
}

func (s *ssaTransformer) transformBlock(c *ircode.Command, depth int) bool {
	if c.Op != ircode.OpBlock && c.Op != ircode.OpIf && c.Op != ircode.OpLoop {
		panic("Not a block")
	}
	for i, c2 := range c.Block {
		if !s.transformCommand(c2, depth) {
			if i+1 < len(c.Block) {
				s.log.AddError(errlog.ErrorUnreachable, c.Block[i+1].Location)
			}
			return false
		}
	}
	return true
}

func (s *ssaTransformer) transformCommand(c *ircode.Command, depth int) bool {
	switch c.Op {
	case ircode.OpBlock:
		return s.transformBlock(c, depth)
	case ircode.OpIf:
		// Transform the condition
		s.transformArguments(c, depth)
		// Transform the if-clause
		m := newVariableInfoScope()
		s.stack = append(s.stack, m)
		ifCompletes := s.transformBlock(c, depth+1)
		s.stack = s.stack[0 : len(s.stack)-1]
		// Transform the else-clause
		if c.Else != nil {
			m2 := newVariableInfoScope()
			s.stack = append(s.stack, m2)
			elseCompletes := s.transformCommand(c.Else, depth+1)
			s.stack = s.stack[0 : len(s.stack)-1]
			if ifCompletes && elseCompletes {
				s.mergeAlternativeScopes(m, m2)
			} else if ifCompletes {
				s.mergeOptionalScope(m)
			} else if elseCompletes {
				s.mergeOptionalScope(m2)
			}
			return ifCompletes || elseCompletes
		} else if ifCompletes {
			s.mergeOptionalScope(m)
		}
	case ircode.OpLoop:
		breakScope := newVariableInfoScope()
		breakScope.loopBreak = true
		s.stack = append(s.stack, breakScope)
		// Variables set inside the loop body are added to this scope
		continueScope := newVariableInfoScope()
		continueScope.loopContinue = true
		s.stack = append(s.stack, continueScope)
		m := newVariableInfoScope()
		s.stack = append(s.stack, m)
		doesLoop := s.transformBlock(c, depth+3)
		s.stack = s.stack[0 : len(s.stack)-3]
		if doesLoop || continueScope.targetCount > 0 {
			s.mergeContinueScope(continueScope, m)
		}
		s.mergeBreakScope(s.stack[len(s.stack)-1], breakScope)
		if breakScope.targetCount == 0 {
			return false
		}
		return true
	case ircode.OpBreak:
		loopDepth := int(c.Args[0].Const.ExprType.IntegerValue.Uint64()) + 1
		var i int
		for i = len(s.stack) - 1; i >= 0; i-- {
			if s.stack[i].loopBreak {
				loopDepth--
				if loopDepth == 0 {
					break
				}
			}
		}
		if loopDepth != 0 {
			panic("Could not find matching loop")
		}
		s.stack[i].targetCount++
		s.mergeJump(s.stack[i], s.stack[:i+1], s.stack[i+2:])
		return false
	case ircode.OpContinue:
		loopDepth := int(c.Args[0].Const.ExprType.IntegerValue.Uint64()) + 1
		var i int
		for i = len(s.stack) - 1; i >= 0; i-- {
			if s.stack[i].loopContinue {
				loopDepth--
				if loopDepth == 0 {
					break
				}
			}
		}
		if loopDepth != 0 {
			panic("Could not find matching continue")
		}
		s.stack[i].targetCount++
		s.mergeJump(s.stack[i], s.stack[:i+1], s.stack[i+1:])
		return false
	case ircode.OpDefVariable:
		s.stack[depth].vars[c.Dest[0].Var] = ssaVariableInfo{v: c.Dest[0].Var}
	case ircode.OpPrintln:
		s.transformArguments(c, depth)
	case ircode.OpAdd,
		ircode.OpSetVariable,
		ircode.OpSet,
		ircode.OpGet,
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

		s.transformArguments(c, depth)
		var v = c.Dest[0].Var
		if s.variableIsLive(v) != -1 {
			v = s.newVariableVersion(v)
		}
		vinfo := ssaVariableInfo{v: v, initialized: true}
		if c.Op == ircode.OpSetVariable && c.Args[0].Const != nil {
			vinfo.value = c.Args[0].Const
		} else {
			vinfo.value = nil
		}
		c.Dest[0].Var = v
		s.setVariableInfo(s.stack[len(s.stack)-1], vinfo)
	default:
		panic("TODO")
	}
	return true
}

func (s *ssaTransformer) transformArguments(c *ircode.Command, depth int) {
	// Evaluate arguments right to left
	for i := len(c.Args) - 1; i >= 0; i-- {
		if c.Args[i].Cmd != nil {
			s.transformCommand(c.Args[i].Cmd, depth)
		} else if c.Args[i].Var.Var != nil {
			vinfo, _ := s.lookupVariableInfo(c.Args[i].Var.Var)
			// Do not replace the first argument to OpSet with a constant
			if vinfo.value != nil && !(c.Op == ircode.OpSet && i == 0) {
				c.Args[i].Const = vinfo.value
				c.Args[i].Var.Var = nil
			} else {
				c.Args[i].Var.Var = vinfo.v
			}
		}
	}
}

func (s *ssaTransformer) searchVariableInfo(v *ircode.Variable, search []*ssaVariableInfoScope) (vinfo ssaVariableInfo, ok bool) {
	for i := len(search) - 1; i >= 0; i-- {
		if search[i].loopBreak {
			continue
		}
		if vinfo, ok = search[i].vars[v]; ok {
			return
		}
	}
	ok = false
	return
}

// lookupVariableInfo creates a phi-function where perhaps necessary.
func (s *ssaTransformer) lookupVariableInfo(v *ircode.Variable) (vinfo ssaVariableInfo, depth int) {
	loops := false
	for depth = len(s.stack) - 1; depth >= 0; depth-- {
		if s.stack[depth].loopBreak {
			continue
		}
		loops = loops || s.stack[depth].loopContinue
		var ok bool
		if vinfo, ok = s.stack[depth].vars[v]; ok {
			break
		}
	}
	if depth < 0 {
		panic("Unknown variable during lookup: " + v.Name)
	}
	// Create (a placeholder) Phi-function
	if loops {
		for i := depth + 1; i < len(s.stack); i++ {
			if s.stack[i].loopContinue {
				v2 := s.newVariableVersion(vinfo.v)
				v2.Phi = []*ircode.Variable{vinfo.v}
				vinfo.v = v2
				s.stack[i].vars[v] = vinfo
			}
		}
	}
	return
}

func (s *ssaTransformer) variableIsLive(v *ircode.Variable) int {
	for i := len(s.stack) - 1; i >= 0; i-- {
		if _, ok := s.stack[i].vars[v.Original]; ok {
			return i
		}
	}
	return -1
}

func (s *ssaTransformer) setVariableInfo(dest *ssaVariableInfoScope, vinfo ssaVariableInfo) {
	dest.vars[vinfo.v.Original] = vinfo
}

func (s *ssaTransformer) newVariableVersion(v *ircode.Variable) *ircode.Variable {
	v.Original.VersionCount++
	return &ircode.Variable{Name: v.Original.Name + "." + strconv.Itoa(v.Original.VersionCount), Type: v.Type, Original: v.Original, Scope: v.Original.Scope}
}

// @param scope must not to be on the scope-stack.
func (s *ssaTransformer) mergeOptionalScope(scope *ssaVariableInfoScope) {
	for v, vinfo := range scope.vars {
		vinfo2, ok := s.searchVariableInfo(v, s.stack)
		if ok {
			s.mergeSingleOptional(s.stack[len(s.stack)-1], vinfo, vinfo2)
		}
	}
}

// @param a1 must not to be on the scope-stack.
// @param a2 must not to be on the scope-stack.
func (s *ssaTransformer) mergeAlternativeScopes(a1 *ssaVariableInfoScope, a2 *ssaVariableInfoScope) {
	for v, vinfo1 := range a1.vars {
		if vinfo2, ok := a2.vars[v]; ok {
			vinfo3 := ssaVariableInfo{}
			vinfo3.v = s.newVariableVersion(v)
			vinfo3.v.Phi = []*ircode.Variable{vinfo1.v, vinfo2.v}
			s.setVariableInfo(s.stack[len(s.stack)-1], vinfo3)
		} else {
			vinfo2, ok := s.searchVariableInfo(v, s.stack)
			if ok {
				s.mergeSingleOptional(s.stack[len(s.stack)-1], vinfo1, vinfo2)
			}
		}
	}
	for v, vinfo2 := range a2.vars {
		if _, ok := a1.vars[v]; !ok {
			vinfo3, ok := s.searchVariableInfo(v, s.stack)
			if ok {
				s.mergeSingleOptional(s.stack[len(s.stack)-1], vinfo2, vinfo3)
			}
		}
	}
}

func (s *ssaTransformer) mergeSingleOptional(dest *ssaVariableInfoScope, vinfo ssaVariableInfo, vinfo2 ssaVariableInfo) {
	/*	depth := s.variableIsLive(vinfo.v.Original)
		if depth == -1 {
			return
		}
		println("MERGING", vinfo.v.Original.Name, vinfo.v.Name)
		vinfo2 := s.stack[depth].vars[vinfo.v.Original]*/
	var phi []*ircode.Variable
	if vinfo2.v.Phi == nil {
		if vinfo.v.Phi == nil {
			phi = s.mergePhi([]*ircode.Variable{vinfo2.v}, []*ircode.Variable{vinfo.v})
		} else {
			phi = s.mergePhi([]*ircode.Variable{vinfo2.v}, vinfo.v.Phi)
		}
	} else {
		if vinfo.v.Phi == nil {
			phi = s.mergePhi(vinfo2.v.Phi, []*ircode.Variable{vinfo.v})
		} else {
			phi = s.mergePhi(vinfo2.v.Phi, vinfo.v.Phi)
		}
	}
	vinfo3 := ssaVariableInfo{v: s.newVariableVersion(vinfo.v)}
	vinfo3.v.Phi = phi
	s.setVariableInfo(dest, vinfo3)
}

func (s *ssaTransformer) mergeContinueScope(continueScope *ssaVariableInfoScope, loopBodyScope *ssaVariableInfoScope) {
	for v, vinfo := range loopBodyScope.vars {
		if phiInfo, ok := continueScope.vars[v]; ok {
			phiInfo.v.Phi = append(phiInfo.v.Phi, vinfo.v)
		}
	}
}

func (s *ssaTransformer) mergeBreakScope(dest *ssaVariableInfoScope, breakScope *ssaVariableInfoScope) {
	for v, vinfo := range breakScope.vars {
		dest.vars[v] = vinfo
	}
}

func (s *ssaTransformer) mergePhi(phi1 []*ircode.Variable, phi2 []*ircode.Variable) []*ircode.Variable {
	phi := make([]*ircode.Variable, len(phi1), len(phi1)+len(phi2))
	copy(phi[0:len(phi1)], phi1)
	// Add all variables from phi2, unless they are already in phi1
	for _, v2 := range phi2 {
		add := true
		for _, v := range phi1 {
			if v == v2 {
				add = false
				break
			}
		}
		if add {
			phi = append(phi, v2)
		}
	}
	return phi
}

// Merges all variables in `scopes` into the `dest` scope.
// Only variables defined in `dest` or `search` are considered.
func (s *ssaTransformer) mergeJump(dest *ssaVariableInfoScope, search []*ssaVariableInfoScope, scopes []*ssaVariableInfoScope) {
	done := make(map[*ircode.Variable]bool)
	for i := len(scopes) - 1; i >= 0; i-- {
		for v, vinfo := range scopes[i].vars {
			if _, ok := done[v]; ok {
				continue
			}
			if vinfo2, ok := dest.vars[v]; ok {
				if vinfo2.v.Phi != nil {
					if vinfo.v.Phi != nil {
						vinfo2.v.Phi = s.mergePhi(vinfo2.v.Phi, vinfo.v.Phi)
					} else {
						vinfo2.v.Phi = s.mergePhi(vinfo2.v.Phi, []*ircode.Variable{vinfo.v})
					}
				} else {
					s.mergeSingleOptional(dest, vinfo, vinfo2)
				}
			} else if _, ok := s.searchVariableInfo(v, search); ok {
				s.setVariableInfo(dest, vinfo)
			}
			done[v] = true
		}
	}
}

// TransformToSSA ...
// The transformer checks the control flow and detects unreachable code.
func TransformToSSA(f *ircode.Function, log *errlog.ErrorLog) {
	s := &ssaTransformer{f: f, log: log}
	m := newVariableInfoScope()
	// Mark all parameters as initialized
	for _, v := range f.Vars {
		if v.Kind == ircode.VarParameter {
			m.vars[v] = ssaVariableInfo{v: v, initialized: true}
		}
	}
	s.stack = append(s.stack, m)
	s.transformBlock(&f.Body, 0)
	s.stack = s.stack[0 : len(s.stack)-1]
}
