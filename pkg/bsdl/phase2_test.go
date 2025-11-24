package bsdl

import (
	"testing"
)

// TestParseSTM32Attributes tests parsing of STM32-specific attributes
func TestParseSTM32Attributes(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	bsdl, err := parser.ParseFile("../../testdata/STM32F303_F334_LQFP64.bsd")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	attrs := bsdl.Entity.GetAttributes()
	t.Logf("Total attributes: %d", len(attrs))

	// Count different attribute types
	specCount := 0
	constCount := 0

	for _, attr := range attrs {
		if attr.Spec != nil {
			specCount++
			t.Logf("Attribute Spec: %s of %s", attr.Spec.Name, attr.Spec.Of)

			// Check specific attributes
			switch attr.Spec.Name {
			case "TAP_SCAN_CLOCK":
				t.Logf("  TAP_SCAN_CLOCK value terms: %d", len(attr.Spec.Is.Terms))
				for i, term := range attr.Spec.Is.Terms {
					t.Logf("    Term %d: tuple=%v, real=%v, ident=%v",
						i, term.Tuple != nil, term.Real != nil, term.Ident != nil)
				}

			case "INSTRUCTION_OPCODE":
				t.Logf("  INSTRUCTION_OPCODE value: %s", attr.Spec.Is.GetConcatenatedString())

			case "INSTRUCTION_CAPTURE":
				t.Logf("  INSTRUCTION_CAPTURE value: %s", attr.Spec.Is.GetConcatenatedString())

			case "IDCODE_REGISTER":
				t.Logf("  IDCODE_REGISTER value: %s", attr.Spec.Is.GetConcatenatedString())
			}
		} else if attr.Constant != nil {
			constCount++
			t.Logf("Constant: %s of type %s", attr.Constant.Name, attr.Constant.Type)
		}
	}

	t.Logf("\nAttribute Specs: %d", specCount)
	t.Logf("Constants: %d", constCount)
}

// TestParseLatticeAttributes tests parsing of Lattice FPGA attributes
func TestParseLatticeAttributes(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	bsdl, err := parser.ParseFile("../../testdata/LFE5U_25F_CABGA381.bsm")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	attrs := bsdl.Entity.GetAttributes()
	t.Logf("Total Lattice attributes: %d", len(attrs))

	for _, attr := range attrs {
		if attr.Spec != nil {
			switch attr.Spec.Name {
			case "INSTRUCTION_LENGTH":
				if val, ok := attr.Spec.Is.GetInteger(); ok {
					t.Logf("INSTRUCTION_LENGTH: %d bits", val)
				}

			case "IDCODE_REGISTER":
				t.Logf("IDCODE_REGISTER: %s", attr.Spec.Is.GetConcatenatedString())
			}
		}
	}
}
