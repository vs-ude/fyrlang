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
	var cmp *ComponentType
	if cmpScope := s.ComponentScope(); cmpScope != nil {
		cmp = cmpScope.Component
	}
	var err error
	loc := ast.Location()
	ft := &FuncType{TypeBase: TypeBase{name: ast.NameToken.StringValue, location: loc, pkg: s.PackageScope().Package}}
	f := &Func{name: ast.NameToken.StringValue, Type: ft, Ast: ast, OuterScope: s, Component: cmp, Location: loc}
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
		if m, ok := t.(*MutableType); ok {
			t = m.Type
		}
		if ptr, ok := t.(*PointerType); ok {
			t = ptr.ElementType
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
		fixTargetGroupSpecifier(ft, f.InnerScope, ast.Type.Location())
		vthis := &Variable{name: "this", Type: makeExprType(ft.Target)}
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
		fixParameterGroupSpecifier(ft, p, i, f.InnerScope)
	}
	for i, p := range f.Type.Out.Params {
		fixReturnGroupSpecifier(ft, p, i, f.InnerScope)
	}
	if err := parseFuncAttribs(ast, f, log); err != nil {
		return nil, err
	}
	return f, nil
}

// If a function parameter has no group specifier but has a pointer type,
// this function adds an implicit group specifier.
// Furthermore, all isolate group specifiers should get an implicit name.
func fixParameterGroupSpecifier(ft *FuncType, p *Parameter, pos int, s *Scope) {
	if TypeHasPointers(p.Type) {
		et := NewExprType(p.Type)
		if et.PointerDestGroupSpecifier != nil {
			if et.PointerDestGroupSpecifier.Kind == GroupSpecifierIsolate {
				et.PointerDestGroupSpecifier.Name = p.Name
			}
		} else {
			// g := &GroupSpecifier{Kind: GroupSpecifierNamed, Name: p.Name, Location: p.Location}
			g := s.LookupOrCreateGroupSpecifier(p.Name, p.Location)
			p.Type = &GroupedType{GroupSpecifier: g, Type: p.Type, TypeBase: TypeBase{location: p.Location, component: p.Type.Component(), pkg: p.Type.Package()}}
		}
	}
}

// If a function return parameter has no group specifier but has a pointer type,
// this function adds an implicit group specifier.
// Furthermore, all isolate group specifiers should get an implicit name.
func fixReturnGroupSpecifier(ft *FuncType, p *Parameter, pos int, s *Scope) {
	if TypeHasPointers(p.Type) {
		et := NewExprType(p.Type)
		if et.PointerDestGroupSpecifier != nil {
			if et.PointerDestGroupSpecifier.Kind == GroupSpecifierIsolate {
				if p.Name == "" {
					// Return parameters can have no name. Construct one for the group
					// that does not depend on the parameter name.
					et.PointerDestGroupSpecifier.Name = strconv.Itoa(pos) + "_return"
				} else {
					et.PointerDestGroupSpecifier.Name = p.Name
				}
			}
		} else {
			// g := &GroupSpecifier{Kind: GroupSpecifierNamed, Name: p.Name, Location: p.Location}
			g := s.LookupOrCreateGroupSpecifier(p.Name, p.Location)
			if p.Name == "" {
				// Return parameters can have no name. Construct one for the group
				// that does not depend on the parameter name.
				g.Name = strconv.Itoa(pos) + "_return"
			}
			p.Type = &GroupedType{GroupSpecifier: g, Type: p.Type, TypeBase: TypeBase{location: p.Location, component: p.Type.Component(), pkg: p.Type.Package()}}
		}
	}
}

// For methods, the target type (i.e. the type of `this`) has an implicit group specifier named `"this"`.
func fixTargetGroupSpecifier(ft *FuncType, s *Scope, loc errlog.LocationRange) {
	if TypeHasPointers(ft.Target) {
		et := NewExprType(ft.Target)
		if et.PointerDestGroupSpecifier == nil {
			// g := &GroupSpecifier{Kind: GroupSpecifierNamed, Name: "this", Location: ft.Location()}
			g := s.LookupOrCreateGroupSpecifier("this", ft.Location())
			ft.Target = &GroupedType{GroupSpecifier: g, Type: ft.Target, TypeBase: TypeBase{location: loc, component: ft.Component(), pkg: ft.Package()}}
		}
	}
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
