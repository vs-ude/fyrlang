package types

import (
	"math/big"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

// ExprType represents type information about an expression.
// It is more powerful than type alone, because it reveals information about
// mutability and groups and can store values in case the expression is constant.
type ExprType struct {
	// Instances of GroupType or MutableType are removed for convenience and
	// factored into the PointerDestMutable and PointerDestGroup properties.
	Type Type
	// Mutable defines whether the value of the expression is mutable.
	Mutable bool
	// PointerDestMutable defines the mutability of value being pointed to.
	// This is required, because the type system distinguishes between the mutability of a pointer
	// and the mutability of the value it is pointing to.
	PointerDestMutable bool
	// The group to which the value belongs
	Group *Group
	// The (default) group to values being pointed to.
	// This is required, because a pointer on the stack belongs to a stack-group,
	// but it might point to an object of another group.
	PointerDestGroup *Group
	StringValue      string
	RuneValue        rune
	IntegerValue     *big.Int
	FloatValue       *big.Float
	BoolValue        bool
	ArrayValue       []*ExprType
	StructValue      map[string]*ExprType
	// HasValue is true if one of the *Value properties holds a value.
	// This does not imply that the expression has a constant value, because
	// an ArrayValue may contain an ExprType that has no value.
	// Use IsConstant() to determine whether an expression is constant.
	HasValue bool
}

// Clone ...
func (et *ExprType) Clone() *ExprType {
	result := &ExprType{}
	result.Type = et.Type
	result.Mutable = et.Mutable
	result.PointerDestMutable = et.PointerDestMutable
	result.Group = et.Group
	result.PointerDestGroup = et.PointerDestGroup
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

// IsPrimitiveConstant is true for all constants of primitive type.
// Constants representing arrays, slices, structs or pointers to structs are not primitive.
// Constant pointers such as the null pointer are primitive as well.
// For primitive constants code generators can easily substitute constant variables with the constant itself.
func (et *ExprType) IsPrimitiveConstant() bool {
	if !et.HasValue {
		return false
	}
	if _, ok := et.Type.(*ArrayType); ok {
		return false
	}
	if _, ok := et.Type.(*SliceType); ok {
		return false
	}
	// This catches integers and constant pointer values (e.g. null).
	if et.IntegerValue != nil {
		return true
	}
	// This must be a pointer to a struct
	if _, ok := et.Type.(*PointerType); ok {
		return false
	}
	if _, ok := et.Type.(*StructType); ok {
		return false
	}
	return true
}

// ToType ...
func (et *ExprType) ToType() Type {
	t := et.Type
	if et.PointerDestMutable {
		t = &MutableType{TypeBase: TypeBase{location: t.Location(), pkg: t.Package()}}
	}
	if et.PointerDestGroup != nil {
		t = &GroupType{TypeBase: TypeBase{location: t.Location(), pkg: t.Package()}, Group: et.PointerDestGroup}
	}
	return t
}

func exprType(n parser.Node) *ExprType {
	return n.TypeAnnotation().(*ExprType)
}

// makeExprType sets the Type property.
// If the Type `t` is MutableType or GroupType, these are dropped and PointerDestMutable/PointerDestGroup are set accordingly.
func makeExprType(t Type) *ExprType {
	e := &ExprType{}
	for {
		switch t2 := t.(type) {
		case *MutableType:
			e.PointerDestMutable = true
			t = t2.Type
			continue
		case *GroupType:
			e.PointerDestGroup = t2.Group
			t = t2.Type
			continue
		}
		break
	}
	e.Type = t
	return e
}

// deriveExprType acts like makeExprType.
// However, before it analyzes `t`, it copies the Mutable, PointerDestMutable, Group and PointerDestGroup properties from `et`.
// For example if `et` is the type of an array expression and `t` is the type of the array elements, then deriveExprType
// can be used to derive the ExprType of array elements.
func deriveExprType(et *ExprType, t Type) *ExprType {
	e := &ExprType{Mutable: et.Mutable, PointerDestMutable: et.PointerDestMutable, Group: et.Group, PointerDestGroup: et.PointerDestGroup}
	for {
		switch t2 := t.(type) {
		case *MutableType:
			e.PointerDestMutable = true
			t = t2.Type
			continue
		case *GroupType:
			e.PointerDestGroup = t2.Group
			t = t2.Type
			continue
		}
		break
	}
	e.Type = t
	return e
}

// derivePointerExprType acts like makeExprType.
// However, before it analyzes `t`, it copies the PointerDestGroup property from `et` and set PointerDestMutable to `false`.
// The Mutable and Group properties are set to `et.PointerDestMutable` and `et.PointerDestGroup`.
// The PointerDestMutable property becomes true if `et.PointerDestMutable` is true and `t` is a MutableType.
// For example if `et` is the type of a slice expression and `t` is the type of the slice elements, then deriveExprType
// can be used to derive the ExprType of slice elements.
func derivePointerExprType(et *ExprType, t Type) *ExprType {
	e := &ExprType{Mutable: et.PointerDestMutable, PointerDestMutable: false, Group: et.Group, PointerDestGroup: et.PointerDestGroup}
	for {
		switch t2 := t.(type) {
		case *MutableType:
			e.PointerDestMutable = et.PointerDestMutable
			t = t2.Type
			continue
		case *GroupType:
			e.PointerDestGroup = t2.Group
			t = t2.Type
			continue
		}
		break
	}
	e.Type = t
	return e
}

func deriveAddressOfExprType(et *ExprType, loc errlog.LocationRange) *ExprType {
	e := &ExprType{Mutable: true, PointerDestMutable: et.Mutable, Group: NewFreeGroup(loc), PointerDestGroup: et.Group}
	e.Type = &PointerType{TypeBase: TypeBase{location: loc}, ElementType: et.Type}
	return e
}

// copyExprType copies the type information from `src` to `dest`.
// It does not copy values stored in ExprType.
func copyExprType(dest *ExprType, src *ExprType) {
	dest.Type = src.Type
	dest.Group = src.Group
	dest.Mutable = src.Mutable
	dest.PointerDestGroup = src.PointerDestGroup
	dest.PointerDestMutable = src.PointerDestMutable
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
	} else if t.Type == nullType || t.Type == voidType || t.Type == structLiteralType {
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
				if err := inferType(t.ArrayValue[i], t.ArrayValue[0], loc, log); err != nil {
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
			return inferType(tleft, &ExprType{Type: floatType}, loc, log)
		}
		if tleft.Type == floatType && tright.Type == integerType {
			return inferType(tright, &ExprType{Type: floatType}, loc, log)
		}
		if tleft.Type == tright.Type {
			return nil
		}
		return log.AddError(errlog.ErrorIncompatibleTypes, loc)
	} else if needsTypeInference(tleft) {
		return inferType(tleft, tright, loc, log)
	} else if needsTypeInference(tright) {
		return inferType(tright, tleft, loc, log)
	}
	if mode == Strict && tleft.PointerDestMutable != tright.PointerDestMutable {
		return log.AddError(errlog.ErrorIncompatibleTypes, loc)
	} else if tleft.PointerDestMutable && !tright.PointerDestMutable && mode == Assignable {
		return log.AddError(errlog.ErrorIncompatibleTypes, loc)
	}
	return checkEqualType(tleft.Type, tright.Type, mode, loc, log)
}

func inferType(et *ExprType, target *ExprType, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	if et.Type == integerType {
		if target.Type == integerType {
			return nil
		} else if target.Type == intType {
			et.Type = target.Type
			return checkIntegerBoundaries(et.IntegerValue, 32, loc, log)
		} else if target.Type == int8Type {
			et.Type = target.Type
			return checkIntegerBoundaries(et.IntegerValue, 8, loc, log)
		} else if target.Type == int16Type {
			et.Type = target.Type
			return checkIntegerBoundaries(et.IntegerValue, 16, loc, log)
		} else if target.Type == int32Type {
			et.Type = target.Type
			return checkIntegerBoundaries(et.IntegerValue, 32, loc, log)
		} else if target.Type == int64Type {
			et.Type = target.Type
			return checkIntegerBoundaries(et.IntegerValue, 64, loc, log)
		} else if target.Type == uintType {
			et.Type = target.Type
			return checkUIntegerBoundaries(et.IntegerValue, 32, loc, log)
		} else if target.Type == uint8Type {
			et.Type = target.Type
			return checkUIntegerBoundaries(et.IntegerValue, 8, loc, log)
		} else if target.Type == uint16Type {
			et.Type = target.Type
			return checkUIntegerBoundaries(et.IntegerValue, 16, loc, log)
		} else if target.Type == uint32Type {
			et.Type = target.Type
			return checkUIntegerBoundaries(et.IntegerValue, 32, loc, log)
		} else if target.Type == uint64Type {
			et.Type = target.Type
			return checkUIntegerBoundaries(et.IntegerValue, 64, loc, log)
		} else if target.Type == floatType || target.Type == float32Type || target.Type == float64Type {
			et.Type = target.Type
			et.FloatValue = big.NewFloat(0)
			et.FloatValue.SetInt(et.IntegerValue)
			et.IntegerValue = nil
			return nil
		}
	} else if et.Type == floatType {
		if target.Type == floatType {
			return nil
		} else if target.Type == float32Type {
			et.Type = target.Type
			return nil
		} else if target.Type == float64Type {
			et.Type = target.Type
			return nil
		}
	} else if et.Type == nullType {
		if IsPointerType(target.Type) || IsSliceType(target.Type) {
			copyExprType(et, target)
			return nil
		}
	} else if et.Type == arrayLiteralType {
		if s, ok := GetSliceType(target.Type); ok {
			tet := derivePointerExprType(target, s.ElementType)
			for _, vet := range et.ArrayValue {
				if needsTypeInference(vet) {
					// TODO: loc is not the optimal location
					if err := inferType(vet, tet, loc, log); err != nil {
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
		} else if a, ok := GetArrayType(target.Type); ok {
			tet := deriveExprType(target, a.ElementType)
			for _, vet := range et.ArrayValue {
				if needsTypeInference(vet) {
					// TODO: loc is not the optimal location
					if err := inferType(vet, tet, loc, log); err != nil {
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
		targetType := target.Type
		isPointer := false
		if ptr, ok := GetPointerType(target.Type); ok {
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
							tet = derivePointerExprType(target, f.Type)
						} else {
							tet = deriveExprType(target, f.Type)
						}
						found = true
						if needsTypeInference(vet) {
							// TODO: loc is not the optimal location
							if err := inferType(vet, tet, loc, log); err != nil {
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
