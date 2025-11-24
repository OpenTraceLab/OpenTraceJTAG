package bsdl

import "testing"

func TestGetBoundaryCells(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("parser init failed: %v", err)
	}

	file, err := parser.ParseFile("../../testdata/STM32F303_F334_LQFP64.bsd")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	info := file.Entity.GetDeviceInfo()
	if info == nil {
		t.Fatalf("missing device info")
	}

	cells, err := file.Entity.GetBoundaryCells()
	if err != nil {
		t.Fatalf("GetBoundaryCells returned error: %v", err)
	}
	if len(cells) != info.BoundaryLength {
		t.Fatalf("expected %d cells, got %d", info.BoundaryLength, len(cells))
	}

	find := func(port string, function string) *BoundaryCell {
		for i := range cells {
			cell := &cells[i]
			if cell.Port == port && cell.Function == function {
				return cell
			}
		}
		return nil
	}

	if cell := find("PA0", "OUTPUT3"); cell == nil {
		t.Fatalf("PA0 OUTPUT3 cell not found")
	} else {
		if cell.Control != 111 {
			t.Fatalf("expected control 111 for PA0, got %d", cell.Control)
		}
		if cell.Safe != "X" {
			t.Fatalf("expected safe X, got %s", cell.Safe)
		}
	}

	if ctrl := find("*", "CONTROL"); ctrl == nil {
		t.Fatalf("CONTROL cell not found")
	}
}
