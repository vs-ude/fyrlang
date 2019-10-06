package types

import (
	"path/filepath"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

// ParseFile ...
func ParseFile(p *Package, f *parser.FileNode, lmap *errlog.LocationMap, log *errlog.ErrorLog) error {
	var cmp *ComponentType
	s := newScope(p.Scope, FileScope, f.Location())
	// Imports and component
	for _, n := range f.Children {
		if impBlock, ok := n.(*parser.ImportBlockNode); ok {
			for _, nImp := range impBlock.Imports {
				if _, ok := nImp.(*parser.LineNode); ok {
					continue
				} else if imp, ok := nImp.(*parser.ImportNode); ok {
					pnew, err := LookupPackage(imp.StringToken.StringValue, p, imp.StringToken.Location, lmap, log)
					if err != nil {
						return err
					}
					err = pnew.Parse(lmap, log)
					if err != nil {
						return err
					}
					ns := &Namespace{Scope: pnew.Scope}
					if imp.NameToken != nil {
						ns.name = imp.NameToken.StringValue
					} else {
						ns.name = filepath.Base(pnew.Path)
					}
					err = s.AddNamespace(ns, imp.Location(), log)
					if err != nil {
						return err
					}
					// Let `pnew` import the runtime as well
					runtime := p.RuntimePackage()
					if runtime != nil {
						pnew.addImport(runtime)
					}
					// Import package `pnew`
					p.addImport(pnew)
				} else {
					panic("Wrong")
				}
			}
		} else if cn, ok := n.(*parser.ComponentNode); ok {
			if cmp != nil {
				return log.AddError(errlog.ErrorComponentTwice, cn.Location())
			}
			s = newScope(s, ComponentScope, s.Location)
			cmp = &ComponentType{Scope: s, TypeBase: TypeBase{name: cn.NameToken.StringValue, location: cn.Location()}}
			s.Component = cmp
			p.Scope.AddType(cmp, log)
		}
	}
	// Declare all named types
	types := make(map[*parser.TypedefNode]Type)
	for _, n := range f.Children {
		if tdef, ok := n.(*parser.TypedefNode); ok {
			var typ Type
			if tdef.GenericParams != nil {
				gt := &GenericType{Type: tdef.Type, Scope: s}
				for _, p := range tdef.GenericParams.Params {
					gt.TypeParameters = append(gt.TypeParameters, &GenericTypeParameter{Name: p.NameToken.StringValue})
				}
				typ = gt
			} else {
				typ = declareType(tdef.Type)
			}
			typ.setName(tdef.NameToken.StringValue)
			err := s.AddType(typ, log)
			if err != nil {
				return err
			}
			types[tdef] = typ
		}
	}
	// Declare functions and attach them to types where applicable
	var funcs []*Func
	for _, n := range f.Children {
		if fn, ok := n.(*parser.FuncNode); ok {
			if fn.GenericParams != nil {
				f, err := declareGenericFunction(fn, s, log)
				f.Component = cmp
				if err != nil {
					return err
				}
			} else {
				f, err := declareFunction(fn, s, log)
				if err != nil {
					return err
				}
				f.Component = cmp
				if f.Target == nil {
					funcs = append(funcs, f)
				}
				if !f.IsGenericMemberFunc() {
					p.Funcs = append(p.Funcs, f)
				}
			}
		} else if en, ok := n.(*parser.ExternNode); ok {
			if en.StringToken.StringValue == "C" {
				for _, een := range en.Elements {
					if fn, ok := een.(*parser.ExternFuncNode); ok {
						f, err := declareExternFunction(fn, s, log)
						if err == nil {
							f.Component = cmp
							funcs = append(funcs, f)
							p.Funcs = append(p.Funcs, f)
						}
					}
				}
			} else {
				log.AddError(errlog.ErrorUnknownLinkage, en.StringToken.Location, en.StringToken.StringValue)
			}
		}
	}
	// Define named types
	for tdef, typ := range types {
		if _, ok := typ.(*GenericType); ok {
			continue
		}
		err := defineType(typ, tdef.Type, s, log)
		if err != nil {
			return err
		}
	}
	// Check correctness of all types.
	// This will define template functions as well.
	for _, typ := range types {
		if _, ok := typ.(*GenericType); ok {
			continue
		}
		err := typ.Check(log)
		if err != nil {
			return err
		}
	}
	for _, f := range funcs {
		err := f.Type.Check(log)
		if err != nil {
			return err
		}
	}
	if cmp != nil {
		if err := cmp.Check(log); err != nil {
			return err
		}
	}
	// Parse all variables and their types
	for _, n := range f.Children {
		en, ok := n.(*parser.ExpressionStatementNode)
		if !ok {
			continue
		}
		if vn, ok := en.Expression.(*parser.VarExpressionNode); ok {
			if err := checkVarExpression(vn, s, log); err != nil {
				return err
			}
			/*
				var assign *VariableAssignment
				if vn.Value != nil {
					assign = &VariableAssignment{value: vn.Value}
					if err := checkExpression(assign.value, s, log); err != nil {
						return err
					}
				}
				var typ Type
				if vn.Type != nil {
					typ, err := parseType(vn.Type, s, log)
					if err != nil {
						return err
					}
					if vn.VarToken.Kind == lexer.TokenVar {
						typ = &MutableType{TypeBase: TypeBase{location: parser.NodeLocation(vn.Type)}, Type: typ}
					}
				}
				for _, name := range vn.Names {
					v := &Variable{name: name.NameToken.StringValue, Type: typ, Component: cmp, Assignment: assign}
					if assign != nil {
						assign.Variables = append(assign.Variables, v)
					}
				}
			*/
		}
	}
	if cmp != nil {
		for _, el := range cmp.Scope.Elements {
			if v, ok := el.(*Variable); ok {
				v.Component = cmp
			}
		}
	}
	// Check all functions, global functions as well as functions attached to types,
	// including generic instance types.
	for _, typ := range types {
		err := checkFuncs(typ, p, log)
		if err != nil {
			return err
		}
	}
	var err error
	for _, f := range funcs {
		err2 := checkFuncs(f.Type, p, log)
		if err2 != nil {
			if err == nil {
				err = err2
			}
		}
		if f.IsExtern {
			continue
		}
		err2 = checkFuncBody(f, log)
		if err2 != nil {
			if err == nil {
				err = err2
			}
		}
	}
	return err
}
