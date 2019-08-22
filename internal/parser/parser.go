package parser

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/lexer"
)

// Parser ...
type Parser struct {
	l          *lexer.Lexer
	savedToken *lexer.Token
	log        *errlog.ErrorLog
}

// NewParser ...
func NewParser(log *errlog.ErrorLog) *Parser {
	return &Parser{log: log}
}

// Parse ...
func (p *Parser) Parse(file int, str string, log *errlog.ErrorLog) (*FileNode, error) {
	var err error
	p.l = lexer.NewLexer(file, str, log)
	n := &FileNode{File: file}
	n.Children, err = p.parseFile()
	return n, err
}

func (p *Parser) parseFile() ([]Node, error) {
	var children []Node
	for {
		if p.peek(lexer.TokenEOF) {
			break
		} else if p.peek(lexer.TokenType) {
			n, err := p.parseTypedef()
			if err != nil {
				// TODO: Skip to save point and continue
				return children, nil
			}
			children = append(children, n)
		} else if t, ok := p.optional(lexer.TokenNewline); ok {
			n := &LineNode{Token: t}
			children = append(children, n)
		} else if p.peek(lexer.TokenImport) {
			n, err := p.parseImportBlock()
			if err != nil {
				// TODO: Skip to save point and continue
				return children, nil
			}
			children = append(children, n)
		} else if t, ok := p.optional(lexer.TokenFunc); ok {
			n := &FuncNode{FuncToken: t}
			err := p.parseFunc(n)
			if err != nil {
				// TODO: Skip to save point and continue
				return children, nil
			}
			children = append(children, n)
		} else if t, ok := p.optional(lexer.TokenMut); ok {
			t2, err := p.expect(lexer.TokenFunc)
			if err != nil {
				return children, nil
			}
			n := &FuncNode{ComponentMutToken: t, FuncToken: t2}
			if err = p.parseFunc(n); err != nil {
				// TODO: Skip to save point and continue
				return children, nil
			}
			children = append(children, n)
		} else if t, ok := p.optional(lexer.TokenComponent); ok {
			n := &ComponentNode{ComponentToken: t}
			var err error
			if n.NameToken, err = p.expect(lexer.TokenIdentifier); err != nil {
				return nil, err
			}
			if n.NewlineToken, err = p.expectMulti(lexer.TokenNewline, lexer.TokenEOF); err != nil {
				return nil, err
			}
			children = append(children, n)
		} else if p.peek(lexer.TokenVar) {
			n, err := p.parseVarExpression()
			if err != nil {
				return nil, err
			}
			n2 := &ExpressionStatementNode{Expression: n}
			if n2.NewlineToken, err = p.expectMulti(lexer.TokenNewline, lexer.TokenEOF); err != nil {
				return nil, err
			}
			children = append(children, n2)
		} else if p.peek(lexer.TokenLet) {
			n, err := p.parseLetExpression()
			if err != nil {
				return nil, err
			}
			n2 := &ExpressionStatementNode{Expression: n}
			if n2.NewlineToken, err = p.expectMulti(lexer.TokenNewline, lexer.TokenEOF); err != nil {
				return nil, err
			}
			children = append(children, n2)
		} else {
			// TODO: Skip to save point and continue
			return nil, p.expectError(lexer.TokenImport, lexer.TokenFunc, lexer.TokenType, lexer.TokenVar, lexer.TokenLet, lexer.TokenComponent)
		}
	}
	return children, nil
}

func (p *Parser) parseImportBlock() (Node, error) {
	n := &ImportBlockNode{}
	var err error
	if n.ImportToken, err = p.expect(lexer.TokenImport); err != nil {
		return nil, err
	}
	if n.OpenToken, err = p.expect(lexer.TokenOpenBraces); err != nil {
		return nil, err
	}
	if n.NewlineToken1, err = p.expect(lexer.TokenNewline); err != nil {
		return nil, err
	}
	var ok bool
	for {
		if n.CloseToken, ok = p.optional(lexer.TokenCloseBraces); ok {
			break
		}
		t, ok2 := p.optional(lexer.TokenNewline)
		if ok2 {
			n.Imports = append(n.Imports, &LineNode{Token: t})
			continue
		}
		im := &ImportNode{}
		im.NameToken, _ = p.optional(lexer.TokenIdentifier)
		if im.StringToken, err = p.expect(lexer.TokenString); err != nil {
			return nil, err
		}
		if im.NewlineToken, err = p.expect(lexer.TokenNewline); err != nil {
			return nil, err
		}
		n.Imports = append(n.Imports, im)
	}
	if n.NewlineToken1, err = p.expect(lexer.TokenNewline); err != nil {
		return nil, err
	}
	return n, nil
}

func (p *Parser) parseTypedef() (Node, error) {
	n := &TypedefNode{}
	var err error
	if n.TypeToken, err = p.expect(lexer.TokenType); err != nil {
		return nil, err
	}
	if n.NameToken, err = p.expect(lexer.TokenIdentifier); err != nil {
		return nil, err
	}
	if p.peek(lexer.TokenLess) {
		if n.GenericParams, err = p.parseGenericParamList(); err != nil {
			return nil, err
		}
	}
	if n.Type, err = p.parseType(); err != nil {
		return nil, err
	}
	if n.NewlineToken, err = p.expectMulti(lexer.TokenNewline, lexer.TokenEOF); err != nil {
		return nil, err
	}
	return n, nil
}

func (p *Parser) parseGenericParamList() (*GenericParamListNode, error) {
	n := &GenericParamListNode{}
	var err error
	var ok bool
	if n.OpenToken, err = p.expect(lexer.TokenLess); err != nil {
		return nil, err
	}
	for {
		if n.CloseToken, ok = p.optional(lexer.TokenGreater); ok {
			break
		}
		pn := &GenericParamNode{}
		if len(n.Params) > 0 {
			if pn.CommaToken, err = p.expect(lexer.TokenComma); err != nil {
				return nil, err
			}
		}
		if pn.NameToken, err = p.expect(lexer.TokenIdentifier); err != nil {
			return nil, err
		}
		n.Params = append(n.Params, pn)
	}
	return n, nil
}

func (p *Parser) parseFunc(n *FuncNode) error {
	var err error
	if n.Type, err = p.parseTypeIntern(false); err != nil {
		return err
	}
	var ok bool
	if n.DotToken, ok = p.optional(lexer.TokenDot); ok {
		if n.NameToken, err = p.expect(lexer.TokenIdentifier); err != nil {
			return err
		}
	} else if name, ok := n.Type.(*NamedTypeNode); ok && name.Namespace == nil {
		n.NameToken = name.NameToken
		n.Type = nil
		if p.peek(lexer.TokenLess) {
			if n.GenericParams, err = p.parseGenericParamList(); err != nil {
				return err
			}
		}
	} else {
		return p.expectError(lexer.TokenDot)
	}
	if n.Params, err = p.parseParameterList(); err != nil {
		return err
	}
	if !p.peek(lexer.TokenOpenBraces) {
		if p.peek(lexer.TokenOpenParanthesis) {
			if n.ReturnParams, err = p.parseParameterList(); err != nil {
				return err
			}
		} else {
			pn := &ParamNode{}
			if err = p.parseParameter(pn); err != nil {
				return err
			}
			n.ReturnParams = &ParamListNode{Params: []*ParamNode{pn}}
		}
	}
	if n.Body, err = p.parseBody(); err != nil {
		return err
	}
	if n.NewlineToken, err = p.expectMulti(lexer.TokenNewline, lexer.TokenEOF); err != nil {
		return err
	}
	return nil
}

func (p *Parser) parseParameterList() (*ParamListNode, error) {
	var err error
	var ok bool
	n := &ParamListNode{}
	if n.OpenToken, err = p.expect(lexer.TokenOpenParanthesis); err != nil {
		return nil, err
	}
	// Parameters
	for {
		if n.CloseToken, ok = p.optional(lexer.TokenCloseParanthesis); ok {
			break
		}
		pn := &ParamNode{}
		if len(n.Params) != 0 {
			if pn.CommaToken, err = p.expect(lexer.TokenComma); err != nil {
				return nil, err
			}
		}
		if err = p.parseParameter(pn); err != nil {
			return nil, err
		}
		n.Params = append(n.Params, pn)
	}
	return n, nil
}

func (p *Parser) parseParameter(pn *ParamNode) error {
	var err error
	if pn.Type, err = p.parseType(); err != nil {
		return err
	}
	// Perhaps the type is not a type, but the parameter name instead
	if !p.peek(lexer.TokenComma) && !p.peek(lexer.TokenOpenBraces) && !p.peek(lexer.TokenCloseParanthesis) && !p.peek(lexer.TokenNewline) && !p.peek(lexer.TokenAssign) && !p.peek(lexer.TokenEOF) {
		pt, ok := pn.Type.(*NamedTypeNode)
		if !ok || pt.Namespace != nil {
			return p.expectError(lexer.TokenComma, lexer.TokenOpenBraces, lexer.TokenCloseParanthesis, lexer.TokenNewline, lexer.TokenAssign)
		}
		pn.NameToken = pt.NameToken
		if pn.Type, err = p.parseType(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) parseType() (Node, error) {
	return p.parseTypeIntern(true)
}

func (p *Parser) parseTypeList() (*TypeListNode, error) {
	n := &TypeListNode{}
	var err error
	var ok bool
	if n.BacktickToken, err = p.expect(lexer.TokenBacktick); err != nil {
		return nil, err
	}
	if n.OpenToken, err = p.expect(lexer.TokenLess); err != nil {
		return nil, err
	}
	for {
		if n.CloseToken, ok = p.optional(lexer.TokenGreater); ok {
			break
		}
		pn := &TypeListElementNode{}
		if len(n.Types) > 0 {
			if pn.CommaToken, err = p.expect(lexer.TokenComma); err != nil {
				return nil, err
			}
		}
		if pn.Type, err = p.parseType(); err != nil {
			return nil, err
		}
		n.Types = append(n.Types, pn)
	}
	return n, nil
}

func (p *Parser) parseTypeIntern(allowScopedName bool) (Node, error) {
	var err error
	var ok bool
	t := p.scan()
	switch t.Kind {
	case lexer.TokenMut:
		n := &MutableTypeNode{MutToken: t}
		if n.Type, err = p.parseTypeIntern(allowScopedName); err != nil {
			return nil, err
		}
		return n, nil
	case lexer.TokenAsterisk, lexer.TokenAmpersand, lexer.TokenTilde, lexer.TokenCaret, lexer.TokenHash:
		n := &PointerTypeNode{PointerToken: t}
		if n.ElementType, err = p.parseTypeIntern(allowScopedName); err != nil {
			return nil, err
		}
		return n, nil
	case lexer.TokenOpenBracket:
		if t2, ok := p.optional(lexer.TokenCloseBracket); ok {
			n := &SliceTypeNode{OpenToken: t, CloseToken: t2}
			if n.ElementType, err = p.parseTypeIntern(allowScopedName); err != nil {
				return nil, err
			}
			return n, nil
		}
		n := &ArrayTypeNode{OpenToken: t}
		if n.Size, err = p.parseExpression(); err != nil {
			return nil, err
		}
		if n.CloseToken, err = p.expect(lexer.TokenCloseBracket); err != nil {
			return nil, err
		}
		if n.ElementType, err = p.parseTypeIntern(allowScopedName); err != nil {
			return nil, err
		}
		return n, nil
	case lexer.TokenIdentifier:
		n := &NamedTypeNode{NameToken: t}
		for allowScopedName {
			var dot *lexer.Token
			if dot, ok = p.optional(lexer.TokenDot); !ok {
				break
			}
			n2 := &NamedTypeNode{NamespaceDotToken: dot, Namespace: n}
			if n2.NameToken, err = p.expect(lexer.TokenIdentifier); err != nil {
				return nil, err
			}
			n = n2
		}
		if p.peek(lexer.TokenBacktick) {
			g := &GenericInstanceTypeNode{Type: n}
			if g.TypeArguments, err = p.parseTypeList(); err != nil {
				return nil, err
			}
			return g, nil
		}
		return n, nil
	case lexer.TokenStruct:
		return p.parseStructType(t)
	case lexer.TokenInterface:
		return p.parseInterfaceType(t)
	case lexer.TokenFunc:
		return p.parseClosureType(t)
	case lexer.TokenColon:
		n := &GroupTypeNode{ColonToken: t}
		if n.GroupNameToken, err = p.expect(lexer.TokenIdentifier); err != nil {
			return nil, err
		}
		n.Type, err = p.parseType()
		if err != nil {
			return nil, err
		}
		return n, nil
	}
	return nil, p.expectError(lexer.TokenMut, lexer.TokenAsterisk, lexer.TokenAmpersand, lexer.TokenTilde, lexer.TokenCaret, lexer.TokenHash, lexer.TokenOpenBracket, lexer.TokenIdentifier)
}

func (p *Parser) parseStructType(structToken *lexer.Token) (*StructTypeNode, error) {
	n := &StructTypeNode{StructToken: structToken}
	var err error
	var ok bool
	if n.OpenToken, err = p.expect(lexer.TokenOpenBraces); err != nil {
		return nil, err
	}
	if n.NewlineToken, err = p.expect(lexer.TokenNewline); err != nil {
		return nil, err
	}
	for {
		if n.CloseToken, ok = p.optional(lexer.TokenCloseBraces); ok {
			break
		}
		if t, ok := p.optional(lexer.TokenNewline); ok {
			n.Fields = append(n.Fields, &LineNode{Token: t})
			continue
		}
		f := &StructFieldNode{}
		if f.Type, err = p.parseType(); err != nil {
			return nil, err
		}
		if f.NewlineToken, ok = p.optional(lexer.TokenNewline); ok {
			n.Fields = append(n.Fields, f)
			continue
		}
		if name, ok := f.Type.(*NamedTypeNode); ok && name.Namespace == nil {
			f.NameToken = name.NameToken
			if f.Type, err = p.parseType(); err != nil {
				return nil, err
			}
		} else {
			p.expectError(lexer.TokenNewline)
		}
		if f.NewlineToken, err = p.expect(lexer.TokenNewline); err != nil {
			return nil, err
		}
		n.Fields = append(n.Fields, f)
	}
	return n, nil
}

func (p *Parser) parseInterfaceType(ifaceToken *lexer.Token) (*InterfaceTypeNode, error) {
	n := &InterfaceTypeNode{InterfaceToken: ifaceToken}
	var err error
	var ok bool
	if n.OpenToken, err = p.expect(lexer.TokenOpenBraces); err != nil {
		return nil, err
	}
	if n.CloseToken, ok = p.optional(lexer.TokenCloseBraces); ok {
		return n, nil
	}
	if n.NewlineToken, err = p.expect(lexer.TokenNewline); err != nil {
		return nil, err
	}
	for {
		if n.CloseToken, ok = p.optional(lexer.TokenCloseBraces); ok {
			break
		}
		if t, ok := p.optional(lexer.TokenNewline); ok {
			n.Fields = append(n.Fields, &LineNode{Token: t})
			continue
		}
		if p.peek(lexer.TokenFunc) {
			f, err := p.parseInterfaceFunc()
			if err != nil {
				return nil, err
			}
			n.Fields = append(n.Fields, f)
			continue
		}
		if t, ok := p.optional(lexer.TokenMut); ok {
			f, err := p.parseInterfaceFunc()
			if err != nil {
				return nil, err
			}
			f.ComponentMutToken = t
			n.Fields = append(n.Fields, f)
			continue
		}
		f := &InterfaceFieldNode{}
		if f.Type, err = p.parseType(); err != nil {
			return nil, err
		}
		if f.NewlineToken, err = p.expect(lexer.TokenNewline); err != nil {
			return nil, err
		}
		n.Fields = append(n.Fields, f)
	}
	return n, nil
}

func (p *Parser) parseInterfaceFunc() (*InterfaceFuncNode, error) {
	n := &InterfaceFuncNode{}
	var err error
	if n.FuncToken, err = p.expect(lexer.TokenFunc); err != nil {
		return nil, err
	}
	n.MutToken, _ = p.optional(lexer.TokenMut)
	if n.PointerToken, err = p.expectMulti(lexer.TokenAmpersand, lexer.TokenAsterisk); err != nil {
		return nil, err
	}
	if n.NameToken, err = p.expect(lexer.TokenIdentifier); err != nil {
		return nil, err
	}
	if n.Params, err = p.parseParameterList(); err != nil {
		return nil, err
	}
	if !p.peek(lexer.TokenNewline) {
		if p.peek(lexer.TokenOpenParanthesis) {
			if n.ReturnParams, err = p.parseParameterList(); err != nil {
				return nil, err
			}
		} else {
			pn := &ParamNode{}
			if err = p.parseParameter(pn); err != nil {
				return nil, err
			}
			n.ReturnParams = &ParamListNode{Params: []*ParamNode{pn}}
		}
	}
	if n.NewlineToken, err = p.expectMulti(lexer.TokenNewline, lexer.TokenEOF); err != nil {
		return nil, err
	}
	return n, nil
}

func (p *Parser) parseClosureType(funcToken *lexer.Token) (Node, error) {
	n := &ClosureTypeNode{FuncToken: funcToken}
	var err error
	if n.Params, err = p.parseParameterList(); err != nil {
		return nil, err
	}
	if !p.peek(lexer.TokenNewline) {
		if p.peek(lexer.TokenOpenParanthesis) {
			if n.ReturnParams, err = p.parseParameterList(); err != nil {
				return nil, err
			}
		} else {
			pn := &ParamNode{}
			if err = p.parseParameter(pn); err != nil {
				return nil, err
			}
			n.ReturnParams = &ParamListNode{Params: []*ParamNode{pn}}
		}
	}
	return n, nil
}

func (p *Parser) parseBody() (*BodyNode, error) {
	n := &BodyNode{}
	var err error
	var ok bool
	if n.OpenToken, err = p.expect(lexer.TokenOpenBraces); err != nil {
		return nil, err
	}
	for {
		if n.CloseToken, ok = p.optional(lexer.TokenCloseBraces); ok {
			break
		}
		var s Node
		if s, err = p.parseStatement(); err != nil {
			return nil, err
		}
		n.Children = append(n.Children, s)
	}
	return n, nil
}

func (p *Parser) parseStatement() (Node, error) {
	if t, ok := p.optional(lexer.TokenNewline); ok {
		return &LineNode{Token: t}, nil
	} else if p.peek(lexer.TokenIf) {
		n, err := p.parseIfStatement()
		if err != nil {
			return nil, err
		}
		if n.NewlineToken, err = p.expect(lexer.TokenNewline); err != nil {
			return nil, err
		}
		return n, nil
	} else if p.peek(lexer.TokenFor) {
		return p.parseForStatement()
	} else if t, ok := p.optional(lexer.TokenContinue); ok {
		nl, err := p.expect(lexer.TokenNewline)
		if err != nil {
			return nil, err
		}
		return &ContinueStatementNode{Token: t, NewlineToken: nl}, nil
	} else if t, ok := p.optional(lexer.TokenBreak); ok {
		nl, err := p.expect(lexer.TokenNewline)
		if err != nil {
			return nil, err
		}
		return &BreakStatementNode{Token: t, NewlineToken: nl}, nil
	} else if t, ok := p.optional(lexer.TokenYield); ok {
		nl, err := p.expect(lexer.TokenNewline)
		if err != nil {
			return nil, err
		}
		return &YieldStatementNode{Token: t, NewlineToken: nl}, nil
	} else if p.peek(lexer.TokenReturn) {
		return p.parseReturnStatement()
	}
	e, err := p.parseExpressionStatement()
	if err != nil {
		return nil, err
	}
	nl, err := p.expect(lexer.TokenNewline)
	if err != nil {
		return nil, err
	}
	return &ExpressionStatementNode{Expression: e, NewlineToken: nl}, nil
}

func (p *Parser) parseExpressionStatement() (Node, error) {
	if p.peek(lexer.TokenVar) {
		return p.parseVarExpression()
	}
	if p.peek(lexer.TokenLet) {
		return p.parseLetExpression()
	}
	e, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if t, ok := p.optionalMulti(lexer.TokenAssign, lexer.TokenAssignPlus, lexer.TokenAssignMinus, lexer.TokenAssignAsterisk, lexer.TokenAssignDivision, lexer.TokenAssignShiftLeft, lexer.TokenAssignShiftRight, lexer.TokenWalrus); ok {
		n := &AssignmentExpressionNode{Left: e, OpToken: t}
		if n.Right, err = p.parseExpression(); err != nil {
			return nil, err
		}
		return n, nil
	} else if t, ok := p.optionalMulti(lexer.TokenInc, lexer.TokenDec); ok {
		return &IncrementExpressionNode{Expression: e, Token: t}, nil
	}
	return e, nil
}

func (p *Parser) parseVarExpression() (*VarExpressionNode, error) {
	t, err := p.expect(lexer.TokenVar)
	if err != nil {
		return nil, err
	}
	n := &VarExpressionNode{VarToken: t}
	var ok bool
	for {
		vn := &VarNameNode{}
		if len(n.Names) > 0 {
			if vn.CommaToken, ok = p.optional(lexer.TokenComma); !ok {
				break
			}
		}
		if vn.NameToken, err = p.expect(lexer.TokenIdentifier); err != nil {
			return nil, err
		}
		n.Names = append(n.Names, vn)
	}
	if !p.peek(lexer.TokenAssign) {
		if n.Type, err = p.parseType(); err != nil {
			return nil, err
		}
	}
	if n.AssignToken, ok = p.optional(lexer.TokenAssign); ok {
		if n.Value, err = p.parseExpression(); err != nil {
			return nil, err
		}
	}
	return n, nil
}

func (p *Parser) parseLetExpression() (*VarExpressionNode, error) {
	t, err := p.expect(lexer.TokenLet)
	if err != nil {
		return nil, err
	}
	n := &VarExpressionNode{VarToken: t}
	var ok bool
	for {
		vn := &VarNameNode{}
		if len(n.Names) > 0 {
			if vn.CommaToken, ok = p.optional(lexer.TokenComma); !ok {
				break
			}
		}
		if vn.NameToken, err = p.expect(lexer.TokenIdentifier); err != nil {
			return nil, err
		}
		n.Names = append(n.Names, vn)
	}
	if !p.peek(lexer.TokenAssign) {
		if n.Type, err = p.parseType(); err != nil {
			return nil, err
		}
	}
	if n.AssignToken, err = p.expect(lexer.TokenAssign); err != nil {
		return nil, err
	}
	if n.Value, err = p.parseExpression(); err != nil {
		return nil, err
	}
	return n, nil
}

func (p *Parser) parseReturnStatement() (*ReturnStatementNode, error) {
	t, err := p.expect(lexer.TokenReturn)
	if err != nil {
		return nil, err
	}
	n := &ReturnStatementNode{ReturnToken: t}
	var ok bool
	if n.NewlineToken, ok = p.optional(lexer.TokenNewline); !ok {
		if n.Value, err = p.parseExpression(); err != nil {
			return nil, err
		}
		if n.NewlineToken, err = p.expect(lexer.TokenNewline); err != nil {
			return nil, err
		}
	}
	return n, nil
}

func (p *Parser) parseIfStatement() (*IfStatementNode, error) {
	t, err := p.expect(lexer.TokenIf)
	if err != nil {
		return nil, err
	}
	n := &IfStatementNode{IfToken: t}
	if n.Statement, err = p.parseExpressionStatement(); err != nil {
		return nil, err
	}
	var ok bool
	if n.SemicolonToken, ok = p.optional(lexer.TokenSemicolon); ok {
		if n.Expression, err = p.parseExpression(); err != nil {
			return nil, err
		}
	} else {
		n.Expression = n.Statement
		n.Statement = nil
	}
	if n.Body, err = p.parseBody(); err != nil {
		return nil, err
	}
	if n.ElseToken, ok = p.optional(lexer.TokenElse); ok {
		if p.peek(lexer.TokenOpenBraces) {
			if n.Else, err = p.parseBody(); err != nil {
				return nil, err
			}
		} else {
			if n.Else, err = p.parseIfStatement(); err != nil {
				return nil, err
			}
		}
	}
	return n, nil
}

func (p *Parser) parseForStatement() (*ForStatementNode, error) {
	t, err := p.expect(lexer.TokenFor)
	if err != nil {
		return nil, err
	}
	n := &ForStatementNode{ForToken: t}
	if !p.peek(lexer.TokenOpenBraces) {
		if !p.peek(lexer.TokenSemicolon) {
			if n.StartStatement, err = p.parseExpressionStatement(); err != nil {
				return nil, err
			}
		}
		var ok bool
		if n.SemicolonToken1, ok = p.optional(lexer.TokenSemicolon); ok {
			if n.SemicolonToken2, ok = p.optional(lexer.TokenSemicolon); ok {
				// Do nothing
			} else {
				if n.Condition, err = p.parseExpression(); err != nil {
					return nil, err
				}
				if n.SemicolonToken2, err = p.expect(lexer.TokenSemicolon); err != nil {
					return nil, err
				}
			}
			if !p.peek(lexer.TokenOpenBraces) {
				if n.IncStatement, err = p.parseExpressionStatement(); err != nil {
					return nil, err
				}
			}
		} else {
			n.Condition = n.StartStatement
			n.StartStatement = nil
		}
	}
	if n.Body, err = p.parseBody(); err != nil {
		return nil, err
	}
	if n.NewlineToken, err = p.expect(lexer.TokenNewline); err != nil {
		return nil, err
	}
	return n, nil
}

func (p *Parser) parseExpression() (Node, error) {
	n, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	return p.parseExpressionList(n)
}

func (p *Parser) parseExpressionList(left Node) (Node, error) {
	n, err := p.parseLogicalOr(left)
	if err != nil {
		return nil, err
	}
	if p.peek(lexer.TokenComma) {
		n2 := &ExpressionListNode{}
		n3 := &ExpressionListElementNode{Expression: n}
		n2.Elements = []*ExpressionListElementNode{n3}
		for {
			t, ok := p.optional(lexer.TokenComma)
			if !ok {
				break
			}
			u, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			if n, err = p.parseLogicalOr(u); err != nil {
				return nil, err
			}
			n2.Elements = append(n2.Elements, &ExpressionListElementNode{CommaToken: t, Expression: n})
		}
		return n2, nil
	}
	return n, nil
}

func (p *Parser) parseSingleExpression() (Node, error) {
	n, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	return p.parseLogicalOr(n)
}

func (p *Parser) parseLogicalOr(left Node) (Node, error) {
	n, err := p.parseLogicalAnd(left)
	if err != nil {
		return nil, err
	}
	for {
		t, ok := p.optional(lexer.TokenLogicalOr)
		if !ok {
			break
		}
		u, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		right, err := p.parseLogicalAnd(u)
		if err != nil {
			return nil, err
		}
		n = &BinaryExpressionNode{Left: n, OpToken: t, Right: right}
	}
	return n, nil
}

func (p *Parser) parseLogicalAnd(left Node) (Node, error) {
	n, err := p.parseComparison(left)
	if err != nil {
		return nil, err
	}
	for {
		t, ok := p.optional(lexer.TokenLogicalAnd)
		if !ok {
			break
		}
		u, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		right, err := p.parseComparison(u)
		if err != nil {
			return nil, err
		}
		n = &BinaryExpressionNode{Left: n, OpToken: t, Right: right}
	}
	return n, nil
}

func (p *Parser) parseComparison(left Node) (Node, error) {
	n, err := p.parseIsType(left)
	if err != nil {
		return nil, err
	}
	for {
		t, ok := p.optionalMulti(lexer.TokenEqual, lexer.TokenNotEqual, lexer.TokenLessOrEqual, lexer.TokenGreaterOrEqual, lexer.TokenLess, lexer.TokenGreater)
		if !ok {
			break
		}
		u, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		right, err := p.parseIsType(u)
		if err != nil {
			return nil, err
		}
		n = &BinaryExpressionNode{Left: n, OpToken: t, Right: right}
	}
	return n, nil
}

func (p *Parser) parseIsType(left Node) (Node, error) {
	n, err := p.parseAddExpression(left)
	if err != nil {
		return nil, err
	}
	if t, ok := p.optionalMulti(lexer.TokenIs, lexer.TokenAs); ok {
		n := &IsTypeExpressionNode{Expression: left, IsToken: t}
		if n.Type, err = p.parseType(); err != nil {
			return nil, err
		}
		return n, nil
	}
	return n, nil
}

func (p *Parser) parseAddExpression(left Node) (Node, error) {
	n, err := p.parseMultiplyExpression(left)
	if err != nil {
		return nil, err
	}
	for {
		t, ok := p.optionalMulti(lexer.TokenPlus, lexer.TokenMinus, lexer.TokenBinaryOr, lexer.TokenCaret)
		if !ok {
			break
		}
		u, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		right, err := p.parseMultiplyExpression(u)
		if err != nil {
			return nil, err
		}
		n = &BinaryExpressionNode{Left: n, OpToken: t, Right: right}
	}
	return n, nil
}

func (p *Parser) parseMultiplyExpression(left Node) (Node, error) {
	for {
		t, ok := p.optionalMulti(lexer.TokenAsterisk, lexer.TokenDivision, lexer.TokenPercent, lexer.TokenAmpersand, lexer.TokenShiftLeft, lexer.TokenShiftRight, lexer.TokenBitClear)
		if !ok {
			break
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpressionNode{Left: left, OpToken: t, Right: right}
	}
	return left, nil
}

func (p *Parser) parseUnary() (Node, error) {
	var err error
	if t, ok := p.optionalMulti(lexer.TokenBang, lexer.TokenCaret, lexer.TokenAsterisk, lexer.TokenAmpersand, lexer.TokenMinus); ok {
		n := &UnaryExpressionNode{OpToken: t}
		if n.Expression, err = p.parseUnary(); err != nil {
			return nil, err
		}
		return n, nil
	}
	n, err := p.parsePrimitive()
	if err != nil {
		return nil, err
	}
	return p.parseAccessExpression(n)
}

func (p *Parser) parseAccessExpression(left Node) (Node, error) {
	var err error
	if t, ok := p.optional(lexer.TokenDot); ok {
		n := &MemberAccessExpressionNode{Expression: left, DotToken: t}
		if n.IdentifierToken, err = p.expect(lexer.TokenIdentifier); err != nil {
			return nil, err
		}
		return p.parseAccessExpression(n)
	} else if t, ok := p.optional(lexer.TokenOpenBracket); ok {
		e, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		t2, err := p.expect(lexer.TokenCloseBracket)
		if err != nil {
			return nil, err
		}
		n := &ArrayAccessExpressionNode{Expression: left, OpenToken: t, Index: e, CloseToken: t2}
		return p.parseAccessExpression(n)
	} else if t, ok := p.optional(lexer.TokenOpenParanthesis); ok {
		n := &MemberCallExpressionNode{Expression: left, OpenToken: t}
		if t, ok := p.optional(lexer.TokenCloseParanthesis); ok {
			n.CloseToken = t
			return n, nil
		}
		args, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if list, ok := args.(*ExpressionListNode); ok {
			n.Arguments = list
		} else {
			n2 := &ExpressionListNode{}
			n3 := &ExpressionListElementNode{Expression: n}
			n2.Elements = []*ExpressionListElementNode{n3}
			n.Arguments = n2
		}
		if n.CloseToken, err = p.expect(lexer.TokenCloseParanthesis); err != nil {
			return nil, err
		}
		return p.parseAccessExpression(n)
	} else if p.peek(lexer.TokenBacktick) {
		n := &GenericInstanceFuncNode{Expression: left}
		if n.TypeArguments, err = p.parseTypeList(); err != nil {
			return nil, err
		}
		return p.parseAccessExpression(n)
	}
	return left, nil
}

func (p *Parser) parsePrimitive() (Node, error) {
	if t, ok := p.optionalMulti(lexer.TokenIdentifier, lexer.TokenComponent); ok {
		return &IdentifierExpressionNode{IdentifierToken: t}, nil
	} else if t, ok := p.optionalMulti(lexer.TokenFalse, lexer.TokenTrue, lexer.TokenNull, lexer.TokenInteger, lexer.TokenHex, lexer.TokenOctal, lexer.TokenFloat, lexer.TokenString, lexer.TokenRune); ok {
		return &ConstantExpressionNode{ValueToken: t}, nil
	} else if p.peek(lexer.TokenOpenBraces) {
		return p.parseStructLiteral()
	} else if p.peek(lexer.TokenOpenBracket) {
		return p.parseArrayLiteral()
	} else if p.peek(lexer.TokenOpenParanthesis) {
		return p.parseParanthesis()
	} else if p.peek(lexer.TokenNew) {
		return p.parseNewExpression()
	} else if p.peek(lexer.TokenAt) {
		return p.parseClosure()
	}
	return nil, p.expectError(lexer.TokenIdentifier, lexer.TokenComponent, lexer.TokenFalse, lexer.TokenTrue, lexer.TokenNull, lexer.TokenInteger, lexer.TokenHex, lexer.TokenOctal, lexer.TokenFloat, lexer.TokenString, lexer.TokenRune, lexer.TokenOpenBraces, lexer.TokenOpenBracket, lexer.TokenOpenParanthesis, lexer.TokenNew)
}

func (p *Parser) parseClosure() (Node, error) {
	n := &ClosureExpressionNode{}
	var err error
	var ok bool
	if n.AtToken, err = p.expect(lexer.TokenAt); err != nil {
		return nil, err
	}
	if n.OpenToken, ok = p.optional(lexer.TokenOpenParanthesis); ok {
		if n.Expression, err = p.parseExpression(); err != nil {
			return nil, err
		}
		if n.CloseToken, err = p.expect(lexer.TokenCloseParanthesis); err != nil {
			return nil, err
		}
		return n, nil
	}
	if n.OpenToken, err = p.expect(lexer.TokenOpenBraces); err != nil {
		return nil, err
	}
	if n.NewlineToken, err = p.expect(lexer.TokenNewline); err != nil {
		return nil, err
	}
	for {
		if n.CloseToken, ok = p.optional(lexer.TokenCloseBraces); ok {
			break
		}
		var s Node
		if s, err = p.parseStatement(); err != nil {
			return nil, err
		}
		n.Children = append(n.Children, s)
	}
	return n, nil
}

func (p *Parser) parseArrayLiteral() (Node, error) {
	t, err := p.expect(lexer.TokenOpenBracket)
	if err != nil {
		return nil, err
	}
	e, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	t2, err := p.expect(lexer.TokenCloseBracket)
	if err != nil {
		return nil, err
	}
	args, ok := e.(*ExpressionListNode)
	if !ok {
		args = &ExpressionListNode{}
		n3 := &ExpressionListElementNode{Expression: e}
		args.Elements = []*ExpressionListElementNode{n3}
	}
	return &ArrayLiteralNode{OpenToken: t, Values: args, CloseToken: t2}, nil
}

func (p *Parser) parseStructLiteral() (*StructLiteralNode, error) {
	var err error
	var ok bool
	n := &StructLiteralNode{}
	if n.OpenToken, err = p.expect(lexer.TokenOpenBraces); err != nil {
		return nil, err
	}
	// Parameters
	for {
		if n.CloseToken, ok = p.optional(lexer.TokenCloseBraces); ok {
			break
		}
		f := &StructLiteralFieldNode{}
		if len(n.Fields) != 0 {
			if f.CommaToken, err = p.expect(lexer.TokenComma); err != nil {
				return nil, err
			}
		}
		if err = p.parseStructLiteralField(f); err != nil {
			return nil, err
		}
		n.Fields = append(n.Fields, f)
	}
	return n, nil
}

func (p *Parser) parseStructLiteralField(f *StructLiteralFieldNode) error {
	var err error
	if f.NameToken, err = p.expect(lexer.TokenIdentifier); err != nil {
		return err
	}
	if f.ColonToken, err = p.expect(lexer.TokenColon); err != nil {
		return err
	}
	if f.Value, err = p.parseSingleExpression(); err != nil {
		return err
	}
	return nil
}

func (p *Parser) parseNewExpression() (*NewExpressionNode, error) {
	t, err := p.expect(lexer.TokenNew)
	if err != nil {
		return nil, err
	}
	n := &NewExpressionNode{NewToken: t}
	if n.Type, err = p.parseType(); err != nil {
		return nil, err
	}
	if p.peek(lexer.TokenOpenBraces) {
		if n.Value, err = p.parseStructLiteral(); err != nil {
			return nil, err
		}
	} else if p.peek(lexer.TokenOpenBracket) {
		if n.Value, err = p.parseArrayLiteral(); err != nil {
			return nil, err
		}
	} else if p.peek(lexer.TokenOpenBraces) {
		if n.Value, err = p.parseSingleExpression(); err != nil {
			return nil, err
		}
	}
	return n, nil
}

func (p *Parser) parseParanthesis() (*ParanthesisExpressionNode, error) {
	t, err := p.expect(lexer.TokenOpenParanthesis)
	if err != nil {
		return nil, err
	}
	n := &ParanthesisExpressionNode{OpenToken: t}
	if n.Expression, err = p.parseExpression(); err != nil {
		return nil, err
	}
	if n.CloseToken, err = p.expect(lexer.TokenCloseParanthesis); err != nil {
		return nil, err
	}
	return n, nil
}

func (p *Parser) expect(tokenKind lexer.TokenKind) (*lexer.Token, error) {
	t := p.scan()
	if t.Kind != tokenKind {
		err := p.log.AddError(errlog.ErrorExpectedToken, t.Location, t.Raw, p.l.TokenKindToString(tokenKind))
		return nil, err
	}
	return t, nil
}

func (p *Parser) expectMulti(tokenKind ...lexer.TokenKind) (*lexer.Token, error) {
	t := p.scan()
	for _, k := range tokenKind {
		if t.Kind == k {
			return t, nil
		}
	}
	var str = []string{t.Raw}
	for _, kind := range tokenKind {
		str = append(str, p.l.TokenKindToString(kind))
	}
	err := p.log.AddError(errlog.ErrorExpectedToken, t.Location, str...)
	return nil, err
}

func (p *Parser) optional(tokenKind lexer.TokenKind) (*lexer.Token, bool) {
	t := p.scan()
	if t.Kind != tokenKind {
		p.savedToken = t
		return nil, false
	}
	return t, true
}

func (p *Parser) optionalMulti(tokenKind ...lexer.TokenKind) (*lexer.Token, bool) {
	t := p.scan()
	for _, k := range tokenKind {
		if t.Kind == k {
			return t, true
		}
	}
	p.savedToken = t
	return nil, false
}

func (p *Parser) peek(tokenKind lexer.TokenKind) bool {
	t := p.scan()
	p.savedToken = t
	return t.Kind == tokenKind
}

func (p *Parser) scan() *lexer.Token {
	if p.savedToken != nil {
		t := p.savedToken
		p.savedToken = nil
		return t
	}
	return p.l.Scan()
}

func (p *Parser) expectError(tokenKind ...lexer.TokenKind) error {
	t := p.scan()
	var str = []string{t.Raw}
	for _, kind := range tokenKind {
		str = append(str, p.l.TokenKindToString(kind))
	}
	err := p.log.AddError(errlog.ErrorExpectedToken, t.Location, str...)
	return err
}
