package types

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

func declareGenericFunction(ast *parser.FuncNode, s *Scope, log *errlog.ErrorLog) (*GenericFunc, error) {
	if ast.GenericParams == nil {
		panic("Wrong")
	}
	f := &GenericFunc{name: ast.NameToken.StringValue, ast: ast}
	for _, p := range ast.GenericParams.Params {
		f.TypeParameters = append(f.TypeParameters, &GenericTypeParameter{Name: p.NameToken.StringValue})
	}
	return f, s.AddElement(f, ast.Location(), log)
}

func declareFunction(ast *parser.FuncNode, s *Scope, log *errlog.ErrorLog) (*Func, error) {
	if ast.GenericParams != nil {
		panic("Wrong")
	}
	var err error
	loc := ast.Location()
	ft := &FuncType{TypeBase: TypeBase{name: ast.NameToken.StringValue, location: loc, pkg: s.PackageScope().Package}}
	f := &Func{name: ast.NameToken.StringValue, Type: ft, Ast: ast, OuterScope: s, Location: loc}
	f.InnerScope = newScope(f.OuterScope, FunctionScope, f.Location)
	f.InnerScope.Func = f
	if ast.Type != nil {
		f.Target, err = declareAndDefineType(ast.Type, s, log)
		if err != nil {
			return nil, err
		}
		t := f.Target
		if m, ok := t.(*MutableType); ok {
			t = m.Type
		}
		if ptr, ok := t.(*PointerType); ok {
			t = ptr.ElementType
		}
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
			// Do not inspect the function signature. This is done upon instantiation
			return f, nil
		default:
			return nil, log.AddError(errlog.ErrorTypeCannotHaveFunc, ast.Location())
		}
	}
	f.Type.In, err = declareAndDefineParams(ast.Params, true, f.InnerScope, log)
	if err != nil {
		return nil, err
	}
	f.Type.Out, err = declareAndDefineParams(ast.ReturnParams, false, f.InnerScope, log)
	if err != nil {
		return nil, err
	}
	if f.Target == nil {
		return f, s.AddElement(f, ast.Location(), log)
	}
	return f, nil
}

func declareExternFunction(ast *parser.ExternFuncNode, s *Scope, log *errlog.ErrorLog) (*Func, error) {
	ft := &FuncType{TypeBase: TypeBase{name: ast.NameToken.StringValue, location: ast.Location(), pkg: s.PackageScope().Package}}
	f := &Func{name: ast.NameToken.StringValue, Type: ft, Ast: nil, OuterScope: s, Location: ast.Location(), IsExtern: true}
	if ast.ExportToken != nil {
		f.IsExported = true
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
	return f, s.AddElement(f, ast.Location(), log)
}
