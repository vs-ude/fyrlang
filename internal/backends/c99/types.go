package c99

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/vs-ude/fyrlang/internal/types"
)

func mapType(mod *Module, t types.Type) *TypeDecl {
	return mapTypeIntern(mod, t, nil, nil)
}

func mapTypeIntern(mod *Module, t types.Type, group *types.Group, mut *types.MutableType) *TypeDecl {
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
		} else if t2 == types.PrimitiveTypeUintptr {
			return NewTypeDecl("uintptr_t")
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
			return NewTypeDecl("uint8_t*")
		} else if t2 == types.PrimitiveTypeIntLiteral {
			panic("Oooops")
		} else if t2 == types.PrimitiveTypeFloatLiteral {
			panic("Oooops")
		} else if t2 == types.PrimitiveTypeArrayLiteral {
			panic("Oooops")
		} else if t2 == types.PrimitiveTypeVoid {
			return NewTypeDecl("void")
		}
	case *types.PointerType:
		if group != nil && group.Kind == types.GroupIsolate {
			// TODO: Use full qualified type signature
			typesig := "->" + t2.ToString()
			typesigMangled := mangleTypeSignature(typesig)
			typename := "t_" + typesigMangled
			if !mod.hasTypeDef(typename) {
				typ := "struct { " + mapTypeIntern(mod, t2.ElementType, nil, nil).ToString("") + "* ptr; uintptr_t group; }"
				tdef := NewTypeDef(typ, typename)
				tdef.Guard = "T_" + typesigMangled
				mod.addTypeDef(tdef)
			}
			return NewTypeDecl(typename)
		}
		d := mapTypeIntern(mod, t2.ElementType, nil, nil)
		d.Code = d.Code + "*"
		return d
	case *types.StructType:
		if t2.IsTypedef() {
			return NewTypeDecl("struct " + mangleTypeName(t2.Package(), t2.Component(), t2.Name()))
		}
		s := &Struct{}
		if t2.BaseType != nil {
			defineStructFieldType(mod, t2.BaseType)
			sf := &StructField{Name: t2.BaseType.Name(), Type: mapTypeIntern(mod, t2.BaseType, nil, nil)}
			s.Fields = append(s.Fields, sf)
		}
		for _, f := range t2.Fields {
			defineStructFieldType(mod, f.Type)
			sf := &StructField{Name: f.Name, Type: mapTypeIntern(mod, f.Type, nil, nil)}
			s.Fields = append(s.Fields, sf)
		}
		return NewTypeDecl(s.ToString(""))
	case *types.AliasType:
		return mapTypeIntern(mod, t2.Alias, group, mut)
	case *types.GenericInstanceType:
		// TODO: Use full qualified type signature
		typesig := t2.ToString()
		typesigMangled := mangleTypeSignature(typesig)
		typename := "t_" + typesigMangled
		if !mod.hasTypeDef(typename) {
			typ := mapTypeIntern(mod, t2.InstanceType, nil, nil).ToString("")
			tdef := NewTypeDef(typ, typename)
			tdef.Guard = "T_" + typesigMangled
			mod.addTypeDef(tdef)
		}
		return NewTypeDecl(typename)
	case *types.GenericType:
		panic("Oooops")
	case *types.GroupType:
		return mapTypeIntern(mod, t2.Type, t2.Group, mut)
	case *types.MutableType:
		return mapTypeIntern(mod, t2.Type, group, t2)
	case *types.ComponentType:
		return NewTypeDecl(mangleTypeName(t2.Package(), nil, t2.Name()))
	case *types.InterfaceType:
		return NewTypeDecl("void*")
		// panic("TODO")
	case *types.ClosureType:
		return NewTypeDecl("void*")
		// panic("TODO")
	case *types.SliceType:
		// TODO: Use full qualified type signature
		typesig := t2.ToString()
		if group != nil && group.Kind == types.GroupIsolate {
			typesig = "->" + typesig
		}
		typesigMangled := mangleTypeSignature(typesig)
		typename := "t_" + typesigMangled
		if !mod.hasTypeDef(typename) {
			var typ string
			if group != nil && group.Kind == types.GroupIsolate {
				typ = "struct { " + mapTypeIntern(mod, t2, nil, mut).ToString("") + " slice; uintptr_t group; }"
			} else {
				typ = "struct { " + mapTypeIntern(mod, t2.ElementType, nil, nil).ToString("") + "* ptr; int size; int cap; }"
			}
			tdef := NewTypeDef(typ, typename)
			tdef.Guard = "T_" + typesigMangled
			mod.addTypeDef(tdef)
		}
		return NewTypeDecl(typename)
	case *types.ArrayType:
		// TODO: Use full qualified type signature
		typesig := t2.ToString()
		typesigMangled := mangleTypeSignature(typesig)
		typename := "t_" + typesigMangled
		if !mod.hasTypeDef(typename) {
			typ := "struct { " + mapTypeIntern(mod, t2.ElementType, group, mut).ToString("") + " arr[" + strconv.FormatUint(t2.Size, 10) + "]; }"
			tdef := NewTypeDef(typ, typename)
			tdef.Guard = "T_" + typesigMangled
			mod.addTypeDef(tdef)
		}
		return NewTypeDecl(typename)
	case *types.FuncType:
		panic("TODO")
	}
	panic("Oooops")
}

func declareNamedType(mod *Module, comp *types.ComponentType, name string, t types.Type) {
	switch t2 := t.(type) {
	case *types.GroupType:
		declareNamedType(mod, comp, name, t2.Type)
		return
	case *types.MutableType:
		declareNamedType(mod, comp, name, t2.Type)
		return
	case *types.AliasType:
		declareNamedType(mod, comp, name, t2.Alias)
		return
	case *types.GenericType:
		// Do nothing by intention
		return
	case *types.ComponentType:
		// Declare all named types
		for name, t := range t2.Scope.Types {
			declareNamedType(mod, t2, name, t)
		}
		return
	case *types.StructType:
		typename := mangleTypeName(mod.Package.TypePackage, comp, t2.Name())
		mod.addTypeDecl(NewTypeDecl("struct " + typename))
		return
	}
	typename := mangleTypeName(mod.Package.TypePackage, comp, name)
	tdef := NewTypeDef(mapTypeIntern(mod, t, nil, nil).ToString(""), typename)
	mod.addTypeDef(tdef)
}

// This function generates C-code that defines named types.
// This is required for structs.
// Other simple types are already defined when they are declared.
func defineNamedType(mod *Module, comp *types.ComponentType, name string, t types.Type) {
	switch t2 := t.(type) {
	case *types.GroupType:
		defineNamedType(mod, comp, name, t2.Type)
		return
	case *types.MutableType:
		defineNamedType(mod, comp, name, t2.Type)
		return
	case *types.AliasType:
		defineNamedType(mod, comp, name, t2.Alias)
		return
	case *types.GenericType:
		// Do nothing by intention
		return
	case *types.ComponentType:
		// Declare all named types
		for name, t := range t2.Scope.Types {
			defineNamedType(mod, t2, name, t)
		}
		return
	case *types.StructType:
		// Struct defined in another package? Do nothing.
		if t2.Package() != mod.Package.TypePackage {
			return
		}
		typename := mangleTypeName(mod.Package.TypePackage, comp, t2.Name())
		// Already defined?
		if mod.hasStructDef(typename) {
			return
		}
		s := &Struct{Name: typename}
		if t2.BaseType != nil {
			defineStructFieldType(mod, t2.BaseType)
			sf := &StructField{Name: t2.BaseType.Name(), Type: mapTypeIntern(mod, t2.BaseType, nil, nil)}
			s.Fields = append(s.Fields, sf)
		}
		for _, f := range t2.Fields {
			defineStructFieldType(mod, f.Type)
			sf := &StructField{Name: f.Name, Type: mapTypeIntern(mod, f.Type, nil, nil)}
			s.Fields = append(s.Fields, sf)
		}
		mod.addStructDef(s)
		return
	}
}

// This function asures that all structs which appear in another struct's fields
// are defined. Otherwise the C-compiler would yield an `has incomplete type` error.
func defineStructFieldType(mod *Module, t types.Type) {
	switch t2 := t.(type) {
	case *types.GroupType:
		defineStructFieldType(mod, t2.Type)
		return
	case *types.MutableType:
		defineStructFieldType(mod, t2.Type)
		return
	case *types.AliasType:
		defineStructFieldType(mod, t2.Alias)
		return
	case *types.GenericType:
		// Do nothing by intention
		return
	case *types.ComponentType:
		panic("Ooooops")
	case *types.StructType:
		if t2.IsTypedef() && t2.Package() == mod.Package.TypePackage {
			defineNamedType(mod, t2.Component(), t2.Name(), t2)
		}
		return
	}
}

func mangleTypeName(p *types.Package, comp *types.ComponentType, name string) string {
	var data string
	if comp == nil {
		data = p.FullPath() + "//" + name
	} else {
		data = p.FullPath() + "//" + comp.Name() + "//" + name
	}
	sum := sha256.Sum256([]byte(data))
	sumHex := hex.EncodeToString(sum[:])
	return name + "_" + sumHex
}

func mangleTypeSignature(typesig string) string {
	sum := sha256.Sum256([]byte(typesig))
	sumHex := hex.EncodeToString(sum[:])
	return sumHex
}
