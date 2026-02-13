package delphi

import (
	"strings"
	"unicode"
)

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
	TokDirective // {$I ...}
	TokLParen
	TokRParen
	TokSemicolon
	TokColon
	TokComma
	TokDot
	TokEquals
	TokLBracket
	TokRBracket
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
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}

		ch := l.input[l.pos]

		switch {
		case ch == '{':
			l.lexBraceComment()
		case ch == '(' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '*':
			l.lexParenComment()
		case ch == '/' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '/':
			l.lexLineComment()
		case ch == '\'':
			l.lexString()
		case ch == '#':
			l.lexCharLiteral()
		case ch == '(':
			l.addToken(TokLParen, "(")
		case ch == ')':
			l.addToken(TokRParen, ")")
		case ch == '[':
			l.addToken(TokLBracket, "[")
		case ch == ']':
			l.addToken(TokRBracket, "]")
		case ch == ';':
			l.addToken(TokSemicolon, ";")
		case ch == ':':
			l.addToken(TokColon, ":")
		case ch == ',':
			l.addToken(TokComma, ",")
		case ch == '.':
			l.addToken(TokDot, ".")
		case ch == '=':
			l.addToken(TokEquals, "=")
		case unicode.IsLetter(rune(ch)) || ch == '_':
			l.lexIdentOrKeyword()
		case unicode.IsDigit(rune(ch)) || ch == '$':
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

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' {
			l.pos++
			l.col++
		} else if ch == '\r' || ch == '\n' {
			l.pos++
			if ch == '\r' && l.pos < len(l.input) && l.input[l.pos] == '\n' {
				l.pos++
			}
			l.line++
			l.col = 1
		} else {
			break
		}
	}
}

func (l *Lexer) lexBraceComment() {
	start := l.pos
	startLine := l.line
	startCol := l.col
	l.pos++ // skip {
	l.col++

	// Check for compiler directive {$...}
	isDirective := l.pos < len(l.input) && l.input[l.pos] == '$'

	for l.pos < len(l.input) && l.input[l.pos] != '}' {
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		}
		l.pos++
		l.col++
	}
	if l.pos < len(l.input) {
		l.pos++ // skip }
		l.col++
	}

	val := l.input[start:l.pos]
	if isDirective {
		l.tokens = append(l.tokens, Token{Type: TokDirective, Value: val, Line: startLine, Col: startCol})
	} else {
		l.tokens = append(l.tokens, Token{Type: TokComment, Value: val, Line: startLine, Col: startCol})
	}
}

func (l *Lexer) lexParenComment() {
	start := l.pos
	startLine := l.line
	startCol := l.col
	l.pos += 2 // skip (*
	l.col += 2

	for l.pos+1 < len(l.input) {
		if l.input[l.pos] == '*' && l.input[l.pos+1] == ')' {
			l.pos += 2
			l.col += 2
			break
		}
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		}
		l.pos++
		l.col++
	}

	l.tokens = append(l.tokens, Token{Type: TokComment, Value: l.input[start:l.pos], Line: startLine, Col: startCol})
}

func (l *Lexer) lexLineComment() {
	start := l.pos
	startCol := l.col
	for l.pos < len(l.input) && l.input[l.pos] != '\n' && l.input[l.pos] != '\r' {
		l.pos++
		l.col++
	}
	l.tokens = append(l.tokens, Token{Type: TokComment, Value: l.input[start:l.pos], Line: l.line, Col: startCol})
}

func (l *Lexer) lexString() {
	start := l.pos
	startCol := l.col
	l.pos++ // skip opening '
	l.col++
	for l.pos < len(l.input) {
		if l.input[l.pos] == '\'' {
			l.pos++
			l.col++
			if l.pos < len(l.input) && l.input[l.pos] == '\'' {
				l.pos++
				l.col++
				continue
			}
			break
		}
		l.pos++
		l.col++
	}
	l.tokens = append(l.tokens, Token{Type: TokString, Value: l.input[start:l.pos], Line: l.line, Col: startCol})
}

func (l *Lexer) lexCharLiteral() {
	start := l.pos
	startCol := l.col
	l.pos++ // skip #
	l.col++
	for l.pos < len(l.input) && unicode.IsDigit(rune(l.input[l.pos])) {
		l.pos++
		l.col++
	}
	l.tokens = append(l.tokens, Token{Type: TokNumber, Value: l.input[start:l.pos], Line: l.line, Col: startCol})
}

func (l *Lexer) lexIdentOrKeyword() {
	start := l.pos
	startCol := l.col
	for l.pos < len(l.input) && (unicode.IsLetter(rune(l.input[l.pos])) || unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '_') {
		l.pos++
		l.col++
	}
	word := l.input[start:l.pos]

	if isPascalKeyword(word) {
		l.tokens = append(l.tokens, Token{Type: TokKeyword, Value: strings.ToLower(word), Line: l.line, Col: startCol})
	} else {
		l.tokens = append(l.tokens, Token{Type: TokIdent, Value: word, Line: l.line, Col: startCol})
	}
}

func (l *Lexer) lexNumber() {
	start := l.pos
	startCol := l.col
	if l.input[l.pos] == '$' {
		l.pos++ // hex prefix
		l.col++
	}
	for l.pos < len(l.input) && (unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '.' ||
		(l.input[l.pos] >= 'a' && l.input[l.pos] <= 'f') ||
		(l.input[l.pos] >= 'A' && l.input[l.pos] <= 'F')) {
		l.pos++
		l.col++
	}
	l.tokens = append(l.tokens, Token{Type: TokNumber, Value: l.input[start:l.pos], Line: l.line, Col: startCol})
}

var pascalKeywords = map[string]bool{
	"unit": true, "program": true, "library": true, "uses": true,
	"interface": true, "implementation": true, "initialization": true,
	"finalization": true, "begin": true, "end": true,
	"type": true, "var": true, "const": true, "procedure": true,
	"function": true, "class": true, "record": true, "object": true,
	"property": true, "inherited": true, "constructor": true,
	"destructor": true, "private": true, "protected": true,
	"public": true, "published": true, "strict": true,
	"if": true, "then": true, "else": true, "case": true,
	"of": true, "for": true, "to": true, "downto": true,
	"do": true, "while": true, "repeat": true, "until": true,
	"with": true, "try": true, "except": true, "finally": true,
	"raise": true, "nil": true, "and": true, "or": true,
	"not": true, "xor": true, "div": true, "mod": true,
	"shl": true, "shr": true, "in": true, "is": true,
	"as": true, "set": true, "array": true, "string": true,
	"packed": true, "file": true, "overload": true, "override": true,
	"virtual": true, "abstract": true, "dynamic": true,
	"reintroduce": true, "external": true,
}

func isPascalKeyword(word string) bool {
	return pascalKeywords[strings.ToLower(word)]
}
