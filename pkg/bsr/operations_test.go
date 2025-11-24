package bsr

import (
	"testing"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
)

func TestEnterExtest(t *testing.T) {
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
	var irOps []jtag.ShiftOp

	idBytes := encodeIDCodes([]uint32{id})
	sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
		if region == jtag.ShiftRegionIR {
			irOps = append(irOps, jtag.ShiftOp{
				Region: region,
				TDI:    append([]byte(nil), tdi...),
				Bits:   bits,
			})
		}
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

	// Enter EXTEST
	if err := bsrCtl.EnterExtest(); err != nil {
		t.Fatalf("EnterExtest failed: %v", err)
	}

	// Find the IR shift with the EXTEST opcode
	var extestIR *jtag.ShiftOp
	for i := range irOps {
		if irOps[i].Bits == 5 { // IR length is 5
			extestIR = &irOps[i]
			break
		}
	}

	if extestIR == nil {
		t.Fatalf("no IR shift found for EXTEST")
	}

	// Verify EXTEST opcode (00000)
	irBits := bytesToBools(extestIR.TDI, extestIR.Bits)
	for i, bit := range irBits {
		if bit {
			t.Errorf("bit %d: expected 0 for EXTEST, got 1", i)
		}
	}
}

func TestSetAllPinsHiZ(t *testing.T) {
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
	var drOps []jtag.ShiftOp

	idBytes := encodeIDCodes([]uint32{id})
	sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
		if region == jtag.ShiftRegionDR {
			if bits == 32 {
				return append([]byte(nil), idBytes...), nil
			}
			drOps = append(drOps, jtag.ShiftOp{
				Region: region,
				TDI:    append([]byte(nil), tdi...),
				Bits:   bits,
			})
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

	// Enter EXTEST
	if err := bsrCtl.EnterExtest(); err != nil {
		t.Fatalf("EnterExtest failed: %v", err)
	}

	// Clear DR ops before SetAllPinsHiZ so we only capture that operation
	drOps = nil

	if err := bsrCtl.SetAllPinsHiZ(); err != nil {
		t.Fatalf("SetAllPinsHiZ failed: %v", err)
	}

	// Find the DR shift
	if len(drOps) == 0 {
		t.Fatalf("no DR shifts recorded")
	}

	// Find the 4-bit DR shift (should be the actual boundary scan shift)
	var drOp *jtag.ShiftOp
	for i := range drOps {
		if drOps[i].Bits == 4 {
			drOp = &drOps[i]
			break
		}
	}

	if drOp == nil {
		t.Fatalf("no 4-bit DR shift found (ops: %v bits)", func() []int {
			bits := make([]int, len(drOps))
			for i, op := range drOps {
				bits[i] = op.Bits
			}
			return bits
		}())
	}

	// Verify all pin states are HiZ
	for _, dev := range bsrCtl.Devices {
		for pinName, ps := range dev.Pins {
			if ps.Mode != PinHiZ {
				t.Errorf("pin %s: expected HiZ mode, got %v", pinName, ps.Mode)
			}
		}
	}
}

func TestDrivePin(t *testing.T) {
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
	var drOps []jtag.ShiftOp

	idBytes := encodeIDCodes([]uint32{id})
	sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
		if region == jtag.ShiftRegionDR {
			if bits == 32 {
				return append([]byte(nil), idBytes...), nil
			}
			drOps = append(drOps, jtag.ShiftOp{
				Region: region,
				TDI:    append([]byte(nil), tdi...),
				Bits:   bits,
			})
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

	// Enter EXTEST
	if err := bsrCtl.EnterExtest(); err != nil {
		t.Fatalf("EnterExtest failed: %v", err)
	}

	// Clear DR ops before DrivePin so we only capture that operation
	drOps = nil

	// Drive PA0 (package pin A0) high
	pinRef := PinRef{
		ChainIndex: 0,
		DeviceName: "DEV0",
		PinName:    "A0",
	}

	if err := bsrCtl.DrivePin(pinRef, true); err != nil {
		t.Fatalf("DrivePin failed: %v", err)
	}

	// Verify pin state
	ps := bsrCtl.GetPinState(pinRef)
	if ps == nil {
		t.Fatalf("pin state not found")
	}
	if ps.Mode != PinOutput {
		t.Errorf("expected Output mode, got %v", ps.Mode)
	}
	if ps.DrivenVal == nil || !*ps.DrivenVal {
		t.Errorf("expected DrivenVal=true, got %v", ps.DrivenVal)
	}

	// Verify DR shift occurred
	if len(drOps) == 0 {
		t.Fatalf("no DR shifts recorded")
	}

	// Find the 4-bit DR shift (should be the actual boundary scan shift)
	var drOp *jtag.ShiftOp
	for i := range drOps {
		if drOps[i].Bits == 4 {
			drOp = &drOps[i]
			break
		}
	}

	if drOp == nil {
		t.Fatalf("no 4-bit DR shift found")
	}

	drBits := bytesToBools(drOp.TDI, drOp.Bits)

	// Boundary register layout (from createTestBSDL):
	// 3: control for PA1
	// 2: output for PA1
	// 1: control for PA0
	// 0: output for PA0

	// PA0 driven high, so bit 0 should be 1
	if !drBits[0] {
		t.Errorf("bit 0 (PA0 output): expected 1, got 0")
	}

	// Control cell (bit 1) should enable output
	// Disable value is 1, so enable should be 0
	if drBits[1] {
		t.Errorf("bit 1 (PA0 control): expected 0 to enable, got 1")
	}
}

func TestCaptureAll(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("parser init failed: %v", err)
	}

	// Create BSDL with input cells
	repo := chain.NewMemoryRepository()
	id := uint32(0x12345678)
	text := `
entity DEV0 is
	attribute INSTRUCTION_LENGTH of DEV0 : entity is 5;
	attribute BOUNDARY_LENGTH of DEV0 : entity is 4;
	attribute INSTRUCTION_OPCODE of DEV0 : entity is
		"BYPASS (11111)," &
		"EXTEST (00000)";
	attribute IDCODE_REGISTER of DEV0 : entity is "00010010001101000101011001111000";
	attribute BOUNDARY_REGISTER of DEV0 : entity is
		"3 (BC_1, PB1, INPUT, X)," &
		"2 (BC_1, *, CONTROL, 1)," &
		"1 (BC_1, PB0, INPUT, X)," &
		"0 (BC_1, PA0, OUTPUT3, X, 2, 1, Z)";
	constant PKG_TEST: PIN_MAP_STRING :=
		"PA0 : A0," &
		"PB0 : B0," &
		"PB1 : B1";
end DEV0;
`

	file, err := parser.ParseString(text)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if _, _, err := repo.AddFile(file); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	sim := jtag.NewSimAdapter(jtag.AdapterInfo{Name: "sim"})
	idBytes := encodeIDCodes([]uint32{id})

	// Return specific pattern for DR capture: 0b1010 (bit 1 and 3 high)
	sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
		if region == jtag.ShiftRegionDR {
			if bits == 32 {
				return append([]byte(nil), idBytes...), nil
			}
			if bits == 4 {
				return []byte{0b1010}, nil // Binary: 1010
			}
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

	// Enter EXTEST
	if err := bsrCtl.EnterExtest(); err != nil {
		t.Fatalf("EnterExtest failed: %v", err)
	}

	// Capture all inputs
	values, err := bsrCtl.CaptureAll()
	if err != nil {
		t.Fatalf("CaptureAll failed: %v", err)
	}

	// Should capture PB0 (bit 1) and PB1 (bit 3)
	// Expected: PB0=false (bit 1=0 in 0b1010), PB1=true (bit 3=1 in 0b1010)

	pb0Ref := PinRef{ChainIndex: 0, DeviceName: "DEV0", PinName: "B0"}
	pb1Ref := PinRef{ChainIndex: 0, DeviceName: "DEV0", PinName: "B1"}

	// Note: bytesToBools unpacks LSB first, so:
	// 0b1010 -> [false, true, false, true]
	// bit 1 (PB0) = true
	// bit 3 (PB1) = true

	if val, ok := values[pb0Ref]; !ok {
		t.Errorf("PB0 not in captured values")
	} else if !val {
		t.Errorf("PB0: expected true, got false")
	}

	if val, ok := values[pb1Ref]; !ok {
		t.Errorf("PB1 not in captured values")
	} else if !val {
		t.Errorf("PB1: expected true, got false")
	}

	// PA0 is output, so it should not be in the captured values
	pa0Ref := PinRef{ChainIndex: 0, DeviceName: "DEV0", PinName: "A0"}
	if _, ok := values[pa0Ref]; ok {
		t.Errorf("PA0 is output, should not be captured")
	}
}

func bytesToBools(buf []byte, bits int) []bool {
	out := make([]bool, bits)
	for i := 0; i < bits; i++ {
		out[i] = buf[i/8]&(1<<(uint(i)%8)) != 0
	}
	return out
}
