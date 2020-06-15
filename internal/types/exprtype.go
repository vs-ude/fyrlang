package types

import (
	"math/big"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

// ExprType represents type information about an expression.
// It is more powerful than type alone, because it can store values in case the expression is constant.
// Furthermore, it exposes mutability and group specifiers by removing them from the Type hierarchy.
type ExprType struct {
	// Instances of GroupedType or MutableType are removed for convenience and
	// factored into the PointerDestMutable and PointerDestGroup properties.
	Type Type
	// Mutable defines whether the value of the expression is mutable.
	Mutable  bool
	Volatile bool
	// Unsafe is true if the value of the expression has been obtained via
	// dereferencing of an unsafe pointer.
	Unsafe bool
	// PointerDestMutable defines the mutability of the value being pointed to.
	// This is required, because the type system distinguishes between the mutability of a pointer
	// and the mutability of the value it is pointing to.
	PointerDestMutable bool
	// The group specifier that applies to the expression value (or null if none was specified).
	GroupSpecifier *GroupSpecifier
	// The group specifier that applies to the  values being pointed to (or null if none was specified).
	// This is required, because a pointer on the stack belongs to a stack-group,
	// but it might point to an object of another group.
	PointerDestGroupSpecifier *GroupSpecifier
	StringValue               string
	RuneValue                 rune
	IntegerValue              *big.Int
	FloatValue                *big.Float
	BoolValue                 bool
	ArrayValue                []*ExprType
	StructValue               map[string]*ExprType
	FuncValue                 *Func
	NamespaceValue            *Namespace
	TypeConversionValue       TypeConversion
	// HasValue is true if one of the *Value properties holds a value.
	// This does not imply that the expression has a constant value, because
	// an ArrayValue may contain an ExprType that has no value.
	// Use IsConstant() to determine whether an expression is constant.
	HasValue bool
}

// TypeConversion ...
type TypeConversion int

const (
	// ConvertStringToPointer ...
	ConvertStringToPointer TypeConversion = 1 + iota
	// ConvertPointerToPointer ...
	ConvertPointerToPointer
	// ConvertIntegerToPointer ...
	ConvertIntegerToPointer
	// ConvertPointerToInteger ...
	ConvertPointerToInteger
	// ConvertSliceToPointer ...
	ConvertSliceToPointer
	// ConvertPointerToSlice ...
	ConvertPointerToSlice
	// ConvertStringToByteSlice ...
	ConvertStringToByteSlice
	// ConvertPointerToString ...
	ConvertPointerToString
	// ConvertByteSliceToString ...
	ConvertByteSliceToString
	// ConvertIntegerToInteger ...
	ConvertIntegerToInteger
	// ConvertFloatToInteger ...
	ConvertFloatToInteger
	// ConvertBoolToInteger ...
	ConvertBoolToInteger
	// ConvertRuneToInteger ...
	ConvertRuneToInteger
	// ConvertIntegerToFloat ...
	ConvertIntegerToFloat
	// ConvertFloatToFloat ...
	ConvertFloatToFloat
	// ConvertIntegerToBool ...
	ConvertIntegerToBool
	// ConvertIntegerToRune ...
	ConvertIntegerToRune
	// ConvertIllegal ...
	ConvertIllegal
)

// Clone ...
func (et *ExprType) Clone() *ExprType {
	result := &ExprType{}
	result.Type = et.Type
	result.Mutable = et.Mutable
	result.Unsafe = et.Unsafe
	result.PointerDestMutable = et.PointerDestMutable
	result.Volatile = et.Volatile
	result.GroupSpecifier = et.GroupSpecifier
	result.PointerDestGroupSpecifier = et.PointerDestGroupSpecifier
	result.StringValue = et.StringValue
	result.RuneValue = et.RuneValue
	result.IntegerValue = et.IntegerValue
	result.FloatValue = et.FloatValue
	result.BoolValue = et.BoolValue
	result.ArrayValue = et.ArrayValue
	result.StructValue = et.StructValue
	result.HasValue = et.HasValue
	return result
}

// IsConstant ...
func (et *ExprType) IsConstant() bool {
	if !et.HasValue {
		return false
	}
	// Only null-slices are constants
	if _, ok := et.Type.(*SliceType); ok {
		if et.IntegerValue != nil {
			return true
		}
		return false
	}
	// Only null-pointers are constants
	if _, ok := et.Type.(*PointerType); ok {
		if et.IntegerValue != nil {
			return true
		}
		return false
	}
	if len(et.ArrayValue) != 0 {
		for _, a := range et.ArrayValue {
			if !a.IsConstant() {
				return false
			}
		}
	} else if len(et.StructValue) != 0 {
		for _, a := range et.ArrayValue {
			if !a.IsConstant() {
				return false
			}
		}
	}
	return true
}

// IsNullValue returns true if the expression is a null pointer or null slice.
func (et *ExprType) IsNullValue() bool {
	if _, ok := et.Type.(*SliceType); ok {
		if et.IntegerValue != nil {
			return true
		}
		return false
	}
	if _, ok := et.Type.(*PointerType); ok {
		if et.IntegerValue != nil {
			return true
		}
		return false
	}
	if _, ok := et.Type.(*FuncType); ok {
		return et.FuncValue == nil
	}
	return false
}

// ToType ...
func (et *ExprType) ToType() Type {
	t := et.Type
	if et.PointerDestMutable || et.Volatile {
		t = &MutableType{TypeBase: TypeBase{location: t.Location(), pkg: t.Package()}, Type: t, Mutable: et.PointerDestMutable, Volatile: et.Volatile}
	}
	if et.PointerDestGroupSpecifier != nil {
		t = &GroupedType{TypeBase: TypeBase{location: t.Location(), pkg: t.Package()}, GroupSpecifier: et.PointerDestGroupSpecifier, Type: t}
	}
	return t
}

func exprType(n parser.Node) *ExprType {
	return n.TypeAnnotation().(*ExprType)
}

// NewExprType ...
func NewExprType(t Type) *ExprType {
	return makeExprType(t)
}

// makeExprType sets the Type property.
// If the Type `t` is MutableType or GroupType, these are dropped and PointerDestMutable/PointerDestGroup are set accordingly.
func makeExprType(t Type) *ExprType {
	e := &ExprType{}
	for {
		switch t2 := t.(type) {
		case *MutableType:
			e.PointerDestMutable = t2.Mutable
			e.Volatile = t2.Volatile
			t = t2.Type
			continue
		case *GroupedType:
			e.PointerDestGroupSpecifier = t2.GroupSpecifier
			t = t2.Type
			continue
		}
		break
	}
	e.Type = t
	return e
}

// DeriveExprType acts like makeExprType.
// However, before it analyzes `t`, it copies the Mutable, PointerDestMutable, Volatile, Unsafe and PointerDestGroupSpecifier properties from `et`.
// For example if `et` is the type of an array expression and `t` is the type of the array elements, then DeriveExprType
// can be used to derive the ExprType of array elements.
func DeriveExprType(et *ExprType, t Type) *ExprType {
	e := &ExprType{Mutable: et.Mutable, PointerDestMutable: et.PointerDestMutable, Unsafe: et.Unsafe, Volatile: et.Volatile, GroupSpecifier: nil /*et.Group*/, PointerDestGroupSpecifier: et.PointerDestGroupSpecifier}
	for {
		switch t2 := t.(type) {
		case *MutableType:
			e.PointerDestMutable = t2.Mutable
			e.Volatile = t2.Volatile
			t = t2.Type
			continue
		case *GroupedType:
			e.PointerDestGroupSpecifier = t2.GroupSpecifier
			t = t2.Type
			continue
		}
		break
	}
	e.Type = t
	return e
}

// DerivePointerExprType acts like makeExprType.
// However, before it analyzes `t`, it copies the PointerDestGroup property from `et` and set PointerDestMutable to `false`.
// The Mutable and Group properties are set to `et.PointerDestMutable` and `et.PointerDestGroup`.
// The PointerDestMutable property becomes true if `et.PointerDestMutable` is true and `t` is a MutableType.
// For example if `et` is the type of a slice expression and `t` is the type of the slice elements, then DerivePointerExprType
// can be used to derive the ExprType of slice elements.
func DerivePointerExprType(et *ExprType, t Type) *ExprType {
	e := &ExprType{Mutable: et.PointerDestMutable, PointerDestMutable: false, Volatile: false, GroupSpecifier: nil /*et.Group*/, PointerDestGroupSpecifier: et.PointerDestGroupSpecifier}
	if pt, ok := GetPointerType(et.Type); ok && pt.Mode == PtrUnsafe {
		e.Unsafe = true
	}
	for {
		switch t2 := t.(type) {
		case *MutableType:
			e.PointerDestMutable = et.PointerDestMutable
			e.Volatile = et.Volatile
			t = t2.Type
			continue
		case *GroupedType:
			e.PointerDestGroupSpecifier = t2.GroupSpecifier
			t = t2.Type
			continue
		}
		break
	}
	e.Type = t
	return e
}

func deriveAddressOfExprType(et *ExprType, loc errlog.LocationRange) *ExprType {
	e := &ExprType{Mutable: true, PointerDestMutable: et.Mutable, Volatile: et.Volatile}
	pt := &PointerType{TypeBase: TypeBase{location: loc}, ElementType: et.ToType()}
	if et.Unsafe {
		pt.Mode = PtrUnsafe
	} else {
		pt.Mode = PtrOwner
	}
	e.Type = pt
	return e
}

func deriveSliceOfExprType(et *ExprType, elementType Type, loc errlog.LocationRange) *ExprType {
	e := &ExprType{Mutable: true, PointerDestMutable: et.Mutable, Volatile: et.Volatile}
	e.Type = &SliceType{TypeBase: TypeBase{location: loc}, ElementType: elementType}
	return e
}

// copyExprType copies the type information from `src` to `dest`.
// It does not copy values stored in ExprType.
func copyExprType(dest *ExprType, src *ExprType) {
	dest.Type = src.Type
	dest.GroupSpecifier = src.GroupSpecifier
	dest.Mutable = src.Mutable
	dest.PointerDestGroupSpecifier = src.PointerDestGroupSpecifier
	dest.Volatile = src.Volatile
	dest.Unsafe = src.Unsafe
	dest.PointerDestMutable = src.PointerDestMutable
}

// CloneExprType copies the type information from `src` to `dest`.
// It does not copy values stored in ExprType.
func CloneExprType(src *ExprType) *ExprType {
	dest := &ExprType{}
	dest.Type = src.Type
	dest.GroupSpecifier = src.GroupSpecifier
	dest.Mutable = src.Mutable
	dest.PointerDestGroupSpecifier = src.PointerDestGroupSpecifier
	dest.PointerDestMutable = src.PointerDestMutable
	dest.Volatile = src.Volatile
	dest.Unsafe = src.Unsafe
	return dest
}

// Checks whether the type `t` can be instantiated.
// For literal types, the function tries to deduce a default type.
func checkInstantiableExprType(t *ExprType, s *Scope, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	if t.Type == integerType {
		if t.IntegerValue.IsInt64() {
			i := t.IntegerValue.Int64()
			if i <= (1<<31)-1 && i >= -(1<<31) {
				t.Type = intType
			} else {
				t.Type = int64Type
			}
		} else if t.IntegerValue.IsUint64() {
			t.Type = uint64Type
		} else {
			log.AddError(errlog.ErrorNumberOutOfRange, loc, t.IntegerValue.String())
		}
	} else if t.Type == floatType {
		if _, acc := t.FloatValue.Float64(); acc == big.Exact {
			if _, acc := t.FloatValue.Float32(); acc == big.Exact {
				t.Type = float32Type
			} else {
				t.Type = float64Type
			}
		} else {
			log.AddError(errlog.ErrorNumberOutOfRange, loc, t.FloatValue.String())
		}
	} else if t.Type == nullType || t.Type == voidType || t.Type == structLiteralType || t.Type == namespaceType {
		// TODO: Use a better string representation of the type
		log.AddError(errlog.ErrorTypeCannotBeInstantiated, loc, t.Type.Name())
	} else if t.Type == arrayLiteralType {
		if len(t.ArrayValue) == 0 {
			log.AddError(errlog.ErrorTypeCannotBeInstantiated, loc, t.Type.Name())
		}
		if err := checkInstantiableExprType(t.ArrayValue[0], s, loc, log); err != nil {
			return err
		}
		for i := 1; i < len(t.ArrayValue); i++ {
			if needsTypeInference(t.ArrayValue[i]) {
				if err := inferType(t.ArrayValue[i], t.ArrayValue[0], false, loc, log); err != nil {
					return err
				}
			} else {
				if err := checkExprEqualType(t.ArrayValue[0], t.ArrayValue[i], Assignable, loc, log); err != nil {
					return err
				}
			}
		}
		t.Type = &ArrayType{TypeBase: TypeBase{location: loc}, Size: uint64(len(t.ArrayValue)), ElementType: t.ArrayValue[0].ToType()}
	}
	return nil
}

func needsTypeInference(t *ExprType) bool {
	return t.Type == floatType || t.Type == integerType || t.Type == nullType || t.Type == arrayLiteralType || t.Type == structLiteralType
}

func checkExprEqualType(tleft *ExprType, tright *ExprType, mode EqualTypeMode, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	if needsTypeInference(tleft) && needsTypeInference(tright) {
		if mode == Assignable {
			panic("Cannot assign to a constant")
		}
		if tleft.Type == integerType && tright.Type == floatType {
			return inferType(tleft, &ExprType{Type: floatType}, false, loc, log)
		}
		if tleft.Type == floatType && tright.Type == integerType {
			return inferType(tright, &ExprType{Type: floatType}, false, loc, log)
		}
		if tleft.Type == tright.Type {
			return nil
		}
		return log.AddError(errlog.ErrorIncompatibleTypes, loc)
	} else if needsTypeInference(tleft) {
		return inferType(tleft, tright, false, loc, log)
	} else if needsTypeInference(tright) {
		return inferType(tright, tleft, false, loc, log)
	}
	if mode == Strict && tleft.PointerDestMutable != tright.PointerDestMutable {
		return log.AddError(errlog.ErrorIncompatibleTypes, loc)
	} else if tleft.PointerDestMutable && !tright.PointerDestMutable && mode == Assignable {
		return log.AddError(errlog.ErrorIncompatibleTypes, loc)
	}
	return checkEqualType(tleft.Type, tright.Type, mode, loc, log)
}

func inferType(et *ExprType, target *ExprType, nested bool, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	tt := StripType(target.Type)
	if et.Type == integerType {
		if tt == integerType {
			return nil
		} else if tt == intType {
			et.Type = target.Type
			return checkIntegerBoundaries(et.IntegerValue, 32, loc, log)
		} else if tt == int8Type {
			et.Type = target.Type
			return checkIntegerBoundaries(et.IntegerValue, 8, loc, log)
		} else if tt == int16Type {
			et.Type = target.Type
			return checkIntegerBoundaries(et.IntegerValue, 16, loc, log)
		} else if tt == int32Type {
			et.Type = target.Type
			return checkIntegerBoundaries(et.IntegerValue, 32, loc, log)
		} else if tt == int64Type {
			et.Type = target.Type
			return checkIntegerBoundaries(et.IntegerValue, 64, loc, log)
		} else if tt == uintType {
			et.Type = target.Type
			return checkUIntegerBoundaries(et.IntegerValue, 32, loc, log)
		} else if tt == uint8Type {
			et.Type = target.Type
			return checkUIntegerBoundaries(et.IntegerValue, 8, loc, log)
		} else if tt == uint16Type {
			et.Type = target.Type
			return checkUIntegerBoundaries(et.IntegerValue, 16, loc, log)
		} else if tt == uint32Type {
			et.Type = target.Type
			return checkUIntegerBoundaries(et.IntegerValue, 32, loc, log)
		} else if tt == uint64Type {
			et.Type = target.Type
			return checkUIntegerBoundaries(et.IntegerValue, 64, loc, log)
		} else if tt == runeType {
			et.Type = target.Type
			return checkUIntegerBoundaries(et.IntegerValue, 16, loc, log)
		} else if tt == uintptrType {
			et.Type = target.Type
			// TODO: The 64 depends on the target plaform
			return checkUIntegerBoundaries(et.IntegerValue, 64, loc, log)
		} else if tt == floatType || tt == float32Type || tt == float64Type {
			et.Type = target.Type
			et.FloatValue = big.NewFloat(0)
			et.FloatValue.SetInt(et.IntegerValue)
			et.IntegerValue = nil
			return nil
		} else if IsUnsafePointerType(tt) {
			// Convert an integer to an unsafe pointer
			et.Type = target.Type
			et.PointerDestMutable = target.PointerDestMutable
			et.Volatile = target.Volatile
			// TODO: The 64 depends on the target plaform
			return checkUIntegerBoundaries(et.IntegerValue, 64, loc, log)
		}
	} else if et.Type == floatType {
		if tt == floatType {
			return nil
		} else if tt == float32Type {
			et.Type = target.Type
			return nil
		} else if tt == float64Type {
			et.Type = target.Type
			return nil
		}
	} else if et.Type == nullType {
		if IsPointerType(tt) || IsSliceType(tt) || IsFuncType(tt) || IsStringType(tt) {
			copyExprType(et, target)
			return nil
		}
	} else if et.Type == arrayLiteralType {
		if s, ok := GetSliceType(tt); ok {
			tet := DerivePointerExprType(target, s.ElementType)
			if nested && len(et.ArrayValue) != 0 && tet.PointerDestGroupSpecifier != nil {
				return log.AddError(errlog.ErrorCannotInferTypeWithGroups, loc)
			}
			for _, vet := range et.ArrayValue {
				if needsTypeInference(vet) {
					// TODO: loc is not the optimal location
					if err := inferType(vet, tet, true, loc, log); err != nil {
						return err
					}
				} else {
					if err := checkExprEqualType(tet, vet, Assignable, loc, log); err != nil {
						return err
					}
				}
			}
			copyExprType(et, target)
			return nil
		} else if a, ok := GetArrayType(tt); ok {
			tet := DeriveExprType(target, a.ElementType)
			if len(et.ArrayValue) != 0 && uint64(len(et.ArrayValue)) != a.Size {
				return log.AddError(errlog.ErrorIncompatibleTypes, loc)
			}
			if nested && len(et.ArrayValue) != 0 && tet.PointerDestGroupSpecifier != nil {
				return log.AddError(errlog.ErrorCannotInferTypeWithGroups, loc)
			}
			for _, vet := range et.ArrayValue {
				if needsTypeInference(vet) {
					// TODO: loc is not the optimal location
					if err := inferType(vet, tet, true, loc, log); err != nil {
						return err
					}
				} else {
					if err := checkExprEqualType(tet, vet, Assignable, loc, log); err != nil {
						return err
					}
				}
			}
			copyExprType(et, target)
			return nil
		}
	} else if et.Type == structLiteralType {
		targetType := tt
		isPointer := false
		if ptr, ok := GetPointerType(tt); ok {
			isPointer = true
			targetType = ptr.ElementType
		}
		if s, ok := GetStructType(targetType); ok {
			for name, vet := range et.StructValue {
				found := false
				for _, f := range s.Fields {
					if f.Name == name {
						var tet *ExprType
						if isPointer {
							tet = DerivePointerExprType(target, f.Type)
						} else {
							tet = DeriveExprType(target, f.Type)
						}
						if nested && tet.PointerDestGroupSpecifier != nil {
							return log.AddError(errlog.ErrorCannotInferTypeWithGroups, loc)
						}
						found = true
						if needsTypeInference(vet) {
							// TODO: loc is not the optimal location
							if err := inferType(vet, tet, true, loc, log); err != nil {
								return err
							}
						} else {
							if err := checkExprEqualType(tet, vet, Assignable, loc, log); err != nil {
								return err
							}
						}
						break
					}
				}
				if !found {
					return log.AddError(errlog.ErrorUnknownField, loc, name)
				}
			}
			copyExprType(et, target)
			return nil
		}
		if s, ok := GetUnionType(targetType); ok {
			if len(et.StructValue) > 1 {
				return log.AddError(errlog.ErrorExcessiveUnionValue, loc)
			}
			for name, vet := range et.StructValue {
				found := false
				for _, f := range s.Fields {
					if f.Name == name {
						var tet *ExprType
						if isPointer {
							tet = DerivePointerExprType(target, f.Type)
						} else {
							tet = DeriveExprType(target, f.Type)
						}
						if nested && tet.PointerDestGroupSpecifier != nil {
							return log.AddError(errlog.ErrorCannotInferTypeWithGroups, loc)
						}
						found = true
						if needsTypeInference(vet) {
							// TODO: loc is not the optimal location
							if err := inferType(vet, tet, true, loc, log); err != nil {
								return err
							}
						} else {
							if err := checkExprEqualType(tet, vet, Assignable, loc, log); err != nil {
								return err
							}
						}
						break
					}
				}
				if !found {
					return log.AddError(errlog.ErrorUnknownField, loc, name)
				}
			}
			copyExprType(et, target)
			return nil
		}
	}
	return log.AddError(errlog.ErrorIncompatibleTypes, loc)
}

func checkExprIntType(et *ExprType, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	target := &ExprType{Type: PrimitiveTypeInt}
	return checkExprEqualType(target, et, Assignable, loc, log)
}

func checkIntegerBoundaries(bigint *big.Int, bits uint, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	if bigint.IsInt64() {
		i := bigint.Int64()
		if i <= (1<<(bits-1))-1 && i >= -(1<<(bits-1)) {
			return nil
		}
	}
	return log.AddError(errlog.ErrorNumberOutOfRange, loc, bigint.String())
}

func checkUIntegerBoundaries(bigint *big.Int, bits uint, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	if bigint.IsUint64() {
		i := bigint.Uint64()
		if i <= (1<<(bits))-1 {
			return nil
		}
	}
	return log.AddError(errlog.ErrorNumberOutOfRange, loc, bigint.String())
}

func checkExprStringType(et *ExprType, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	target := &ExprType{Type: PrimitiveTypeString}
	return checkExprEqualType(target, et, Assignable, loc, log)
}
