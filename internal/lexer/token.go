package lexer

import (
	"math/big"
	"sort"

	"github.com/vs-ude/fyrlang/internal/errlog"
)

// TokenKind ...
type TokenKind int

const (
	// TokenDot ...
	TokenDot TokenKind = 1 + iota
	// TokenIdentifier ...
	TokenIdentifier
	// TokenOctal ...
	TokenOctal
	// TokenHex ...
	TokenHex
	// TokenInteger ...
	TokenInteger
	// TokenFloat ...
	TokenFloat
	// TokenString ...
	TokenString
	// TokenRune ...
	TokenRune
	// TokenLineComment ...
	TokenLineComment
	// TokenBlockComment ...
	TokenBlockComment
	// TokenDivision ...
	TokenDivision
	// TokenAsterisk ...
	TokenAsterisk
	// TokenAmpersand ...
	TokenAmpersand
	// TokenTilde ...
	TokenTilde
	// TokenHash ...
	TokenHash
	// TokenBang ...
	TokenBang
	// TokenCaret ...
	TokenCaret
	// TokenPlus ...
	TokenPlus
	// TokenMinus ...
	TokenMinus
	// TokenPercent ...
	TokenPercent
	// TokenLogicalOr ...
	TokenLogicalOr
	// TokenLogicalAnd ...
	TokenLogicalAnd
	// TokenBinaryOr ...
	TokenBinaryOr
	// TokenBitClear ...
	TokenBitClear
	// TokenIs ...
	TokenIs
	// TokenAs ...
	TokenAs
	// TokenMut ...
	TokenMut
	// TokenDual ...
	TokenDual
	// TokenType ...
	TokenType
	// TokenStruct ...
	TokenStruct
	// TokenInterface ...
	TokenInterface
	// TokenUnion ...
	TokenUnion
	// TokenVar ...
	TokenVar
	// TokenLet ...
	TokenLet
	// TokenIf ...
	TokenIf
	// TokenFor ...
	TokenFor
	// TokenElse ...
	TokenElse
	// TokenInc ...
	TokenInc
	// TokenDec ...
	TokenDec
	// TokenBreak ...
	TokenBreak
	// TokenContinue ...
	TokenContinue
	// TokenReturn ...
	TokenReturn
	// TokenYield ...
	TokenYield
	// TokenComponent ...
	TokenComponent
	// TokenImport ...
	TokenImport
	// TokenFalse ...
	TokenFalse
	// TokenTrue ...
	TokenTrue
	// TokenNull ...
	TokenNull
	// TokenAt ...
	TokenAt
	// TokenOpenParanthesis ...
	TokenOpenParanthesis
	// TokenCloseParanthesis ...
	TokenCloseParanthesis
	// TokenOpenBraces ...
	TokenOpenBraces
	// TokenCloseBraces ...
	TokenCloseBraces
	// TokenOpenBracket ...
	TokenOpenBracket
	// TokenCloseBracket ...
	TokenCloseBracket
	// TokenBacktick ...
	TokenBacktick
	// TokenBacktickDot ...
	TokenBacktickDot
	// TokenComma ...
	TokenComma
	// TokenColon ...
	TokenColon
	// TokenSemicolon ...
	TokenSemicolon
	// TokenFunc ...
	TokenFunc
	// TokenAssign ...
	TokenAssign
	// TokenAssignPlus ...
	TokenAssignPlus
	// TokenAssignMinus ...
	TokenAssignMinus
	// TokenAssignAsterisk ...
	TokenAssignAsterisk
	// TokenAssignDivision ...
	TokenAssignDivision
	// TokenAssignShiftLeft ...
	TokenAssignShiftLeft
	// TokenAssignShiftRight ...
	TokenAssignShiftRight
	// TokenAssignBinaryOr ...
	TokenAssignBinaryOr
	// TokenAssignBinaryAnd ...
	TokenAssignBinaryAnd
	// TokenAssignPercent ...
	TokenAssignPercent
	// TokenAssignAndCaret ...
	TokenAssignAndCaret
	// TokenAssignCaret ...
	TokenAssignCaret
	// TokenWalrus ...
	TokenWalrus
	// TokenShiftLeft ...
	TokenShiftLeft
	// TokenShiftRight ...
	TokenShiftRight
	// TokenEqual ...
	TokenEqual
	// TokenNotEqual ...
	TokenNotEqual
	// TokenLessOrEqual ...
	TokenLessOrEqual
	// TokenGreaterOrEqual ...
	TokenGreaterOrEqual
	// TokenLess ...
	TokenLess
	// TokenGreater ...
	TokenGreater
	// TokenNew ...
	TokenNew
	// TokenNewSlice ...
	TokenNewSlice
	// TokenArrow ...
	TokenArrow
	// TokenExtern ...
	TokenExtern
	// TokenEllipsis ...
	TokenEllipsis
	// TokenNewline ...
	TokenNewline
	// TokenError ...
	TokenError
	// TokenEOF ...
	TokenEOF
	// TokenVolatile ...
	TokenVolatile
)

// Token ...
type Token struct {
	Raw          string
	Kind         TokenKind
	StringValue  string
	RuneValue    rune
	IntegerValue *big.Int
	FloatValue   *big.Float
	ErrorCode    errlog.ErrorCode
	Location     errlog.LocationRange
}

type tokenDefinition struct {
	str  string
	kind TokenKind
}

type tokenizer struct {
	table       [256][]*tokenDefinition
	identifiers map[string]*tokenDefinition
	defs        map[TokenKind]string
}

func newTokenizer() *tokenizer {
	t := &tokenizer{identifiers: make(map[string]*tokenDefinition), defs: make(map[TokenKind]string)}
	return t
}

func (t *tokenizer) addTokenDefinition(str string, kind TokenKind) *tokenDefinition {
	t.defs[kind] = str
	td := &tokenDefinition{str: str, kind: kind}
	if (str[0] >= 'a' && str[0] <= 'z') || (str[0] >= 'A' && str[0] <= 'Z') || str[0] == '_' {
		t.identifiers[str] = td
	}
	t.table[str[0]] = append(t.table[str[0]], td)
	return td
}

func (t *tokenizer) tokenKindToString(kind TokenKind) string {
	return t.defs[kind]
}

func (t *tokenizer) polish() {
	for i := range t.table {
		if t.table[i] == nil {
			continue
		}
		tokenLess := func(a, b int) bool {
			return len(t.table[i][a].str) >= len(t.table[i][b].str)
		}
		sort.Slice(t.table[i], tokenLess)
	}
}

func (t *tokenizer) scan(file int, str string, i int) (token *Token, pos int) {
	if len(str) == i {
		loc := t.encodeRange(file, str, i-1, i)
		return &Token{Kind: TokenEOF, Location: loc}, i
	}
	ch := str[i]
	// Identifier ...
	if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' {
		j := i + 1
		for ; j < len(str); j++ {
			ch = str[j]
			if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
				break
			}
		}
		ident := str[i:j]
		if ident == "new" && j+2 <= len(str) && str[j] == '[' && str[j+1] == ']' {
			j += 2
			ident = "new[]"
		}
		if td, ok := t.identifiers[ident]; ok {
			token := &Token{Kind: td.kind, StringValue: ident, Location: t.encodeRange(file, str, i, j)}
			return token, j
		}
		return &Token{Kind: TokenIdentifier, StringValue: ident, Location: t.encodeRange(file, str, i, j)}, j
	}
	// Hex number
	if ch == '0' && i+1 < len(str) && str[i+1] == 'x' {
		j := i + 2
		ch = str[j]
		if j >= len(str) || ((ch < '0' || ch > '9') && (ch < 'a' || ch > 'f')) {
			j2 := t.skipIdentifierOrNumber(str, j)
			loc := t.encodeRange(file, str, j, j2)
			return t.errorToken(errlog.ErrorIllegalNumber, loc), j2
		}
		for ; j < len(str); j++ {
			ch = str[j]
			if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
				break
			}
		}
		if j < len(str) {
			ch = str[j]
			// No identifier must follow a number without a space in between
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' {
				j2 := t.skipIdentifierOrNumber(str, j)
				loc := t.encodeRange(file, str, j, j2)
				return t.errorToken(errlog.ErrorIllegalNumber, loc), j2
			}
		}
		value := new(big.Int)
		value.SetString(str[i+2:j], 16)
		return &Token{Kind: TokenHex, IntegerValue: value, Location: t.encodeRange(file, str, i, j)}, j
	}
	// Octal number
	if ch == '0' && i+1 < len(str) && (str[i+1] >= '0' && str[i+1] <= '7') {
		j := i + 1
		for ; j < len(str); j++ {
			ch = str[j]
			if ch < '0' || ch > '7' {
				break
			}
		}
		if j < len(str) {
			ch = str[j]
			// No identifier must follow a number without a space in between
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '8' && ch <= '9') || ch == '_' {
				j2 := t.skipIdentifierOrNumber(str, j)
				loc := t.encodeRange(file, str, j, j2)
				return t.errorToken(errlog.ErrorIllegalNumber, loc), j2
			}
		}
		value := new(big.Int)
		value.SetString(str[i+1:j], 8)
		token := &Token{Kind: TokenOctal, IntegerValue: value, Location: t.encodeRange(file, str, i, j)}
		return token, j
	}
	// Number ...
	if ch >= '0' && ch <= '9' {
		isFloat := false
		j := i + 1
		for ; j < len(str); j++ {
			ch = str[j]
			if ch < '0' || ch > '9' {
				break
			}
		}
		// 12.34 ...
		if j+1 < len(str) && str[j] == '.' && str[j+1] >= '0' && str[j+1] <= '9' {
			isFloat = true
			j++
			for ; j < len(str); j++ {
				ch = str[j]
				if ch < '0' || ch > '9' {
					break
				}
			}
			// 12.34e56 or 12e56
			if j+1 < len(str) && str[j] == 'e' && str[j+1] >= '0' && str[j+1] <= '9' {
				j++
				for ; j < len(str); j++ {
					ch = str[j]
					if ch < '0' || ch > '9' {
						break
					}
				}
			}
		}
		if j < len(str) {
			ch = str[j]
			// No identifier must follow a number without a space in between
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' {
				j2 := t.skipIdentifierOrNumber(str, j)
				loc := t.encodeRange(file, str, j, j2)
				return t.errorToken(errlog.ErrorIllegalNumber, loc), j2
			}
		}
		if isFloat {
			value := new(big.Float)
			value.SetString(str[i:j])
			return &Token{Kind: TokenFloat, FloatValue: value, Location: t.encodeRange(file, str, i, j)}, j
		}
		value := new(big.Int)
		value.SetString(str[i:j], 10)
		return &Token{Kind: TokenInteger, IntegerValue: value, Location: t.encodeRange(file, str, i, j)}, j
	}
	if ch == '"' {
		value := make([]byte, 0, 100)
		j := i + 1
		for ; j < len(str); j++ {
			ch = str[j]
			if ch == '"' {
				break
			}
			if ch == '\\' {
				k, s, _, ok := t.scanEscapeSequence(str, j+1)
				if !ok {
					j2 := t.skipString(str, j)
					loc := t.encodeRange(file, str, j, j2)
					return t.errorToken(errlog.ErrorIllegalString, loc), j2
				}
				j = k
				value = append(value, []byte(s)...)
			} else {
				value = append(value, ch)
			}
		}
		if ch != '"' {
			j2 := t.skipString(str, j)
			loc := t.encodeRange(file, str, j, j2)
			return t.errorToken(errlog.ErrorIllegalString, loc), j2
		}
		token := &Token{Kind: TokenString, StringValue: string(value), Location: t.encodeRange(file, str, i, j+1)}
		return token, j + 1
	}
	if ch == '\'' {
		var value rune
		j := i + 1
		ch = str[j]
		if ch == '\\' {
			k, _, v, ok := t.scanEscapeSequence(str, j+1)
			if !ok {
				j2 := t.skipRune(str, j)
				loc := t.encodeRange(file, str, j, j2)
				return t.errorToken(errlog.ErrorIllegalRune, loc), j2
			}
			j = k
			value = v
		} else {
			value = rune(ch)
		}
		j++
		if j == len(str) || str[j] != '\'' {
			j2 := t.skipRune(str, j)
			loc := t.encodeRange(file, str, j, j2)
			return t.errorToken(errlog.ErrorIllegalRune, loc), j2
		}
		token := &Token{Kind: TokenRune, RuneValue: value, Location: t.encodeRange(file, str, i, j+1)}
		return token, j + 1
	}
	// An operator
	tokens := t.table[ch]
	for _, td := range tokens {
		match := true
		for k := 1; k < len(td.str); k++ {
			if str[i+k] != td.str[k] {
				match = false
				break
			}
		}
		if match {
			token := &Token{Kind: td.kind, StringValue: td.str, Location: t.encodeRange(file, str, i, i+len(td.str))}
			return token, i + len(td.str)
		}
	}
	loc := t.encodeRange(file, str, i, i+1)
	return t.errorToken(errlog.ErrorIllegalCharacter, loc), i + 1
}

func (t *tokenizer) scanEscapeSequence(str string, i int) (pos int, result string, resultRune rune, ok bool) {
	ch := str[i]
	switch ch {
	case 'a':
		return i, "\a", 7, true
	case 'b':
		return i, "\b", 8, true
	case 'f':
		return i, "\f", 12, true
	case 'n':
		return i, "\n", 10, true
	case 'r':
		return i, "\r", 13, true
	case 't':
		return i, "\t", 9, true
	case 'v':
		return i, "\v", 11, true
	case '\\':
		return i, "\\", 0x5c, true
	case '"':
		return i, "\"", 0x22, true
	case '\'':
		return i, "'", 0x27, true
	case 'x':
		println("hex rune")
		pos, resultRune, ok = t.scanHexRune(str, i+1, 1)
		result = string(resultRune)
		return
	case 'u':
		pos, resultRune, ok = t.scanHexRune(str, i+1, 2)
		result = string(resultRune)
		return
	case 'U':
		pos, resultRune, ok = t.scanHexRune(str, i+1, 4)
		result = string(resultRune)
		return
	case '0', '1', '2', '3', '4', '5', '6', '7':
		if i+3 > len(str) {
			return i, "", 0, false
		}
	}
	return i, "", 0, false
}

func (t *tokenizer) scanHexRune(str string, i int, n int) (pos int, result rune, ok bool) {
	if i+2*n > len(str) {
		return len(str) - 1, 0, false
	}
	for k := 0; k < n; k++ {
		var r rune
		ch := str[i+2*k+1]
		if ch >= '0' && ch <= '9' {
			r = rune(ch - '0')
		} else if ch >= 'a' && ch <= 'f' {
			r = rune(ch - 'a' + 10)
		} else {
			return i + 2*k + 1, 0, false
		}
		ch = str[i+2*k]
		if ch >= '0' && ch <= '9' {
			r = r + rune(ch-'0')<<4
		} else if ch >= 'a' && ch <= 'f' {
			r = r + rune(ch-'a'+10)<<4
		} else {
			return i + 2*k + 1, 0, false
		}
		result <<= 8
		result += r
	}
	return i + 2*n - 1, result, true
}

func (t *tokenizer) skipIdentifierOrNumber(str string, i int) (pos int) {
	for ; i < len(str); i++ {
		ch := str[i]
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && ch != '_' {
			break
		}
	}
	return i
}

func (t *tokenizer) skipString(str string, i int) (pos int) {
	escaped := false
	for ; i < len(str); i++ {
		ch := str[i]
		if !escaped && ch == '\\' {
			escaped = true
			continue
		}
		if (!escaped && ch == '"') || ch == '\n' || ch == '\r' {
			break
		}
		escaped = false
	}
	return i
}

func (t *tokenizer) skipRune(str string, i int) (pos int) {
	escaped := false
	for ; i < len(str); i++ {
		ch := str[i]
		if !escaped && ch == '\\' {
			escaped = true
			continue
		}
		if (!escaped && ch == '\'') || ch == '\n' || ch == '\r' {
			break
		}
		escaped = false
	}
	return i
}

func (t *tokenizer) encodeRange(file int, str string, from int, to int) errlog.LocationRange {
	var line = 1
	var pos = 1
	for i := 0; i < from; i++ {
		if str[i] == '\n' {
			line++
			pos = 1
		} else {
			pos++
		}
	}
	fromLine := line
	fromPos := pos
	for i := from; i < to; i++ {
		if str[i] == '\n' {
			line++
			pos = 1
		} else {
			pos++
		}
	}
	return errlog.EncodeLocationRange(file, fromLine, fromPos, line, pos)
}

func (t *tokenizer) errorToken(code errlog.ErrorCode, loc errlog.LocationRange) *Token {
	return &Token{Kind: TokenError, Location: loc, ErrorCode: code}
}
