package schematic

import (
	"strings"
	"testing"
)

func TestParseMinimalSchematic(t *testing.T) {
	input := `(kicad_sch
		(version 20250114)
		(generator "eeschema")
		(generator_version "9.0")
		(uuid 862335ee-c981-4fe1-9eb9-84db19301dd4)
		(paper "A4")
		(lib_symbols)
		(sheet_instances
			(path "/"
				(page "1")
			)
		)
	)`

	sch, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse schematic: %v", err)
	}

	if sch.Version != 20250114 {
		t.Errorf("Expected version 20250114, got %d", sch.Version)
	}

	if sch.Generator != "eeschema" {
		t.Errorf("Expected generator 'eeschema', got '%s'", sch.Generator)
	}

	if sch.GeneratorVer != "9.0" {
		t.Errorf("Expected generator version '9.0', got '%s'", sch.GeneratorVer)
	}

	if sch.Paper != "A4" {
		t.Errorf("Expected paper 'A4', got '%s'", sch.Paper)
	}

	if len(sch.SheetInstances) != 1 {
		t.Errorf("Expected 1 sheet instance, got %d", len(sch.SheetInstances))
	}
}

func TestParseSchematicWithSymbol(t *testing.T) {
	input := `(kicad_sch
		(version 20231120)
		(generator "eeschema")
		(uuid test-uuid)
		(paper "A4")
		(lib_symbols
			(symbol "Device:R"
				(property "Reference" "R" (at 0 0 0))
				(property "Value" "R" (at 0 0 0))
				(pin passive line (at -2.54 0 0) (length 2.54)
					(name "1")
					(number "1")
				)
				(pin passive line (at 2.54 0 180) (length 2.54)
					(name "2")
					(number "2")
				)
			)
		)
		(symbol (lib_id "Device:R")
			(at 100 50 0)
			(unit 1)
			(uuid sym-uuid-1)
			(property "Reference" "R1" (at 100 45 0))
			(property "Value" "10k" (at 100 55 0))
		)
	)`

	sch, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse schematic: %v", err)
	}

	if len(sch.LibSymbols) != 1 {
		t.Errorf("Expected 1 lib symbol, got %d", len(sch.LibSymbols))
	}

	if len(sch.Symbols) != 1 {
		t.Errorf("Expected 1 symbol instance, got %d", len(sch.Symbols))
	}

	if sch.Symbols[0].LibID != "Device:R" {
		t.Errorf("Expected lib_id 'Device:R', got '%s'", sch.Symbols[0].LibID)
	}

	// Test GetSymbol helper
	r1 := sch.GetSymbol("R1")
	if r1 == nil {
		t.Error("GetSymbol('R1') returned nil")
	}

	// Test GetAllReferences
	refs := sch.GetAllReferences()
	if len(refs) != 1 || refs[0] != "R1" {
		t.Errorf("Expected refs ['R1'], got %v", refs)
	}
}

func TestParseSchematicWithWires(t *testing.T) {
	input := `(kicad_sch
		(version 20231120)
		(generator "eeschema")
		(uuid test-uuid)
		(paper "A4")
		(lib_symbols)
		(wire (pts (xy 100 50) (xy 150 50))
			(stroke (width 0) (type default))
			(uuid wire-1)
		)
		(wire (pts (xy 150 50) (xy 150 100))
			(stroke (width 0) (type default))
			(uuid wire-2)
		)
		(junction (at 150 50) (diameter 0) (color 0 0 0 0)
			(uuid junc-1)
		)
	)`

	sch, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse schematic: %v", err)
	}

	if len(sch.Wires) != 2 {
		t.Errorf("Expected 2 wires, got %d", len(sch.Wires))
	}

	if len(sch.Junctions) != 1 {
		t.Errorf("Expected 1 junction, got %d", len(sch.Junctions))
	}
}

func TestParseSchematicWithLabels(t *testing.T) {
	input := `(kicad_sch
		(version 20231120)
		(generator "eeschema")
		(uuid test-uuid)
		(paper "A4")
		(lib_symbols)
		(label "VCC" (at 100 50 0)
			(effects (font (size 1.27 1.27)))
			(uuid label-1)
		)
		(global_label "GND" (shape input) (at 100 100 0)
			(effects (font (size 1.27 1.27)))
			(uuid glabel-1)
		)
	)`

	sch, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse schematic: %v", err)
	}

	if len(sch.Labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(sch.Labels))
	}

	if sch.Labels[0].Text != "VCC" {
		t.Errorf("Expected label text 'VCC', got '%s'", sch.Labels[0].Text)
	}

	if len(sch.GlobalLabels) != 1 {
		t.Errorf("Expected 1 global label, got %d", len(sch.GlobalLabels))
	}

	if sch.GlobalLabels[0].Text != "GND" {
		t.Errorf("Expected global label text 'GND', got '%s'", sch.GlobalLabels[0].Text)
	}

	// Test GetLabels helper
	labels := sch.GetLabels()
	if len(labels) != 2 {
		t.Errorf("Expected 2 total labels, got %d", len(labels))
	}
}

func TestParseInvalidRoot(t *testing.T) {
	input := `(kicad_pcb (version 20231120))`

	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Error("Expected error for wrong root node type")
	}
}

func TestParseFile(t *testing.T) {
	// Test with actual test file
	sch, err := ParseFile("../../../testdata/test/test.kicad_sch")
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}

	if sch.Version == 0 {
		t.Error("Version should not be 0")
	}

	if sch.Paper != "A4" {
		t.Errorf("Expected paper 'A4', got '%s'", sch.Paper)
	}
}
