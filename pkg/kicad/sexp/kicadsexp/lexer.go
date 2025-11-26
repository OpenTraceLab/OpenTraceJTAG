package kicadsexp

import (
	"bufio"
	"fmt"
	"io"
	"unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenLeftParen
	TokenRightParen
	TokenSymbol
	TokenString
)

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value string
}

// Lexer tokenizes S-expressions from an io.Reader
type Lexer struct {
	reader *bufio.Reader
	peeked *rune
}

// NewLexer creates a new lexer
func NewLexer(r io.Reader) *Lexer {
	return &Lexer{
		reader: bufio.NewReader(r),
	}
}

// NextToken reads the next token from the input
func (l *Lexer) NextToken() (Token, error) {
	// Skip whitespace and comments
	for {
		ch, err := l.peek()
		if err != nil {
			if err == io.EOF {
				return Token{Type: TokenEOF}, nil
			}
			return Token{}, err
		}

		// Skip whitespace
		if unicode.IsSpace(ch) {
			l.read() // consume it
			continue
		}

		// Skip comments (from # to end of line)
		if ch == '#' {
			// Skip until newline
			for {
				c, err := l.read()
				if err != nil || c == '\n' {
					break
				}
			}
			continue
		}

		break
	}

	// Read the actual token
	ch, err := l.peek()
	if err != nil {
		if err == io.EOF {
			return Token{Type: TokenEOF}, nil
		}
		return Token{}, err
	}

	switch ch {
	case '(':
		l.read()
		return Token{Type: TokenLeftParen, Value: "("}, nil

	case ')':
		l.read()
		return Token{Type: TokenRightParen, Value: ")"}, nil

	case '"':
		return l.readString()

	default:
		return l.readSymbol()
	}
}

// peek looks at the next rune without consuming it
func (l *Lexer) peek() (rune, error) {
	if l.peeked != nil {
		return *l.peeked, nil
	}

	ch, _, err := l.reader.ReadRune()
	if err != nil {
		return 0, err
	}

	l.peeked = &ch
	return ch, nil
}

// read consumes and returns the next rune
func (l *Lexer) read() (rune, error) {
	if l.peeked != nil {
		ch := *l.peeked
		l.peeked = nil
		return ch, nil
	}

	ch, _, err := l.reader.ReadRune()
	return ch, err
}

// readString reads a quoted string
func (l *Lexer) readString() (Token, error) {
	// Consume opening quote
	l.read()

	var result []rune
	for {
		ch, err := l.read()
		if err != nil {
			if err == io.EOF {
				return Token{}, fmt.Errorf("unexpected EOF in string")
			}
			return Token{}, err
		}

		if ch == '"' {
			// Check if it's an escaped quote
			next, err := l.peek()
			if err == nil && next == '"' {
				// Escaped quote - consume it and add to string
				l.read()
				result = append(result, '"')
				continue
			}
			// End of string
			break
		}

		if ch == '\\' {
			// Handle escape sequences
			next, err := l.read()
			if err != nil {
				return Token{}, fmt.Errorf("unexpected EOF after backslash")
			}
			switch next {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case 'r':
				result = append(result, '\r')
			case '\\':
				result = append(result, '\\')
			case '"':
				result = append(result, '"')
			default:
				// Unknown escape - just include it
				result = append(result, next)
			}
			continue
		}

		result = append(result, ch)
	}

	return Token{Type: TokenString, Value: string(result)}, nil
}

// readSymbol reads an unquoted symbol (identifier, number, etc.)
func (l *Lexer) readSymbol() (Token, error) {
	var result []rune

	for {
		ch, err := l.peek()
		if err != nil {
			if err == io.EOF {
				break
			}
			return Token{}, err
		}

		// Stop at delimiters
		if unicode.IsSpace(ch) || ch == '(' || ch == ')' || ch == '"' {
			break
		}

		// Consume and add to symbol
		l.read()
		result = append(result, ch)
	}

	if len(result) == 0 {
		return Token{}, fmt.Errorf("empty symbol")
	}

	return Token{Type: TokenSymbol, Value: string(result)}, nil
}
