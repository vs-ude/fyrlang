package types

import (
	"math/big"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

// ExprType ...
type ExprType struct {
	// Instances of GroupType or MutableType are removed for convenience and
	// factors into the PointerDestMutable and PointerDestGroup properties.
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
	HasValue         bool
}

func exprType(n parser.Node) *ExprType {
	return n.TypeAnnotation().(*ExprType)
}

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
	} else if t.Type == nullType || t.Type == voidType {
		// TODO: Use a better string representation of the type
		log.AddError(errlog.ErrorTypeCannotBeInstantiated, loc, t.Type.Name())
	} else if t.Type == arrayLiteralType {
		panic("TODO")
	}
	// TODO: Literal types
	return nil
}

func needsTypeInference(t *ExprType) bool {
	return t.Type == floatType || t.Type == integerType || t.Type == nullType || t.Type == arrayLiteralType
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
		if tleft.Type == nullType && tright.Type == nullType {
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
		if isPointerType(target.Type) || isSliceType(target.Type) {
			et.Type = target.Type
			et.PointerDestGroup = target.PointerDestGroup
			et.PointerDestMutable = target.PointerDestMutable
			return nil
		}
	} else if et.Type == arrayLiteralType {
		if isSliceType(target.Type) {
			et.Type = target.Type
			et.PointerDestGroup = target.PointerDestGroup
			et.PointerDestMutable = target.PointerDestMutable
			panic("TODO")
		} else if a, ok := getArrayType(target.Type); ok {
			tet := makeExprType(a.ElementType)
			tet.Group = target.Group
			tet.Mutable = target.Mutable
			tet.PointerDestGroup = target.PointerDestGroup
			tet.PointerDestMutable = target.PointerDestMutable
			for _, vet := range et.ArrayValue {
				// TODO: loc is not the optimal location
				if err := inferType(vet, tet, loc, log); err != nil {
					return err
				}
			}
			et.Type = target.Type
			et.Group = target.Group
			et.Mutable = target.Mutable
			et.PointerDestGroup = target.PointerDestGroup
			et.PointerDestMutable = target.PointerDestMutable
			return nil
		}
	}
	return log.AddError(errlog.ErrorIncompatibleTypes, loc)
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
