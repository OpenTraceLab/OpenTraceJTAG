package bsr

import (
	"fmt"
	"testing"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
)

// Helper function to create test BSDL
func createTestBSDL(name string, id uint32, irLen, boundaryLen int) string {
	return fmt.Sprintf(`
entity %s is
	attribute INSTRUCTION_LENGTH of %s : entity is %d;
	attribute BOUNDARY_LENGTH of %s : entity is %d;
	attribute INSTRUCTION_OPCODE of %s : entity is
		"BYPASS (11111)," &
		"EXTEST (00000)," &
		"SAMPLE (10101)";
	attribute IDCODE_REGISTER of %s : entity is "%s";
	attribute BOUNDARY_REGISTER of %s : entity is
		"3 (BC_1, *, CONTROL, 1)," &
		"2 (BC_1, PA1, OUTPUT3, X, 3, 1, Z)," &
		"1 (BC_1, *, CONTROL, 1)," &
		"0 (BC_1, PA0, OUTPUT3, X, 1, 1, Z)";
	constant PKG_TEST: PIN_MAP_STRING :=
		"PA0 : A0," &
		"PA1 : A1";
end %s;
`, name, name, irLen, name, boundaryLen, name, name, idToBinary(id),
		name, name)
}

func idToBinary(id uint32) string {
	result := ""
	for i := 31; i >= 0; i-- {
		if id&(1<<uint(i)) != 0 {
			result += "1"
		} else {
			result += "0"
		}
	}
	return result
}

func TestNewController(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("parser init failed: %v", err)
	}

	// Create 2-device chain
	repo := chain.NewMemoryRepository()
	ids := []uint32{0x12345678, 0x87654321}

	for i, id := range ids {
		name := fmt.Sprintf("DEV%d", i)
		text := createTestBSDL(name, id, 5, 4)
		file, err := parser.ParseString(text)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if _, _, err := repo.AddFile(file); err != nil {
			t.Fatalf("AddFile failed: %v", err)
		}
	}

	// Create simulator
	sim := jtag.NewSimAdapter(jtag.AdapterInfo{Name: "sim"})
	idBytes := encodeIDCodes(ids)
	sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
		if region == jtag.ShiftRegionDR && bits == len(ids)*32 {
			return append([]byte(nil), idBytes...), nil
		}
		return make([]byte, (bits+7)/8), nil
	}

	// Discover chain
	chainCtl := chain.NewController(sim, repo)
	ch, err := chainCtl.Discover(2)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Create BSR controller
	bsrCtl, err := NewController(ch)
	if err != nil {
		t.Fatalf("NewController failed: %v", err)
	}

	// Verify devices
	if len(bsrCtl.Devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(bsrCtl.Devices))
	}

	// Verify each device has pins
	for i, dev := range bsrCtl.Devices {
		if dev.boundaryLength != 4 {
			t.Errorf("device %d: expected boundary length 4, got %d", i, dev.boundaryLength)
		}

		// Should have 2 IO pins (PA0, PA1)
		if len(dev.Pins) != 2 {
			t.Errorf("device %d: expected 2 pins, got %d", i, len(dev.Pins))
		}

		// Verify pin states initialized to HiZ
		for pinName, ps := range dev.Pins {
			if ps.Mode != PinHiZ {
				t.Errorf("device %d pin %s: expected HiZ mode, got %v", i, pinName, ps.Mode)
			}
			if ps.DrivenVal != nil {
				t.Errorf("device %d pin %s: expected nil DrivenVal, got %v", i, pinName, *ps.DrivenVal)
			}
		}
	}
}

func TestDRLayout(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("parser init failed: %v", err)
	}

	// Create 2-device chain with different boundary lengths
	repo := chain.NewMemoryRepository()
	ids := []uint32{0x11111111, 0x22222222}
	boundarySizes := []int{4, 8}

	for i, id := range ids {
		name := fmt.Sprintf("DEV%d", i)

		// Generate boundary register cells
		var boundaryCells string
		for j := 0; j < boundarySizes[i]; j++ {
			if j > 0 {
				boundaryCells += " &\n\t\t"
			}
			cellNum := boundarySizes[i] - 1 - j
			if j < boundarySizes[i]-1 {
				boundaryCells += fmt.Sprintf("\"%d (BC_1, *, CONTROL, 0),\"", cellNum)
			} else {
				boundaryCells += fmt.Sprintf("\"%d (BC_1, *, CONTROL, 0)\"", cellNum)
			}
		}

		text := fmt.Sprintf(`
entity %s is
	attribute INSTRUCTION_LENGTH of %s : entity is 5;
	attribute BOUNDARY_LENGTH of %s : entity is %d;
	attribute INSTRUCTION_OPCODE of %s : entity is
		"BYPASS (11111)," &
		"EXTEST (00000)";
	attribute IDCODE_REGISTER of %s : entity is "%s";
	attribute BOUNDARY_REGISTER of %s : entity is
		%s;
end %s;
`, name, name, name, boundarySizes[i], name, name, idToBinary(id), name, boundaryCells, name)

		file, err := parser.ParseString(text)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if _, _, err := repo.AddFile(file); err != nil {
			t.Fatalf("AddFile failed: %v", err)
		}
	}

	sim := jtag.NewSimAdapter(jtag.AdapterInfo{Name: "sim"})
	idBytes := encodeIDCodes(ids)
	sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
		if region == jtag.ShiftRegionDR && bits == len(ids)*32 {
			return append([]byte(nil), idBytes...), nil
		}
		return make([]byte, (bits+7)/8), nil
	}

	chainCtl := chain.NewController(sim, repo)
	ch, err := chainCtl.Discover(2)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	bsrCtl, err := NewController(ch)
	if err != nil {
		t.Fatalf("NewController failed: %v", err)
	}

	// Verify DR layout
	// Total bits = 4 + 8 = 12
	expectedTotal := 4 + 8
	if bsrCtl.Layout.TotalBits != expectedTotal {
		t.Errorf("expected total bits %d, got %d", expectedTotal, bsrCtl.Layout.TotalBits)
	}

	if len(bsrCtl.Layout.Cells) != expectedTotal {
		t.Errorf("expected %d cells, got %d", expectedTotal, len(bsrCtl.Layout.Cells))
	}

	// Verify order: TDO device (index 1) first, then TDI device (index 0)
	// First 8 bits should map to device 1
	for i := 0; i < 8; i++ {
		if bsrCtl.Layout.Cells[i].DeviceIndex != 1 {
			t.Errorf("bit %d: expected device 1, got %d", i, bsrCtl.Layout.Cells[i].DeviceIndex)
		}
	}

	// Next 4 bits should map to device 0
	for i := 8; i < 12; i++ {
		if bsrCtl.Layout.Cells[i].DeviceIndex != 0 {
			t.Errorf("bit %d: expected device 0, got %d", i, bsrCtl.Layout.Cells[i].DeviceIndex)
		}
	}
}

func TestAllPins(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("parser init failed: %v", err)
	}

	repo := chain.NewMemoryRepository()
	id := uint32(0x12345678)
	text := createTestBSDL("DEV0", id, 5, 4)

	file, err := parser.ParseString(text)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if _, _, err := repo.AddFile(file); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	sim := jtag.NewSimAdapter(jtag.AdapterInfo{Name: "sim"})
	idBytes := encodeIDCodes([]uint32{id})
	sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
		if region == jtag.ShiftRegionDR && bits == 32 {
			return append([]byte(nil), idBytes...), nil
		}
		return make([]byte, (bits+7)/8), nil
	}

	chainCtl := chain.NewController(sim, repo)
	ch, err := chainCtl.Discover(1)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	bsrCtl, err := NewController(ch)
	if err != nil {
		t.Fatalf("NewController failed: %v", err)
	}

	// Get all pins
	allPins := bsrCtl.AllPins()
	if len(allPins) != 2 {
		t.Fatalf("expected 2 pins, got %d", len(allPins))
	}

	// Verify pin refs
	for _, ref := range allPins {
		if ref.DeviceName != "DEV0" {
			t.Errorf("expected device name DEV0, got %s", ref.DeviceName)
		}
		if ref.ChainIndex != 0 {
			t.Errorf("expected chain index 0, got %d", ref.ChainIndex)
		}
		if ref.PinName != "A0" && ref.PinName != "A1" {
			t.Errorf("unexpected pin name: %s", ref.PinName)
		}
	}
}

// Helper to encode IDCODEs as bytes
func encodeIDCodes(ids []uint32) []byte {
	bits := make([]bool, len(ids)*32)
	idx := 0
	for _, id := range ids {
		for i := 0; i < 32; i++ {
			bits[idx] = (id>>uint(i))&1 == 1
			idx++
		}
	}
	return boolsToBytes(bits)
}

func boolsToBytes(bits []bool) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, bit := range bits {
		if bit {
			out[i/8] |= 1 << (uint(i) % 8)
		}
	}
	return out
}
