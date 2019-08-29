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
	scopeCount    int
	locationStack []errlog.LocationRange
	location      errlog.LocationRange
}

// AccessChainBuilder ...
type AccessChainBuilder struct {
	OutputType *types.ExprType
	Cmd        *Command
	b          *Builder
}

// NewBuilder ...
func NewBuilder(name string, t *types.FuncType) *Builder {
	b := &Builder{scopeCount: 1}
	b.Func = NewFunction(name, t)
	b.current = &b.Func.Body
	b.current.Scope.ID = b.scopeCount
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

// SetLocation ...
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
		dest = b.newVariable(value.Type())
	}
	// TODO: Safety b.compareTypes(dest.Type, value.Type())
	c := &Command{Op: OpSetVariable, Dest: []VariableUsage{{Var: dest}}, Args: []Argument{value}, Type: dest.Type, Location: b.location}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Add ...
func (b *Builder) Add(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newVariable(value1.Type())
	} else {
		// TODO: Safety b.compareTypes(dest.Type, value1.Type())
	}
	// TODO: Safety b.compareTypes(value1.Type(), value2.Type())
	c := &Command{Op: OpAdd, Dest: []VariableUsage{{Var: dest}}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location}
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
		dest = b.newVariable(&types.ExprType{Type: types.PrimitiveTypeBool})
	} else if dest.Type.Type != types.PrimitiveTypeBool {
		panic("Not a bool")
	}
	c := &Command{Op: op, Dest: []VariableUsage{{Var: dest}}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Compare ...
func (b *Builder) Compare(op Operation, dest *Variable, value1, value2 Argument) *Variable {
	t := value1.Type().Type
	if t != types.PrimitiveTypeBool && !types.IsIntegerType(t) && !types.IsFloatType(t) && !types.IsPointerType(t) && !types.IsSliceType(t) {
		panic("Not comparable")
	}
	t = value2.Type().Type
	if t != types.PrimitiveTypeBool && !types.IsIntegerType(t) && !types.IsFloatType(t) && !types.IsPointerType(t) && !types.IsSliceType(t) {
		panic("Not comparable")
	}
	if op != OpEqual && op != OpNotEqual && op != OpLess && op != OpGreater && op != OpLessOrEqual && op != OpGreaterOrEqual {
		panic("Wrong op")
	}
	if dest == nil {
		dest = b.newVariable(&types.ExprType{Type: types.PrimitiveTypeBool})
	} else if dest.Type.Type != types.PrimitiveTypeBool {
		panic("Not a bool")
	}
	c := &Command{Op: op, Dest: []VariableUsage{{Var: dest}}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// If ...
func (b *Builder) If(value Argument) {
	if value.Type().Type != types.PrimitiveTypeBool {
		panic("Not a bool")
	}
	c := &Command{Op: OpIf, Args: []Argument{value}, Type: nil, Scope: newScope(b.current.Scope), Location: b.location}
	b.scopeCount++
	c.Scope.ID = b.scopeCount
	b.current.Block = append(b.current.Block, c)
	b.stack = append(b.stack, b.current)
	b.current = c
}

// Else ...
func (b *Builder) Else() {
	if b.current.Op != OpIf {
		panic("Else without if")
	}
	c := &Command{Op: OpBlock, Type: nil, Scope: newScope(b.current.Scope), Location: b.location}
	b.scopeCount++
	c.Scope.ID = b.scopeCount
	b.current.Else = c
	b.stack = append(b.stack, b.current)
	b.current = c
}

// Loop ...
func (b *Builder) Loop() {
	c := &Command{Op: OpLoop, Type: nil, Scope: newScope(b.current.Scope), Location: b.location}
	b.scopeCount++
	c.Scope.ID = b.scopeCount
	b.current.Block = append(b.current.Block, c)
	b.stack = append(b.stack, b.current)
	b.current = c
	b.loopCount++
}

// Break ...
func (b *Builder) Break(loopDepth int) {
	if loopDepth < 0 || loopDepth >= b.loopCount {
		panic("Invalid loop depth")
	}
	c := &Command{Op: OpBreak, Args: []Argument{NewIntArg(loopDepth)}, Type: nil, Location: b.location}
	b.current.Block = append(b.current.Block, c)
}

// Continue ...
func (b *Builder) Continue(loopDepth int) {
	if loopDepth < 0 || loopDepth >= b.loopCount {
		panic("Invalid loop depth")
	}
	c := &Command{Op: OpContinue, Args: []Argument{NewIntArg(loopDepth)}, Type: nil, Location: b.location}
	b.current.Block = append(b.current.Block, c)
}

// End ...
func (b *Builder) End() {
	if len(b.stack) == 0 {
		panic("End without if or loop")
	}
	if b.current.Op == OpLoop {
		b.loopCount--
	}
	b.current = b.stack[len(b.stack)-1]
	b.stack = b.stack[0 : len(b.stack)-1]
}

// Println ...
func (b *Builder) Println(args ...Argument) {
	c := &Command{Op: OpPrintln, Args: args, Location: b.location}
	b.current.Block = append(b.current.Block, c)
}

// Get ...
func (b *Builder) Get(dest *Variable, source Argument) AccessChainBuilder {
	c := &Command{Op: OpGet, Dest: []VariableUsage{{Var: dest}}, Args: []Argument{source}, Type: source.Type(), Location: b.location}
	return AccessChainBuilder{Cmd: c, OutputType: c.Type, b: b}
}

// Set ...
func (b *Builder) Set(dest *Variable) AccessChainBuilder {
	if dest == nil {
		panic("Set with dest nil")
	}
	c := &Command{Op: OpSet, Dest: []VariableUsage{{Var: dest}}, Args: []Argument{NewVarArg(dest)}, Type: dest.Type, Location: b.location}
	return AccessChainBuilder{Cmd: c, OutputType: dest.Type, b: b}
}

// DefineVariable ...
func (b *Builder) DefineVariable(name string, t *types.ExprType) *Variable {
	v := b.newVariable(t)
	v.Name = name
	c := &Command{Op: OpDefVariable, Dest: []VariableUsage{{Var: v}}, Type: t, Location: b.location}
	b.current.Block = append(b.current.Block, c)
	return v
}

// Finalize ...
func (b *Builder) Finalize() {
	if len(b.stack) != 0 {
		panic("Finalize before all if's and loop's are closed")
	}
}

func (b *Builder) newVariable(t *types.ExprType) *Variable {
	v := &Variable{Name: "%" + strconv.Itoa(len(b.Func.vars)), Type: t, Scope: b.current.Scope}
	v.Original = v
	b.Func.vars = append(b.Func.vars, v)
	return v
}

// Slice ...
func (ab AccessChainBuilder) Slice(left, right Argument, resultType *types.ExprType) AccessChainBuilder {
	if left.Type().Type != types.PrimitiveTypeInt || right.Type().Type != types.PrimitiveTypeInt {
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
	if ab.Cmd.Op == OpGet {
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
	if ab.Cmd.Op == OpGet {
		ab.Cmd.Type = ab.OutputType
	}
	if ab.Cmd.Op == OpSet {
		// The access chain does not modify the Args[0] variable. Do not set a Dest[0].
		// In this case, the destination variable is not known or the destination is on the heap anyway.
		ab.Cmd.Dest = nil
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
	if ab.Cmd.Op == OpGet {
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// StructField ...
// Accesses the field in a struct value (as in val.field in C)
func (ab AccessChainBuilder) StructField(field *types.StructField, resultType *types.ExprType) AccessChainBuilder {
	st, ok := types.GetStructType(ab.OutputType.Type)
	if !ok {
		panic("Not a struct")
	}
	index := st.FieldIndex(field)
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessStruct, FieldIndex: index, InputType: ab.OutputType, OutputType: resultType})
	ab.OutputType = resultType
	if ab.Cmd.Op == OpGet {
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
	st, ok := types.GetStructType(p)
	if !ok {
		panic("Not a struct")
	}
	ab.OutputType = resultType
	index := st.FieldIndex(field)
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessPointerToStruct, FieldIndex: index, InputType: ab.OutputType, OutputType: resultType})
	ab.OutputType = resultType
	if ab.Cmd.Op == OpGet {
		ab.Cmd.Type = ab.OutputType
	}
	if ab.Cmd.Op == OpSet {
		// The access chain does not modify the Args[0] variable.,Do not set a Dest[0].
		// In this case, the destination variable is not known or the destination is on the heap anyway.
		ab.Cmd.Dest = nil
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
	if ab.Cmd.Op == OpSet {
		// The access chain does not modify the Args[0] variable.,Do not set a Dest[0].
		// In this case, the destination variable is not known or the destination is on the heap anyway.
		ab.Cmd.Dest = nil
	}
	if ab.Cmd.Op == OpGet {
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// AddressOf ...
func (ab AccessChainBuilder) AddressOf(resultType *types.ExprType) AccessChainBuilder {
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessDereferencePointer, InputType: ab.OutputType, OutputType: resultType})
	ab.OutputType = resultType
	if ab.Cmd.Op == OpSet {
		// The access chain does not modify the Args[0] variable.,Do not set a Dest[0].
		// In this case, the destination variable is not known or the destination is on the heap anyway.
		ab.Cmd.Dest = nil
	}
	if ab.Cmd.Op == OpGet {
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// SetValue ...
// Terminates the access chain building.
func (ab AccessChainBuilder) SetValue(value Argument) {
	if ab.Cmd.Op != OpSet {
		panic("Not a set operation")
	}
	ab.b.current.Block = append(ab.b.current.Block, ab.Cmd)
	// If there is no access chain, generate a SetVariable instruction instead
	if len(ab.Cmd.AccessChain) == 0 {
		ab.Cmd.Op = OpSetVariable
		ab.Cmd.Args = []Argument{value}
	} else {
		ab.Cmd.Args = append(ab.Cmd.Args, value)
	}
}

// GetValue ...
// Terminates the access chain building.
func (ab AccessChainBuilder) GetValue() *Variable {
	if ab.Cmd.Op != OpGet {
		panic("Not a get operation")
	}
	ab.b.current.Block = append(ab.b.current.Block, ab.Cmd)
	if ab.Cmd.Dest[0].Var == nil {
		ab.Cmd.Dest[0].Var = ab.b.newVariable(ab.Cmd.Type)
	}
	return ab.Cmd.Dest[0].Var
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
