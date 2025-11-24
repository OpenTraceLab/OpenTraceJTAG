package bsdl

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// BSDLLexer defines the lexical structure for BSDL files
// BSDL is based on VHDL syntax with specific keywords and tokens
var BSDLLexer = lexer.MustSimple([]lexer.SimpleRule{
	// Comments - VHDL style (-- to end of line)
	{Name: "Comment", Pattern: `--[^\n]*`},

	// Whitespace
	{Name: "Whitespace", Pattern: `[\s\t\n\r]+`},

	// Keywords (case-insensitive in VHDL/BSDL)
	// Entity structure
	{Name: "KwEntity", Pattern: `(?i)\bENTITY\b`},
	{Name: "KwIs", Pattern: `(?i)\bIS\b`},
	{Name: "KwEnd", Pattern: `(?i)\bEND\b`},

	// Generic and port clauses
	{Name: "KwGeneric", Pattern: `(?i)\bGENERIC\b`},
	{Name: "KwPort", Pattern: `(?i)\bPORT\b`},

	// Use clause
	{Name: "KwUse", Pattern: `(?i)\bUSE\b`},
	{Name: "KwAll", Pattern: `(?i)\bALL\b`},

	// Attribute keywords
	{Name: "KwAttribute", Pattern: `(?i)\bATTRIBUTE\b`},
	{Name: "KwOf", Pattern: `(?i)\bOF\b`},
	{Name: "KwConstant", Pattern: `(?i)\bCONSTANT\b`},

	// Port modes
	{Name: "KwIn", Pattern: `(?i)\bIN\b`},
	{Name: "KwOut", Pattern: `(?i)\bOUT\b`},
	{Name: "KwInout", Pattern: `(?i)\bINOUT\b`},
	{Name: "KwBuffer", Pattern: `(?i)\bBUFFER\b`},
	{Name: "KwLinkage", Pattern: `(?i)\bLINKAGE\b`},

	// Types
	{Name: "KwBit", Pattern: `(?i)\bBIT\b`},
	{Name: "KwBitVector", Pattern: `(?i)\bBIT_VECTOR\b`},
	{Name: "KwString", Pattern: `(?i)\bSTRING\b`},
	{Name: "KwInteger", Pattern: `(?i)\bINTEGER\b`},
	{Name: "KwReal", Pattern: `(?i)\bREAL\b`},
	{Name: "KwBoolean", Pattern: `(?i)\bBOOLEAN\b`},

	// Boolean literals
	{Name: "KwTrue", Pattern: `(?i)\bTRUE\b`},
	{Name: "KwFalse", Pattern: `(?i)\bFALSE\b`},

	// Operators and punctuation
	{Name: "Assign", Pattern: `:=`},
	{Name: "Colon", Pattern: `:`},
	{Name: "Semicolon", Pattern: `;`},
	{Name: "Comma", Pattern: `,`},
	{Name: "Dot", Pattern: `\.`},
	{Name: "Range", Pattern: `\.\.`},
	{Name: "Concat", Pattern: `&`},
	{Name: "Arrow", Pattern: `=>`},

	// Parentheses and brackets
	{Name: "LParen", Pattern: `\(`},
	{Name: "RParen", Pattern: `\)`},
	{Name: "LBracket", Pattern: `\[`},
	{Name: "RBracket", Pattern: `\]`},

	// Literals
	// String literals with escape sequences
	{Name: "String", Pattern: `"(?:[^"\\]|\\.)*"`},

	// Binary literals (e.g., "00101" for opcodes)
	{Name: "BinaryLit", Pattern: `"[01]+"`},

	// Bit string literals (e.g., X"FF", B"1010")
	{Name: "BitString", Pattern: `[XxBbOo]"[0-9A-Fa-f]+"`},

	// Numbers
	{Name: "Real", Pattern: `[-+]?[0-9]+\.[0-9]+([eE][-+]?[0-9]+)?`},
	{Name: "Integer", Pattern: `[-+]?[0-9]+`},

	// Identifiers (must come after keywords)
	// VHDL allows letters, digits, and underscores, but cannot start with digit
	{Name: "Ident", Pattern: `[a-zA-Z][a-zA-Z0-9_]*`},

	// Special identifier (used for wildcards in boundary scan)
	{Name: "Asterisk", Pattern: `\*`},
})
