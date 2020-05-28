package types

import (
	"path/filepath"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

type file struct {
	types map[*parser.TypedefNode]Type
	funcs []*Func
	p     *Package
	s     *Scope
	cmp   *ComponentType
	fnode *parser.FileNode
	lmap  *errlog.LocationMap
	log   *errlog.ErrorLog
}

func newFile(p *Package, f *parser.FileNode, lmap *errlog.LocationMap, log *errlog.ErrorLog) *file {
	return &file{types: make(map[*parser.TypedefNode]Type), p: p, fnode: f, lmap: lmap, log: log}
}

func (f *file) parseAndDeclare() error {
	f.s = newScope(f.p.Scope, FileScope, f.fnode.Location())
	// Imports and component
	for _, n := range f.fnode.Children {
		if impBlock, ok := n.(*parser.ImportBlockNode); ok {
			for _, nImp := range impBlock.Imports {
				if _, ok := nImp.(*parser.LineNode); ok {
					continue
				} else if imp, ok := nImp.(*parser.ImportNode); ok {
					pnew, err := LookupPackage(imp.StringToken.StringValue, f.p, imp.StringToken.Location, f.lmap, f.log)
					if err != nil {
						return err
					}
					err = pnew.Parse(f.lmap, f.log)
					if err != nil {
						return err
					}
					ns := &Namespace{Scope: pnew.Scope}
					if imp.NameToken != nil {
						ns.name = imp.NameToken.StringValue
					} else {
						ns.name = filepath.Base(pnew.Path)
					}
					err = f.s.AddNamespace(ns, imp.Location(), f.log)
					if err != nil {
						return err
					}
					// Let `pnew` import the runtime as well
					runtime := f.p.RuntimePackage()
					if runtime != nil {
						pnew.addImport(runtime)
					}
					// Import package `pnew`
					f.p.addImport(pnew)
				} else {
					panic("Wrong")
				}
			}
		} else if cn, ok := n.(*parser.ComponentNode); ok {
			if f.cmp != nil {
				return f.log.AddError(errlog.ErrorComponentTwice, cn.Location())
			}
			f.s = newScope(f.s, ComponentScope, f.s.Location)
			f.cmp = &ComponentType{Scope: f.s, TypeBase: TypeBase{name: cn.NameToken.StringValue, location: cn.Location()}}
			f.s.Component = f.cmp
			// TODO: Parse everything
			if err := parseComponentAttribs(cn, f.cmp, f.log); err != nil {
				return err
			}
			f.p.Scope.AddType(f.cmp, f.log)
		}
	}
	// Declare all named types
	for _, n := range f.fnode.Children {
		if tdef, ok := n.(*parser.TypedefNode); ok {
			var typ Type
			if tdef.GenericParams != nil {
				gt := &GenericType{Type: tdef.Type, Scope: f.s}
				for _, p := range tdef.GenericParams.Params {
					gt.TypeParameters = append(gt.TypeParameters, &GenericTypeParameter{Name: p.NameToken.StringValue})
				}
				typ = gt
			} else {
				typ = declareType(tdef.Type)
			}
			typ.SetName(tdef.NameToken.StringValue)
			err := f.s.AddType(typ, f.log)
			if err != nil {
				return err
			}
			f.types[tdef] = typ
		}
	}
	// Declare functions and attach them to types where applicable
	for _, n := range f.fnode.Children {
		if fn, ok := n.(*parser.FuncNode); ok {
			if fn.GenericParams != nil {
				_, err := declareGenericFunction(fn, f.s, f.log)
				if err != nil {
					return err
				}
			} else {
				fdecl, err := declareFunction(fn, f.s, f.log)
				if err != nil {
					return err
				}
				if fdecl.Type.Target == nil {
					f.funcs = append(f.funcs, fdecl)
				}
				if !fdecl.IsGenericMemberFunc() {
					f.p.Funcs = append(f.p.Funcs, fdecl)
					// Parse the function a second time, but this time
					// the `dual` keyword means non-mutable.
					if fdecl.DualIsMut {
						if fdecl.Type.Target == nil {
							panic("Oooops")
						}
						f.s.dualIsMut = -1
						fnClone := fn.Clone().(*parser.FuncNode)
						fdecl2, err := declareFunction(fnClone, f.s, f.log)
						f.s.dualIsMut = 0
						if err != nil {
							return err
						}
						fdecl2.Component = f.cmp
						fdecl.DualFunc = fdecl2
					}
				}
			}
		} else if en, ok := n.(*parser.ExternNode); ok {
			if en.StringToken.StringValue == "C" {
				for _, een := range en.Elements {
					if fn, ok := een.(*parser.ExternFuncNode); ok {
						fdecl, err := declareExternFunction(fn, f.s, f.log)
						if err == nil {
							fdecl.Component = f.cmp
							f.funcs = append(f.funcs, fdecl)
							f.p.Funcs = append(f.p.Funcs, fdecl)
						}
					}
				}
			} else {
				return f.log.AddError(errlog.ErrorUnknownLinkage, en.StringToken.Location, en.StringToken.StringValue)
			}
		}
	}
	return nil
}

func (f *file) defineTypes() error {
	// Define named types
	for tdef, typ := range f.types {
		if _, ok := typ.(*GenericType); ok {
			continue
		}
		err := defineType(typ, tdef.Type, f.s, f.log)
		if err != nil {
			return err
		}
		if err := parseTypeAttribs(tdef, typ, f.log); err != nil {
			return err
		}
	}
	// Check correctness of all types.
	// This will define template functions as well.
	for _, typ := range f.types {
		if _, ok := typ.(*GenericType); ok {
			continue
		}
		err := typ.Check(f.log)
		if err != nil {
			return err
		}
	}
	for _, fdecl := range f.funcs {
		err := fdecl.Type.Check(f.log)
		if err != nil {
			return err
		}
	}
	if f.cmp != nil {
		if err := f.cmp.Check(f.log); err != nil {
			return err
		}
	}
	return nil
}

func (f *file) defineGlobalVars() error {
	// Parse all variables and their types
	for _, n := range f.fnode.Children {
		en, ok := n.(*parser.ExpressionStatementNode)
		if !ok {
			continue
		}
		if vn, ok := en.Expression.(*parser.VarExpressionNode); ok {
			if err := checkVarExpression(vn, f.s, f.log); err != nil {
				return err
			}
			if vn.Value != nil {
				if f.cmp == nil {
					f.p.VarExpressions = append(f.p.VarExpressions, vn)
				}
			}
		}
	}
	if f.cmp != nil {
		for _, el := range f.cmp.Scope.Elements {
			if v, ok := el.(*Variable); ok {
				v.Component = f.cmp
			}
		}
	}
	return nil
}

func (f *file) defineFuns() error {
	// Check all functions, global functions as well as functions attached to types,
	// including generic instance types.
	for _, typ := range f.types {
		err := checkFuncs(typ, f.p, f.log)
		if err != nil {
			return err
		}
	}
	var err error
	for _, fdecl := range f.funcs {
		err2 := checkFuncs(fdecl.Type, f.p, f.log)
		if err2 != nil {
			if err == nil {
				err = err2
			}
		}
		if fdecl.IsExtern {
			continue
		}
		err2 = checkFuncBody(fdecl, f.log)
		if err2 != nil {
			if err == nil {
				err = err2
			}
		}
	}
	return err
}
