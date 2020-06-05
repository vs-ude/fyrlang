package types

import (
	"path/filepath"
	"unicode"

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
	s := f.s
	// Components and declare all named elements
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
					if !pnew.parsed {
						err = pnew.Parse(f.lmap, f.log)
						if err != nil {
							return err
						}
						// Let `pnew` import the runtime as well
						runtime := f.p.RuntimePackage()
						if runtime != nil {
							pnew.addImport(runtime)
						}
					}
					ns := &Namespace{Scope: pnew.Scope}
					if imp.NameToken != nil {
						ns.name = imp.NameToken.StringValue
					} else {
						ns.name = filepath.Base(pnew.Path)
					}
					err = s.AddNamespace(ns, imp.Location(), f.log)
					if err != nil {
						return err
					}
					// Import package `pnew`
					f.p.addImport(pnew)
				} else {
					panic("Wrong")
				}
			}
		} else if cn, ok := n.(*parser.ComponentNode); ok {
			cmpScope := newScope(f.p.Scope, ComponentScope, cn.Location())
			cmp := &ComponentType{ComponentScope: cmpScope, TypeBase: TypeBase{name: cn.NameToken.StringValue, pkg: f.s.PackageScope().Package, location: cn.Location()}}
			cmp.SetScope(f.s)
			cmpScope.Component = cmp
			if err := parseComponentAttribs(cn, cmp, f.log); err != nil {
				return err
			}
			fileScope := newScope(cmpScope, ComponentFileScope, cn.Location())
			fileScope.Component = cmp
			f.componentScopes[cmp] = fileScope
			f.s.AddType(cmp, f.log)
			f.components[cn] = cmp
			s = fileScope
		}
	}
	var cmp *ComponentType
	s = f.s
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
	s = f.s
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
					if cmp != nil {
						cmp.Funcs = append(cmp.Funcs, fdecl)
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

func (f *file) declareComponents() error {
	// for cn, cmp := range f.components {
	// TODO: Parse everything inside the curly braces
	// }
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
	// Parse all variables and check their types
	var parserError error
	var cmp *ComponentType
	s := f.s
	for _, n := range f.fnode.Children {
		if en, ok := n.(*parser.ExpressionStatementNode); ok {
			if vn, ok := en.Expression.(*parser.VarExpressionNode); ok {
				v, err := checkGlobalVarExpression(vn, s, cmp, f.log)
				if err != nil {
					return err
				}
				// Global variables must be initialized
				if vn.Value == nil {
					err := f.log.AddError(errlog.ErrorUninitializedVariable, vn.Location(), v.name)
					if parserError == nil {
						parserError = err
					}
					continue
				}
				if cmp == nil {
					f.p.VarExpressions = append(f.p.VarExpressions, vn)
				} else {
					cf := &ComponentField{Var: v, Expression: vn}
					cmp.Fields = append(cmp.Fields, cf)
				}
			}
		} else if un, ok := n.(*parser.UseNode); ok {
			t, err := s.LookupNamedType(un.Component, f.log)
			if err != nil {
				parserError = err
				continue
			}
			usecmp, ok := t.(*ComponentType)
			if !ok {
				parserError = f.log.AddError(errlog.ErrorIncompatibleTypes, un.Component.Location())
			}
			var name string
			if un.NameToken != nil {
				name = un.NameToken.StringValue
			}
			if err := usecmp.Check(f.log); err != nil {
				parserError = err
				continue
			}
			if cmp.UsesComponent(usecmp, false) {
				return f.log.AddError(errlog.ErrorDoubleComponentUsage, un.Location(), cmp.name, usecmp.name)
			}
			if usecmp.UsesComponent(cmp, true) {
				return f.log.AddError(errlog.ErrorCircularComponentUsage, un.Location(), cmp.name, usecmp.name)
			}
			usage := &ComponentUsage{name: name, Type: usecmp, Location: un.Location()}
			if name != "" {
				s.AddElement(usage, un.Location(), f.log)
			}
			if cmp == nil {
				f.p.ComponentsUsed = append(f.p.ComponentsUsed, usage)
			} else {
				cmp.ComponentsUsed = append(cmp.ComponentsUsed, usage)
			}
		} else if cn, ok := n.(*parser.ComponentNode); ok {
			cmp = f.components[cn]
			s = f.componentScopes[cmp]
		}
	}
	return parserError
}

func (f *file) defineComponents() error {
	for _, cmp := range f.components {
		s := f.componentScopes[cmp]
		for _, usage := range cmp.ComponentsUsed {
			if usage.name == "" {
				for n, e := range usage.Type.ComponentScope.Elements {
					// Only add symbols with capital letter or exported elements
					name := []rune(n)
					if !unicode.IsUpper(name[0]) && !isElementExported(e) {
						continue
					}
					if _, ok := s.Parent.Elements[n]; !ok {
						if _, ok := s.Elements[n]; !ok {
							s.AddElement(e, usage.Location, f.log)
						}
					}
				}
				for n, t := range usage.Type.ComponentScope.Types {
					// Only add symbols with capital letter
					name := []rune(n)
					if !unicode.IsUpper(name[0]) {
						continue
					}
					if _, ok := s.Parent.Types[n]; !ok {
						s.AddType(t, f.log)
					}
				}
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
