package ssa

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

type ssaTransformer struct {
	f                   *ircode.Function
	log                 *errlog.ErrorLog
	namedGroupVariables map[string]*GroupVariable
	topLevelScope       *ssaScope
}

func (s *ssaTransformer) transformBlock(c *ircode.Command, vs *ssaScope) bool {
	if c.Op != ircode.OpBlock && c.Op != ircode.OpIf && c.Op != ircode.OpLoop {
		panic("Not a block")
	}
	for i, c2 := range c.Block {
		if !s.transformCommand(c2, vs) {
			if i+1 < len(c.Block) && c.Block[i+1].Op != ircode.OpCloseScope {
				s.log.AddError(errlog.ErrorUnreachable, c.Block[i+1].Location)
			}
			return false
		}
	}
	return true
}

func (s *ssaTransformer) transformCommand(c *ircode.Command, vs *ssaScope) bool {
	switch c.Op {
	case ircode.OpBlock:
		s.transformBlock(c, vs)
	case ircode.OpIf:
		c.Scope.GroupInfo = vs.newScopedGroupVariable(c.Scope)
		// visit the condition
		s.transformArguments(c, vs)
		// visit the if-clause
		ifScope := newScope()
		ifScope.parent = vs
		ifScope.kind = scopeIf
		ifCompletes := s.transformBlock(c, ifScope)
		// visit the else-clause
		if c.Else != nil {
			c.Else.Scope.GroupInfo = vs.newScopedGroupVariable(c.Else.Scope)
			elseScope := newScope()
			elseScope.parent = vs
			elseScope.kind = scopeIf
			elseCompletes := s.transformBlock(c.Else, elseScope)
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
		c.Scope.GroupInfo = vs.newScopedGroupVariable(c.Scope)
		loopScope := newScope()
		loopScope.parent = vs
		loopScope.kind = scopeLoop
		doesLoop := s.transformBlock(c, loopScope)
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
			if loopScope.kind == scopeLoop {
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
			if loopScope.kind == scopeLoop {
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
		if types.TypeHasPointers(c.Dest[0].Type.Type) {
			if c.Dest[0].Kind == ircode.VarParameter && c.Dest[0].Type.PointerDestGroup != nil {
				// Parameters with pointers have groups already when the function is being called.
				grp := c.Dest[0].Type.PointerDestGroup
				if grp.Kind == types.GroupIsolate {
					setGroupVariable(c.Dest[0], vs.newGroupVariable())
				} else {
					setGroupVariable(c.Dest[0], vs.newNamedGroupVariable(grp.Name))
				}
			} else {
				setGroupVariable(c.Dest[0], vs.newGroupVariable())
			}
		}
	case ircode.OpPrintln:
		s.transformArguments(c, vs)
	case ircode.OpSet:
		s.transformArguments(c, vs)
		gDest := accessChainGroupVariable(c, vs, s.log)
		gSrc := argumentGroupVariable(c, c.Args[len(c.Args)-1], vs, c.Location)
		/* gv := */ vs.merge(gDest, gSrc, nil, c, s.log)
		if len(c.Dest) == 1 {
			_, dest := vs.lookupVariable(c.Dest[0])
			if dest == nil {
				panic("Oooops, variable does not exist")
			}
			if !ircode.IsVarInitialized(dest) {
				s.log.AddError(errlog.ErrorUninitializedVariable, c.Location, dest.Original.Name)
			}
			// if gv != dest.GroupInfo {
			v := vs.newVariableVersion(dest)
			//		setGroupVariable(v, gv)
			c.Dest[0] = v
		}
	case ircode.OpGet:
		s.transformArguments(c, vs)
		v := vs.createDestinationVariable(c)
		if types.TypeHasPointers(v.Type.Type) {
			// The group resulting in the Get operation becomes the group of the destination
			setGroupVariable(v, accessChainGroupVariable(c, vs, s.log))
		}
		// The destination variable is now initialized
		v.IsInitialized = true
	case ircode.OpSetVariable:
		s.transformArguments(c, vs)
		v := vs.createDestinationVariable(c)
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
			setGroupVariable(v, argumentGroupVariable(c, c.Args[0], vs, c.Location))
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

		s.transformArguments(c, vs)
		v := vs.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
		// No groups to update here, because these ops work on primitive types.
	case ircode.OpArray,
		ircode.OpStruct:

		s.transformArguments(c, vs)
		v := vs.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
		// If the type has pointers, update the group variable
		if types.TypeHasPointers(v.Type.Type) {
			var gv *GroupVariable
			for i := range c.Args {
				gArg := argumentGroupVariable(c, c.Args[i], vs, c.Location)
				if gArg != nil {
					if gv == nil {
						gv = gArg
					} else {
						gv = vs.merge(gv, gArg, nil, c, s.log)
					}
				}
			}
			if gv == nil {
				gv = vs.newGroupVariable()
			}
			setGroupVariable(v, gv)
		}
	case ircode.OpOpenScope,
		ircode.OpCloseScope:
		// Do nothing by intention
	default:
		panic("Ooop")
	}
	return true
}

func (s *ssaTransformer) transformArguments(c *ircode.Command, vs *ssaScope) {
	// Evaluate arguments right to left
	for i := len(c.Args) - 1; i >= 0; i-- {
		if c.Args[i].Cmd != nil {
			s.transformCommand(c.Args[i].Cmd, vs)
		} else if c.Args[i].Var != nil {
			_, v2 := vs.lookupVariable(c.Args[i].Var)
			if v2 == nil {
				panic("Oooops, variable does not exist " + c.Args[i].Var.Name)
			}
			if !ircode.IsVarInitialized(v2) {
				s.log.AddError(errlog.ErrorUninitializedVariable, c.Location, v2.Original.Name)
			}
			// Do not replace the first argument to OpSet with a constant
			if v2.Type.IsPrimitiveConstant() && !(c.Op == ircode.OpSet && i == 0) {
				c.Args[i].Const = &ircode.Constant{ExprType: v2.Type}
				c.Args[i].Var = nil
			} else {
				c.Args[i].Var = v2
			}
			if c.Args[i].Var != nil {
				gv := groupVar(v2)
				if gv != nil {
					_, gv2 := vs.lookupGroup(gv)
					// If the group of the variable changed in the meantime, update the variable such that it refers to its current group
					if gv != gv2 {
						v := vs.newVariableUsageVersion(c.Args[i].Var)
						setGroupVariable(v, gv2)
						c.Args[i].Var = v
					}
				}
			}
		}
	}
}

func (s *ssaTransformer) transformScope(block *ircode.Command, vs *ssaScope) {
	// Update all groups in the command block such that they reflect the computed group mergers
	for _, c := range block.Block {
		for _, arg := range c.Args {
			if arg.Var != nil {
				if arg.Var.GroupInfo != nil {
					_, arg.Var.GroupInfo = vs.lookupGroup(arg.Var.GroupInfo.(*GroupVariable))
				}
			} else if arg.Const != nil {
				if arg.Const.GroupInfo != nil {
					_, arg.Const.GroupInfo = vs.lookupGroup(arg.Const.GroupInfo.(*GroupVariable))
				}
			}
		}
		for _, dest := range c.Dest {
			if dest.GroupInfo != nil {
				_, dest.GroupInfo = vs.lookupGroup(dest.GroupInfo.(*GroupVariable))
			}
		}
	}
	// Find all groups in this scope which do not merge or phi any other group.
	// Those are assigned real variables which point to the underlying memory group.
	for gv, gvNew := range vs.groups {
		// Ignore groups that have been merged by others.
		// Ignore named groups, since these are not free'd and their pointers are
		// passed along with the function arguments.
		if gv.Constraints.NamedGroup != "" {
			continue
		}
		// Consider all groups that have no input-ties to any other group.
		// Ignore groups which are never associated with any allocation.
		if gv.scope != vs || len(gv.In) != 0 || gv != gvNew || vs.NoAllocations(gv) {
			continue
		}
		t := &types.ExprType{Type: &types.PointerType{Mode: types.PtrUnsafe, ElementType: types.PrimitiveTypeVoid}}
		v := &ircode.Variable{Name: gv.Name, Type: t, Scope: block.Scope}
		v.Original = v
		s.f.Vars = append(s.f.Vars, v)
		c := &ircode.Command{Op: ircode.OpDefVariable, Dest: []*ircode.Variable{v}, Type: v.Type, Location: block.Location, Scope: block.Scope}
		openScope := block.Block[0]
		if openScope.Op != ircode.OpOpenScope {
			panic("Oooops")
		}
		openScope.Block = append(openScope.Block, c)

		if !vs.groupVariableMergesOuterScope(gv) {
			println(gv.GroupVariableName(), gv.Allocations, vs.NoAllocations(gv))
			c := &ircode.Command{Op: ircode.OpFree, Args: []ircode.Argument{ircode.NewVarArg(v)}, Location: block.Location, Scope: block.Scope}
			closeScope := block.Block[len(block.Block)-1]
			if closeScope.Op != ircode.OpCloseScope {
				panic("Oooops")
			}
			closeScope.Block = append(closeScope.Block, c)
		}
	}
	/*
		// Determine all groups that must be freed
		for _, gv := range vs.groups {
			// Consider all groups that have no ties to the outer scope
			if gv.scope != vs || len(gv.Out) != 0 {
				continue
			}

		}
	*/
}

// TransformToSSA checks the control flow and detects unreachable code.
// Thereby it translates IR-code Variables into Single-Static-Assignment which is
// required for further optimizations and code analysis.
// Furthermore, it checks groups and whether the IR-code (and therefore the original code)
// comply with the grouping rules.
func TransformToSSA(f *ircode.Function, log *errlog.ErrorLog) {
	s := &ssaTransformer{f: f, log: log, topLevelScope: newScope()}
	s.namedGroupVariables = make(map[string]*GroupVariable)
	s.topLevelScope.kind = scopeFunc
	// Mark all parameters as initialized
	for _, v := range f.Vars {
		if v.Kind == ircode.VarParameter {
			v.IsInitialized = true
			// m.vars[v] = v
			if v.Type.PointerDestGroup != nil {
				gv := s.topLevelScope.newNamedGroupVariable(v.Type.PointerDestGroup.Name)
				s.namedGroupVariables[v.Type.PointerDestGroup.Name] = gv
				setGroupVariable(v, gv)
			}
			s.topLevelScope.vars[v] = v
		}
	}
	f.Body.Scope.GroupInfo = s.topLevelScope.newScopedGroupVariable(f.Body.Scope)
	s.transformBlock(&f.Body, s.topLevelScope)
	s.transformScope(&f.Body, s.topLevelScope)
}
