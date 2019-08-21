package types

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

// Find all functions attached to a type by traversing the type graph.
// Check the implementation of each function.
// This is required, because template types instantiate new functions, which must be checked.
func checkFuncs(t Type, log *errlog.ErrorLog) error {
	// TODO: Ignore types defined in a different package
	switch t2 := t.(type) {
	case *AliasType:
		if t2.TypeBase.funcsChecked {
			return nil
		}
		t2.funcsChecked = true
		for _, f := range t2.Funcs {
			if err := checkFuncs(f.Type, log); err != nil {
				return err
			}
			if err := checkFuncBody(f, log); err != nil {
				return err
			}
		}
		return checkFuncs(t2.Alias, log)
	case *PointerType:
		if t2.TypeBase.funcsChecked {
			return nil
		}
		t2.funcsChecked = true
		return checkFuncs(t2.ElementType, log)
	case *MutableType:
		if t2.TypeBase.funcsChecked {
			return nil
		}
		t2.funcsChecked = true
		return checkFuncs(t2.Type, log)
	case *SliceType:
		if t2.TypeBase.funcsChecked {
			return nil
		}
		t2.funcsChecked = true
		return checkFuncs(t2.ElementType, log)
	case *ArrayType:
		if t2.TypeBase.funcsChecked {
			return nil
		}
		t2.funcsChecked = true
		return checkFuncs(t2.ElementType, log)
	case *StructType:
		if t2.TypeBase.funcsChecked {
			return nil
		}
		t2.funcsChecked = true
		if t2.BaseType != nil {
			if err := checkFuncs(t2.BaseType, log); err != nil {
				return err
			}
		}
		for _, iface := range t2.Interfaces {
			if err := checkFuncs(iface, log); err != nil {
				return err
			}
		}
		for _, f := range t2.Funcs {
			if err := checkFuncs(f.Type, log); err != nil {
				return err
			}
			if err := checkFuncBody(f, log); err != nil {
				return err
			}
		}
		return nil
	case *InterfaceType:
		if t2.TypeBase.funcsChecked {
			return nil
		}
		t2.funcsChecked = true
		for _, b := range t2.BaseTypes {
			if err := checkFuncs(b, log); err != nil {
				return err
			}
		}
		for _, f := range t2.Funcs {
			if err := checkFuncs(f.FuncType, log); err != nil {
				return err
			}
		}
		return nil
	case *ClosureType:
		if t2.TypeBase.funcsChecked {
			return nil
		}
		t2.funcsChecked = true
		return checkFuncs(t2.FuncType, log)
	case *GroupType:
		if t2.TypeBase.funcsChecked {
			return nil
		}
		t2.funcsChecked = true
		return checkFuncs(t2.Type, log)
	case *FuncType:
		if t2.TypeBase.funcsChecked {
			return nil
		}
		t2.funcsChecked = true
		// TODO: Check that target is acceptable
		for _, p := range t2.In.Params {
			if err := checkFuncs(p.Type, log); err != nil {
				return err
			}
		}
		for _, p := range t2.Out.Params {
			if err := checkFuncs(p.Type, log); err != nil {
				return err
			}
		}
		return nil
	case *GenericInstanceType:
		if t2.TypeBase.funcsChecked {
			return nil
		}
		t2.funcsChecked = true
		if err := checkFuncs(t2.InstanceType, log); err != nil {
			return err
		}
		for _, f := range t2.Funcs {
			if err := checkFuncs(f.Type, log); err != nil {
				return err
			}
			if err := checkFuncBody(f, log); err != nil {
				return err
			}
		}
		return nil
	case *GenericType:
		// Do nothing
		return nil
	case *PrimitiveType:
		// Do nothing
		return nil
	}
	panic("Wrong")
}

func checkFuncBody(f *Func, log *errlog.ErrorLog) error {
	for _, p := range f.Type.In.Params {
		et := makeExprType(p.Type)
		et.Group = f.InnerScope.Group
		if err := f.InnerScope.AddElement(&Variable{name: p.Name, Component: f.Component, Type: et}, p.Location, log); err != nil {
			return err
		}
	}
	for _, p := range f.Type.Out.Params {
		if p.Name == "" {
			continue
		}
		et := makeExprType(p.Type)
		et.Group = f.InnerScope.Group
		et.Mutable = true
		if err := f.InnerScope.AddElement(&Variable{name: p.Name, Component: f.Component, Type: et}, p.Location, log); err != nil {
			return err
		}
	}
	println("CHECK FUNC", f.Name())
	err := checkBody(f.Ast.Body, f.InnerScope, log)
	return err
}

func checkBody(ast *parser.BodyNode, s *Scope, log *errlog.ErrorLog) error {
	var err error
	for _, ch := range ast.Children {
		err2 := checkStatement(ch, s, log)
		if err2 != nil {
			if err == nil {
				err = err2
			}
		}
	}
	return nil
}
