package ircode

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/types"
)

// Operation ...
type Operation int

const (
	// OpSetVariable sets a variable.
	// The first argument is a variable.
	OpSetVariable Operation = 1 + iota
	// OpDefVariable defines a new variable without assigning to it
	OpDefVariable
	// OpIf is an if clause, which takes a boolean expression as its only argument.
	OpIf
	// OpLoop repeats the commands in its branch expression for ever.
	// Use OpBreak to leave the loop.
	OpLoop
	// OpBreak leaves a OpLoop of OpIf.
	// The first argument is an int which describes which loop/if to exit.
	// A value of 0 exits the current loop/if, a value of 1 the parent loop/if.
	OpBreak
	// OpContinue jumps to the beginning of an OpLoop (not OpIf).
	// A value of 0 exits the current loop, a value of 1 the parent loop.
	OpContinue
	// OpBlock represented by the operations in the Block field.
	OpBlock
	// OpAdd adds numerical values.
	OpAdd
	// OpSub ...
	OpSub
	// OpMul ...
	OpMul
	// OpDiv ...
	OpDiv
	// OpRemainder ...
	OpRemainder
	// OpBinaryXor ...
	OpBinaryXor
	// OpBinaryOr ...
	OpBinaryOr
	// OpBinaryAnd ...
	OpBinaryAnd
	// OpShiftLeft ...
	OpShiftLeft
	// OpShiftRight ...
	OpShiftRight
	// OpBitClear ...
	OpBitClear
	// OpLogicalOr ...
	OpLogicalOr
	// OpLogicalAnd ...
	OpLogicalAnd
	// OpEqual ...
	OpEqual
	// OpNotEqual ...
	OpNotEqual
	// OpLess ...
	OpLess
	// OpGreater ...
	OpGreater
	// OpLessOrEqual ...
	OpLessOrEqual
	// OpGreaterOrEqual ...
	OpGreaterOrEqual
	// OpNot ...
	OpNot
	// OpMinusSign ...
	OpMinusSign
	// OpBitwiseComplement ...
	OpBitwiseComplement
	// OpPrintln outputs its argument.
	OpPrintln
	// OpGet retrieves a value from its first argument via an access chain.
	// The following arguments are subject to the access chain.
	OpGet
	// OpSet sets a value in its destination variable via an access chain.
	// All arguments except the last one are subject to the access chain.
	// The last argument is the value to set.
	OpSet
	// OpArray ...
	OpArray
	// OpStruct ...
	OpStruct
)

// AccessKind ...
type AccessKind int

const (
	// AccessStruct ...
	// The expected argument is an integer denoting the field-index inside the struct.
	AccessStruct AccessKind = 1 + iota
	// AccessPointerToStruct ...
	// The expected argument is an integer denoting the field-index inside the struct.
	AccessPointerToStruct
	// AccessArrayIndex ...
	// The expected argument is an integer denoting the array-index.
	AccessArrayIndex
	// AccessSliceIndex ...
	// The expected argument is an integer denoting the array-index.
	AccessSliceIndex
	// AccessUnsafeArrayIndex ...
	AccessUnsafeArrayIndex
	// AccessDereferencePointer ...
	AccessDereferencePointer
	// AccessSlice ...
	// The two expected arguments are the left-offset and size of the slice.
	AccessSlice
	// AccessCast ...
	// The argument must exist, but is ignored.
	// The Access Chain Element must bear the type to which the cast converts.
	AccessCast
	// AccessAddressOf takes the address (in form of a pointer) of the element accessed by the AccessChain.
	AccessAddressOf
	// AccessInc increases an integener number by 1.
	// AccessInc can only appear at the end of an access chain and can only be used
	// in association with OpSet, but not OpGet.
	AccessInc
	// AccessDec ...
	AccessDec
)

// VariableKind ...
type VariableKind int

const (
	// VarDefault is a variable that has its counterpart in the high-level AST.
	VarDefault = 0
	// VarParameter is the parameter of a function.
	VarParameter = 1
	// VarTemporary is a generated variable that has no counterpart in the high-level AST.
	VarTemporary = 2
)

// CommandScope ...
type CommandScope struct {
	ID     int
	Parent *CommandScope
}

// Variable ...
type Variable struct {
	Name string
	Kind VariableKind
	// Type of the variable.
	// If the variable has a constant value, it is stored here as well.
	Type  *types.ExprType
	Scope *CommandScope
	// Used for SSA
	Phi []*Variable
	// Used for SSA
	Original *Variable
	// VersionCount is used during SSA transformation to track
	// how many additional versions of this variable exist.
	VersionCount int
	// A Sticky variable cannot be optimized away by inlining,
	// because its address is taken.
	Sticky bool
	// The group of the variable during initial assignment
	Group *types.Group
	// The group to which pointers inside this variable point.
	// Initially this is the free group.
	PointerDestGroup *types.Group
	// This value is useless if the variable is a Phi variable.
	// Use IsVarInitialized() instead.
	IsInitialized bool
	// Used to a traversal algorithm
	marked bool
}

// VariableUsage ...
type VariableUsage struct {
	Var              *Variable
	Group            *types.Group
	PointerDestGroup *types.Group
}

// Constant ...
type Constant struct {
	// Type and value of the constant
	ExprType *types.ExprType
}

// Argument ...
// An argument is either a variable, the result of another command, or a constant.
type Argument struct {
	Var      VariableUsage
	Cmd      *Command
	Const    *Constant
	Location errlog.LocationRange
}

// Command ...
type Command struct {
	// Dest may be null, if the command is inlined or if it represents a void operation
	Dest []VariableUsage
	// Return-type of the command.
	// This must be the same ExprType as the one stored in Dest[0].Var.Type.
	Type *types.ExprType
	// The operation performed by the command
	Op Operation
	// Arguments to the operation
	Args []Argument
	// Optional block of commands nested inside this command
	Block []*Command
	// Optional else-block of commands nested inside this command
	Else *Command
	// Optional
	AccessChain []AccessChainElement
	// Used by Loop, If, Else
	Scope    *CommandScope
	Location errlog.LocationRange
}

// AccessChainElement ...
type AccessChainElement struct {
	InputType  *types.ExprType
	OutputType *types.ExprType
	Kind       AccessKind
	// Used when accessing a struct
	Field    *types.StructField
	Location errlog.LocationRange
}

// Type ...
func (arg *Argument) Type() *types.ExprType {
	if arg.Var.Var != nil {
		return arg.Var.Var.Type
	}
	if arg.Cmd != nil {
		return arg.Cmd.Type
	}
	return arg.Const.ExprType
}

// ToString ...
func (arg *Argument) ToString() string {
	if arg.Var.Var != nil {
		return arg.Var.ToString()
	}
	if arg.Cmd != nil {
		return arg.Cmd.ToString("")
	}
	return arg.Const.ToString()
}

// ToString ...
func (vu *VariableUsage) ToString() string {
	if vu.Var == nil {
		panic("No variable")
	}
	//	return vu.Var.ToString() + "." + vu.Group.ToString()
	if vu.Group == nil {
		return vu.Var.ToString()
	}
	return vu.Var.ToString() + "." + vu.Group.ToString()
}

func newScope(parent *CommandScope) *CommandScope {
	return &CommandScope{Parent: parent}
}

// HasParent implements the types.GroupScope interface
func (s *CommandScope) HasParent(parent types.GroupScope) bool {
	p, ok := parent.(*CommandScope)
	if !ok {
		panic("Oooops")
	}
	if s.Parent == nil {
		return false
	}
	if s.Parent == p {
		return true
	}
	return s.Parent.HasParent(parent)
}

// ToString ...
func (v *Variable) ToString() string {
	return v.Name
}

// IsOriginal ...
func (v *Variable) IsOriginal() bool {
	return v.Original == v
}

// IsVarInitialized ...
func IsVarInitialized(v *Variable) bool {
	if v.Phi != nil {
		v.marked = true
		for _, v2 := range v.Phi {
			if v2.marked {
				continue
			}
			if !IsVarInitialized(v2) {
				return false
			}
		}
		v.marked = false
		return true
	}
	return v.IsInitialized
}

// ToString ...
func (c *Constant) ToString() string {
	return constToString(c.ExprType)
}

func constToString(et *types.ExprType) string {
	if types.IsIntegerType(et.Type) {
		return et.IntegerValue.Text(10)
	}
	if types.IsFloatType(et.Type) {
		return et.FloatValue.Text('f', 5)
	}
	if et.Type == types.PrimitiveTypeString {
		return et.StringValue
	}
	if et.Type == types.PrimitiveTypeBool {
		if et.BoolValue {
			return "true"
		}
		return "false"
	}
	if types.IsArrayType(et.Type) {
		str := "["
		for i, element := range et.ArrayValue {
			if i > 0 {
				str += ", "
			}
			str += constToString(element)
		}
		return str + "]"
	}
	if types.IsSliceType(et.Type) {
		if et.IntegerValue != nil {
			if et.IntegerValue.Uint64() != 0 {
				panic("Oooops")
			}
			return "null-slice"
		}
		str := "&["
		for i, element := range et.ArrayValue {
			if i > 0 {
				str += ", "
			}
			str += constToString(element)
		}
		return str + "]"
	}
	if ptr, ok := types.GetPointerType(et.Type); ok {
		if et.IntegerValue != nil {
			if et.IntegerValue.Uint64() == 0 {
				return "null"
			}
			return "0x" + et.IntegerValue.Text(16)
		}
		_, ok := types.GetStructType(ptr.ElementType)
		if !ok {
			panic("Oooops")
		}
		str := "&{"
		i := 0
		for name, element := range et.StructValue {
			if i > 0 {
				str += ", "
			}
			str += name + ": " + constToString(element)
			i++
		}
		return str + "}"
	}
	if _, ok := types.GetStructType(et.Type); ok {
		str := "{"
		i := 0
		for name, element := range et.StructValue {
			if i > 0 {
				str += ", "
			}
			str += name + ": " + constToString(element)
			i++
		}
		return str + "}"
	}
	fmt.Printf("%T\n", et.Type)
	panic("TODO")
}

// ToString ...
func (cmd *Command) ToString(indent string) string {
	switch cmd.Op {
	case OpBlock:
		var str string
		for _, c := range cmd.Block {
			str += c.ToString(indent+"    ") + "\n"
		}
		return str
	case OpIf:
		str := indent + "if " + cmd.Args[0].ToString() + " { // " + strconv.Itoa(cmd.Scope.ID) + "\n"
		for _, c := range cmd.Block {
			str += c.ToString(indent+"    ") + "\n"
		}
		str += indent + "}"
		if cmd.Else != nil {
			str += " else { // " + strconv.Itoa(cmd.Else.Scope.ID) + "\n"
			str += cmd.Else.ToString(indent)
			str += indent + "}"
		}
		return str
	case OpSetVariable:
		return indent + cmd.Dest[0].ToString() + " = " + cmd.Args[0].ToString()
	case OpDefVariable:
		return indent + "def " + cmd.Dest[0].ToString() + " " + cmd.Dest[0].Var.Type.Type.ToString()
	case OpLoop:
		str := indent + "loop { // " + strconv.Itoa(cmd.Scope.ID) + "\n"
		for _, c := range cmd.Block {
			str += c.ToString(indent+"    ") + "\n"
		}
		str += indent + "}"
		return str
	case OpBreak:
		return indent + "break " + cmd.Args[0].ToString()
	case OpContinue:
		return indent + "continue " + cmd.Args[0].ToString()
	case OpAdd:
		return indent + cmd.Dest[0].ToString() + " = add(" + argsToString(cmd.Args) + ")"
	case OpSub:
		return indent + cmd.Dest[0].ToString() + " = sub(" + argsToString(cmd.Args) + ")"
	case OpMul:
		return indent + cmd.Dest[0].ToString() + " = mul(" + argsToString(cmd.Args) + ")"
	case OpDiv:
		return indent + cmd.Dest[0].ToString() + " = div(" + argsToString(cmd.Args) + ")"
	case OpRemainder:
		return indent + cmd.Dest[0].ToString() + " = remainder(" + argsToString(cmd.Args) + ")"
	case OpBinaryXor:
		return indent + cmd.Dest[0].ToString() + " = xor(" + argsToString(cmd.Args) + ")"
	case OpBinaryOr:
		return indent + cmd.Dest[0].ToString() + " = or(" + argsToString(cmd.Args) + ")"
	case OpBinaryAnd:
		return indent + cmd.Dest[0].ToString() + " = and(" + argsToString(cmd.Args) + ")"
	case OpShiftLeft:
		return indent + cmd.Dest[0].ToString() + " = shift_left(" + argsToString(cmd.Args) + ")"
	case OpShiftRight:
		return indent + cmd.Dest[0].ToString() + " = shift_right(" + argsToString(cmd.Args) + ")"
	case OpBitClear:
		return indent + cmd.Dest[0].ToString() + " = bit_clear(" + argsToString(cmd.Args) + ")"
	case OpLogicalAnd:
		return indent + cmd.Dest[0].ToString() + " = logical_and(" + argsToString(cmd.Args) + ")"
	case OpLogicalOr:
		return indent + cmd.Dest[0].ToString() + " = logical_or(" + argsToString(cmd.Args) + ")"
	case OpEqual:
		return indent + cmd.Dest[0].ToString() + " = eq(" + argsToString(cmd.Args) + ")"
	case OpNotEqual:
		return indent + cmd.Dest[0].ToString() + " = neq(" + argsToString(cmd.Args) + ")"
	case OpLess:
		return indent + cmd.Dest[0].ToString() + " = less(" + argsToString(cmd.Args) + ")"
	case OpGreater:
		return indent + cmd.Dest[0].ToString() + " = greater(" + argsToString(cmd.Args) + ")"
	case OpLessOrEqual:
		return indent + cmd.Dest[0].ToString() + " = leq(" + argsToString(cmd.Args) + ")"
	case OpGreaterOrEqual:
		return indent + cmd.Dest[0].ToString() + " = gew(" + argsToString(cmd.Args) + ")"
	case OpNot:
		return indent + cmd.Dest[0].ToString() + " = not(" + cmd.Args[0].ToString() + ")"
	case OpMinusSign:
		return indent + cmd.Dest[0].ToString() + " = minus(" + cmd.Args[0].ToString() + ")"
	case OpBitwiseComplement:
		return indent + cmd.Dest[0].ToString() + " = complement(" + cmd.Args[0].ToString() + ")"
	case OpPrintln:
		return indent + "println(" + argsToString(cmd.Args) + ")"
	case OpGet:
		str := indent + cmd.Dest[0].ToString() + " = " + cmd.Args[0].ToString()
		str += accessChainToString(cmd.AccessChain, cmd.Args[1:])
		return str
	case OpSet:
		str := ""
		if cmd.Dest[0].Var != nil {
			str += indent + cmd.Dest[0].ToString() + " <= "
		}
		if cmd.AccessChain[len(cmd.AccessChain)-1].Kind == AccessInc || cmd.AccessChain[len(cmd.AccessChain)-1].Kind == AccessDec {
			str += cmd.Args[0].ToString() + accessChainToString(cmd.AccessChain, cmd.Args[1:])
		} else {
			str += cmd.Args[0].ToString() + accessChainToString(cmd.AccessChain, cmd.Args[1:]) + " = "
			str += cmd.Args[len(cmd.Args)-1].ToString()
		}
		return str
	case OpArray:
		return indent + cmd.Dest[0].ToString() + " = [" + argsToString(cmd.Args) + "]"
	case OpStruct:
		return indent + cmd.Dest[0].ToString() + " = {" + argsToString(cmd.Args) + "}"
	}
	println(cmd.Op)
	panic("TODO")
}

func accessChainToString(chain []AccessChainElement, args []Argument) string {
	str := ""
	i := 0
	for _, ac := range chain {
		switch ac.Kind {
		case AccessArrayIndex, AccessSliceIndex:
			if ac.Kind == AccessArrayIndex {
				str += " ["
			} else {
				str += " *["
			}
			str += args[i].ToString()
			str += "]"
			i++
		case AccessSlice:
			str += " ["
			str += args[i].ToString()
			str += ":"
			str += args[i+1].ToString()
			str += "]"
			i += 2
		case AccessStruct, AccessPointerToStruct:
			if ac.Kind == AccessStruct {
				str += "."
			} else {
				str += "->"
			}
			str += ac.Field.Name
		case AccessDereferencePointer:
			str += " *"
		case AccessAddressOf:
			str += " & "
		case AccessInc:
			str += "++"
		case AccessDec:
			str += "--"
		default:
			panic("TODO")
		}
	}
	return str
}

func argsToString(args []Argument) string {
	var str string
	for i, a := range args {
		if i != 0 {
			str += ", "
		}
		str += a.ToString()
	}
	return str
}

/*
// PhiToString ...
func (cmd *Command) PhiToString() string {
	return cmd.phiToString(make(map[*Variable]bool))
}

func (cmd *Command) phiToString(done map[*Variable]bool) string {
	var result string
	for _, vu := range cmd.Dest {
		if vu.Var.Phi != nil {
			if _, ok := done[vu.Var]; !ok {
				result += singlePhiToString(vu.Var, done)
				done[vu.Var] = true
			}
		}
	}
	for _, a := range cmd.Args {
		if a.Var.Var != nil && a.Var.Var.Phi != nil {
			if _, ok := done[a.Var.Var]; !ok {
				result += singlePhiToString(a.Var.Var, done)
				done[a.Var.Var] = true
			}
		} else if a.Cmd != nil {
			result += a.Cmd.phiToString(done)
		}
	}
	if cmd.Block != nil {
		for _, c := range cmd.Block {
			result += c.phiToString(done)
		}
	}
	if cmd.Else != nil {
		result += cmd.Else.phiToString(done)
	}
	return result
}

func singlePhiToString(v *Variable, done map[*Variable]bool) string {
	str := v.Name + " = phi("
	for i, p := range v.Phi {
		if i > 0 {
			str += ", "
		}
		str += p.Name
	}
	str += ")\n"
	for _, phi := range v.Phi {
		if phi.Phi != nil {
			if _, ok := done[phi]; !ok {
				str += singlePhiToString(phi, done)
				done[phi] = true
			}
		}
	}
	return str
}

// PhiGroupsToString ...
func (cmd *Command) PhiGroupsToString() string {
	return cmd.phiGroupsToString(make(map[*Group]bool))
}

func (cmd *Command) phiGroupsToString(done map[*Group]bool) string {
	var result string
	for _, vu := range cmd.Dest {
		if vu.Group.Pointer != nil && len(vu.Group.Pointer.Groups) != 0 {
			if _, ok := done[vu.Group.Pointer]; !ok {
				result += singlePhiGroupToString(vu.Group.Pointer, done)
				done[vu.Group.Pointer] = true
			}
		}
		if vu.Group.Borrow != nil && len(vu.Group.Borrow.Groups) != 0 {
			if _, ok := done[vu.Group.Borrow]; !ok {
				result += singlePhiGroupToString(vu.Group.Borrow, done)
				done[vu.Group.Borrow] = true
			}
		}
	}
	for _, a := range cmd.Args {
		if a.Var.Var != nil {
			if a.Var.Group.Pointer != nil && len(a.Var.Group.Pointer.Groups) != 0 {
				if _, ok := done[a.Var.Group.Pointer]; !ok {
					result += singlePhiGroupToString(a.Var.Group.Pointer, done)
					done[a.Var.Group.Pointer] = true
				}
			}
			if a.Var.Group.Borrow != nil && len(a.Var.Group.Borrow.Groups) != 0 {
				if _, ok := done[a.Var.Group.Borrow]; !ok {
					result += singlePhiGroupToString(a.Var.Group.Borrow, done)
					done[a.Var.Group.Borrow] = true
				}
			}
		} else if a.Cmd != nil {
			result += a.Cmd.phiGroupsToString(done)
		}
	}
	if cmd.Block != nil {
		for _, c := range cmd.Block {
			result += c.phiGroupsToString(done)
		}
	}
	if cmd.Else != nil {
		result += cmd.Else.phiGroupsToString(done)
	}
	return result
}

func singlePhiGroupToString(g *Group, done map[*Group]bool) string {
	var str string
	if g.Kind == GroupPhi {
		str = strconv.Itoa(g.id) + " = phi-group("
	} else if g.Kind == GroupGamma {
		str = strconv.Itoa(g.id) + " = gamma-group("
	} else {
		panic("Should not happen")
	}
	for i, p := range g.Groups {
		if i > 0 {
			str += ", "
		}
		str += p.ToString()
	}
	str += ")\n"
	for _, g2 := range g.Groups {
		if g2.Kind == GroupPhi || g2.Kind == GroupGamma {
			if _, ok := done[g2]; !ok {
				str += singlePhiGroupToString(g2, done)
				done[g2] = true
			}
		}
	}
	return str
}
*/

/*******************************************************
 *
 * Convenience functions and constants
 *
 *******************************************************/

// TrueConstant ...
var TrueConstant = &Constant{ExprType: &types.ExprType{Type: types.PrimitiveTypeBool, BoolValue: true, HasValue: true}}

// FalseConstant ...
var FalseConstant = &Constant{ExprType: &types.ExprType{Type: types.PrimitiveTypeBool, BoolValue: false, HasValue: true}}

// NewDefaultCompositeConstant ...
func NewDefaultCompositeConstant(t *types.ExprType) *Constant {
	// TODO
	c := &Constant{ExprType: t}
	return c
}

// NewStringArg ...
func NewStringArg(s string) Argument {
	return Argument{Const: &Constant{ExprType: &types.ExprType{Type: types.PrimitiveTypeString, StringValue: s, HasValue: true}}}
}

// NewIntArg ...
func NewIntArg(i int) Argument {
	bigint := big.NewInt(int64(i))
	return Argument{Const: &Constant{ExprType: &types.ExprType{Type: types.PrimitiveTypeInt, IntegerValue: bigint, HasValue: true}}}
}

// NewBoolArg ...
func NewBoolArg(b bool) Argument {
	return Argument{Const: &Constant{ExprType: &types.ExprType{Type: types.PrimitiveTypeBool, BoolValue: b, HasValue: true}}}
}

// NewVarArg ...
func NewVarArg(v *Variable) Argument {
	return Argument{Var: VariableUsage{Var: v}}
}

// NewConstArg ...
func NewConstArg(c *Constant) Argument {
	return Argument{Const: c}
}
