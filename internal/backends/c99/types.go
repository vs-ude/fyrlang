package c99

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

func mapType(mod *Module, t types.Type) *TypeDecl {
	return mapTypeIntern(mod, t, nil, false, false)
}

func mapParameterType(mod *Module, t types.Type) *TypeDecl {
	if t2, ok := t.(*types.GroupedType); ok {
		t = t2.Type
	}
	return mapTypeIntern(mod, t, nil, false, false)
}

func mapExprType(mod *Module, t *types.ExprType) *TypeDecl {
	return mapTypeIntern(mod, t.Type, t.PointerDestGroupSpecifier, t.PointerDestMutable, t.Volatile)
}

func mapVarExprType(mod *Module, t *types.ExprType) *TypeDecl {
	return mapTypeIntern(mod, t.Type, nil, t.PointerDestMutable, t.Volatile)
}

// Temporary variables are never marked as volatile, because they can at most
// contain a value that has been read from volatile memory.
func mapTmpVarExprType(mod *Module, t *types.ExprType) *TypeDecl {
	return mapTypeIntern(mod, t.Type, nil, t.PointerDestMutable, false)
}

func mapSlicePointerExprType(mod *Module, t *types.ExprType) *TypeDecl {
	sl, ok := types.GetSliceType(t.Type)
	if !ok {
		panic("Ooooops")
	}
	tdecl := mapTypeIntern(mod, sl.ElementType, nil, t.PointerDestMutable, t.Volatile)
	return &TypeDecl{Code: tdecl.Code + "*"}
}

func mapTypeIntern(mod *Module, t types.Type, group *types.GroupSpecifier, mut bool, volatile bool) *TypeDecl {
	tdecl := mapTypeIntern2(mod, t, group, mut)
	if volatile {
		if _, ok := types.GetPointerType(t); ok {
			tdecl.Code += " volatile"
		} else {
			tdecl.Code = "volatile " + tdecl.Code
		}
	}
	return tdecl
}

func mapTypeIntern2(mod *Module, t types.Type, group *types.GroupSpecifier, mut bool) *TypeDecl {
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
			// TODO: What about ->string
			return defineString(mod)
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
		// Use a fat pointer?
		if group != nil && group.Kind != types.GroupSpecifierNamed {
			// TODO: Use full qualified type signature
			typesig := "->" + t2.ToString()
			typesigMangled := mangleTypeSignature(typesig)
			typename := "t_" + typesigMangled
			if !mod.hasTypeDef(typename) {
				typ := "struct { " + mapTypeIntern(mod, t2.ElementType, nil, mut, false).ToString("") + "* ptr; uintptr_t group; }"
				tdef := NewTypeDef(typ, typename)
				tdef.Guard = "T_" + typesigMangled
				mod.addTypeDef(tdef)
			}
			return NewTypeDecl(typename)
		}
		d := mapTypeIntern(mod, t2.ElementType, nil, mut, false)
		d.Code = d.Code + "*"
		return d
	case *types.StructType:
		if t2.IsTypedef() {
			// A named struct
			return NewTypeDecl("struct " + mangleTypeName(t2.Package(), t2.Component(), t2.Name()))
		}
		return defineAnonymousStruct(mod, t2)
	case *types.UnionType:
		if t2.IsTypedef() {
			// A named struct
			return NewTypeDecl("union " + mangleTypeName(t2.Package(), t2.Component(), t2.Name()))
		}
		return defineAnonymousUnion(mod, t2)
	case *types.AliasType:
		return mapTypeIntern(mod, t2.Alias, group, mut, false)
	case *types.GenericInstanceType:
		return mapTypeIntern(mod, t2.InstanceType, nil, false, false)
	case *types.GenericType:
		panic("Oooops")
	case *types.GroupedType:
		return mapTypeIntern(mod, t2.Type, t2.GroupSpecifier, mut, false)
	case *types.MutableType:
		return mapTypeIntern(mod, t2.Type, group, mut || t2.Mutable, t2.Volatile)
	case *types.ComponentType:
		return NewTypeDecl("struct " + mangleTypeName(t2.Package(), nil, t2.Name()))
	case *types.InterfaceType:
		return NewTypeDecl("void*")
		// panic("TODO")
	case *types.ClosureType:
		return NewTypeDecl("void*")
		// panic("TODO")
	case *types.SliceType:
		return defineSliceType(mod, t2, group, mut)
	case *types.ArrayType:
		// TODO: Use full qualified type signature
		typesig := t2.ToString()
		typesigMangled := mangleTypeSignature(typesig)
		typename := "t_" + typesigMangled
		if !mod.hasTypeDef(typename) {
			typ := "struct { " + mapTypeIntern(mod, t2.ElementType, group, mut, false).ToString("") + " arr[" + strconv.FormatUint(t2.Size, 10) + "]; }"
			tdef := NewTypeDef(typ, typename)
			tdef.Guard = "T_" + typesigMangled
			mod.addTypeDef(tdef)
		}
		return NewTypeDecl(typename)
	case *types.FuncType:
		// TODO: Use full qualified type signature
		typesig := t2.ToFunctionSignature()
		typesigMangled := mangleTypeSignature(typesig)
		typename := "t_" + typesigMangled
		if !mod.hasTypeDef(typename) {
			irft := ircode.NewFunctionType(t2)
			ft := &FunctionType{}
			for _, p := range irft.In {
				ft.Parameters = append(ft.Parameters, &FunctionParameter{Name: "p_" + p.Name, Type: mapType(mod, p.Type)})
			}
			for _, g := range irft.GroupSpecifiers {
				ft.Parameters = append(ft.Parameters, &FunctionParameter{Name: "g_" + g.Name, Type: &TypeDecl{Code: "uintptr_t*"}})
			}
			if len(irft.Out) == 0 {
				ft.ReturnType = NewTypeDecl("void")
			} else {
				ft.ReturnType = mapType(mod, irft.ReturnType())
			}
			tdef := ft.TypeDef(typename)
			tdef.Guard = "T_" + typesigMangled
			mod.addTypeDef(tdef)
		}
		return NewTypeDecl(typename)
	}
	panic("Oooops")
}

func declareNamedType(mod *Module, comp *types.ComponentType, name string, t types.Type) {
	switch t2 := t.(type) {
	case *types.GroupedType:
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
		for name, t := range t2.ComponentScope.Types {
			declareNamedType(mod, t2, name, t)
		}
		return
	case *types.StructType:
		typename := mangleTypeName(mod.Package.TypePackage, comp, t2.Name())
		mod.addTypeDecl(NewTypeDecl("struct " + typename))
		return
	case *types.UnionType:
		typename := mangleTypeName(mod.Package.TypePackage, comp, t2.Name())
		mod.addTypeDecl(NewTypeDecl("union " + typename))
		return
	}
	typename := mangleTypeName(mod.Package.TypePackage, comp, name)
	tdef := NewTypeDef(mapTypeIntern(mod, t, nil, false, false).ToString(""), typename)
	mod.addTypeDef(tdef)
}

// This function generates C-code that defines named types.
// This is required for structs.
// Other simple types are already defined when they are declared.
func defineNamedType(mod *Module, comp *types.ComponentType, name string, t types.Type) {
	switch t2 := t.(type) {
	case *types.GroupedType:
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
		// Component defined in another package? Do nothing.
		if t2.Package() != mod.Package.TypePackage {
			return
		}
		typename := mangleTypeName(mod.Package.TypePackage, nil, t2.Name())
		// Already defined?
		if mod.hasStructDef(typename) {
			return
		}
		// Declare all named types inside the component
		for name, t := range t2.ComponentScope.Types {
			defineNamedType(mod, t2, name, t)
		}
		s := &Struct{Name: typename}
		// Variable used to store the group pointer.
		sf := &StructField{Name: "__gptr", Type: mapTypeIntern(mod, types.PrimitiveTypeUintptr, nil, false, false)}
		s.Fields = append(s.Fields, sf)
		for _, f := range t2.Fields {
			defineStructFieldType(mod, f.Var.Type.Type)
			sf := &StructField{Name: f.Var.Name(), Type: mapTypeIntern(mod, f.Var.Type.Type, nil, false, false)}
			s.Fields = append(s.Fields, sf)
		}
		mod.addStructDef(s)
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
		for _, f := range t2.Fields {
			defineStructFieldType(mod, f.Type)
			sf := &StructField{Name: f.Name, Type: mapTypeIntern(mod, f.Type, nil, false, false)}
			s.Fields = append(s.Fields, sf)
		}
		mod.addStructDef(s)
		return
	case *types.UnionType:
		// Union defined in another package? Do nothing.
		if t2.Package() != mod.Package.TypePackage {
			return
		}
		typename := mangleTypeName(mod.Package.TypePackage, comp, t2.Name())
		// Already defined?
		if mod.hasUnionDef(typename) {
			return
		}
		s := &Union{Name: typename}
		for _, f := range t2.Fields {
			defineStructFieldType(mod, f.Type)
			sf := &StructField{Name: f.Name, Type: mapTypeIntern(mod, f.Type, nil, false, false)}
			s.Fields = append(s.Fields, sf)
		}
		mod.addUnionDef(s)
		return
	}
}

// This function asures that all structs which appear in another struct's fields
// are defined. Otherwise the C-compiler would yield an `has incomplete type` error.
func defineStructFieldType(mod *Module, t types.Type) {
	switch t2 := t.(type) {
	case *types.GroupedType:
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

func defineAnonymousStruct(mod *Module, st *types.StructType) *TypeDecl {
	// TODO: Use full qualified type signature
	typesig := st.ToString()
	typesigMangled := mangleTypeSignature(typesig)
	structName := "s_" + typesigMangled
	typename := "struct " + structName
	if !mod.hasStructDef(structName) {
		mod.addStructDefPre(structName)
		tdecl := NewTypeDecl(typename)
		mod.addTypeDecl(tdecl)
		s := &Struct{Name: structName}
		for _, f := range st.Fields {
			defineStructFieldType(mod, f.Type)
			sf := &StructField{Name: f.Name, Type: mapTypeIntern(mod, f.Type, nil, false, false)}
			s.Fields = append(s.Fields, sf)
		}
		s.Guard = "T_" + typesigMangled
		mod.addStructDef(s)
	}
	return NewTypeDecl(typename)
}

func defineAnonymousUnion(mod *Module, st *types.UnionType) *TypeDecl {
	// TODO: Use full qualified type signature
	typesig := st.ToString()
	typesigMangled := mangleTypeSignature(typesig)
	structName := "s_" + typesigMangled
	typename := "union " + structName
	if !mod.hasUnionDef(structName) {
		mod.addUnionDefPre(structName)
		tdecl := NewTypeDecl(typename)
		mod.addTypeDecl(tdecl)
		s := &Union{Name: structName}
		for _, f := range st.Fields {
			defineStructFieldType(mod, f.Type)
			sf := &StructField{Name: f.Name, Type: mapTypeIntern(mod, f.Type, nil, false, false)}
			s.Fields = append(s.Fields, sf)
		}
		s.Guard = "T_" + typesigMangled
		mod.addUnionDef(s)
	}
	return NewTypeDecl(typename)
}

func defineString(mod *Module) *TypeDecl {
	typename := "t_string"
	if !mod.hasTypeDef(typename) {
		s := &Struct{Name: "s_string"}
		sf := &StructField{Name: "size", Type: &TypeDecl{Code: "int"}}
		s.Fields = append(s.Fields, sf)
		sf = &StructField{Name: "data", Type: &TypeDecl{Code: "uint8_t*"}}
		s.Fields = append(s.Fields, sf)
		tdef := NewTypeDef(s.ToString(""), typename)
		tdef.Guard = "T_STRING"
		mod.addTypeDef(tdef)
	}
	return NewTypeDecl(typename)
}

func defineSliceType(mod *Module, t *types.SliceType, group *types.GroupSpecifier, mut bool) *TypeDecl {
	typesig := t.ToString()
	if group != nil && group.Kind != types.GroupSpecifierNamed {
		typesig = group.ToString() + typesig
	}
	typesigMangled := mangleTypeSignature(typesig)
	typename := "t_" + typesigMangled
	if !mod.hasTypeDef(typename) {
		var typ string
		if group != nil && group.Kind != types.GroupSpecifierNamed {
			typ = "struct { " + mapTypeIntern(mod, t, nil, mut, false).ToString("") + " slice; uintptr_t group; }"
		} else {
			typ = "struct { " + mapTypeIntern(mod, t.ElementType, nil, mut, false).ToString("") + "* ptr; int size; int cap; }"
		}
		tdef := NewTypeDef(typ, typename)
		tdef.Guard = "T_" + typesigMangled
		mod.addTypeDef(tdef)
	}
	return NewTypeDecl(typename)
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
