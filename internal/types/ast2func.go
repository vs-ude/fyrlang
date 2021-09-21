package types

import (
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
	name := ast.NameToken.StringValue
	loc := ast.Location()
	ft := &FuncType{TypeBase: TypeBase{name: name, location: loc, pkg: s.PackageScope().Package}}
	f := &Func{name: name, Type: ft, Ast: ast, OuterScope: s, Location: loc, Component: s.Component}
	f.InnerScope = newScope(f.OuterScope, FunctionScope, f.Location)
	f.InnerScope.Func = f

	specifiers := make(map[string]bool)

	// Member function?
	if ast.Type != nil {
		// A dual member function?
		if mt, ok := ast.Type.(*parser.PointerTypeNode); ok && mt.MutableToken.Kind == lexer.TokenDual {
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
		// The target for a destructor is always a mutable reference to the type it is destructing.
		if name == "__dtor__" {
			ptr, ok := t.(*PointerType)
			if !ok || !ptr.Mutable {
				return nil, log.AddError(errlog.ErrorWrongTypeForDestructor, ast.Type.Location())
			}
			t = ptr.ElementType
			targetIsPointer = true
		} else {
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
		tthis := NewExprType(ft.Target)
		// If the target is not a pointer, then this is a value and it can be modified.
		// The same applies to destructors.
		if !targetIsPointer {
			tthis.Mutable = true
		}
		vthis := &Variable{name: "this", Type: tthis}
		f.InnerScope.AddElement(vthis, ast.Type.Location(), log)
		specifiers["this"] = true
	}
	f.Type.In, err = declareAndDefineParams(ast.Params, true, specifiers, f.InnerScope, log)
	if err != nil {
		return nil, err
	}
	f.Type.Out, err = declareAndDefineParams(ast.ReturnParams, false, specifiers, f.InnerScope, log)
	if err != nil {
		return nil, err
	}
	if err := parseFuncAttribs(ast, f, log); err != nil {
		return nil, err
	}
	return f, nil
}

func declareExternFunction(ast *parser.ExternFuncNode, s *Scope, log *errlog.ErrorLog) (*Func, error) {
	ft := &FuncType{TypeBase: TypeBase{name: ast.NameToken.StringValue, location: ast.Location(), pkg: s.PackageScope().Package}}
	f := &Func{name: ast.NameToken.StringValue, Type: ft, Ast: nil, OuterScope: s, Location: ast.Location(), IsExtern: true}
	if err := parseExternFuncAttribs(ast, f, log); err != nil {
		return nil, err
	}
	specifiers := make(map[string]bool)
	p, err := declareAndDefineParams(ast.Params, true, specifiers, s, log)
	if err != nil {
		return nil, err
	}
	ft.In = p
	p, err = declareAndDefineParams(ast.ReturnParams, false, specifiers, s, log)
	if err != nil {
		return nil, err
	}
	ft.Out = p
	return f, nil
}
