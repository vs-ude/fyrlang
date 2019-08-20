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
	return f, s.AddElement(f, ast.LocationRange(), log)
}

func declareFunction(ast *parser.FuncNode, s *Scope, log *errlog.ErrorLog) (*Func, error) {
	if ast.GenericParams != nil {
		panic("Wrong")
	}
	var err error
	loc := parser.NodeLocation(ast)
	ft := &FuncType{TypeBase: TypeBase{name: ast.NameToken.StringValue, location: loc}}
	f := &Func{name: ast.NameToken.StringValue, Type: ft, ast: ast, OuterScope: s, Location: loc}
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
				return nil, log.AddError(errlog.ErrorDuplicateScopeName, ast.LocationRange(), f.name)
			}
			target.Funcs = append(target.Funcs, f)
		case *AliasType:
			if target.HasMember(f.name) {
				return nil, log.AddError(errlog.ErrorDuplicateScopeName, ast.LocationRange(), f.name)
			}
			target.Funcs = append(target.Funcs, f)
		case *GenericInstanceType:
			if target.HasMember(f.name) {
				return nil, log.AddError(errlog.ErrorDuplicateScopeName, ast.LocationRange(), f.name)
			}
			target.Funcs = append(target.Funcs, f)
		case *GenericType:
			if target.HasMember(f.name) {
				return nil, log.AddError(errlog.ErrorDuplicateScopeName, ast.LocationRange(), f.name)
			}
			target.Funcs = append(target.Funcs, f)
			// Do not inspect the function signature. This is done upon instantiation
			return f, nil
		default:
			return nil, log.AddError(errlog.ErrorTypeCannotHaveFunc, ast.LocationRange())
		}
	}
	f.Type.In, err = declareAndDefineParams(ast.Params, true, s, log)
	if err != nil {
		return nil, err
	}
	f.Type.Out, err = declareAndDefineParams(ast.ReturnParams, false, s, log)
	if err != nil {
		return nil, err
	}
	if f.Target == nil {
		return f, s.AddElement(f, ast.LocationRange(), log)
	}
	return f, nil
}
