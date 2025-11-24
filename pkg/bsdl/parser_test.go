package bsdl

import (
	"testing"
)

func TestParseSimpleEntity(t *testing.T) {
	input := `
	entity TEST_CHIP is
	end TEST_CHIP;
	`

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	bsdl, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if bsdl.Entity == nil {
		t.Fatal("Entity is nil")
	}

	if bsdl.Entity.Name != "TEST_CHIP" {
		t.Errorf("Expected entity name 'TEST_CHIP', got '%s'", bsdl.Entity.Name)
	}
}

func TestParseEntityWithGeneric(t *testing.T) {
	input := `
	entity TEST_CHIP is
		generic (PHYSICAL_PIN_MAP : string := "PKG_120LQFP");
	end TEST_CHIP;
	`

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	bsdl, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if bsdl.Entity.Generic == nil {
		t.Fatal("Generic clause is nil")
	}

	if len(bsdl.Entity.Generic.Generics) != 1 {
		t.Fatalf("Expected 1 generic, got %d", len(bsdl.Entity.Generic.Generics))
	}

	gen := bsdl.Entity.Generic.Generics[0]
	if gen.Name != "PHYSICAL_PIN_MAP" {
		t.Errorf("Expected generic name 'PHYSICAL_PIN_MAP', got '%s'", gen.Name)
	}

	if gen.Type != "string" {
		t.Errorf("Expected type 'string', got '%s'", gen.Type)
	}

	if gen.DefaultValue == nil {
		t.Fatal("Default value is nil")
	}

	if gen.DefaultValue.GetValue() != "PKG_120LQFP" {
		t.Errorf("Expected default value 'PKG_120LQFP', got '%s'", gen.DefaultValue.GetValue())
	}
}

func TestParseEntityWithPorts(t *testing.T) {
	input := `
	entity TEST_CHIP is
		port (
			PA_00 : inout bit;
			PA_01 : inout bit;
			JTG_TCK : in bit;
			JTG_TDO : out bit
		);
	end TEST_CHIP;
	`

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	bsdl, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if bsdl.Entity.Port == nil {
		t.Fatal("Port clause is nil")
	}

	if len(bsdl.Entity.Port.Ports) != 4 {
		t.Fatalf("Expected 4 ports, got %d", len(bsdl.Entity.Port.Ports))
	}

	// Test first port
	port0 := bsdl.Entity.Port.Ports[0]
	if port0.Name != "PA_00" {
		t.Errorf("Expected port name 'PA_00', got '%s'", port0.Name)
	}
	if port0.Mode != "inout" {
		t.Errorf("Expected mode 'inout', got '%s'", port0.Mode)
	}
	if port0.Type.Name != "bit" {
		t.Errorf("Expected type 'bit', got '%s'", port0.Type.Name)
	}

	// Test input port
	port2 := bsdl.Entity.Port.Ports[2]
	if port2.Mode != "in" {
		t.Errorf("Expected mode 'in', got '%s'", port2.Mode)
	}

	// Test output port
	port3 := bsdl.Entity.Port.Ports[3]
	if port3.Mode != "out" {
		t.Errorf("Expected mode 'out', got '%s'", port3.Mode)
	}
}

func TestParseEntityWithUseClause(t *testing.T) {
	input := `
	entity TEST_CHIP is
		use STD_1149_1_2001.all;
	end TEST_CHIP;
	`

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	bsdl, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	useClause := bsdl.Entity.GetUseClause()
	if useClause == nil {
		t.Fatal("Use clause is nil")
	}

	if useClause.Package != "STD_1149_1_2001" {
		t.Errorf("Expected package 'STD_1149_1_2001', got '%s'", useClause.Package)
	}

	if useClause.Dot != "all" {
		t.Errorf("Expected 'all', got '%s'", useClause.Dot)
	}
}

func TestParseConstantAttribute(t *testing.T) {
	input := `
	entity TEST_CHIP is
		constant PKG_120LQFP: PIN_MAP_STRING := "PA_00 : 14," & "PA_01 : 15,";
	end TEST_CHIP;
	`

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	bsdl, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	attrs := bsdl.Entity.GetAttributes()
	if len(attrs) == 0 {
		t.Fatal("No attributes parsed")
	}

	attr := attrs[0]
	if attr.Constant == nil {
		t.Fatal("Expected constant attribute")
	}

	if attr.Constant.Name != "PKG_120LQFP" {
		t.Errorf("Expected constant name 'PKG_120LQFP', got '%s'", attr.Constant.Name)
	}

	if attr.Constant.Type != "PIN_MAP_STRING" {
		t.Errorf("Expected type 'PIN_MAP_STRING', got '%s'", attr.Constant.Type)
	}

	// Test string concatenation
	concatenated := attr.Constant.Value.GetConcatenatedString()
	expected := "PA_00 : 14,PA_01 : 15,"
	if concatenated != expected {
		t.Errorf("Expected concatenated string '%s', got '%s'", expected, concatenated)
	}
}

func TestParseAttributeSpec(t *testing.T) {
	input := `
	entity TEST_CHIP is
		attribute INSTRUCTION_LENGTH of TEST_CHIP: entity is 5;
	end TEST_CHIP;
	`

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	bsdl, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	attrs := bsdl.Entity.GetAttributes()
	if len(attrs) == 0 {
		t.Fatal("No attributes parsed")
	}

	attr := attrs[0]
	if attr.Spec == nil {
		t.Fatal("Expected attribute specification")
	}

	if attr.Spec.Name != "INSTRUCTION_LENGTH" {
		t.Errorf("Expected attribute name 'INSTRUCTION_LENGTH', got '%s'", attr.Spec.Name)
	}

	if attr.Spec.Of != "TEST_CHIP" {
		t.Errorf("Expected 'TEST_CHIP', got '%s'", attr.Spec.Of)
	}

	if attr.Spec.EntityType != "entity" {
		t.Errorf("Expected entity type 'entity', got '%s'", attr.Spec.EntityType)
	}

	// Check the value is 5
	if val, ok := attr.Spec.Is.GetInteger(); !ok || val != 5 {
		t.Errorf("Expected integer value 5, got %v (ok=%v)", val, ok)
	}
}

func TestParseSampleBSDLFile(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	bsdl, err := parser.ParseFile("../../testdata/adsp-21562_adsp-21563_adsp-21565_lqfp_bsdl.bsdl")
	if err != nil {
		t.Fatalf("Failed to parse sample BSDL file: %v", err)
	}

	if bsdl.Entity == nil {
		t.Fatal("Entity is nil")
	}

	expectedName := "ADSP21562_ADSP21563_ADSP21565"
	if bsdl.Entity.Name != expectedName {
		t.Errorf("Expected entity name '%s', got '%s'", expectedName, bsdl.Entity.Name)
	}

	// Check generic clause
	if bsdl.Entity.Generic == nil {
		t.Fatal("Generic clause is nil")
	}

	// Check port clause
	if bsdl.Entity.Port == nil {
		t.Fatal("Port clause is nil")
	}

	t.Logf("Successfully parsed entity '%s' with %d ports", bsdl.Entity.Name, len(bsdl.Entity.Port.Ports))
}
