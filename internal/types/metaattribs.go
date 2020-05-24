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
