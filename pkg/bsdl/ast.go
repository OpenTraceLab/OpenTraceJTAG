package bsdl

// BSDLFile represents a complete BSDL file
// A BSDL file typically contains one entity declaration
type BSDLFile struct {
	Entity *Entity `@@`
}

// Entity represents the top-level BSDL entity declaration
// Example: entity CHIP_NAME is ... end CHIP_NAME;
type Entity struct {
	Name       string             `KwEntity @Ident KwIs`
	Generic    *GenericClause     `@@?`
	Port       *PortClause        `@@?`
	Decls      []*EntityDecl      `@@*`
	EndName    string             `KwEnd ( KwEntity )? @Ident? Semicolon`
}

// EntityDecl represents declarations within the entity
type EntityDecl struct {
	UseClause *UseClause  `  @@`
	Attribute *Attribute  `| @@`
}

// GetUseClause returns the first use clause if present
func (e *Entity) GetUseClause() *UseClause {
	for _, decl := range e.Decls {
		if decl.UseClause != nil {
			return decl.UseClause
		}
	}
	return nil
}

// GetAttributes returns all attributes
func (e *Entity) GetAttributes() []*Attribute {
	var attrs []*Attribute
	for _, decl := range e.Decls {
		if decl.Attribute != nil {
			attrs = append(attrs, decl.Attribute)
		}
	}
	return attrs
}

// GenericClause represents the generic parameters
// Example: generic (PHYSICAL_PIN_MAP : string := "PKG_120LQFP");
type GenericClause struct {
	Generics []*Generic `KwGeneric LParen ( @@ ( Semicolon @@ )* )? RParen Semicolon`
}

// Generic represents a single generic parameter
type Generic struct {
	Name         string  `@Ident`
	Type         string  `Colon @( Ident | KwString | KwInteger | KwReal | KwBoolean )`
	DefaultValue *String `( Assign @@ )?`
}

// PortClause represents the port declarations
// Example: port ( PA_00 : inout bit; ... );
type PortClause struct {
	Ports []*Port `KwPort LParen ( @@ ( Semicolon @@ )* Semicolon? )? RParen Semicolon`
}

// Port represents a single port declaration
type Port struct {
	Name string     `@Ident`
	Mode string     `Colon @( KwIn | KwOut | KwInout | KwBuffer | KwLinkage )`
	Type *PortType  `@@`
}

// PortType represents the type of a port
type PortType struct {
	Name  string     `@( KwBit | KwBitVector | KwString )`
	Range *RangeSpec `@@?`
}

// RangeSpec represents an array range (e.g., (7 downto 0))
type RangeSpec struct {
	Start     int    `LParen @Integer`
	Direction string `@Ident`
	End       int    `@Integer RParen`
}

// UseClause represents a use clause
// Example: use STD_1149_1_2001.all;
type UseClause struct {
	Package string `KwUse @Ident`
	Dot     string `Dot @( Ident | KwAll ) Semicolon`
}

// Attribute represents an attribute declaration or specification
// BSDL uses attributes extensively for configuration
type Attribute struct {
	Constant *ConstantAttribute `  @@`
	Spec     *AttributeSpec     `| @@`
}

// ConstantAttribute represents a constant declaration
// Example: constant PKG_120LQFP: PIN_MAP_STRING := "PA_00 : 14," & ...
type ConstantAttribute struct {
	Name  string       `KwConstant @Ident`
	Type  string       `Colon @Ident`
	Value *Expression  `Assign @@ Semicolon`
}

// AttributeSpec represents an attribute specification
// Example: attribute INSTRUCTION_LENGTH of CHIP: entity is 5;
type AttributeSpec struct {
	Name       string      `KwAttribute @Ident`
	Of         string      `KwOf @Ident`
	EntityType string      `Colon @( Ident | KwEntity | "signal" | KwConstant )`
	Is         *Expression `KwIs @@ Semicolon`
}

// Expression represents a value expression
// Can be a simple value or a concatenated string expression
type Expression struct {
	Terms []*ExpressionTerm `@@ ( Concat @@ )*`
}

// ExpressionTerm represents a single term in an expression
type ExpressionTerm struct {
	String  *String  `  @@`
	Integer *int     `| @Integer`
	Real    *float64 `| @Real`
	Binary  *string  `| @BinaryLit`
	Ident   *string  `| @Ident`
	Tuple   *Tuple   `| @@`
	Boolean *bool    `| ( @KwTrue | KwFalse )`
}

// Tuple represents a parenthesized list of values
// Example: (35.0e6, BOTH)
type Tuple struct {
	Values []*Expression `LParen @@ ( Comma @@ )* RParen`
}

// String represents a string literal
type String struct {
	Value string `@String`
}

// GetValue returns the string value without quotes
func (s *String) GetValue() string {
	if len(s.Value) >= 2 && s.Value[0] == '"' && s.Value[len(s.Value)-1] == '"' {
		return s.Value[1 : len(s.Value)-1]
	}
	return s.Value
}

// GetConcatenatedString returns the full string from a concatenated expression
func (e *Expression) GetConcatenatedString() string {
	result := ""
	for _, term := range e.Terms {
		if term.String != nil {
			result += term.String.GetValue()
		}
	}
	return result
}

// GetInteger returns the integer value if the expression is a simple integer
func (e *Expression) GetInteger() (int, bool) {
	if len(e.Terms) == 1 && e.Terms[0].Integer != nil {
		return *e.Terms[0].Integer, true
	}
	return 0, false
}
