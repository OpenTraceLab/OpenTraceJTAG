package bsdl

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/participle/v2"
)

// Parser represents a BSDL file parser
type Parser struct {
	parser *participle.Parser[BSDLFile]
}

// NewParser creates a new BSDL parser instance
func NewParser() (*Parser, error) {
	parser, err := participle.Build[BSDLFile](
		participle.Lexer(BSDLLexer),
		participle.Elide("Comment", "Whitespace"),
		participle.UseLookahead(2),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build parser: %w", err)
	}

	return &Parser{parser: parser}, nil
}

// Parse parses a BSDL file from a reader
func (p *Parser) Parse(r io.Reader) (*BSDLFile, error) {
	bsdl, err := p.parser.Parse("", r)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	return bsdl, nil
}

// ParseString parses a BSDL file from a string
func (p *Parser) ParseString(input string) (*BSDLFile, error) {
	bsdl, err := p.parser.ParseString("", input)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	return bsdl, nil
}

// ParseFile parses a BSDL file from a file path
func (p *Parser) ParseFile(filename string) (*BSDLFile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return p.Parse(file)
}
