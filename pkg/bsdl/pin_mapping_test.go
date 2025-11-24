package bsdl

import (
	"path/filepath"
	"testing"
)

func TestPinMappingReturnsInputCell(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	bsdlFile, err := parser.ParseFile(filepath.Join("../../testdata", "STM32F303_F334_LQFP64.bsd"))
	if err != nil {
		t.Fatalf("Failed to parse BSDL: %v", err)
	}
	
	pm := ExtractPinMapping(bsdlFile)
	
	// PA5 should map to INPUT cell (94), not OUTPUT cell (95)
	idx := pm.GetBSRIndex("PA5")
	t.Logf("PA5 BSR index: %d", idx)
	
	if idx != 94 {
		t.Errorf("Expected PA5 to map to INPUT cell 94, got %d", idx)
	}
	
	// Verify it's actually an INPUT cell
	cells, err := bsdlFile.Entity.GetBoundaryCells()
	if err != nil {
		t.Fatalf("Failed to get boundary cells: %v", err)
	}
	
	for _, cell := range cells {
		if cell.Number == idx && cell.Port == "PA5" {
			t.Logf("Cell %d: Port=%s, Function=%s", cell.Number, cell.Port, cell.Function)
			if cell.Function != "INPUT" {
				t.Errorf("Expected INPUT function, got %s", cell.Function)
			}
			break
		}
	}
}
