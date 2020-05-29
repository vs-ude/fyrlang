package types

import (
	"path/filepath"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

type file struct {
	types           map[*parser.TypedefNode]Type
	components      map[*parser.ComponentNode]*ComponentType
	componentScopes map[*ComponentType]*Scope
	funcs           []*Func
	p               *Package
	s               *Scope
	fnode           *parser.FileNode
	lmap            *errlog.LocationMap
	log             *errlog.ErrorLog
}

func newFile(p *Package, f *parser.FileNode, lmap *errlog.LocationMap, log *errlog.ErrorLog) *file {
	return &file{types: make(map[*parser.TypedefNode]Type), components: make(map[*parser.ComponentNode]*ComponentType), componentScopes: make(map[*ComponentType]*Scope), p: p, fnode: f, lmap: lmap, log: log}
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
			s := newScope(f.p.Scope, ComponentScope, cn.Location())
			cmp := &ComponentType{ComponentScope: s, TypeBase: TypeBase{name: cn.NameToken.StringValue, location: cn.Location()}}
			cmp.SetScope(f.s)
			s.Component = cmp
			fileScope := newScope(f.s, ComponentFileScope, cn.Location())
			fileScope.Component = cmp
			f.componentScopes[cmp] = fileScope
			f.s.AddType(cmp, f.log)
			f.components[cn] = cmp
		}
	}
	var cmp *ComponentType
	s := f.s
	// Declare all named types
	for _, n := range f.fnode.Children {
		if tdef, ok := n.(*parser.TypedefNode); ok {
			var typ Type
			if tdef.GenericParams != nil {
				gt := &GenericType{Type: tdef.Type}
				for _, p := range tdef.GenericParams.Params {
					gt.TypeParameters = append(gt.TypeParameters, &GenericTypeParameter{Name: p.NameToken.StringValue})
				}
				typ = gt
			} else {
				typ = declareType(tdef.Type)
			}
			typ.SetName(tdef.NameToken.StringValue)
			err := s.AddType(typ, f.log)
			if err != nil {
				return err
			}
			f.types[tdef] = typ
			typ.SetScope(s)
			typ.SetComponent(cmp)
		} else if cn, ok := n.(*parser.ComponentNode); ok {
			cmp = f.components[cn]
			s = f.componentScopes[cmp]
		}
	}
	cmp = nil
	// Declare functions and attach them to types where applicable
	for _, n := range f.fnode.Children {
		if fn, ok := n.(*parser.FuncNode); ok {
			if fn.GenericParams != nil {
				fdecl, err := declareGenericFunction(fn, s, f.log)
				if err != nil {
					return err
				}
				fdecl.Component = cmp
			} else {
				fdecl, err := declareFunction(fn, s, f.log)
				if err != nil {
					return err
				}
				fdecl.Component = cmp
				// Not a member function?
				if fdecl.Type.Target == nil {
					f.funcs = append(f.funcs, fdecl)
					if err := s.AddElement(fdecl, fn.Location(), f.log); err != nil {
						return err
					}
				}
				if !fdecl.IsGenericMemberFunc() {
					f.p.Funcs = append(f.p.Funcs, fdecl)
					// Parse the function a second time, but this time
					// the `dual` keyword means non-mutable.
					if fdecl.DualIsMut {
						// It must be a member function due to parsing.
						if fdecl.Type.Target == nil {
							panic("Oooops")
						}
						s.dualIsMut = -1
						fnClone := fn.Clone().(*parser.FuncNode)
						fdecl2, err := declareFunction(fnClone, s, f.log)
						s.dualIsMut = 0
						if err != nil {
							return err
						}
						fdecl.DualFunc = fdecl2
					}
				}
			}
		} else if en, ok := n.(*parser.ExternNode); ok {
			if en.StringToken.StringValue == "C" {
				for _, een := range en.Elements {
					if fn, ok := een.(*parser.ExternFuncNode); ok {
						fdecl, err := declareExternFunction(fn, s, f.log)
						if err == nil {
							err = s.AddElement(fdecl, een.Location(), f.log)
						}
						if err == nil {
							fdecl.Component = cmp
							f.funcs = append(f.funcs, fdecl)
							f.p.Funcs = append(f.p.Funcs, fdecl)
						}
					}
				}
			} else {
				return f.log.AddError(errlog.ErrorUnknownLinkage, en.StringToken.Location, en.StringToken.StringValue)
			}
		} else if cn, ok := n.(*parser.ComponentNode); ok {
			cmp = f.components[cn]
			s = f.componentScopes[cmp]
		}
	}
	return nil
}

func (f *file) defineComponents() error {
	for cn, cmp := range f.components {
		// TODO: Parse everything
		if err := parseComponentAttribs(cn, cmp, f.log); err != nil {
			return err
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
		err := defineType(typ, tdef.Type, typ.Scope(), f.log)
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
	for _, cmp := range f.components {
		if err := cmp.Check(f.log); err != nil {
			return err
		}
	}
	return nil
}

func (f *file) defineGlobalVars() error {
	// Parse all variables and their types
	var cmp *ComponentType
	s := f.s
	for _, n := range f.fnode.Children {
		en, ok := n.(*parser.ExpressionStatementNode)
		if !ok {
			continue
		}
		if vn, ok := en.Expression.(*parser.VarExpressionNode); ok {
			if err := checkGlobalVarExpression(vn, s, cmp, f.log); err != nil {
				return err
			}
			if vn.Value != nil {
				if cmp == nil {
					f.p.VarExpressions = append(f.p.VarExpressions, vn)
				} else {
					// TODO
				}
			}
		} else if cn, ok := n.(*parser.ComponentNode); ok {
			cmp = f.components[cn]
			s = f.componentScopes[cmp]
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
