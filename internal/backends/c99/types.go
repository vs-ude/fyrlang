package c99

import "github.com/vs-ude/fyrlang/internal/types"

func mapType(t types.Type) *TypeDecl {
	switch t2 := t.(type) {
	case *types.PrimitiveType:
		if t2 == types.PrimitiveTypeInt {
			return NewTypeDecl("int")
		} else if t2 == types.PrimitiveTypeInt8 {
			return NewTypeDecl("int8_t")
		} else if t2 == types.PrimitiveTypeInt16 {
			return NewTypeDecl("int16_t")
		} else if t2 == types.PrimitiveTypeInt32 {
			return NewTypeDecl("int32_t")
		} else if t2 == types.PrimitiveTypeInt64 {
			return NewTypeDecl("int64_t")
		} else if t2 == types.PrimitiveTypeUint {
			return NewTypeDecl("unsigned int")
		} else if t2 == types.PrimitiveTypeUint8 {
			return NewTypeDecl("uint8_t")
		} else if t2 == types.PrimitiveTypeUint16 {
			return NewTypeDecl("uint16_t")
		} else if t2 == types.PrimitiveTypeUint32 {
			return NewTypeDecl("uint32_t")
		} else if t2 == types.PrimitiveTypeUint64 {
			return NewTypeDecl("uint64_t")
		} else if t2 == types.PrimitiveTypeFloat32 {
			return NewTypeDecl("float")
		} else if t2 == types.PrimitiveTypeFloat64 {
			return NewTypeDecl("double")
		} else if t2 == types.PrimitiveTypeBool {
			return NewTypeDecl("bool")
		} else if t2 == types.PrimitiveTypeByte {
			return NewTypeDecl("uint8_t")
		} else if t2 == types.PrimitiveTypeRune {
			return NewTypeDecl("uint16_t")
		} else if t2 == types.PrimitiveTypeNull {
			panic("Oooops")
		} else if t2 == types.PrimitiveTypeString {
			panic("TODO")
		} else if t2 == types.PrimitiveTypeIntLiteral {
			panic("Oooops")
		} else if t2 == types.PrimitiveTypeFloatLiteral {
			panic("Oooops")
		} else if t2 == types.PrimitiveTypeArrayLiteral {
			panic("Oooops")
		} else if t2 == types.PrimitiveTypeVoid {
			return NewTypeDecl("void")
		}
	}
	panic("Oooops")
}
