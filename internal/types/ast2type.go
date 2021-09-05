package types

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/lexer"
	"github.com/vs-ude/fyrlang/internal/parser"
)

func parseType(ast parser.Node, s *Scope, log *errlog.ErrorLog) (Type, error) {
	if n, ok := ast.(*parser.NamedTypeNode); ok {
		return s.LookupNamedType(n, log)
	}
	t := declareType(ast)
	err := defineType(t, ast, s, log)
	if err != nil {
		return nil, err
	}
	if err = t.Check(log); err != nil {
		return nil, err
	}
	if err = checkFuncs(t, s.InstantiatingPackage(), log); err != nil {
		return nil, err
	}
	return t, nil
}

func declareAndDefineType(ast parser.Node, s *Scope, log *errlog.ErrorLog) (Type, error) {
	if n, ok := ast.(*parser.NamedTypeNode); ok {
		return s.LookupNamedType(n, log)
	}
	t := declareType(ast)
	err := defineType(t, ast, s, log)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func declareType(ast parser.Node) Type {
	if _, ok := ast.(*parser.NamedTypeNode); ok {
		return &AliasType{TypeBase: TypeBase{location: ast.Location()}}
	} else if _, ok := ast.(*parser.PointerTypeNode); ok {
		return &PointerType{TypeBase: TypeBase{location: ast.Location()}}
	} else if _, ok := ast.(*parser.SliceTypeNode); ok {
		return &SliceType{TypeBase: TypeBase{location: ast.Location()}}
	} else if _, ok := ast.(*parser.ArrayTypeNode); ok {
		return &ArrayType{TypeBase: TypeBase{location: ast.Location()}}
	} else if _, ok := ast.(*parser.StructTypeNode); ok {
		return &StructType{TypeBase: TypeBase{location: ast.Location()}}
	} else if _, ok := ast.(*parser.UnionTypeNode); ok {
		return &UnionType{TypeBase: TypeBase{location: ast.Location()}}
	} else if _, ok := ast.(*parser.InterfaceTypeNode); ok {
		return &InterfaceType{TypeBase: TypeBase{location: ast.Location()}}
	} else if _, ok := ast.(*parser.ClosureTypeNode); ok {
		return &ClosureType{TypeBase: TypeBase{location: ast.Location()}}
	} else if _, ok := ast.(*parser.FuncTypeNode); ok {
		return &FuncType{TypeBase: TypeBase{location: ast.Location()}}
	} else if _, ok := ast.(*parser.TypeQualifierNode); ok {
		return &QualifiedType{TypeBase: TypeBase{location: ast.Location()}}
	} else if _, ok := ast.(*parser.GenericInstanceTypeNode); ok {
		return &GenericInstanceType{TypeArguments: make(map[string]Type), TypeBase: TypeBase{location: ast.Location()}}
	}
	panic("AST is not a type")
}

func defineType(t Type, ast parser.Node, s *Scope, log *errlog.ErrorLog) error {
	if n, ok := ast.(*parser.NamedTypeNode); ok {
		return defineAliasType(t.(*AliasType), n, s, log)
	} else if n, ok := ast.(*parser.PointerTypeNode); ok {
		return definePointerType(t.(*PointerType), n, s, log)
	} else if n, ok := ast.(*parser.SliceTypeNode); ok {
		return defineSliceType(t.(*SliceType), n, s, log)
	} else if n, ok := ast.(*parser.ArrayTypeNode); ok {
		return defineArrayType(t.(*ArrayType), n, s, log)
	} else if n, ok := ast.(*parser.StructTypeNode); ok {
		return defineStructType(t.(*StructType), n, s, log)
	} else if n, ok := ast.(*parser.UnionTypeNode); ok {
		return defineUnionType(t.(*UnionType), n, s, log)
	} else if n, ok := ast.(*parser.InterfaceTypeNode); ok {
		return defineInterfaceType(t.(*InterfaceType), n, s, log)
	} else if n, ok := ast.(*parser.ClosureTypeNode); ok {
		return defineClosureType(t.(*ClosureType), n, s, log)
	} else if n, ok := ast.(*parser.FuncTypeNode); ok {
		return defineFuncType(t.(*FuncType), n, s, log)
	} else if n, ok := ast.(*parser.TypeQualifierNode); ok {
		return defineQualifiedType(t.(*QualifiedType), n, s, log)
	} else if n, ok := ast.(*parser.GenericInstanceTypeNode); ok {
		return defineGenericInstanceType(t.(*GenericInstanceType), n, s, log)
	}
	panic("AST is not a type")
}

func defineAliasType(t *AliasType, n *parser.NamedTypeNode, s *Scope, log *errlog.ErrorLog) error {
	t.pkg = s.PackageScope().Package
	componentScope := s.ComponentScope()
	if componentScope != nil {
		t.component = componentScope.Component
	}
	var err error
	if t.Alias, err = s.LookupNamedType(n, log); err != nil {
		return err
	}
	return nil
}

func definePointerType(t *PointerType, n *parser.PointerTypeNode, s *Scope, log *errlog.ErrorLog) error {
	t.pkg = s.PackageScope().Package
	componentScope := s.ComponentScope()
	if componentScope != nil {
		t.component = componentScope.Component
	}
	if n.GroupSpecifier != nil {
		t.GroupSpecifier = defineGroupSpecifier(n.GroupSpecifier, log)
	}
	if n.MutableToken.Kind == lexer.TokenDual {
		dualIsMut := s.DualIsMut()
		if dualIsMut == 0 {
			return log.AddError(errlog.ErrorDualOutsideDualFunction, n.MutableToken.Location)
		}
		t.Mutable = dualIsMut == 1
	}
	switch n.PointerToken.Kind {
	case lexer.TokenAmpersand:
		t.Mode = PtrReference
	case lexer.TokenAsterisk:
		t.Mode = PtrOwner
	case lexer.TokenHash:
		t.Mode = PtrUnsafe
	}
	var err error
	if t.ElementType, err = declareAndDefineType(n.ElementType, s, log); err != nil {
		return err
	}
	t.location = n.Location()
	return nil
}

func defineSliceType(t *SliceType, n *parser.SliceTypeNode, s *Scope, log *errlog.ErrorLog) error {
	t.pkg = s.PackageScope().Package
	componentScope := s.ComponentScope()
	if componentScope != nil {
		t.component = componentScope.Component
	}
	var err error
	if t.ElementType, err = declareAndDefineType(n.ElementType, s, log); err != nil {
		return err
	}
	return nil
}

func defineArrayType(t *ArrayType, n *parser.ArrayTypeNode, s *Scope, log *errlog.ErrorLog) error {
	t.pkg = s.PackageScope().Package
	componentScope := s.ComponentScope()
	if componentScope != nil {
		t.component = componentScope.Component
	}
	var err error
	if t.ElementType, err = declareAndDefineType(n.ElementType, s, log); err != nil {
		return err
	}
	var tok *lexer.Token
	if c, ok := n.Size.(*parser.ConstantExpressionNode); ok {
		tok = c.ValueToken
	} else {
		tok, err = computeIntegerToken(n.Size, s, errlog.ErrorArraySizeInteger, log)
		if err != nil {
			return err
		}
	}
	if tok.Kind != lexer.TokenInteger {
		return log.AddError(errlog.ErrorArraySizeInteger, n.Location())
	}
	if !tok.IntegerValue.IsUint64() {
		return log.AddError(errlog.ErrorArraySizeInteger, n.Location())
	}
	t.Size = tok.IntegerValue.Uint64()
	return nil
}

func defineClosureType(t *ClosureType, n *parser.ClosureTypeNode, s *Scope, log *errlog.ErrorLog) error {
	t.pkg = s.PackageScope().Package
	componentScope := s.ComponentScope()
	if componentScope != nil {
		t.component = componentScope.Component
	}
	f := &FuncType{TypeBase: TypeBase{name: t.name, location: t.location}}
	p, err := declareAndDefineParams(n.Params, true, s, log)
	if err != nil {
		return err
	}
	f.In = p
	p, err = declareAndDefineParams(n.ReturnParams, false, s, log)
	if err != nil {
		return err
	}
	f.Out = p
	t.FuncType = f
	// This scope is used to fix the group specifiers only
	innerScope := newScope(s, FunctionScope, n.Location())
	for i, p := range f.In.Params {
		fixParameterGroupSpecifier(f, p, i, innerScope, log)
	}
	for i, p := range f.Out.Params {
		fixReturnGroupSpecifier(f, p, i, innerScope, log)
	}
	return nil
}

func defineFuncType(t *FuncType, n *parser.FuncTypeNode, s *Scope, log *errlog.ErrorLog) error {
	t.pkg = s.PackageScope().Package
	componentScope := s.ComponentScope()
	if componentScope != nil {
		t.component = componentScope.Component
	}
	var err error
	t.In, err = declareAndDefineParams(n.Params, true, s, log)
	if err != nil {
		return err
	}
	t.Out, err = declareAndDefineParams(n.ReturnParams, false, s, log)
	if err != nil {
		return err
	}
	// This scope is used to fix the group specifiers only
	innerScope := newScope(s, FunctionScope, n.Location())
	for i, p := range t.In.Params {
		fixParameterGroupSpecifier(t, p, i, innerScope, log)
	}
	for i, p := range t.Out.Params {
		fixReturnGroupSpecifier(t, p, i, innerScope, log)
	}
	return nil
}

func defineStructType(t *StructType, n *parser.StructTypeNode, s *Scope, log *errlog.ErrorLog) error {
	t.pkg = s.PackageScope().Package
	componentScope := s.ComponentScope()
	if componentScope != nil {
		t.component = componentScope.Component
	}
	var err error
	names := make(map[string]bool)
	ifaces := make(map[*InterfaceType]bool)
	for _, fn := range n.Fields {
		if _, ok := fn.(*parser.LineNode); ok {
			continue
		}
		if sn, ok := fn.(*parser.StructFieldNode); ok {
			// An interface of base type?
			if sn.NameToken == nil {
				baseType, err := declareAndDefineType(sn.Type, s, log)
				if err != nil {
					return err
				}
				if ifaceType, ok := baseType.(*InterfaceType); ok {
					if _, ok = ifaces[ifaceType]; ok {
						return log.AddError(errlog.ErrorStructDuplicateInterface, sn.Location())
					}
					ifaces[ifaceType] = true
					t.Interfaces = append(t.Interfaces, ifaceType)
					continue
				}
				structType, ok := baseType.(*StructType)
				if !ok {
					return log.AddError(errlog.ErrorStructBaseType, sn.Location())
				}
				if t.BaseType != nil {
					return log.AddError(errlog.ErrorStructSingleBaseType, sn.Location())
				}
				t.BaseType = structType
				f := &StructField{Name: structType.name, Type: structType, IsBaseType: true}
				if _, ok := names[f.Name]; ok {
					return log.AddError(errlog.ErrorStructDuplicateField, sn.NameToken.Location, f.Name)
				}
				names[f.Name] = true
				t.Fields = append(t.Fields, f)
			} else {
				f := &StructField{}
				f.Name = sn.NameToken.StringValue
				f.Type, err = declareAndDefineType(sn.Type, s, log)
				if err != nil {
					return err
				}
				if _, ok := names[f.Name]; ok {
					return log.AddError(errlog.ErrorStructDuplicateField, sn.NameToken.Location, f.Name)
				}
				names[f.Name] = true
				t.Fields = append(t.Fields, f)
			}
			continue
		}
		panic("Should not be here")
	}
	return nil
}

func defineUnionType(t *UnionType, n *parser.UnionTypeNode, s *Scope, log *errlog.ErrorLog) error {
	t.pkg = s.PackageScope().Package
	componentScope := s.ComponentScope()
	if componentScope != nil {
		t.component = componentScope.Component
	}
	var err error
	names := make(map[string]bool)
	for _, fn := range n.Fields {
		if _, ok := fn.(*parser.LineNode); ok {
			continue
		}
		if sn, ok := fn.(*parser.StructFieldNode); ok {
			f := &StructField{}
			f.Name = sn.NameToken.StringValue
			f.Type, err = declareAndDefineType(sn.Type, s, log)
			if err != nil {
				return err
			}
			if _, ok := names[f.Name]; ok {
				return log.AddError(errlog.ErrorStructDuplicateField, sn.NameToken.Location, f.Name)
			}
			names[f.Name] = true
			t.Fields = append(t.Fields, f)
		}
	}
	return nil
}

func defineInterfaceType(t *InterfaceType, n *parser.InterfaceTypeNode, s *Scope, log *errlog.ErrorLog) error {
	t.pkg = s.PackageScope().Package
	componentScope := s.ComponentScope()
	if componentScope != nil {
		t.component = componentScope.Component
	}
	names := make(map[string]bool)
	ifaces := make(map[*InterfaceType]bool)
	for _, fn := range n.Fields {
		if _, ok := fn.(*parser.LineNode); ok {
			continue
		}
		if ifn, ok := fn.(*parser.InterfaceFuncNode); ok {
			target := &PointerType{GroupSpecifier: NewGroupSpecifier("this", ifn.Location()), Mutable: ifn.MutToken != nil, Mode: PtrReference, ElementType: t, TypeBase: TypeBase{location: t.Location()}}
			ft := &FuncType{TypeBase: TypeBase{name: ifn.NameToken.StringValue, location: ifn.Location()}, Target: target}
			f := &InterfaceFunc{Name: ifn.NameToken.StringValue, FuncType: ft}
			if _, ok := names[f.Name]; ok {
				return log.AddError(errlog.ErrorInterfaceDuplicateFunc, ifn.Location(), f.Name)
			}
			names[f.Name] = true
			p, err := declareAndDefineParams(ifn.Params, true, s, log)
			if err != nil {
				return err
			}
			f.FuncType.In = p
			p, err = declareAndDefineParams(ifn.ReturnParams, false, s, log)
			if err != nil {
				return err
			}
			f.FuncType.Out = p
			t.Funcs = append(t.Funcs, f)
			continue
		}
		if ifn, ok := fn.(*parser.InterfaceFieldNode); ok {
			typ, err := declareAndDefineType(ifn.Type, s, log)
			if err != nil {
				return err
			}
			ifaceType, ok := typ.(*InterfaceType)
			if !ok {
				return log.AddError(errlog.ErrorInterfaceBaseType, ifn.Location())
			}
			if _, ok = ifaces[ifaceType]; ok {
				return log.AddError(errlog.ErrorInterfaceDuplicateInterface, ifn.Location())
			}
			ifaces[ifaceType] = true
			t.BaseTypes = append(t.BaseTypes, ifaceType)
			continue
		}
		panic("Oooops")
	}
	return nil
}

func defineGroupSpecifier(n *parser.GroupSpecifierNode, log *errlog.ErrorLog) *GroupSpecifier {
	gs := &GroupSpecifier{Location: n.Location()}
	for _, g := range n.Groups {
		ge := GroupSpecifierElement{Name: g.NameToken.StringValue, Arrow: g.ArrowToken != nil}
		gs.Elements = append(gs.Elements, ge)
	}
	return gs
}

func defineQualifiedType(t *QualifiedType, n *parser.TypeQualifierNode, s *Scope, log *errlog.ErrorLog) error {
	t.pkg = s.PackageScope().Package
	componentScope := s.ComponentScope()
	if componentScope != nil {
		t.component = componentScope.Component
	}
	if n.VolatileToken != nil {
		t.Volatile = true
	}
	var err error
	if t.Type, err = declareAndDefineType(n.Type, s, log); err != nil {
		return err
	}
	if m, ok := t.Type.(*QualifiedType); ok {
		t.Const = t.Const || m.Const
		t.Volatile = t.Volatile || m.Volatile
		t.Type = m.Type
	}
	return nil
}

func defineGenericInstanceType(t *GenericInstanceType, n *parser.GenericInstanceTypeNode, s *Scope, log *errlog.ErrorLog) error {
	// The GenericInstanceType belongs to the package in which the generic has been instantiated, and
	// not to the package in which the generic has been defined.
	t.pkg = s.InstantiatingPackage()
	// t.pkg = s.PackageScope().Package
	componentScope := s.ComponentScope()
	if componentScope != nil {
		t.component = componentScope.Component
	}
	basetype, err := s.LookupNamedType(n.Type, log)
	if err != nil {
		return err
	}
	var ok bool
	t.BaseType, ok = basetype.(*GenericType)
	if !ok {
		return log.AddError(errlog.ErrorNotAGenericType, n.Type.Location())
	}
	t.name = t.BaseType.Name()
	// The generic type is instantiated in a scope that belongs to the package in which the generic has been defined.
	t.GenericScope = newScope(t.BaseType.Scope(), GenericTypeScope, s.Location)
	// However, note that the functions in this scope (the member functions of the generic) need to be generated in the
	// package that instantiated the generic (not in the one that defined the generic).
	t.GenericScope.Package = t.pkg
	t.GenericScope.AddType(t, log)
	if len(n.TypeArguments.Types) != len(t.BaseType.TypeParameters) {
		return log.AddError(errlog.ErrorWrongTypeArgumentCount, n.TypeArguments.Location())
	}
	for i, arg := range n.TypeArguments.Types {
		pt, err := declareAndDefineType(arg.Type, s, log)
		if err != nil {
			return err
		}
		name := t.BaseType.TypeParameters[i].Name
		//pt.setName(name)
		t.TypeArguments[name] = pt
		t.GenericScope.AddTypeByName(pt, name, log)
	}
	// TODO: Use a unique type signature
	typesig := t.ToString()
	if equivalent, ok := t.pkg.lookupGenericInstanceType(typesig); ok {
		t.equivalent = equivalent
		t.InstanceType = equivalent.InstanceType
	} else {
		t.InstanceType, err = declareAndDefineType(t.BaseType.Type, t.GenericScope, log)
		if err == nil {
			t.pkg.registerGenericInstanceType(typesig, t)
		}
	}
	return err
}

func declareAndDefineParams(list *parser.ParamListNode, mustBeNamed bool, s *Scope, log *errlog.ErrorLog) (*ParameterList, error) {
	var err error
	pl := &ParameterList{}
	if list != nil {
		for _, pn := range list.Params {
			p := &Parameter{Location: pn.Location()}
			if pn.NameToken != nil {
				p.Name = pn.NameToken.StringValue
			} else if mustBeNamed {
				return nil, log.AddError(errlog.ErrorUnnamedParameter, pn.Location(), p.Name)
			}
			if p.Type, err = declareAndDefineType(pn.Type, s, log); err != nil {
				return nil, err
			}
			for _, p2 := range pl.Params {
				if p2.Name != "" && p2.Name == p.Name {
					return nil, log.AddError(errlog.ErrorDuplicateParameter, pn.Location(), p.Name)
				}
			}
			pl.Params = append(pl.Params, p)
		}
	}
	return pl, nil
}

func computeIntegerToken(n parser.Node, s *Scope, errorCode errlog.ErrorCode, log *errlog.ErrorLog) (*lexer.Token, error) {
	if err := checkExpression(n, s, log); err != nil {
		return nil, err
	}
	et := exprType(n)
	if !et.IsConstant() {
		return nil, log.AddError(errorCode, n.Location())
	}
	return &lexer.Token{Kind: lexer.TokenInteger, Location: n.Location(), IntegerValue: et.IntegerValue}, nil
}
