package lexer

import "github.com/vs-ude/fyrlang/internal/errlog"

// Lexer ...
type Lexer struct {
	t    *tokenizer
	file int
	str  string
	pos  int
	log  *errlog.ErrorLog
}

// NewLexer ...
func NewLexer(file int, str string, log *errlog.ErrorLog) *Lexer {
	t := newTokenizer()
	t.addTokenDefinition("/*", TokenBlockComment)
	t.addTokenDefinition("//", TokenLineComment)
	t.addTokenDefinition("/", TokenDivision)
	t.addTokenDefinition("*", TokenAsterisk)
	t.addTokenDefinition("&", TokenAmpersand)
	t.addTokenDefinition("(", TokenOpenParanthesis)
	t.addTokenDefinition(")", TokenCloseParanthesis)
	t.addTokenDefinition("{", TokenOpenBraces)
	t.addTokenDefinition("}", TokenCloseBraces)
	t.addTokenDefinition("[", TokenOpenBracket)
	t.addTokenDefinition("]", TokenCloseBracket)
	t.addTokenDefinition("`", TokenBacktick)
	t.addTokenDefinition(".", TokenDot)
	t.addTokenDefinition("`.", TokenBacktickDot)
	t.addTokenDefinition(",", TokenComma)
	t.addTokenDefinition(":", TokenColon)
	t.addTokenDefinition(";", TokenSemicolon)
	t.addTokenDefinition("~", TokenTilde)
	t.addTokenDefinition("#", TokenHash)
	t.addTokenDefinition("!", TokenBang)
	t.addTokenDefinition("^", TokenCaret)
	t.addTokenDefinition("+", TokenPlus)
	t.addTokenDefinition("-", TokenMinus)
	t.addTokenDefinition("%", TokenPercent)
	t.addTokenDefinition("||", TokenLogicalOr)
	t.addTokenDefinition("&&", TokenLogicalAnd)
	t.addTokenDefinition("|", TokenBinaryOr)
	t.addTokenDefinition("&^", TokenBitClear)
	t.addTokenDefinition("=", TokenAssign)
	t.addTokenDefinition("+=", TokenAssignPlus)
	t.addTokenDefinition("-=", TokenAssignMinus)
	t.addTokenDefinition("*=", TokenAssignAsterisk)
	t.addTokenDefinition("/=", TokenAssignDivision)
	t.addTokenDefinition("<<=", TokenAssignShiftLeft)
	t.addTokenDefinition(">>=", TokenAssignShiftRight)
	t.addTokenDefinition("|=", TokenAssignBinaryOr)
	t.addTokenDefinition("&=", TokenAssignBinaryAnd)
	t.addTokenDefinition("%=", TokenAssignPercent)
	t.addTokenDefinition("^=", TokenAssignCaret)
	t.addTokenDefinition("&^=", TokenAssignAndCaret)
	t.addTokenDefinition(":=", TokenWalrus)
	t.addTokenDefinition("<<", TokenShiftLeft)
	t.addTokenDefinition(">>", TokenShiftRight)
	t.addTokenDefinition("==", TokenEqual)
	t.addTokenDefinition("!=", TokenNotEqual)
	t.addTokenDefinition("<=", TokenLessOrEqual)
	t.addTokenDefinition(">=", TokenGreaterOrEqual)
	t.addTokenDefinition("<", TokenLess)
	t.addTokenDefinition(">", TokenGreater)
	t.addTokenDefinition("++", TokenInc)
	t.addTokenDefinition("--", TokenDec)
	t.addTokenDefinition("->", TokenArrow)
	t.addTokenDefinition("...", TokenEllipsis)
	t.addTokenDefinition("@", TokenAt)
	t.addTokenDefinition("is", TokenIs)
	t.addTokenDefinition("as", TokenAs)
	t.addTokenDefinition("func", TokenFunc)
	t.addTokenDefinition("mut", TokenMut)
	t.addTokenDefinition("volatile", TokenVolatile)
	t.addTokenDefinition("dual", TokenDual)
	t.addTokenDefinition("type", TokenType)
	t.addTokenDefinition("struct", TokenStruct)
	t.addTokenDefinition("interface", TokenInterface)
	t.addTokenDefinition("union", TokenUnion)
	t.addTokenDefinition("import", TokenImport)
	t.addTokenDefinition("var", TokenVar)
	t.addTokenDefinition("if", TokenIf)
	t.addTokenDefinition("for", TokenFor)
	t.addTokenDefinition("else", TokenElse)
	t.addTokenDefinition("break", TokenBreak)
	t.addTokenDefinition("continue", TokenContinue)
	t.addTokenDefinition("yield", TokenYield)
	t.addTokenDefinition("return", TokenReturn)
	t.addTokenDefinition("true", TokenTrue)
	t.addTokenDefinition("false", TokenFalse)
	t.addTokenDefinition("null", TokenNull)
	t.addTokenDefinition("component", TokenComponent)
	t.addTokenDefinition("new[]", TokenNewSlice)
	t.addTokenDefinition("new", TokenNew)
	t.addTokenDefinition("extern", TokenExtern)
	t.addTokenDefinition("use", TokenUse)
	t.addTokenDefinition("delete", TokenDelete)
	t.addTokenDefinition("const", TokenConst)
	t.addTokenDefinition("\r\n", TokenNewline)
	t.addTokenDefinition("\n", TokenNewline)
	t.polish()
	l := &Lexer{t: t, str: str, file: file, log: log}
	return l
}

// TokenKindToString ...
func (l *Lexer) TokenKindToString(kind TokenKind) string {
	switch kind {
	case TokenIdentifier:
		return "identifier"
	case TokenString:
		return "string literal"
	case TokenRune:
		return "rune literal"
	case TokenInteger:
		return "integer number"
	case TokenFloat:
		return "floating point number"
	case TokenHex:
		return "hex number"
	case TokenOctal:
		return "octal number"
	}
	return l.t.tokenKindToString(kind)
}

// Scan ...
func (l *Lexer) Scan() *Token {
	start := l.pos
	var token *Token
	for {
		l.skipWhitespace()
		if l.pos == len(l.str) {
			token = &Token{Kind: TokenEOF}
			break
		}
		token, l.pos = l.t.scan(l.file, l.str, l.pos)
		if token.Kind == TokenError {
			l.log.AddError(token.ErrorCode, token.Location)
			continue
		}
		if token.Kind == TokenLineComment {
			l.skipLineComment()
		} else if token.Kind == TokenBlockComment {
			l.skipBlockComment()
		} else {
			break
		}
	}
	token.Raw = l.str[start:l.pos]
	return token
}

func (l *Lexer) skipLineComment() {
	for ; l.pos < len(l.str); l.pos++ {
		ch := l.str[l.pos]
		if ch == '\n' || (ch == '\r' && l.pos+1 < len(l.str) && l.str[l.pos+1] == '\n') {
			break
		}
	}
}

func (l *Lexer) skipBlockComment() {
	for ; l.pos < len(l.str); l.pos++ {
		ch := l.str[l.pos]
		if ch == '*' && l.pos+1 < len(l.str) && l.str[l.pos+1] == '/' {
			l.pos += 2
			break
		}
	}
}

func (l *Lexer) skipWhitespace() {
	for ; l.pos < len(l.str); l.pos++ {
		ch := l.str[l.pos]
		if ch != ' ' && ch != '\t' {
			break
		}
	}
}
