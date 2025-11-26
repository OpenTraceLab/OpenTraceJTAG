package kicadsexp

import (
	"fmt"
	"io"
)

// Parser parses S-expressions from a lexer
type Parser struct {
	lexer   *Lexer
	current Token
}

// NewParser creates a new parser from an io.Reader
func NewParser(r io.Reader) *Parser {
	return &Parser{
		lexer: NewLexer(r),
	}
}

// ParseAll parses all top-level S-expressions from the input
func (p *Parser) ParseAll() ([]Sexp, error) {
	var result []Sexp

	// Read first token
	tok, err := p.lexer.NextToken()
	if err != nil {
		return nil, err
	}
	p.current = tok

	// Parse all expressions until EOF
	for p.current.Type != TokenEOF {
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		result = append(result, expr)

		// Get next token for next iteration
		tok, err := p.lexer.NextToken()
		if err != nil {
			return nil, err
		}
		p.current = tok
	}

	return result, nil
}

// parseExpr parses a single S-expression
func (p *Parser) parseExpr() (Sexp, error) {
	switch p.current.Type {
	case TokenLeftParen:
		return p.parseList()

	case TokenSymbol, TokenString:
		// Atom - return as symbol
		sym := Symbol(p.current.Value)
		return sym, nil

	case TokenRightParen:
		return nil, fmt.Errorf("unexpected ')'")

	case TokenEOF:
		return nil, fmt.Errorf("unexpected EOF")

	default:
		return nil, fmt.Errorf("unexpected token type: %v", p.current.Type)
	}
}

// parseList parses a list: ( ... )
func (p *Parser) parseList() (Sexp, error) {
	// Current token should be '('
	if p.current.Type != TokenLeftParen {
		return nil, fmt.Errorf("expected '(', got %v", p.current.Type)
	}

	var elements []Sexp

	// Read elements until we hit ')'
	for {
		// Get next token
		tok, err := p.lexer.NextToken()
		if err != nil {
			return nil, err
		}
		p.current = tok

		// Check for end of list
		if p.current.Type == TokenRightParen {
			break
		}

		// Check for EOF
		if p.current.Type == TokenEOF {
			return nil, fmt.Errorf("unexpected EOF in list")
		}

		// Parse the element
		elem, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
	}

	return &List{elements: elements}, nil
}
