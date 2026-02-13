package tsql

import (
	"strings"
	"unicode"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenKeyword
	TokenIdent
	TokenNumber
	TokenString
	TokenOperator
	TokenPunctuation
	TokenGO        // batch separator
	TokenComment
	TokenNewline
)

type Token struct {
	Type    TokenType
	Value   string
	Line    int
	Col     int
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

		// Line comments
		if l.pos+1 < len(l.input) && l.input[l.pos:l.pos+2] == "--" {
			l.readLineComment()
			continue
		}

		// Block comments
		if l.pos+1 < len(l.input) && l.input[l.pos:l.pos+2] == "/*" {
			l.readBlockComment()
			continue
		}

		// Strings
		if ch == '\'' {
			l.readString()
			continue
		}

		// Quoted identifiers
		if ch == '[' {
			l.readBracketIdent()
			continue
		}
		if ch == '"' {
			l.readQuotedIdent()
			continue
		}

		// Numbers
		if ch >= '0' && ch <= '9' {
			l.readNumber()
			continue
		}

		// Identifiers / keywords
		if isIdentStart(ch) {
			l.readIdentOrKeyword()
			continue
		}

		// Newlines (for GO detection)
		if ch == '\n' {
			l.tokens = append(l.tokens, Token{Type: TokenNewline, Value: "\n", Line: l.line, Col: l.col})
			l.line++
			l.col = 1
			l.pos++
			continue
		}
		if ch == '\r' {
			l.pos++
			if l.pos < len(l.input) && l.input[l.pos] == '\n' {
				l.pos++
			}
			l.tokens = append(l.tokens, Token{Type: TokenNewline, Value: "\n", Line: l.line, Col: l.col})
			l.line++
			l.col = 1
			continue
		}

		// Operators and punctuation
		l.readOperatorOrPunct()
	}

	l.tokens = append(l.tokens, Token{Type: TokenEOF, Line: l.line, Col: l.col})

	// Post-process: detect GO batch separators
	l.detectGO()

	return l.tokens
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' {
			l.pos++
			l.col++
		} else {
			break
		}
	}
}

func (l *Lexer) readLineComment() {
	start := l.pos
	startLine := l.line
	startCol := l.col
	for l.pos < len(l.input) && l.input[l.pos] != '\n' && l.input[l.pos] != '\r' {
		l.pos++
		l.col++
	}
	l.tokens = append(l.tokens, Token{Type: TokenComment, Value: l.input[start:l.pos], Line: startLine, Col: startCol})
}

func (l *Lexer) readBlockComment() {
	start := l.pos
	startLine := l.line
	startCol := l.col
	l.pos += 2
	l.col += 2
	for l.pos+1 < len(l.input) {
		if l.input[l.pos] == '*' && l.input[l.pos+1] == '/' {
			l.pos += 2
			l.col += 2
			break
		}
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
	l.tokens = append(l.tokens, Token{Type: TokenComment, Value: l.input[start:l.pos], Line: startLine, Col: startCol})
}

func (l *Lexer) readString() {
	startLine := l.line
	startCol := l.col
	l.pos++ // skip opening quote
	l.col++
	var b strings.Builder
	b.WriteByte('\'')
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\'' {
			l.pos++
			l.col++
			if l.pos < len(l.input) && l.input[l.pos] == '\'' {
				b.WriteString("''")
				l.pos++
				l.col++
				continue
			}
			b.WriteByte('\'')
			break
		}
		if ch == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		b.WriteByte(ch)
		l.pos++
	}
	l.tokens = append(l.tokens, Token{Type: TokenString, Value: b.String(), Line: startLine, Col: startCol})
}

func (l *Lexer) readBracketIdent() {
	startLine := l.line
	startCol := l.col
	l.pos++ // skip [
	l.col++
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != ']' {
		l.pos++
		l.col++
	}
	val := l.input[start:l.pos]
	if l.pos < len(l.input) {
		l.pos++ // skip ]
		l.col++
	}
	l.tokens = append(l.tokens, Token{Type: TokenIdent, Value: val, Line: startLine, Col: startCol})
}

func (l *Lexer) readQuotedIdent() {
	startLine := l.line
	startCol := l.col
	l.pos++ // skip "
	l.col++
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		l.pos++
		l.col++
	}
	val := l.input[start:l.pos]
	if l.pos < len(l.input) {
		l.pos++ // skip "
		l.col++
	}
	l.tokens = append(l.tokens, Token{Type: TokenIdent, Value: val, Line: startLine, Col: startCol})
}

func (l *Lexer) readNumber() {
	start := l.pos
	startLine := l.line
	startCol := l.col
	for l.pos < len(l.input) && (l.input[l.pos] >= '0' && l.input[l.pos] <= '9' || l.input[l.pos] == '.') {
		l.pos++
		l.col++
	}
	l.tokens = append(l.tokens, Token{Type: TokenNumber, Value: l.input[start:l.pos], Line: startLine, Col: startCol})
}

func (l *Lexer) readIdentOrKeyword() {
	start := l.pos
	startLine := l.line
	startCol := l.col
	for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
		l.pos++
		l.col++
	}
	val := l.input[start:l.pos]

	if isKeyword(val) {
		l.tokens = append(l.tokens, Token{Type: TokenKeyword, Value: strings.ToUpper(val), Line: startLine, Col: startCol})
	} else {
		l.tokens = append(l.tokens, Token{Type: TokenIdent, Value: val, Line: startLine, Col: startCol})
	}
}

func (l *Lexer) readOperatorOrPunct() {
	startLine := l.line
	startCol := l.col
	ch := l.input[l.pos]
	l.pos++
	l.col++

	switch ch {
	case '(', ')', ',', ';', '.', '=', '<', '>', '+', '-', '*', '/', '%', '!', '@', '#':
		l.tokens = append(l.tokens, Token{Type: TokenPunctuation, Value: string(ch), Line: startLine, Col: startCol})
	default:
		l.tokens = append(l.tokens, Token{Type: TokenOperator, Value: string(ch), Line: startLine, Col: startCol})
	}
}

// detectGO converts IDENT "GO" at the start of a line to TokenGO.
func (l *Lexer) detectGO() {
	for i := range l.tokens {
		if l.tokens[i].Type == TokenKeyword && l.tokens[i].Value == "GO" {
			// Check it's at the start of a line (preceded by newline or start of file)
			atLineStart := i == 0
			if !atLineStart {
				for j := i - 1; j >= 0; j-- {
					if l.tokens[j].Type == TokenNewline {
						atLineStart = true
						break
					}
					if l.tokens[j].Type != TokenComment {
						break
					}
				}
			}
			if atLineStart {
				l.tokens[i].Type = TokenGO
			}
		}
	}
}

func isIdentStart(ch byte) bool {
	return ch == '_' || ch == '#' || ch == '@' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9') || ch == '$'
}

func isKeyword(s string) bool {
	_, ok := tsqlKeywords[strings.ToUpper(s)]
	return ok
}

var tsqlKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true, "INTO": true,
	"UPDATE": true, "DELETE": true, "CREATE": true, "ALTER": true, "DROP": true,
	"TABLE": true, "VIEW": true, "PROCEDURE": true, "PROC": true, "FUNCTION": true,
	"TRIGGER": true, "INDEX": true, "SCHEMA": true, "DATABASE": true,
	"BEGIN": true, "END": true, "IF": true, "ELSE": true, "WHILE": true,
	"RETURN": true, "RETURNS": true, "DECLARE": true, "SET": true,
	"EXEC": true, "EXECUTE": true, "GO": true,
	"AS": true, "ON": true, "AND": true, "OR": true, "NOT": true, "NULL": true,
	"IS": true, "IN": true, "EXISTS": true, "BETWEEN": true, "LIKE": true,
	"JOIN": true, "INNER": true, "LEFT": true, "RIGHT": true, "OUTER": true,
	"CROSS": true, "FULL": true, "APPLY": true,
	"GROUP": true, "BY": true, "ORDER": true, "HAVING": true,
	"UNION": true, "ALL": true, "EXCEPT": true, "INTERSECT": true,
	"TOP": true, "DISTINCT": true, "WITH": true,
	"CASE": true, "WHEN": true, "THEN": true, "NOCOUNT": true,
	"PRIMARY": true, "KEY": true, "FOREIGN": true, "REFERENCES": true,
	"CONSTRAINT": true, "UNIQUE": true, "CHECK": true, "DEFAULT": true,
	"INT": true, "BIGINT": true, "SMALLINT": true, "TINYINT": true,
	"VARCHAR": true, "NVARCHAR": true, "CHAR": true, "NCHAR": true,
	"TEXT": true, "NTEXT": true, "BIT": true, "DECIMAL": true,
	"NUMERIC": true, "FLOAT": true, "REAL": true, "MONEY": true,
	"DATE": true, "DATETIME": true, "DATETIME2": true, "TIME": true,
	"IDENTITY": true, "OUTPUT": true, "INSERTED": true, "DELETED": true,
	"FOR": true, "AFTER": true, "INSTEAD": true, "OF": true,
	"VALUES": true, "OVER": true, "PARTITION": true,
	"TRY": true, "CATCH": true, "THROW": true,
	"TYPE": true, "CURSOR": true, "FETCH": true, "NEXT": true,
	"OPEN": true, "CLOSE": true, "DEALLOCATE": true,
	"MERGE": true, "MATCHED": true, "TARGET": true, "SOURCE": true,
	"OPTION": true, "RECOMPILE": true, "NOLOCK": true,
	"REPLACE": true, "MAX": true,
}

// Unexported helper used by the lexer but also exported for callers needing char classification.
func IsSpace(r rune) bool {
	return unicode.IsSpace(r)
}
