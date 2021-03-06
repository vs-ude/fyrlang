package ircode

import (
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/types"
)

// Builder ...
type Builder struct {
	Func          *Function
	current       *Command
	stack         []*Command
	loopCount     int
	locationStack []errlog.LocationRange
	location      errlog.LocationRange
	iterOffset    int
}

// AccessChainBuilder ...
type AccessChainBuilder struct {
	OutputType *types.ExprType
	Cmd        *Command
	b          *Builder
}

// NewBuilder ...
func NewBuilder(f *Function) *Builder {
	b := &Builder{}
	b.Func = f
	b.current = &b.Func.Body
	b.openScope()
	return b
}

// SaveLocation ...
func (b *Builder) SaveLocation() {
	b.locationStack = append(b.locationStack, b.location)
}

// RestoreLocation ...
func (b *Builder) RestoreLocation() {
	b.location = b.locationStack[len(b.locationStack)-1]
	b.locationStack = b.locationStack[:len(b.locationStack)-1]
}

// SetLocation sets the location in the sources for which IR-code is bring built
func (b *Builder) SetLocation(loc errlog.LocationRange) {
	b.location = loc
}

// Location ...
func (b *Builder) Location() errlog.LocationRange {
	return b.location
}

// SetVariable ...
func (b *Builder) SetVariable(dest *Variable, value Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value.Type()))
	}
	// TODO: Safety b.compareTypes(dest.Type, value.Type())
	c := &Command{Op: OpSetVariable, Dest: []*Variable{dest}, Args: []Argument{value}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Add ...
func (b *Builder) Add(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value1.Type()))
	} else {
		// TODO: Safety b.compareTypes(dest.Type, value1.Type())
	}
	// TODO: Safety b.compareTypes(value1.Type(), value2.Type())
	c := &Command{Op: OpAdd, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Sub ...
func (b *Builder) Sub(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value1.Type()))
	}
	c := &Command{Op: OpSub, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Mul ...
func (b *Builder) Mul(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value1.Type()))
	}
	c := &Command{Op: OpMul, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Div ...
func (b *Builder) Div(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value1.Type()))
	}
	c := &Command{Op: OpDiv, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Remainder ...
func (b *Builder) Remainder(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value1.Type()))
	}
	c := &Command{Op: OpRemainder, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// BinaryXor ...
func (b *Builder) BinaryXor(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value1.Type()))
	}
	c := &Command{Op: OpBinaryXor, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// BinaryOr ...
func (b *Builder) BinaryOr(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value1.Type()))
	}
	c := &Command{Op: OpBinaryOr, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// BinaryAnd ...
func (b *Builder) BinaryAnd(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value1.Type()))
	}
	c := &Command{Op: OpBinaryAnd, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// ShiftLeft ...
func (b *Builder) ShiftLeft(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value1.Type()))
	}
	c := &Command{Op: OpShiftLeft, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// ShiftRight ...
func (b *Builder) ShiftRight(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value1.Type()))
	}
	c := &Command{Op: OpShiftRight, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// BitClear ...
func (b *Builder) BitClear(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value1.Type()))
	}
	c := &Command{Op: OpBitClear, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// BooleanOp ...
func (b *Builder) BooleanOp(op Operation, dest *Variable, value1, value2 Argument) *Variable {
	if value1.Type().Type != types.PrimitiveTypeBool || value2.Type().Type != types.PrimitiveTypeBool {
		panic("Not a bool")
	}
	if op != OpLogicalAnd && op != OpLogicalOr {
		panic("Wrong op")
	}
	if dest == nil {
		dest = b.newTempVariable(&types.ExprType{Type: types.PrimitiveTypeBool})
	} else if dest.Type.Type != types.PrimitiveTypeBool {
		panic("Not a bool")
	}
	c := &Command{Op: op, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// BooleanNot ...
func (b *Builder) BooleanNot(dest *Variable, value Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(&types.ExprType{Type: types.PrimitiveTypeBool})
	}
	c := &Command{Op: OpNot, Dest: []*Variable{dest}, Args: []Argument{value}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// MinusSign ...
func (b *Builder) MinusSign(dest *Variable, value Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(&types.ExprType{Type: types.PrimitiveTypeBool})
	}
	c := &Command{Op: OpMinusSign, Dest: []*Variable{dest}, Args: []Argument{value}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// BitwiseComplement ...
func (b *Builder) BitwiseComplement(dest *Variable, value Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(types.CloneExprType(value.Type()))
	}
	c := &Command{Op: OpBitwiseComplement, Dest: []*Variable{dest}, Args: []Argument{value}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// SizeOf ...
func (b *Builder) SizeOf(dest *Variable, typeArg types.Type) *Variable {
	if dest == nil {
		dest = b.newTempVariable(&types.ExprType{Type: types.PrimitiveTypeInt})
	}
	c := &Command{Op: OpSizeOf, Dest: []*Variable{dest}, Args: []Argument{}, TypeArgs: []types.Type{typeArg}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Compare ...
func (b *Builder) Compare(op Operation, dest *Variable, value1, value2 Argument) *Variable {
	t := value1.Type().Type
	if t != types.PrimitiveTypeBool && !types.IsIntegerType(t) && !types.IsFloatType(t) && !types.IsPointerType(t) && !types.IsSliceType(t) && !types.IsFuncType(t) {
		panic("Not comparable")
	}
	t = value2.Type().Type
	if t != types.PrimitiveTypeBool && !types.IsIntegerType(t) && !types.IsFloatType(t) && !types.IsPointerType(t) && !types.IsSliceType(t) && !types.IsFuncType(t) {
		panic("Not comparable")
	}
	if op != OpEqual && op != OpNotEqual && op != OpLess && op != OpGreater && op != OpLessOrEqual && op != OpGreaterOrEqual {
		panic("Wrong op")
	}
	if dest == nil {
		dest = b.newTempVariable(&types.ExprType{Type: types.PrimitiveTypeBool})
	} else if dest.Type.Type != types.PrimitiveTypeBool {
		panic("Not a bool")
	}
	c := &Command{Op: op, Dest: []*Variable{dest}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Len ...
func (b *Builder) Len(dest *Variable, slice Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(&types.ExprType{Type: types.PrimitiveTypeInt})
	}
	c := &Command{Op: OpLen, Dest: []*Variable{dest}, Args: []Argument{slice}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Cap ...
func (b *Builder) Cap(dest *Variable, slice Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(&types.ExprType{Type: types.PrimitiveTypeInt})
	}
	c := &Command{Op: OpCap, Dest: []*Variable{dest}, Args: []Argument{slice}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Append ...
func (b *Builder) Append(dest *Variable, args []Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(args[0].Type())
	}
	c := &Command{Op: OpAppend, Dest: []*Variable{dest}, Args: args, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// GroupOf ...
func (b *Builder) GroupOf(dest *Variable, arg Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(&types.ExprType{Type: types.PrimitiveTypeUintptr})
	}
	c := &Command{Op: OpGroupOf, Dest: []*Variable{dest}, Args: []Argument{arg}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Free ...
func (b *Builder) Free(arg Argument) {
	c := &Command{Op: OpFree, Args: []Argument{arg}, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
}

// Struct ...
func (b *Builder) Struct(dest *Variable, t *types.ExprType, values []Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(t)
	}
	c := &Command{Op: OpStruct, Dest: []*Variable{dest}, Args: values, Type: t, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Array ...
func (b *Builder) Array(dest *Variable, t *types.ExprType, values []Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(t)
	}
	c := &Command{Op: OpArray, Dest: []*Variable{dest}, Args: values, Type: t, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Malloc ...
func (b *Builder) Malloc(dest *Variable, t *types.ExprType) *Variable {
	if dest == nil {
		dest = b.newTempVariable(t)
	}
	c := &Command{Op: OpMalloc, Dest: []*Variable{dest}, Type: t, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// MallocSlice ...
func (b *Builder) MallocSlice(dest *Variable, t *types.ExprType, length Argument, capacity Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(t)
	}
	c := &Command{Op: OpMallocSlice, Dest: []*Variable{dest}, Args: []Argument{length, capacity}, Type: t, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// StringConcat ...
func (b *Builder) StringConcat(dest *Variable, t *types.ExprType, str1 Argument, str2 Argument) *Variable {
	if dest == nil {
		dest = b.newTempVariable(t)
	}
	c := &Command{Op: OpStringConcat, Dest: []*Variable{dest}, Args: []Argument{str1, str2}, Type: t, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

func (b *Builder) openScope() {
	c := &Command{Op: OpOpenScope, Args: nil, Type: nil, Scope: b.current.Scope, Location: b.location}
	b.current.Block = append(b.current.Block, c)
}

func (b *Builder) closeScope() {
	c := &Command{Op: OpCloseScope, Args: nil, Type: nil, Scope: b.current.Scope, Location: b.location}
	b.current.Block = append(b.current.Block, c)
}

// If ...
func (b *Builder) If(value Argument) {
	if value.Type().Type != types.PrimitiveTypeBool {
		panic("Not a bool")
	}
	c := &Command{Op: OpIf, Args: []Argument{value}, Type: nil, Scope: newScope(b.current.Scope), Location: b.location}
	b.current.Block = append(b.current.Block, c)
	b.stack = append(b.stack, b.current)
	b.current = c
	b.openScope()
}

// Else ...
func (b *Builder) Else() {
	if b.current.Op != OpIf {
		panic("Else without if")
	}
	c := &Command{Op: OpBlock, Type: nil, Scope: newScope(b.current.Scope), Location: b.location}
	b.current.Else = c
	b.stack = append(b.stack, b.current)
	b.current = c
	b.openScope()
}

// Loop ...
func (b *Builder) Loop() {
	c := &Command{Op: OpLoop, Type: nil, Scope: newScope(b.current.Scope), Location: b.location}
	b.current.Block = append(b.current.Block, c)
	b.stack = append(b.stack, b.current)
	b.current = c
	b.loopCount++
	b.openScope()
}

// LoopIter ...
func (b *Builder) LoopIter() {
	if b.current.Op != OpLoop {
		panic("Oooops")
	}
	b.iterOffset = len(b.current.Block)
}

// LoopIterEnd ...
func (b *Builder) LoopIterEnd() {
	if b.current.Op != OpLoop {
		panic("Oooops")
	}
	// Move the comands for the iter-expression into `IterBlock`
	b.current.IterBlock = make([]*Command, len(b.current.Block)-b.iterOffset)
	copy(b.current.IterBlock, b.current.Block[b.iterOffset:])
	b.current.Block = b.current.Block[:b.iterOffset]
}

// Break ...
func (b *Builder) Break(loopDepth int) {
	if loopDepth < 0 || loopDepth >= b.loopCount {
		panic("Invalid loop depth")
	}
	c := &Command{Op: OpBreak, Args: []Argument{NewIntArg(loopDepth)}, Type: nil, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
}

// Continue ...
func (b *Builder) Continue(loopDepth int) {
	if loopDepth < 0 || loopDepth >= b.loopCount {
		panic("Invalid loop depth")
	}
	c := &Command{Op: OpContinue, Args: []Argument{NewIntArg(loopDepth)}, Type: nil, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
}

// End ...
func (b *Builder) End() {
	if len(b.stack) == 0 {
		panic("End without if or loop")
	}
	b.closeScope()
	if b.current.Op == OpLoop {
		b.loopCount--
	}
	b.current = b.stack[len(b.stack)-1]
	b.stack = b.stack[0 : len(b.stack)-1]
}

// Println ...
func (b *Builder) Println(args ...Argument) {
	c := &Command{Op: OpPrintln, Args: args, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
}

// Return ...
func (b *Builder) Return(returnType types.Type, args ...Argument) {
	c := &Command{Op: OpReturn, Args: args, TypeArgs: []types.Type{returnType}, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
}

// Delete ...
func (b *Builder) Delete(typ types.Type, ptr Argument, dtor Argument) {
	_, ok := types.GetFuncType(dtor.Type().Type)
	if !ok {
		panic("Not an function")
	}
	c := &Command{Op: OpDelete, Args: []Argument{ptr, dtor}, TypeArgs: []types.Type{typ}, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
}

// Panic ...
func (b *Builder) Panic(arg Argument) {
	c := &Command{Op: OpPanic, Args: []Argument{arg}, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
}

// Assert ...
func (b *Builder) Assert(arg Argument) {
	c := &Command{Op: OpAssert, Args: []Argument{arg}, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
}

// Get ...
func (b *Builder) Get(dest *Variable, source Argument) AccessChainBuilder {
	c := &Command{Op: OpGet, Dest: []*Variable{dest}, Args: []Argument{source}, Type: source.Type(), Location: b.location, Scope: b.current.Scope}
	return AccessChainBuilder{Cmd: c, OutputType: c.Type, b: b}
}

// Set ...
func (b *Builder) Set(dest *Variable) AccessChainBuilder {
	if dest == nil {
		panic("Set with dest nil")
	}
	c := &Command{Op: OpSet, Dest: []*Variable{dest}, Args: []Argument{NewVarArg(dest)}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	return AccessChainBuilder{Cmd: c, OutputType: dest.Type, b: b}
}

// SetAndOp ...
func (b *Builder) SetAndOp(dest *Variable, op Operation) AccessChainBuilder {
	if dest == nil {
		panic("Set with dest nil")
	}
	if !IsSet(op) {
		panic("Wrong op")
	}
	c := &Command{Op: op, Dest: []*Variable{dest}, Args: []Argument{NewVarArg(dest)}, Type: dest.Type, Location: b.location, Scope: b.current.Scope}
	return AccessChainBuilder{Cmd: c, OutputType: dest.Type, b: b}
}

// GetForeignGroup ...
func (b *Builder) GetForeignGroup(dest *Variable, source Argument) AccessChainBuilder {
	c := &Command{Op: OpGetForeignGroup, Dest: []*Variable{dest}, Args: []Argument{source}, Type: source.Type(), Location: b.location, Scope: b.current.Scope}
	return AccessChainBuilder{Cmd: c, OutputType: c.Type, b: b}
}

// Take ...
func (b *Builder) Take(dest *Variable, source *Variable) AccessChainBuilder {
	c := &Command{Op: OpTake, Dest: []*Variable{dest, source}, Args: []Argument{NewVarArg(source)}, Type: source.Type, Location: b.location, Scope: b.current.Scope}
	return AccessChainBuilder{Cmd: c, OutputType: c.Type, b: b}
}

// Call ...
func (b *Builder) Call(dest []*Variable, args []Argument) []*Variable {
	ft, ok := types.GetFuncType(args[0].Type().Type)
	if !ok {
		panic("Not an function")
	}
	if dest == nil {
		for _, p := range ft.Out.Params {
			dest = append(dest, b.newTempVariable(types.NewExprType(p.Type)))
		}
	}
	if len(dest) != len(ft.Out.Params) {
		panic("Return parameter mismatch")
	}
	c := &Command{Op: OpCall, Dest: dest, Args: args, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// DefineVariable ...
func (b *Builder) DefineVariable(name string, t *types.ExprType) *Variable {
	v := b.newVariable(t, name)
	c := &Command{Op: OpDefVariable, Dest: []*Variable{v}, Type: t, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return v
}

// DefineGlobalVariable ...
func (b *Builder) DefineGlobalVariable(name string, t *types.ExprType) *Variable {
	v := b.newVariable(t, name)
	v.Kind = VarGlobal
	c := &Command{Op: OpDefVariable, Dest: []*Variable{v}, Type: t, Location: b.location, Scope: b.current.Scope}
	b.current.Block = append(b.current.Block, c)
	return v
}

// Finalize ...
func (b *Builder) Finalize() {
	if len(b.stack) != 0 {
		panic("Finalize before all if's and loop's are closed")
	}
	b.closeScope()
}

func (b *Builder) newVariable(t *types.ExprType, name string) *Variable {
	v := &Variable{Name: name, Type: t, Scope: b.current.Scope}
	v.Original = v
	b.Func.Vars = append(b.Func.Vars, v)
	return v
}

func (b *Builder) newTempVariable(t *types.ExprType) *Variable {
	v := &Variable{Name: "%" + strconv.Itoa(len(b.Func.Vars)), Type: t, Scope: b.current.Scope}
	v.Original = v
	v.Kind = VarTemporary
	b.Func.Vars = append(b.Func.Vars, v)
	return v
}

// Slice ...
func (ab AccessChainBuilder) Slice(left, right Argument, resultType *types.ExprType) AccessChainBuilder {
	if left.Flags != ArgumentIsMissing && left.Type().Type != types.PrimitiveTypeInt {
		panic("Array index is not an int")
	}
	if right.Flags != ArgumentIsMissing && right.Type().Type != types.PrimitiveTypeInt {
		panic("Array index is not an int")
	}
	// Append the arguments to the access chain command
	ab.Cmd.Args = append(ab.Cmd.Args, left)
	ab.Cmd.Args = append(ab.Cmd.Args, right)
	if types.IsSliceType(ab.OutputType.Type) {
		ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessSlice, InputType: ab.OutputType, OutputType: resultType})
		ab.OutputType = resultType
	} else if types.IsArrayType(ab.OutputType.Type) {
		ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessSlice, InputType: ab.OutputType, OutputType: resultType})
		ab.OutputType = resultType
	} else {
		panic("Neither a slice nor an array. Cannot slice it")
	}
	if ab.Cmd.Op == OpGet || ab.Cmd.Op == OpGetForeignGroup || ab.Cmd.Op == OpTake {
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// SliceIndex ...
func (ab AccessChainBuilder) SliceIndex(index Argument, resultType *types.ExprType) AccessChainBuilder {
	if !types.IsSliceType(ab.OutputType.Type) {
		panic("Not a slice")
	}
	if index.Type().Type != types.PrimitiveTypeInt {
		panic("Slice index is not an int")
	}
	ab.Cmd.Args = append(ab.Cmd.Args, index)
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessSliceIndex, InputType: ab.OutputType, OutputType: resultType})
	ab.OutputType = resultType
	if ab.Cmd.Op == OpGet || ab.Cmd.Op == OpGetForeignGroup {
		ab.Cmd.Type = ab.OutputType
	} else if IsSet(ab.Cmd.Op) {
		// The access chain does not modify the Args[0] variable. Do not set a Dest[0].
		// In this case, the destination variable is not known or the destination is on the heap anyway.
		ab.Cmd.Dest = nil
	} else if ab.Cmd.Op == OpTake {
		ab.Cmd.Type = ab.OutputType
		ab.Cmd.Dest[1] = nil
	}
	return ab
}

// ArrayIndex ...
func (ab AccessChainBuilder) ArrayIndex(arg Argument, resultType *types.ExprType) AccessChainBuilder {
	if !types.IsArrayType(ab.OutputType.Type) {
		panic("Not an array")
	}
	if arg.Type().Type != types.PrimitiveTypeInt {
		panic("Array index is not an int")
	}
	ab.Cmd.Args = append(ab.Cmd.Args, arg)
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessArrayIndex, InputType: ab.OutputType, OutputType: resultType})
	ab.OutputType = resultType
	if ab.Cmd.Op == OpGet || ab.Cmd.Op == OpGetForeignGroup || ab.Cmd.Op == OpTake {
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// StructField ...
// Accesses the field in a struct value (as in val.field in C)
func (ab AccessChainBuilder) StructField(field *types.StructField, resultType *types.ExprType) AccessChainBuilder {
	if _, ok := types.GetStructType(ab.OutputType.Type); !ok {
		if _, ok := types.GetUnionType(ab.OutputType.Type); !ok {
			panic("Not a struct")
		}
	}
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessStruct, Field: field, InputType: ab.OutputType, OutputType: resultType})
	ab.OutputType = resultType
	if ab.Cmd.Op == OpGet || ab.Cmd.Op == OpGetForeignGroup || ab.Cmd.Op == OpTake {
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// PointerStructField ...
// Accesses a field in a struct via a pointer (as in ptr->field in C)
func (ab AccessChainBuilder) PointerStructField(field *types.StructField, resultType *types.ExprType) AccessChainBuilder {
	p, ok := types.GetPointerType(ab.OutputType.Type)
	if !ok {
		panic("Not an pointer")
	}
	if _, ok := types.GetStructType(p.ElementType); !ok {
		if _, ok := types.GetUnionType(p.ElementType); !ok {
			panic("Not a struct")
		}
	}
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessPointerToStruct, Field: field, InputType: ab.OutputType, OutputType: resultType})
	ab.OutputType = resultType
	if ab.Cmd.Op == OpGet || ab.Cmd.Op == OpGetForeignGroup {
		ab.Cmd.Type = ab.OutputType
	} else if IsSet(ab.Cmd.Op) {
		// The access chain does not modify the Args[0] variable. Do not set a Dest[0].
		// In this case, the destination variable is not known or the destination is on the heap anyway.
		ab.Cmd.Dest = nil
	} else if ab.Cmd.Op == OpTake {
		ab.Cmd.Dest[1] = nil
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// DereferencePointer ...
func (ab AccessChainBuilder) DereferencePointer(resultType *types.ExprType) AccessChainBuilder {
	if !types.IsPointerType(ab.OutputType.Type) {
		panic("Not a pointer")
	}
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessDereferencePointer, InputType: ab.OutputType, OutputType: resultType})
	ab.OutputType = resultType
	if IsSet(ab.Cmd.Op) {
		// The access chain does not modify the Args[0] variable. Do not set a Dest[0].
		// In this case, the destination variable is not known or the destination is on the heap anyway.
		ab.Cmd.Dest = nil
	} else if ab.Cmd.Op == OpTake {
		ab.Cmd.Dest[1] = nil
		ab.Cmd.Type = ab.OutputType
	} else if ab.Cmd.Op == OpGet || ab.Cmd.Op == OpGetForeignGroup {
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// AddressOf ...
func (ab AccessChainBuilder) AddressOf(resultType *types.ExprType) AccessChainBuilder {
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessAddressOf, InputType: ab.OutputType, OutputType: resultType})
	ab.OutputType = resultType
	if IsSet(ab.Cmd.Op) {
		// The access chain does not modify the Args[0] variable. Do not set a Dest[0].
		// In this case, the destination variable is not known or the destination is on the heap anyway.
		ab.Cmd.Dest = nil
	} else if ab.Cmd.Op == OpTake {
		ab.Cmd.Dest[1] = nil
		ab.Cmd.Type = ab.OutputType
	} else if ab.Cmd.Op == OpGet || ab.Cmd.Op == OpGetForeignGroup {
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// Cast ...
func (ab AccessChainBuilder) Cast(resultType *types.ExprType) AccessChainBuilder {
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessCast, InputType: ab.OutputType, OutputType: resultType})
	ab.OutputType = resultType
	if ab.Cmd.Op == OpGet || ab.Cmd.Op == OpGetForeignGroup {
		ab.Cmd.Type = ab.OutputType
	} else if IsSet(ab.Cmd.Op) {
		// The access chain does not modify the Args[0] variable. Do not set a Dest[0].
		// In this case, the destination variable is not known or the destination is on the heap anyway.
		ab.Cmd.Dest = nil
	} else if ab.Cmd.Op == OpTake {
		ab.Cmd.Dest[1] = nil
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// SetValue ...
// Terminates the access chain building.
func (ab AccessChainBuilder) SetValue(value Argument) {
	if !IsSet(ab.Cmd.Op) {
		panic("Not a set operation")
	}
	ab.b.current.Block = append(ab.b.current.Block, ab.Cmd)
	// If there is no access chain, generate a SetVariable instruction instead
	if ab.Cmd.Op == OpSet && len(ab.Cmd.AccessChain) == 0 {
		ab.Cmd.Op = OpSetVariable
		ab.Cmd.Args = []Argument{value}
	} else {
		ab.Cmd.Args = append(ab.Cmd.Args, value)
	}
}

// GetValue ...
// Terminates the access chain building.
func (ab AccessChainBuilder) GetValue() *Variable {
	if ab.Cmd.Op != OpGet && ab.Cmd.Op != OpGetForeignGroup {
		panic("Not a get operation")
	}
	if ab.Cmd.Op == OpGetForeignGroup {
		ab.Cmd.Type = types.NewExprType(types.PrimitiveTypeUintptr)
	}
	ab.b.current.Block = append(ab.b.current.Block, ab.Cmd)
	if ab.Cmd.Dest[0] == nil {
		ab.Cmd.Dest = []*Variable{ab.b.newTempVariable(ab.Cmd.Type)}
	}
	return ab.Cmd.Dest[0]
}

// GetVoid terminates the access chain building.
// Call it instead of `GetValue` when the access chain returns void, which can be
// the case for function calls.
func (ab AccessChainBuilder) GetVoid() {
	if ab.Cmd.Op != OpGet {
		panic("Not a get operation")
	}
	ab.b.current.Block = append(ab.b.current.Block, ab.Cmd)
	if len(ab.Cmd.Dest) != 0 && ab.Cmd.Dest[0] != nil {
		panic("Oooops")
	}
}

// Increment ...
// Terminates the access chain building.
func (ab AccessChainBuilder) Increment() {
	if ab.Cmd.Op != OpSet {
		panic("Not a set operation")
	}
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessInc, InputType: ab.OutputType, OutputType: ab.OutputType})
	ab.b.current.Block = append(ab.b.current.Block, ab.Cmd)
}

// Decrement ...
// Terminates the access chain building.
func (ab AccessChainBuilder) Decrement() {
	if ab.Cmd.Op != OpSet {
		panic("Not a set operation")
	}
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessDec, InputType: ab.OutputType, OutputType: ab.OutputType})
	ab.b.current.Block = append(ab.b.current.Block, ab.Cmd)
}

// TakeValue ...
// Terminates the access chain building.
func (ab AccessChainBuilder) TakeValue() *Variable {
	if ab.Cmd.Op != OpTake {
		panic("Not a take operation")
	}
	ab.b.current.Block = append(ab.b.current.Block, ab.Cmd)
	if ab.Cmd.Dest[0] == nil {
		ab.Cmd.Dest[0] = ab.b.newTempVariable(ab.Cmd.Type)
	}
	return ab.Cmd.Dest[0]
}

// IsSet ...
func IsSet(op Operation) bool {
	switch op {
	case OpSet,
		OpSetAndAdd,
		OpSetAndSub,
		OpSetAndMul,
		OpSetAndDiv,
		OpSetAndRemainder,
		OpSetAndBinaryAnd,
		OpSetAndBinaryOr,
		OpSetAndBinaryXor,
		OpSetAndBitClear,
		OpSetAndShiftLeft,
		OpSetAndShiftRight:
		return true
	}
	return false
}
