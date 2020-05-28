package types

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

func parseExternFuncAttribs(ast *parser.ExternFuncNode, f *Func, log *errlog.ErrorLog) error {
	if ast.Attributes == nil {
		return nil
	}
	for _, a := range ast.Attributes.Attributes {
		switch n := a.(type) {
		case *parser.MetaAttributeNode:
			switch n.NameToken.StringValue {
			case "export":
				if n.Values != nil {
					return log.AddError(errlog.ErrorUnexpectedMetaAttributeParam, n.Values.Location(), n.NameToken.StringValue)
				}
				f.IsExported = true
			default:
				return log.AddError(errlog.ErrorUnknownMetaAttribute, n.NameToken.Location, n.NameToken.StringValue)
			}
		case *parser.LineNode:
			// Do nothing
		default:
			panic("Oooops")
		}
	}
	return nil
}

func parseFuncAttribs(ast *parser.FuncNode, f *Func, log *errlog.ErrorLog) error {
	if ast.Attributes == nil {
		return nil
	}
	for _, a := range ast.Attributes.Attributes {
		switch n := a.(type) {
		case *parser.MetaAttributeNode:
			switch n.NameToken.StringValue {
			case "export":
				if f.Component == nil {
					return log.AddError(errlog.ErrorExportOutsideComponent, n.NameToken.Location)
				}
				if n.Values != nil {
					return log.AddError(errlog.ErrorUnexpectedMetaAttributeParam, n.Values.Location(), n.NameToken.StringValue)
				}
				f.IsExported = true
			case "isr":
				if f.Component != nil && !f.Component.IsStatic {
					return log.AddError(errlog.ErrorISRInWrongContext, n.NameToken.Location)
				}
				if f.Type.Target != nil || f.IsGenericMemberFunc() {
					return log.AddError(errlog.ErrorISRInWrongContext, n.NameToken.Location)
				}
				if n.Values != nil {
					return log.AddError(errlog.ErrorUnexpectedMetaAttributeParam, n.Values.Location(), n.NameToken.StringValue)
				}
				f.IsInterruptServiceRoutine = true
			case "nomangle":
				if f.Type.Target != nil || f.IsGenericMemberFunc() {
					return log.AddError(errlog.ErrorNoMangleInWrongContext, n.NameToken.Location)
				}
				if n.Values != nil {
					return log.AddError(errlog.ErrorUnexpectedMetaAttributeParam, n.Values.Location(), n.NameToken.StringValue)
				}
				f.NoNameMangling = true
			default:
				return log.AddError(errlog.ErrorUnknownMetaAttribute, n.NameToken.Location, n.NameToken.StringValue)
			}
		case *parser.LineNode:
			// Do nothing
		default:
			panic("Oooops")
		}
	}
	return nil
}

func parseGenericFuncAttribs(ast *parser.FuncNode, f *GenericFunc, log *errlog.ErrorLog) error {
	if ast.Attributes == nil {
		return nil
	}
	for _, a := range ast.Attributes.Attributes {
		switch n := a.(type) {
		case *parser.MetaAttributeNode:
			switch n.NameToken.StringValue {
			case "export":
				return log.AddError(errlog.ErrorExportInWrongContext, n.NameToken.Location)
			case "isr":
				return log.AddError(errlog.ErrorISRInWrongContext, n.NameToken.Location)
			case "nomangle":
				return log.AddError(errlog.ErrorNoMangleInWrongContext, n.NameToken.Location)
			default:
				return log.AddError(errlog.ErrorUnknownMetaAttribute, n.NameToken.Location, n.NameToken.StringValue)
			}
		case *parser.LineNode:
			// Do nothing
		default:
			panic("Oooops")
		}
	}
	return nil
}

func parseVarAttribs(ast *parser.VarExpressionNode, v *Variable, log *errlog.ErrorLog) error {
	if ast.Attributes == nil {
		return nil
	}
	for _, a := range ast.Attributes.Attributes {
		switch n := a.(type) {
		case *parser.MetaAttributeNode:
			switch n.NameToken.StringValue {
			case "export":
				if v.Component == nil {
					return log.AddError(errlog.ErrorExportOutsideComponent, n.NameToken.Location)
				}
				if n.Values != nil {
					return log.AddError(errlog.ErrorUnexpectedMetaAttributeParam, n.Values.Location(), n.NameToken.StringValue)
				}
				v.IsExported = true
			default:
				return log.AddError(errlog.ErrorUnknownMetaAttribute, n.NameToken.Location, n.NameToken.StringValue)
			}
		case *parser.LineNode:
			// Do nothing
		default:
			panic("Oooops")
		}
	}
	return nil
}

func parseTypeAttribs(ast *parser.TypedefNode, t Type, log *errlog.ErrorLog) error {
	if ast.Attributes == nil {
		return nil
	}
	for _, a := range ast.Attributes.Attributes {
		switch n := a.(type) {
		case *parser.MetaAttributeNode:
			switch n.NameToken.StringValue {
			case "concurrent":
				if n.Values != nil {
					return log.AddError(errlog.ErrorUnexpectedMetaAttributeParam, n.Values.Location(), n.NameToken.StringValue)
				}
				if st, ok := GetStructType(t); ok {
					st.IsConcurrent = true
				} else {
					return log.AddError(errlog.ErrorConcurrentInWrongContext, n.NameToken.Location)
				}
			default:
				return log.AddError(errlog.ErrorUnknownMetaAttribute, n.NameToken.Location, n.NameToken.StringValue)
			}
		case *parser.LineNode:
			// Do nothing
		default:
			panic("Oooops")
		}
	}
	return nil
}

func parseComponentAttribs(ast *parser.ComponentNode, t *ComponentType, log *errlog.ErrorLog) error {
	if ast.Attributes == nil {
		return nil
	}
	for _, a := range ast.Attributes.Attributes {
		switch n := a.(type) {
		case *parser.MetaAttributeNode:
			switch n.NameToken.StringValue {
			case "static":
				if n.Values != nil {
					return log.AddError(errlog.ErrorUnexpectedMetaAttributeParam, n.Values.Location(), n.NameToken.StringValue)
				}
				t.IsStatic = true
			default:
				return log.AddError(errlog.ErrorUnknownMetaAttribute, n.NameToken.Location, n.NameToken.StringValue)
			}
		case *parser.LineNode:
			// Do nothing
		default:
			panic("Oooops")
		}
	}
	return nil
}
