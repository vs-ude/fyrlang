package ssa

import (
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

type ssaTransformer struct {
	f                   *ircode.Function
	stack               []*ssaScope
	log                 *errlog.ErrorLog
	namedGroupVariables map[string]*ircode.Variable
}

type ssaScope struct {
	vars         map[*ircode.Variable]*ircode.Variable
	loopBreak    bool
	loopContinue bool
	targetCount  int
}

func newVariableInfoScope() *ssaScope {
	return &ssaScope{vars: make(map[*ircode.Variable]*ircode.Variable)}
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
		// Make sure that all variables use their latest group-variables
		s.consolidateScope(s.stack[len(s.stack)-1])
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
		// Make sure that all variables use their latest group-variables
		s.consolidateScope(s.stack[len(s.stack)-1])
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
		// Remove the loop-body-scope and the continue-scope from the stack
		s.stack = s.stack[:len(s.stack)-2]
		if doesLoop || continueScope.targetCount > 0 {
			println("MERGE continue", len(continueScope.vars))
			s.mergeIntoContinueScope(continueScope, m)
			// Merge the continue scope into the break scope
			s.mergeBreakScope(s.stack[len(s.stack)-1], continueScope)
		}
		// Remove the break-scope from the stack and merge the break-scope into the remaining stack
		s.stack = s.stack[:len(s.stack)-1]
		println("BREAK SCOPE", len(breakScope.vars))
		s.mergeBreakScope(s.stack[len(s.stack)-1], breakScope)
		if breakScope.targetCount == 0 {
			return false
		}
		println("LOOP returns")
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
		// Merge variables into s.stack[i]. This is the break-scope.
		// Merge only variables which are known in s.stack[:i], which is everything outside the loop.
		// Everything else is a variable local to the loop.
		// Merge all scopes in the range stack[i+2:] excluding the continue scope.
		// Thus, all scopes inside the loop are merged into the break-scope, but only if the variables are used outside the loop, too.
		s.mergeJump(s.stack[i], s.stack[:i], s.stack[i+2:])
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
		// Merge variables in s.stack[i]. This is the continue-scope.
		// Merge only variables which are known in s.stack[:i], which is everything outside the loop.
		// Everything else is a variable local to the loop.
		// Merge all scopes in stack[i+1:]. stack[i] is the continue scope, which is skipped.
		// Thus, all scopes inside the loop are merged into the break-scope, but only if the variables are used outside the loop, too.
		s.mergeJump(s.stack[i], s.stack[:i-1], s.stack[i+1:])
		return false
	case ircode.OpDefVariable:
		s.stack[depth].vars[c.Dest[0]] = c.Dest[0]
		if c.Dest[0].Kind == ircode.VarParameter && c.Dest[0].Type.PointerDestGroup != nil {
			// Parameters with pointers have groups already when the function is being called.
			grp := c.Dest[0].Type.PointerDestGroup
			if grp.Kind == types.GroupIsolate {
				c.Dest[0].GroupVariable = s.newFreeGroupVariable(c)
			} else {
				c.Dest[0].GroupVariable = s.newNamedGroupVariable(grp.Name, c)
			}
		}
		// If the variable has pointers, it must have a group variable that maintains the memory to which these pointers lead.
		if c.Dest[0].GroupVariable == nil && types.TypeHasPointers(c.Dest[0].Type.Type) {
			// So far, the group variable is not initialized
			c.Dest[0].GroupVariable = &ircode.Variable{Kind: ircode.VarGroup, Name: "gu_" + strconv.Itoa(uniqueNameCount)}
			c.Dest[0].GroupVariable.Original = c.Dest[0].GroupVariable
			uniqueNameCount++
		}
	case ircode.OpPrintln:
		s.transformArguments(c, depth)
	case ircode.OpSet:
		s.transformArguments(c, depth)
		gDest := s.accessChainGroupVariable(c)
		gSrc := s.argumentGroupVariable(c, c.Args[len(c.Args)-1], c.Location)
		gv := s.gamma(c, gDest, gSrc)
		if len(c.Dest) == 1 {
			dest, depth := s.lookupVariable(c.Dest[0])
			if depth < 0 {
				panic("Oooops, variable does not exist")
			}
			if !ircode.IsVarInitialized(dest) {
				s.log.AddError(errlog.ErrorUninitializedVariable, c.Location, dest.Original.Name)
			}
			v := s.newVariableUsageVersion(dest)
			// Do not call setGroupVariable, because this is no change of the memory group.
			v.GroupVariable = gv
			s.setVariableInfo(s.stack[len(s.stack)-1], v)
			c.Dest[0] = v
		}
	case ircode.OpGet:
		s.transformArguments(c, depth)
		v := s.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
		if types.TypeHasPointers(v.Type.Type) {
			// The group resulting in the Get operation becomes the group of the destination
			s.setGroupVariable(v, s.accessChainGroupVariable(c))
		}
	case ircode.OpSetVariable:
		s.transformArguments(c, depth)
		v := s.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
		// If assigning a constant, change the variable's ExprType, because it carries the constant value.
		// This way the compiler knows that the current value of this variable is a constant.
		if c.Args[0].Const != nil {
			v.Type = c.Args[0].Const.ExprType
		}
		// If the type has pointers, update the group variable
		if types.TypeHasPointers(v.Type.Type) {
			// The group of the argument becomes the group of the destination
			gArg := s.argumentGroupVariable(c, c.Args[0], c.Location)
			s.setGroupVariable(v, gArg)
		}
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
		ircode.OpBitwiseComplement:

		s.transformArguments(c, depth)
		v := s.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
		// No groups to update here, because these ops work on primitive types.
	case ircode.OpArray,
		ircode.OpStruct:

		s.transformArguments(c, depth)
		v := s.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
		// If the type has pointers, update the group variable
		if types.TypeHasPointers(v.Type.Type) {
			var gv *ircode.Variable
			for i := range c.Args {
				gArg := s.argumentGroupVariable(c, c.Args[i], c.Location)
				if gArg != nil {
					if gv == nil {
						gv = gArg
					} else {
						gv = s.gamma(c, gv, gArg)
					}
				}
			}
			if gv == nil {
				gv = s.newFreeGroupVariable(c)
			}
			s.setGroupVariable(v, gv)
		}
	default:
		panic("TODO")
	}
	return true
}

func (s *ssaTransformer) createDestinationVariable(c *ircode.Command) *ircode.Variable {
	if len(c.Dest) != 1 {
		panic("Ooooops")
	}
	// Create a new version of the destination variable when required
	var v = c.Dest[0]
	if _, depth := s.lookupVariable(v); depth >= 0 {
		// If the variable has been defined or assigned so far, create a new version of it.
		v = s.newVariableVersion(v)
		c.Dest[0] = v
	}
	// Make this (version of) variable visible in the stack
	s.setVariableInfo(s.stack[len(s.stack)-1], v)
	return v
}

func (s *ssaTransformer) setGroupVariable(v *ircode.Variable, gv *ircode.Variable) {
	if v.GroupVariable != gv {
		v.HasGroupVariableChange = true
		v.GroupVariable = gv
	}
}

func (s *ssaTransformer) gamma(c *ircode.Command, gv1, gv2 *ircode.Variable) *ircode.Variable {
	if gv1 == gv2 {
		return gv1
	}

	// TODO: Check if these variables are already in a gamma relationship to reduce useless gammas
	if gv1.Scope.HasParent(gv2.Scope) {
		tmp := gv1
		gv1 = gv2
		gv2 = tmp
	}
	gv := s.newVariableVersion(gv1)
	gv.Gamma = []*ircode.Variable{gv1, gv2}
	s.stack[len(s.stack)-1].vars[gv1] = gv
	s.stack[len(s.stack)-1].vars[gv2] = gv
	s.stack[len(s.stack)-1].vars[gv] = gv
	println("GAMMA", gv.ToString(), "=", gv1.ToString(), gv2.ToString())
	c.Gammas = append(c.Gammas, gv)
	return gv
}

func (s *ssaTransformer) transformArguments(c *ircode.Command, depth int) {
	// Evaluate arguments right to left
	for i := len(c.Args) - 1; i >= 0; i-- {
		if c.Args[i].Cmd != nil {
			s.transformCommand(c.Args[i].Cmd, depth)
		} else if c.Args[i].Var != nil {
			vinfo, depth := s.lookupVariable(c.Args[i].Var)
			if depth < 0 {
				panic("Oooops, variable does not exist")
			}
			if !ircode.IsVarInitialized(vinfo) {
				s.log.AddError(errlog.ErrorUninitializedVariable, c.Location, vinfo.Original.Name)
			}
			// Do not replace the first argument to OpSet with a constant
			if vinfo.Type.IsPrimitiveConstant() && !(c.Op == ircode.OpSet && i == 0) {
				// println("SUBSTITUTE", vinfo.Name, vinfo.Type.Type.ToString())
				c.Args[i].Const = &ircode.Constant{ExprType: vinfo.Type}
				c.Args[i].Var = nil
			} else {
				c.Args[i].Var = vinfo
			}
			if c.Args[i].Var != nil {
				gv := s.lookupGroupVariable(vinfo.GroupVariable, nil)
				// If the group of the variable changed in the meantime, update the variable such that it refers to its current group
				if gv != vinfo.GroupVariable {
					v := s.newVariableUsageVersion(c.Args[i].Var)
					s.setGroupVariable(v, gv)
					//					println("NEW USAGE OF", c.Args[i].Var.ToString(), "becomes", v.ToString())
					s.setVariableInfo(s.stack[len(s.stack)-1], v)
					c.Args[i].Var = v
				}
			}
		}
	}
}

func (s *ssaTransformer) argumentGroupVariable(c *ircode.Command, arg ircode.Argument, loc errlog.LocationRange) *ircode.Variable {
	if arg.Var != nil {
		return arg.Var.GroupVariable
	}
	// If the const contains heap allocated data, attach a group variable
	if types.TypeHasPointers(arg.Const.ExprType.Type) {
		g := s.newFreeGroupVariable(c)
		s.setVariableInfo(s.stack[len(s.stack)-1], g)
		return g
	}
	return nil
}

func (s *ssaTransformer) accessChainGroupVariable(c *ircode.Command) *ircode.Variable {
	// Shortcut in case the result of the access chain carries no pointers at all.
	if !types.TypeHasPointers(c.Type.Type) {
		return nil
	}
	if len(c.AccessChain) == 0 {
		panic("No access chain")
	}
	if c.Args[0].Var == nil {
		panic("Access chain is not accessing a variable")
	}
	/*
		if c.Args[0].Var.GroupVariable == nil {
			// TODO remove debug
			loc := c.Location.From
			file := uint64(loc) >> 48
			line := int((uint64(loc) & 0xffff00000000) >> 32)
			pos := int(uint64(loc) & 0xffffffff)
			println(file, line, pos)
			panic("Access chain is accessing a variable that has no group variable, but it has pointers: " + c.Args[0].Var.ToString())
		}
	*/
	// The variable on which this access chain starts is stored as local variable in a scope.
	// Thus, the group of this value is a scoped group.
	valueGroup := c.Args[0].Var.Scope.GroupVariable
	// The variable on which this access chain starts might have pointers.
	// Determine to group to which these pointers are pointing.
	ptrDestGroup := c.Args[0].Var.GroupVariable
	if ptrDestGroup == nil {
		// The variable has no pointers. In this case the only possible operation is to take the address of take a slice.
		ptrDestGroup = valueGroup
	}
	for _, ac := range c.AccessChain {
		switch ac.Kind {
		case ircode.AccessAddressOf:
			// The result of `&expr` must be a pointer.
			pt, ok := types.GetPointerType(ac.OutputType.Type)
			if !ok {
				panic("Output is not a pointer")
			}
			// The resulting pointer is assigned to an unsafe pointer? -> give up
			if pt.Mode == types.PtrUnsafe {
				return nil
			}
			if ptrDestGroup.Kind == ircode.VarIsolatedGroup {
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value is in turn an isolate pointer. But that is ok, since the type system has this information in form of a GroupType.
				ptrDestGroup = valueGroup
			} else {
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value may contain further pointers to a group stored in `ptrDestGroup`.
				// Pointers and all pointers from there on must point to the same group (unless it is an isolate pointer).
				// Therefore, the `valueGroup` and `ptrDestGroup` must be merged into a gamma-group.
				if valueGroup != ptrDestGroup {
					ptrDestGroup = s.gamma(c, valueGroup, ptrDestGroup)
				}
			}
			// The value is now a temporary variable on the stack.
			// Therefore its group is a scoped group
			valueGroup = c.Scope.GroupVariable
		case ircode.AccessSlice:
			// The result of `expr[a:b]` must be a slice.
			_, ok := types.GetSliceType(ac.OutputType.Type)
			if !ok {
				panic("Output is not a slice")
			}
			if ptrDestGroup.Kind == ircode.VarIsolatedGroup {
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value is in turn an isolate pointer. But that is ok, since the type system has this information in form of a GroupType.
				ptrDestGroup = valueGroup
			} else {
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value may contain further pointers to a group stored in `ptrDestGroup`.
				// Pointers and all pointers from there on must point to the same group (unless it is an isolate pointer).
				// Therefore, the `valueGroup` and `ptrDestGroup` must be merged into a gamma-group.
				if valueGroup != ptrDestGroup {
					ptrDestGroup = s.gamma(c, valueGroup, ptrDestGroup)
				}
			}
			// The value is now a temporary variable on the stack.
			// Therefore its group is a scoped group
			valueGroup = c.Scope.GroupVariable
		case ircode.AccessStruct:
			_, ok := types.GetStructType(ac.InputType.Type)
			if !ok {
				panic("Not a struct")
			}
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = s.newIsolatedGroupVariable(c)
			}
		case ircode.AccessPointerToStruct:
			pt, ok := types.GetPointerType(ac.InputType.Type)
			if !ok {
				panic("Not a pointer")
			}
			_, ok = types.GetStructType(pt.ElementType)
			if !ok {
				panic("Not a struct")
			}
			// Following an unsafe pointer -> give up
			if pt.Mode == types.PtrUnsafe {
				return nil
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = s.newIsolatedGroupVariable(c)
			}
		case ircode.AccessArrayIndex:
			_, ok := types.GetArrayType(ac.InputType.Type)
			if !ok {
				panic("Not a struct")
			}
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = s.newIsolatedGroupVariable(c)
			}
		case ircode.AccessSliceIndex:
			_, ok := types.GetSliceType(ac.InputType.Type)
			if !ok {
				panic("Not a slice")
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = s.newIsolatedGroupVariable(c)
			}
		case ircode.AccessDereferencePointer:
			pt, ok := types.GetPointerType(ac.InputType.Type)
			if !ok {
				panic("Not a pointer")
			}
			// Following an unsafe pointer -> give up
			if pt.Mode == types.PtrUnsafe {
				return nil
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = s.newIsolatedGroupVariable(c)
			}
		default:
			panic("Oooops")
		}
	}
	return ptrDestGroup
}

func (s *ssaTransformer) searchVariable(v *ircode.Variable, search []*ssaScope) (result *ircode.Variable, ok bool) {
	for i := len(search) - 1; i >= 0; i-- {
		if search[i].loopBreak {
			continue
		}
		if result, ok = search[i].vars[v]; ok {
			return
		}
	}
	ok = false
	return
}

// lookupVariable creates a phi-function where perhaps necessary.
func (s *ssaTransformer) lookupVariable(v *ircode.Variable) (result *ircode.Variable, depth int) {
	loops := false
	for depth = len(s.stack) - 1; depth >= 0; depth-- {
		if s.stack[depth].loopBreak {
			continue
		}
		loops = loops || s.stack[depth].loopContinue
		var ok bool
		if result, ok = s.stack[depth].vars[v]; ok {
			break
		}
	}
	if depth < 0 {
		// panic("Unknown variable during lookup: " + v.Name)
		return
	}
	// Create (a placeholder) Phi-function
	if loops {
		for i := depth + 1; i < len(s.stack); i++ {
			if s.stack[i].loopContinue {
				var newGroup *ircode.Variable
				if result.GroupVariable != nil {
					newGroup = s.newVariableVersion(result.GroupVariable)
					newGroup.Phi = []*ircode.Variable{result.GroupVariable}
					s.stack[i].vars[result.GroupVariable] = newGroup
				}
				newResult := s.newVariableVersion(result)
				newResult.Phi = []*ircode.Variable{result}
				newResult.GroupVariable = newGroup
				println("CREATE PHI", v.Original.ToString(), result.ToString(), newResult.ToString())
				result = newResult
				s.stack[i].vars[v.Original] = result
			}
		}
	}
	return
}

func (s *ssaTransformer) lookupGroupVariable(gv *ircode.Variable, stack []*ssaScope) (result *ircode.Variable) {
	if stack == nil {
		stack = s.stack
	}
	result = gv
	for {
		var depth int
		for depth = len(stack) - 1; depth >= 0; depth-- {
			if stack[depth].loopBreak {
				continue
			}
			if result2, ok := stack[depth].vars[result]; ok {
				if result == result2 {
					return
				}
				// Try again
				result = result2
				break
			}
		}
		if depth < 0 {
			return
		}
	}
}

/*
// variableIsLive returns true if the variable has been defined or assigned already.
func (s *ssaTransformer) variableIsLive(v *ircode.Variable) bool {
	for i := len(s.stack) - 1; i >= 0; i-- {
		if _, ok := s.stack[i].vars[v.Original]; ok {
			return true
		}
	}
	return false
}
*/

func (s *ssaTransformer) setVariableInfo(dest *ssaScope, v *ircode.Variable) {
	dest.vars[v.Original] = v
}

// newVariableVersion creates a new version of a variable due to an assignment.
func (s *ssaTransformer) newVariableVersion(v *ircode.Variable) *ircode.Variable {
	v.Original.VersionCount++
	result := &ircode.Variable{Name: v.Original.Name + "." + strconv.Itoa(v.Original.VersionCount), Type: v.Type, Original: v.Original, Scope: v.Original.Scope, Kind: v.Kind, GroupVariable: v.GroupVariable, IsInitialized: v.IsInitialized}
	result.Assignment = result
	return result
}

// newVariableUsageVersion creates a new version of a variable due a usage of the variable.
func (s *ssaTransformer) newVariableUsageVersion(v *ircode.Variable) *ircode.Variable {
	v.Original.VersionCount++
	result := &ircode.Variable{Name: v.Original.Name + "." + strconv.Itoa(v.Original.VersionCount), Type: v.Type, Original: v.Original, Scope: v.Original.Scope, Kind: v.Kind, GroupVariable: v.GroupVariable, IsInitialized: v.IsInitialized}
	result.Assignment = v.Assignment
	return result
}

var uniqueNameCount = 1

func (s *ssaTransformer) newIsolatedGroupVariable(c *ircode.Command) *ircode.Variable {
	// TODO Location
	v := &ircode.Variable{Kind: ircode.VarIsolatedGroup, Name: "gi_" + strconv.Itoa(uniqueNameCount), Scope: c.Scope, Type: &types.ExprType{Type: types.PrimitiveTypeVoid}, IsInitialized: true}
	uniqueNameCount++
	v.Original = v
	v.Assignment = v
	s.setVariableInfo(s.stack[len(s.stack)-1], v)
	return v
}

func (s *ssaTransformer) newNamedGroupVariable(name string, c *ircode.Command) *ircode.Variable {
	if v, ok := s.namedGroupVariables[name]; ok {
		return v
	}
	// TODO Location
	v := &ircode.Variable{Kind: ircode.VarNamedGroup, Name: "gn_" + name, Scope: c.Scope, Type: &types.ExprType{Type: types.PrimitiveTypeVoid}, IsInitialized: true}
	v.Original = v
	v.Assignment = v
	s.setVariableInfo(s.stack[len(s.stack)-1], v)
	s.namedGroupVariables[name] = v
	return v
}

func (s *ssaTransformer) newFreeGroupVariable(c *ircode.Command) *ircode.Variable {
	// TODO Location
	v := &ircode.Variable{Kind: ircode.VarGroup, Name: "gf_" + strconv.Itoa(uniqueNameCount), Scope: c.Scope, Type: &types.ExprType{Type: types.PrimitiveTypeVoid}, IsInitialized: true}
	uniqueNameCount++
	v.Original = v
	v.Assignment = v
	s.setVariableInfo(s.stack[len(s.stack)-1], v)
	return v
}

func (s *ssaTransformer) newPhiGroupVariable(gv1, gv2 *ircode.Variable) *ircode.Variable {
	scope := gv1.Scope
	if gv1.Scope.HasParent(gv2.Scope) {
		scope = gv2.Scope
	}
	gv := &ircode.Variable{Kind: ircode.VarGroup, Type: &types.ExprType{Type: types.PrimitiveTypeVoid}, Scope: scope}
	gv.Phi = []*ircode.Variable{gv1, gv2}
	gv.Original = gv
	gv.Assignment = gv
	gv.Name = "gp_" + strconv.Itoa(uniqueNameCount)
	uniqueNameCount++
	s.setVariableInfo(s.stack[len(s.stack)-1], gv)
	return gv
}

// consolidateScope ensures that all variables visible in the scope use their latets group-variable update.
// This is a prerequisite for merging.
func (s *ssaTransformer) consolidateScope(scope *ssaScope) {
	stack := []*ssaScope{scope}
	for original, v := range scope.vars {
		// Merge normal variables only. Group pseudo-variables are not merged
		if v.Kind != ircode.VarDefault && v.Kind != ircode.VarParameter && v.Kind != ircode.VarTemporary {
			continue
		}
		if v.GroupVariable == nil {
			continue
		}
		gv := s.lookupGroupVariable(v.GroupVariable, stack)
		if gv != v.GroupVariable {
			// println("CONSOLIDATE", v.ToString())
			vNew := s.newVariableUsageVersion(v)
			vNew.GroupVariable = gv
			scope.vars[original] = vNew
			// println("-->", vNew.ToString())
		}
	}
}

// `scope` must not to be on the scope-stack.
func (s *ssaTransformer) mergeOptionalScope(scope *ssaScope) {
	println("MERGE OPTIONAL")
	s.consolidateScope(scope)
	// Merge the group variables first
	for gvPrevOptional, gvLatestOptional := range scope.vars {
		// Merge normal variables only. Group pseudo-variables are not merged
		if gvPrevOptional.Kind == ircode.VarDefault || gvPrevOptional.Kind == ircode.VarParameter || gvPrevOptional.Kind == ircode.VarTemporary {
			continue
		}
		// Is the variable `v` in use in the stack (in the version of `vinfo2`) ?
		// If so, merge it. Otherwise it ends its life in `scope` and must not be merged.
		gvPrev, ok := s.searchVariable(gvPrevOptional, s.stack)
		if ok && gvPrev != gvLatestOptional {
			gvLatestOptional = s.lookupGroupVariable(gvLatestOptional, []*ssaScope{scope})
			gvNew := s.newVariableVersion(gvPrev)
			gvNew.Phi = []*ircode.Variable{gvPrev, gvLatestOptional}
			s.stack[len(s.stack)-1].vars[gvPrev] = gvNew
			s.stack[len(s.stack)-1].vars[gvLatestOptional] = gvNew
			s.stack[len(s.stack)-1].vars[gvNew] = gvNew
		}
	}
	// Merge all normal variables
	for v, vinfo := range scope.vars {
		// Merge normal variables only. Group pseudo-variables are not merged here
		if v.Kind != ircode.VarDefault && v.Kind != ircode.VarParameter && v.Kind != ircode.VarTemporary {
			continue
		}
		// Is the variable `v` in use in the stack (in the version of `vinfo2`) ?
		// If so, merge it. Otherwise it ends its life in `scope` and must not be merged.
		vinfo2, ok := s.searchVariable(v, s.stack)
		// println("merge TEST", v.ToString())
		if ok {
			// println("MERGE", v.ToString(), vinfo2.ToString())
			// On the stack, the variable is known as `vinfo2`.
			// On `scope`, the variable is known as `vinfo`.
			s.mergeSingleOptional(s.stack[len(s.stack)-1], vinfo, vinfo2)
		}
	}
}

// `a1` must not to be on the scope-stack.
// `a2` must not to be on the scope-stack.
func (s *ssaTransformer) mergeAlternativeScopes(a1 *ssaScope, a2 *ssaScope) {
	println("MERGE ALTERNATIVE")
	s.consolidateScope(a1)
	s.consolidateScope(a2)
	// Merge the group variables first
	for gvPrevOptional, gvLatestOptional := range a1.vars {
		// Merge normal variables only. Group pseudo-variables are not merged
		if gvPrevOptional.Kind == ircode.VarDefault || gvPrevOptional.Kind == ircode.VarParameter || gvPrevOptional.Kind == ircode.VarTemporary {
			continue
		}
		println("try alt", gvPrevOptional.ToString(), gvLatestOptional.ToString())
		// Is the variable `v` in use in the stack (in the version of `vinfo2`) ?
		// If so, merge it. Otherwise it ends its life in `scope` and must not be merged.
		gvPrev, ok := s.searchVariable(gvPrevOptional, s.stack)
		if ok {
			println("try alt2", gvPrev.ToString(), gvLatestOptional.ToString())
		}
		if ok && gvPrev != gvLatestOptional {
			if gvLatestOptional2, ok := a2.vars[gvPrev]; ok {
				// Group variable exists in both scopes
				gvLatestOptional = s.lookupGroupVariable(gvLatestOptional, []*ssaScope{a1})
				gvLatestOptional2 = s.lookupGroupVariable(gvLatestOptional2, []*ssaScope{a2})
				gvNew := s.newVariableVersion(gvPrev)
				gvNew.Phi = []*ircode.Variable{gvLatestOptional, gvLatestOptional2}
				s.stack[len(s.stack)-1].vars[gvPrev] = gvNew
				s.stack[len(s.stack)-1].vars[gvLatestOptional] = gvNew
				s.stack[len(s.stack)-1].vars[gvLatestOptional2] = gvNew
				s.stack[len(s.stack)-1].vars[gvNew] = gvNew
				println("NEW", gvPrev.ToString(), gvNew.ToString())
			} else {
				// Group variable exists in `a1` only
				gvLatestOptional = s.lookupGroupVariable(gvLatestOptional, []*ssaScope{a1})
				gvNew := s.newVariableVersion(gvPrev)
				gvNew.Phi = []*ircode.Variable{gvPrev, gvLatestOptional}
				s.stack[len(s.stack)-1].vars[gvPrev] = gvNew
				s.stack[len(s.stack)-1].vars[gvLatestOptional] = gvNew
				s.stack[len(s.stack)-1].vars[gvNew] = gvNew
				println("SINGLE1", gvPrev.ToString(), gvNew.ToString())
			}
		}
	}
	for gvPrevOptional2, gvLatestOptional2 := range a2.vars {
		// Merge normal variables only. Group pseudo-variables are not merged
		if gvPrevOptional2.Kind == ircode.VarDefault || gvPrevOptional2.Kind == ircode.VarParameter || gvPrevOptional2.Kind == ircode.VarTemporary {
			continue
		}
		// Is the variable `v` in use in the stack (in the version of `vinfo2`) ?
		// If so, merge it. Otherwise it ends its life in `scope` and must not be merged.
		gvPrev, ok := s.searchVariable(gvPrevOptional2, s.stack)
		if ok && gvPrev != gvLatestOptional2 {
			// Variable exists in `a2` only?
			if _, ok := a1.vars[gvPrev]; !ok {
				gvLatestOptional2 = s.lookupGroupVariable(gvLatestOptional2, []*ssaScope{a2})
				gvNew := s.newVariableVersion(gvPrev)
				gvNew.Phi = []*ircode.Variable{gvPrev, gvLatestOptional2}
				s.stack[len(s.stack)-1].vars[gvPrev] = gvNew
				s.stack[len(s.stack)-1].vars[gvLatestOptional2] = gvNew
				s.stack[len(s.stack)-1].vars[gvNew] = gvNew
				println("SINGLE2", gvPrev.ToString(), gvNew.ToString())
			}
		}
	}
	// Merge all normal variables
	for v, vinfo1 := range a1.vars {
		// Merge normal variables only. Group pseudo-variables are not merged here
		if v.Kind != ircode.VarDefault && v.Kind != ircode.VarParameter && v.Kind != ircode.VarTemporary {
			continue
		}
		vPrev, ok := s.searchVariable(v, s.stack)
		if !ok {
			continue
		}
		// Variable exists in both scopes?
		if vinfo2, ok := a2.vars[v]; ok {
			vinfo3 := s.newVariableVersion(v)
			vinfo3.Phi = []*ircode.Variable{vinfo1, vinfo2}
			if vinfo3.GroupVariable != nil {
				vinfo3.GroupVariable = s.newPhiGroupVariable(vinfo1.GroupVariable, vinfo2.GroupVariable)
			}
			s.setVariableInfo(s.stack[len(s.stack)-1], vinfo3)
		} else {
			// Variable exists in `a1` only
			s.mergeSingleOptional(s.stack[len(s.stack)-1], vPrev, vinfo1)
		}
	}
	for v, vinfo2 := range a2.vars {
		// Merge normal variables only. Group pseudo-variables are not merged here
		if v.Kind != ircode.VarDefault && v.Kind != ircode.VarParameter && v.Kind != ircode.VarTemporary {
			continue
		}
		// Variable exists in `a2` only?
		if _, ok := a1.vars[v]; !ok {
			vinfo3, ok := s.searchVariable(v, s.stack)
			if ok {
				s.mergeSingleOptional(s.stack[len(s.stack)-1], vinfo2, vinfo3)
			}
		}
	}
}

// Creates a phi variable in scope `dest`.
// The variable `vinfo` becomes a phi-variable of `vinfo` and `vinfo2`.
// `vinfo` and `vinfo2` are versions of the same original variable.
func (s *ssaTransformer) mergeSingleOptional(dest *ssaScope, vinfo *ircode.Variable, vinfo2 *ircode.Variable) {
	/*
		var phi []*ircode.Variable
		if vinfo2.Phi == nil {
			if vinfo.Phi == nil {
				phi = s.mergePhi([]*ircode.Variable{vinfo2}, []*ircode.Variable{vinfo})
			} else {
				phi = s.mergePhi([]*ircode.Variable{vinfo2}, vinfo.Phi)
			}
		} else {
			if vinfo.Phi == nil {
				phi = s.mergePhi(vinfo2.Phi, []*ircode.Variable{vinfo})
			} else {
				phi = s.mergePhi(vinfo2.Phi, vinfo.Phi)
			}
		}
	*/
	vinfo3 := s.newVariableVersion(vinfo)
	vinfo3.Phi = []*ircode.Variable{vinfo, vinfo2}
	if vinfo.GroupVariable != nil {
		latest := s.lookupGroupVariable(vinfo.GroupVariable, nil)
		if _, ok := s.searchVariable(vinfo2.GroupVariable, s.stack); ok {
			// The group of vinfo2 is known in the destination scope already.
			// This implies that the group of vinfo2 is a new version (or the same version)
			// of the group used by vinfo.
			// So just lookup the latest version
			vinfo3.GroupVariable = latest
		} else {
			println("GENERATE PHI GROUP", vinfo.ToString(), vinfo2.ToString())
			// The group of vinfo2 has not made it into the destination scope yet.
			// This must be because the group was set in the merged scope.
			vinfo3.GroupVariable = s.newPhiGroupVariable(latest, vinfo2.GroupVariable)
		}
	}
	s.setVariableInfo(dest, vinfo3)
}

func (s *ssaTransformer) mergeIntoContinueScope(continueScope *ssaScope, loopBodyScope *ssaScope) {
	for v, vinfo := range loopBodyScope.vars {
		if phiInfo, ok := continueScope.vars[v]; ok {
			phiInfo.Phi = append(phiInfo.Phi, vinfo)
			println("CONTINUE PHI", phiInfo.ToString(), vinfo.ToString())
		}
	}
}

func (s *ssaTransformer) mergeBreakScope(dest *ssaScope, breakScope *ssaScope) {
	for v, vinfo := range breakScope.vars {
		println("BREAK MERGE", v.ToString(), vinfo.ToString())
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
func (s *ssaTransformer) mergeJump(dest *ssaScope, search []*ssaScope, scopes []*ssaScope) {
	println("MERGE JUMP scopes", len(scopes))
	done := make(map[*ircode.Variable]bool)
	// Merge all scopes from top to bottom.
	// Do not merge a variable twice.
	for i := len(scopes) - 1; i >= 0; i-- {
		// Merge all variables in the current scope
		for v, vinfo := range scopes[i].vars {
			println("MERGE-JUMP", v.ToString())
			// Do not merge any variable twice
			if _, ok := done[v]; ok {
				continue
			}
			// The variable exists at the destination scope? -> need to create a phi-variable
			if vinfo2, ok := dest.vars[v]; ok {
				s.mergeSingleOptional(dest, vinfo, vinfo2)
			} else if _, ok := s.searchVariable(v, search); ok {
				// The variable exists at a parent scope of the destination scope.
				// So just set it in the destination scope. It will be merged with this parent scope later.
				s.setVariableInfo(dest, vinfo)
			}
			done[v] = true
		}
	}
}

// TransformToSSA checks the control flow and detects unreachable code.
// Thereby it translates IR-code Variables into Single-Static-Assignment which is
// required for further optimizations and code analysis.
func TransformToSSA(f *ircode.Function, log *errlog.ErrorLog) {
	s := &ssaTransformer{f: f, log: log}
	s.namedGroupVariables = make(map[string]*ircode.Variable)
	m := newVariableInfoScope()
	// Mark all parameters as initialized
	for _, v := range f.Vars {
		if v.Kind == ircode.VarParameter {
			v.IsInitialized = true
			m.vars[v] = v
		}
	}
	s.stack = append(s.stack, m)
	s.transformBlock(&f.Body, 0)
	s.stack = s.stack[0 : len(s.stack)-1]
	s.computeGroupBlock(&f.Body)
}
