package ircode

import (
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
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
	OutputType IType
	Cmd        *Command
	b          *Builder
}

// NewBuilder ...
func NewBuilder(name string) *Builder {
	b := &Builder{scopeCount: 1}
	b.Func = NewFunction(name)
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
	b.compareTypes(dest.Type, value.Type())
	c := &Command{Op: OpSetVariable, Dest: []VariableUsage{{Var: dest}}, Args: []Argument{value}, Type: dest.Type, Location: b.location}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// Add ...
func (b *Builder) Add(dest *Variable, value1, value2 Argument) *Variable {
	if dest == nil {
		dest = b.newVariable(value1.Type())
	} else {
		b.compareTypes(dest.Type, value1.Type())
	}
	b.compareTypes(value1.Type(), value2.Type())
	c := &Command{Op: OpAdd, Dest: []VariableUsage{{Var: dest}}, Args: []Argument{value1, value2}, Type: dest.Type, Location: b.location}
	b.current.Block = append(b.current.Block, c)
	return dest
}

// If ...
func (b *Builder) If(value Argument) {
	b.compareTypes(BoolType, value.Type())
	c := &Command{Op: OpIf, Args: []Argument{value}, Type: VoidType, Scope: newScope(b.current.Scope), Location: b.location}
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
	c := &Command{Op: OpBlock, Type: VoidType, Scope: newScope(b.current.Scope), Location: b.location}
	b.scopeCount++
	c.Scope.ID = b.scopeCount
	b.current.Else = c
	b.stack = append(b.stack, b.current)
	b.current = c
}

// Loop ...
func (b *Builder) Loop() {
	c := &Command{Op: OpLoop, Type: VoidType, Scope: newScope(b.current.Scope), Location: b.location}
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
	c := &Command{Op: OpBreak, Args: []Argument{NewIntArg(loopDepth)}, Type: VoidType, Location: b.location}
	b.current.Block = append(b.current.Block, c)
}

// Continue ...
func (b *Builder) Continue(loopDepth int) {
	if loopDepth < 0 || loopDepth >= b.loopCount {
		panic("Invalid loop depth")
	}
	c := &Command{Op: OpContinue, Args: []Argument{NewIntArg(loopDepth)}, Type: VoidType, Location: b.location}
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
	b.current.Block = append(b.current.Block, c)
	return AccessChainBuilder{Cmd: c, OutputType: c.Type, b: b}
}

// Set ...
func (b *Builder) Set(dest *Variable) AccessChainBuilder {
	if dest == nil {
		panic("Set with dest nil")
	}
	c := &Command{Op: OpSet, Dest: []VariableUsage{{Var: dest}}, Args: []Argument{NewVarArg(dest)}, Type: dest.Type, Location: b.location}
	b.current.Block = append(b.current.Block, c)
	return AccessChainBuilder{Cmd: c, OutputType: dest.Type, b: b}
}

// DefineVariable ...
func (b *Builder) DefineVariable(t IType, name string) *Variable {
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

func (b *Builder) newVariable(t IType) *Variable {
	v := &Variable{Name: "%" + strconv.Itoa(len(b.Func.vars)), Type: t, Scope: b.current.Scope}
	v.Original = v
	b.Func.vars = append(b.Func.vars, v)
	return v
}

func (b *Builder) compareTypes(t1, t2 IType) {
	if CompareTypes(t1, t2) {
		return
	}
	panic("Incompatible types")
}

// Slice ...
func (ab AccessChainBuilder) Slice(left, right Argument, mode PointerMode) AccessChainBuilder {
	if left.Type() != IntType || right.Type() != IntType {
		panic("Array index is not an int")
	}
	ab.Cmd.Args = append(ab.Cmd.Args, left)
	ab.Cmd.Args = append(ab.Cmd.Args, right)
	st, ok := ab.OutputType.(*SliceType)
	if ok {
		// TODO: Check whether size of the slice is static
		outType := NewSliceType(st.Array.Element, -1, mode)
		ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessSlice, InputType: ab.OutputType, OutputType: outType})
		ab.OutputType = outType
	} else if at, ok := ab.OutputType.(*ArrayType); ok {
		outType := NewSliceType(at.Element, at.Size, mode)
		ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessSlice, InputType: ab.OutputType, OutputType: outType})
		ab.OutputType = outType
	} else {
		panic("Neither a slice nor an array. Cannot slice it")
	}
	if ab.Cmd.Op == OpGet {
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// SliceIndex ...
func (ab AccessChainBuilder) SliceIndex(arg Argument) AccessChainBuilder {
	var st *SliceType
	var ok bool
	st, ok = ab.OutputType.(*SliceType)
	if !ok {
		panic("Not a slice")
	}
	if arg.Type() != IntType {
		panic("Slice index is not an int")
	}
	ab.Cmd.Args = append(ab.Cmd.Args, arg)
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessSliceIndex, InputType: ab.OutputType, OutputType: st.Array.Element})
	ab.OutputType = st.Array.Element
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

// ArrayIndex ...
func (ab AccessChainBuilder) ArrayIndex(arg Argument) AccessChainBuilder {
	var at *ArrayType
	var ok bool
	at, ok = ab.OutputType.(*ArrayType)
	if !ok {
		panic("Not an array")
	}
	if arg.Type() != IntType {
		panic("Array index is not an int")
	}
	ab.Cmd.Args = append(ab.Cmd.Args, arg)
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessArrayIndex, InputType: ab.OutputType, OutputType: at.Element})
	ab.OutputType = at.Element
	if ab.Cmd.Op == OpGet {
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// Struct ...
func (ab AccessChainBuilder) Struct(field *StructField) AccessChainBuilder {
	return ab.accessStruct(field, AccessStruct)
}

// PointerToStruct ...
func (ab AccessChainBuilder) PointerToStruct(field *StructField) AccessChainBuilder {
	pt, ok := ab.OutputType.(*PointerType)
	if !ok {
		panic("Not a pointer")
	}
	ab.OutputType = pt.Element
	if ab.Cmd.Op == OpSet {
		// The access chain does not modify the Args[0] variable.,Do not set a Dest[0].
		// In this case, the destination variable is not known or the destination is on the heap anyway.
		ab.Cmd.Dest = nil
	}
	return ab.accessStruct(field, AccessPointerToStruct)
}

func (ab AccessChainBuilder) accessStruct(field *StructField, kind AccessKind) AccessChainBuilder {
	var st *StructType
	var ok bool
	st, ok = ab.OutputType.(*StructType)
	if !ok {
		panic("Not a struct")
	}
	index := st.FieldIndex(field)
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: kind, FieldIndex: index, InputType: ab.OutputType, OutputType: field.Type})
	ab.OutputType = field.Type
	if ab.Cmd.Op == OpGet {
		ab.Cmd.Type = ab.OutputType
	}
	return ab
}

// DereferencePointer ...
func (ab AccessChainBuilder) DereferencePointer() AccessChainBuilder {
	pt, ok := ab.OutputType.(*PointerType)
	if !ok {
		panic("Not a pointer")
	}
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessDereferencePointer, InputType: ab.OutputType, OutputType: pt.Element})
	ab.OutputType = pt.Element
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
func (ab AccessChainBuilder) AddressOf(mode PointerMode) AccessChainBuilder {
	outType := NewPointerType(ab.OutputType, mode)
	ab.Cmd.AccessChain = append(ab.Cmd.AccessChain, AccessChainElement{Kind: AccessDereferencePointer, InputType: ab.OutputType, OutputType: outType})
	ab.OutputType = outType
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
	resultType := ab.Cmd.Type
	if len(ab.Cmd.AccessChain) != 0 {
		resultType = ab.Cmd.AccessChain[len(ab.Cmd.AccessChain)-1].OutputType
	}
	if !CompareTypes(resultType, value.Type()) {
		println(resultType.ToString())
		println(value.Type().ToString())
		panic("Type mismatch")
	}
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
	if ab.Cmd.Dest[0].Var == nil {
		ab.Cmd.Dest[0].Var = ab.b.newVariable(ab.Cmd.Type)
	}
	return ab.Cmd.Dest[0].Var
}
