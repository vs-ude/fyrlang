package types

import (
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/lexer"
	"github.com/vs-ude/fyrlang/internal/parser"
)

func declareGenericFunction(ast *parser.FuncNode, s *Scope, log *errlog.ErrorLog) (*GenericFunc, error) {
	if ast.GenericParams == nil {
		panic("Wrong")
	}
	var cmp *ComponentType
	if cmpScope := s.ComponentScope(); cmpScope != nil {
		cmp = cmpScope.Component
	}
	f := &GenericFunc{name: ast.NameToken.StringValue, ast: ast, Component: cmp}
	for _, p := range ast.GenericParams.Params {
		f.TypeParameters = append(f.TypeParameters, &GenericTypeParameter{Name: p.NameToken.StringValue})
	}
	if err := parseGenericFuncAttribs(ast, f, log); err != nil {
		return nil, err
	}
	return f, s.AddElement(f, ast.Location(), log)
}

func declareFunction(ast *parser.FuncNode, s *Scope, log *errlog.ErrorLog) (*Func, error) {
	if ast.GenericParams != nil {
		panic("Wrong")
	}
	var err error
	var name string
	// Destructor ?
	if ast.TildeToken != nil {
		name = "__dtor__"
	} else {
		name = ast.NameToken.StringValue
	}
	loc := ast.Location()
	ft := &FuncType{TypeBase: TypeBase{name: name, location: loc, pkg: s.PackageScope().Package}}
	// Destructor ?
	if ast.TildeToken != nil {
		ft.IsDestructor = true
	}
	f := &Func{name: name, Type: ft, Ast: ast, OuterScope: s, Location: loc, Component: s.Component}
	f.InnerScope = newScope(f.OuterScope, FunctionScope, f.Location)
	f.InnerScope.Func = f
	if ast.Type != nil {
		if mt, ok := ast.Type.(*parser.MutableTypeNode); ok && mt.MutToken.Kind == lexer.TokenDual {
			if s.dualIsMut != -1 {
				f.DualIsMut = true
				s.dualIsMut = 1
				f.InnerScope.dualIsMut = 1
			} else {
				f.InnerScope.dualIsMut = -1
			}
		}
		ft.Target, err = declareAndDefineType(ast.Type, s, log)
		if err != nil {
			return nil, err
		}
		t := ft.Target
		targetIsPointer := false
		// The target for a destructor is always a pointer to the type it is destructing.
		if ast.TildeToken != nil {
			if _, ok := t.(*StructType); !ok {
				return nil, log.AddError(errlog.ErrorWrongTypeForDestructor, ast.Type.Location())
			}
			targetIsPointer = true
			ft.Target = &MutableType{Mutable: true, Type: &PointerType{Mode: PtrOwner, ElementType: ft.Target}}
		} else {
			if m, ok := t.(*MutableType); ok {
				t = m.Type
			}
			if ptr, ok := t.(*PointerType); ok {
				t = ptr.ElementType
				targetIsPointer = true
			}
		}
		if s.dualIsMut != -1 {
			// Do not register the dual function with its target type.
			switch target := t.(type) {
			case *StructType:
				if target.HasMember(f.name) {
					return nil, log.AddError(errlog.ErrorDuplicateScopeName, ast.Location(), f.name)
				}
				target.Funcs = append(target.Funcs, f)
			case *AliasType:
				if target.HasMember(f.name) {
					return nil, log.AddError(errlog.ErrorDuplicateScopeName, ast.Location(), f.name)
				}
				target.Funcs = append(target.Funcs, f)
			case *GenericInstanceType:
				if target.HasMember(f.name) {
					return nil, log.AddError(errlog.ErrorDuplicateScopeName, ast.Location(), f.name)
				}
				target.Funcs = append(target.Funcs, f)
			case *GenericType:
				if target.HasMember(f.name) {
					return nil, log.AddError(errlog.ErrorDuplicateScopeName, ast.Location(), f.name)
				}
				target.Funcs = append(target.Funcs, f)
				if err := parseFuncAttribs(ast, f, log); err != nil {
					return nil, err
				}
				// Do not inspect the function signature. This is done upon instantiation
				return f, nil
			default:
				return nil, log.AddError(errlog.ErrorTypeCannotHaveFunc, ast.Location())
			}
		}
		fixTargetGroupSpecifier(ft, f.InnerScope, ast.Type.Location(), log)
		tthis := makeExprType(ft.Target)
		// If the target is not a pointer, then this is a value and it can be modified.
		// The same applies to destructors.
		if !targetIsPointer {
			tthis.Mutable = true
		}
		vthis := &Variable{name: "this", Type: tthis}
		f.InnerScope.AddElement(vthis, ast.Type.Location(), log)
	}
	f.Type.In, err = declareAndDefineParams(ast.Params, true, f.InnerScope, log)
	if err != nil {
		return nil, err
	}
	f.Type.Out, err = declareAndDefineParams(ast.ReturnParams, false, f.InnerScope, log)
	if err != nil {
		return nil, err
	}
	for i, p := range f.Type.In.Params {
		fixParameterGroupSpecifier(ft, p, i, f.InnerScope, log)
	}
	for i, p := range f.Type.Out.Params {
		fixReturnGroupSpecifier(ft, p, i, f.InnerScope, log)
	}
	if err := parseFuncAttribs(ast, f, log); err != nil {
		return nil, err
	}
	return f, nil
}

// If a function parameter has no group specifier but has a pointer type,
// this function adds an implicit group specifier.
// Furthermore, all isolate group specifiers should get an implicit name.
func fixParameterGroupSpecifier(ft *FuncType, p *Parameter, pos int, s *Scope, log *errlog.ErrorLog) {
	if TypeHasPointers(p.Type) {
		et := NewExprType(p.Type)
		if et.PointerDestGroupSpecifier != nil {
			name := et.PointerDestGroupSpecifier.Name
			if name == "" {
				name = p.Name
			}
			g, _ := s.LookupOrCreateGroupSpecifier(name, p.Location, et.PointerDestGroupSpecifier.Kind, log)
			p.Type = replaceGrouping(p.Type, g)
		} else {
			g, _ := s.LookupOrCreateGroupSpecifier(p.Name, p.Location, GroupSpecifierNamed, log)
			p.Type = &GroupedType{GroupSpecifier: g, Type: p.Type, TypeBase: TypeBase{location: p.Location, component: p.Type.Component(), pkg: p.Type.Package()}}
		}
	}
}

// If a function return parameter has no group specifier but has a pointer type,
// this function adds an implicit group specifier.
// Furthermore, all isolate group specifiers should get an implicit name.
func fixReturnGroupSpecifier(ft *FuncType, p *Parameter, pos int, s *Scope, log *errlog.ErrorLog) {
	if TypeHasPointers(p.Type) {
		et := NewExprType(p.Type)
		var name string
		if et.PointerDestGroupSpecifier != nil && et.PointerDestGroupSpecifier.Name != "" {
			name = et.PointerDestGroupSpecifier.Name
		} else if p.Name != "" {
			name = p.Name
		} else {
			// Return parameters can have no name. Construct one for the group
			// that does not depend on the parameter name.
			name = strconv.Itoa(pos) + "_return"
		}
		if et.PointerDestGroupSpecifier != nil {
			g, _ := s.LookupOrCreateGroupSpecifier(name, p.Location, et.PointerDestGroupSpecifier.Kind, log)
			p.Type = replaceGrouping(p.Type, g)
		} else {
			g, _ := s.LookupOrCreateGroupSpecifier(name, p.Location, GroupSpecifierNamed, log)
			p.Type = &GroupedType{GroupSpecifier: g, Type: p.Type, TypeBase: TypeBase{location: p.Location, component: p.Type.Component(), pkg: p.Type.Package()}}
		}
	}
}

// For methods, the target type (i.e. the type of `this`) has an implicit group specifier named `"this"`.
func fixTargetGroupSpecifier(ft *FuncType, s *Scope, loc errlog.LocationRange, log *errlog.ErrorLog) {
	if TypeHasPointers(ft.Target) {
		et := NewExprType(ft.Target)
		if et.PointerDestGroupSpecifier == nil {
			g, _ := s.LookupOrCreateGroupSpecifier("this", loc, GroupSpecifierNamed, log)
			ft.Target = &GroupedType{GroupSpecifier: g, Type: ft.Target, TypeBase: TypeBase{location: loc, component: ft.Component(), pkg: ft.Package()}}
		}
	}
}

func replaceGrouping(t Type, g *GroupSpecifier) Type {
	grp, ok := t.(*GroupedType)
	if !ok {
		panic("Oooops")
	}
	grp.GroupSpecifier = g
	return t
}

func declareExternFunction(ast *parser.ExternFuncNode, s *Scope, log *errlog.ErrorLog) (*Func, error) {
	ft := &FuncType{TypeBase: TypeBase{name: ast.NameToken.StringValue, location: ast.Location(), pkg: s.PackageScope().Package}}
	f := &Func{name: ast.NameToken.StringValue, Type: ft, Ast: nil, OuterScope: s, Location: ast.Location(), IsExtern: true}
	if err := parseExternFuncAttribs(ast, f, log); err != nil {
		return nil, err
	}
	p, err := declareAndDefineParams(ast.Params, true, s, log)
	if err != nil {
		return nil, err
	}
	ft.In = p
	p, err = declareAndDefineParams(ast.ReturnParams, false, s, log)
	if err != nil {
		return nil, err
	}
	ft.Out = p
	return f, nil
}
