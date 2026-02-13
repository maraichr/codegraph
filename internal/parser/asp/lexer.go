package asp

import (
	"strings"
	"unicode"
)

// TokenType represents a VBScript token type.
type TokenType int

const (
	TokEOF TokenType = iota
	TokKeyword
	TokIdent
	TokString
	TokNumber
	TokOperator
	TokNewline
	TokComment
	TokLineContinuation
	TokLParen
	TokRParen
	TokComma
	TokDot
	TokEquals
	TokAmpersand
)

type Token struct {
	Type  TokenType
	Value string
	Line  int
	Col   int
}

type Lexer struct {
	input  string
	pos    int
	line   int
	col    int
	tokens []Token
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: input, line: 1, col: 1}
}

func (l *Lexer) Tokenize() []Token {
	for l.pos < len(l.input) {
		l.skipSpaces()
		if l.pos >= len(l.input) {
			break
		}

		ch := l.input[l.pos]

		switch {
		case ch == '\r' || ch == '\n':
			l.addToken(TokNewline, "\n")
			if ch == '\r' && l.pos < len(l.input) && l.input[l.pos] == '\n' {
				l.pos++
			}
			l.line++
			l.col = 1
		case ch == '\'':
			l.lexComment()
		case ch == '"':
			l.lexString()
		case ch == '(':
			l.addToken(TokLParen, "(")
		case ch == ')':
			l.addToken(TokRParen, ")")
		case ch == ',':
			l.addToken(TokComma, ",")
		case ch == '.':
			l.addToken(TokDot, ".")
		case ch == '=':
			l.addToken(TokEquals, "=")
		case ch == '&':
			l.addToken(TokAmpersand, "&")
		case ch == '_' && l.peekNewline():
			l.addToken(TokLineContinuation, "_")
			// Skip the newline after continuation
			l.skipToNewline()
		case unicode.IsLetter(rune(ch)) || ch == '_':
			l.lexIdentOrKeyword()
		case unicode.IsDigit(rune(ch)):
			l.lexNumber()
		default:
			l.addToken(TokOperator, string(ch))
		}
	}

	l.tokens = append(l.tokens, Token{Type: TokEOF, Line: l.line, Col: l.col})
	return l.tokens
}

func (l *Lexer) addToken(typ TokenType, val string) {
	l.tokens = append(l.tokens, Token{Type: typ, Value: val, Line: l.line, Col: l.col})
	l.pos++
	l.col++
}

func (l *Lexer) skipSpaces() {
	for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t') {
		l.pos++
		l.col++
	}
}

func (l *Lexer) peekNewline() bool {
	i := l.pos + 1
	for i < len(l.input) && (l.input[i] == ' ' || l.input[i] == '\t') {
		i++
	}
	return i < len(l.input) && (l.input[i] == '\r' || l.input[i] == '\n')
}

func (l *Lexer) skipToNewline() {
	for l.pos < len(l.input) && l.input[l.pos] != '\r' && l.input[l.pos] != '\n' {
		l.pos++
		l.col++
	}
	if l.pos < len(l.input) {
		if l.input[l.pos] == '\r' {
			l.pos++
		}
		if l.pos < len(l.input) && l.input[l.pos] == '\n' {
			l.pos++
		}
		l.line++
		l.col = 1
	}
}

func (l *Lexer) lexComment() {
	start := l.pos
	startCol := l.col
	for l.pos < len(l.input) && l.input[l.pos] != '\r' && l.input[l.pos] != '\n' {
		l.pos++
		l.col++
	}
	l.tokens = append(l.tokens, Token{Type: TokComment, Value: l.input[start:l.pos], Line: l.line, Col: startCol})
}

func (l *Lexer) lexString() {
	start := l.pos
	startCol := l.col
	l.pos++ // skip opening "
	l.col++
	for l.pos < len(l.input) {
		if l.input[l.pos] == '"' {
			l.pos++
			l.col++
			// Check for escaped quote ""
			if l.pos < len(l.input) && l.input[l.pos] == '"' {
				l.pos++
				l.col++
				continue
			}
			break
		}
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		}
		l.pos++
		l.col++
	}
	l.tokens = append(l.tokens, Token{Type: TokString, Value: l.input[start:l.pos], Line: l.line, Col: startCol})
}

func (l *Lexer) lexIdentOrKeyword() {
	start := l.pos
	startCol := l.col
	for l.pos < len(l.input) && (unicode.IsLetter(rune(l.input[l.pos])) || unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '_') {
		l.pos++
		l.col++
	}
	word := l.input[start:l.pos]

	if isVBKeyword(word) {
		l.tokens = append(l.tokens, Token{Type: TokKeyword, Value: strings.ToLower(word), Line: l.line, Col: startCol})
	} else {
		l.tokens = append(l.tokens, Token{Type: TokIdent, Value: word, Line: l.line, Col: startCol})
	}
}

func (l *Lexer) lexNumber() {
	start := l.pos
	startCol := l.col
	for l.pos < len(l.input) && (unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '.') {
		l.pos++
		l.col++
	}
	l.tokens = append(l.tokens, Token{Type: TokNumber, Value: l.input[start:l.pos], Line: l.line, Col: startCol})
}

var vbKeywords = map[string]bool{
	"function": true, "sub": true, "class": true, "end": true,
	"dim": true, "set": true, "const": true, "if": true,
	"then": true, "else": true, "elseif": true, "for": true,
	"next": true, "do": true, "loop": true, "while": true,
	"wend": true, "select": true, "case": true, "with": true,
	"property": true, "get": true, "let": true, "new": true,
	"call": true, "exit": true, "public": true, "private": true,
	"byref": true, "byval": true, "option": true, "explicit": true,
	"nothing": true, "true": true, "false": true, "and": true,
	"or": true, "not": true, "mod": true, "is": true,
	"each": true, "in": true, "to": true, "step": true,
	"redim": true, "preserve": true, "on": true, "error": true,
	"resume": true, "goto": true,
}

func isVBKeyword(word string) bool {
	return vbKeywords[strings.ToLower(word)]
}
