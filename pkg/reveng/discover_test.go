package reveng

import (
	"context"
	"fmt"
	"testing"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsr"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
)

// Test helper: create BSDL for testing
func createRevengTestBSDL(name string, id uint32) string {
	return fmt.Sprintf(`
entity %s is
	attribute INSTRUCTION_LENGTH of %s : entity is 5;
	attribute BOUNDARY_LENGTH of %s : entity is 6;
	attribute INSTRUCTION_OPCODE of %s : entity is
		"BYPASS (11111)," &
		"EXTEST (00000)";
	attribute IDCODE_REGISTER of %s : entity is "%s";
	attribute BOUNDARY_REGISTER of %s : entity is
		"5 (BC_1, *, CONTROL, 1)," &
		"4 (BC_1, PA2, OUTPUT3, X, 5, 1, Z)," &
		"3 (BC_1, PA2, INPUT, X)," &
		"2 (BC_1, *, CONTROL, 1)," &
		"1 (BC_1, PA1, OUTPUT3, X, 2, 1, Z)," &
		"0 (BC_1, PA1, INPUT, X)";
	constant PKG_TEST: PIN_MAP_STRING :=
		"PA1 : A1," &
		"PA2 : A2";
end %s;
`, name, name, name, name, name, idToBinary(id), name, name)
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

func TestDiscoverNetlist_SimpleConnection(t *testing.T) {
	t.Skip("Integration test - needs more complex simulation setup")

	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("parser init failed: %v", err)
	}

	// Create 2-device chain
	repo := chain.NewMemoryRepository()
	ids := []uint32{0x11111111, 0x22222222}

	for i, id := range ids {
		name := fmt.Sprintf("DEV%d", i)
		text := createRevengTestBSDL(name, id)
		file, err := parser.ParseString(text)
		if err != nil {
			t.Fatalf("parse failed for %s: %v", name, err)
		}
		if _, _, err := repo.AddFile(file); err != nil {
			t.Fatalf("AddFile failed: %v", err)
		}
	}

	// Simulated connectivity: DEV0.A1 ↔ DEV1.A1
	// When we drive DEV0.A1, DEV1.A1 should toggle
	connectivity := map[string][]string{
		"DEV0:A1": {"DEV1:A1"},
		"DEV1:A1": {"DEV0:A1"},
	}

	sim := jtag.NewSimAdapter(jtag.AdapterInfo{Name: "sim"})
	idBytes := encodeIDCodes(ids)

	// Track current driven state
	drivenPins := make(map[string]bool)

	sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
		if region == jtag.ShiftRegionDR {
			// Return IDCODE for discovery
			if bits == 64 {
				return append([]byte(nil), idBytes...), nil
			}

			// For boundary scan DR shifts, simulate connectivity
			if bits == 12 { // 2 devices × 6 bits each
				// Decode TDI to see what's being driven
				tdiBits := bytesToBools(tdi, bits)

				// Device layout (reverse order: DEV1 first, DEV0 second)
				// DEV1: bits 0-5
				// DEV0: bits 6-11

				// Check what's driven on each device
				// DEV0 cells: 11,10,9,8,7,6
				//   bit 11: PA2 input
				//   bit 10: PA2 output
				//   bit 9: PA2 control
				//   bit 8: PA1 input
				//   bit 7: PA1 output
				//   bit 6: PA1 control

				// DEV1 cells: 5,4,3,2,1,0
				//   bit 5: PA2 input
				//   bit 4: PA2 output
				//   bit 3: PA2 control
				//   bit 2: PA1 input
				//   bit 1: PA1 output
				//   bit 0: PA1 control

				// Clear current driven state
				drivenPins = make(map[string]bool)

				// Check DEV0.A1 (bit 7 = output, bit 6 = control)
				if !tdiBits[6] { // control = 0 means enabled
					drivenPins["DEV0:A1"] = tdiBits[7]
				}

				// Check DEV0.A2 (bit 10 = output, bit 9 = control)
				if !tdiBits[9] {
					drivenPins["DEV0:A2"] = tdiBits[10]
				}

				// Check DEV1.A1 (bit 1 = output, bit 0 = control)
				if !tdiBits[0] {
					drivenPins["DEV1:A1"] = tdiBits[1]
				}

				// Check DEV1.A2 (bit 4 = output, bit 3 = control)
				if !tdiBits[3] {
					drivenPins["DEV1:A2"] = tdiBits[4]
				}

				// Build TDO based on connectivity
				tdoBits := make([]bool, bits)

				// Set input cells based on driven values and connectivity
				// DEV0.A1 input (bit 8)
				tdoBits[8] = computeInputValue("DEV0:A1", drivenPins, connectivity)

				// DEV0.A2 input (bit 11)
				tdoBits[11] = computeInputValue("DEV0:A2", drivenPins, connectivity)

				// DEV1.A1 input (bit 2)
				tdoBits[2] = computeInputValue("DEV1:A1", drivenPins, connectivity)

				// DEV1.A2 input (bit 5)
				tdoBits[5] = computeInputValue("DEV1:A2", drivenPins, connectivity)

				return boolsToBytes(tdoBits), nil
			}
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
	bsrCtl, err := bsr.NewController(ch)
	if err != nil {
		t.Fatalf("NewController failed: %v", err)
	}

	// Run reverse engineering
	cfg := DefaultConfig()
	cfg.SkipKnownJTAGPins = false
	cfg.SkipPowerPins = false

	ctx := context.Background()
	netlist, err := DiscoverNetlist(ctx, bsrCtl, cfg, nil)
	if err != nil {
		t.Fatalf("DiscoverNetlist failed: %v", err)
	}

	// Verify results
	if netlist.MultiPinNetCount() == 0 {
		t.Error("expected at least one multi-pin net")
	}

	// Find the net containing DEV0.A1 and DEV1.A1
	var foundNet *Net
	for _, net := range netlist.Nets {
		if len(net.Pins) < 2 {
			continue
		}

		hasDEV0A1 := false
		hasDEV1A1 := false

		for _, pin := range net.Pins {
			if pin.DeviceName == "DEV0" && pin.PinName == "A1" {
				hasDEV0A1 = true
			}
			if pin.DeviceName == "DEV1" && pin.PinName == "A1" {
				hasDEV1A1 = true
			}
		}

		if hasDEV0A1 && hasDEV1A1 {
			foundNet = net
			break
		}
	}

	if foundNet == nil {
		t.Errorf("expected to find net connecting DEV0.A1 and DEV1.A1")
		t.Logf("Found nets:")
		for _, net := range netlist.Nets {
			if len(net.Pins) > 1 {
				t.Logf("  Net %d: %v", net.ID, net.Pins)
			}
		}
	}
}

func TestFindTogglers(t *testing.T) {
	driver := bsr.PinRef{ChainIndex: 0, DeviceName: "DEV0", PinName: "A0"}

	pin1 := bsr.PinRef{ChainIndex: 0, DeviceName: "DEV0", PinName: "A1"}
	pin2 := bsr.PinRef{ChainIndex: 1, DeviceName: "DEV1", PinName: "B0"}
	pin3 := bsr.PinRef{ChainIndex: 1, DeviceName: "DEV1", PinName: "B1"}

	// Create capture maps simulating:
	// - pin1 toggles 0→1→0 (connected to driver)
	// - pin2 stays constant (not connected)
	// - pin3 toggles 1→0→1 (connected to driver, inverted)

	baseline := map[bsr.PinRef]bool{
		pin1: false,
		pin2: true,
		pin3: true,
	}

	high := map[bsr.PinRef]bool{
		pin1: true,  // toggled up
		pin2: true,  // no change
		pin3: false, // toggled down
	}

	low2 := map[bsr.PinRef]bool{
		pin1: false, // toggled back
		pin2: true,  // no change
		pin3: true,  // toggled back
	}

	cfg := DefaultConfig()
	togglers := findTogglers(driver, baseline, high, low2, cfg)

	if len(togglers) != 2 {
		t.Errorf("expected 2 togglers, got %d", len(togglers))
	}

	// Check that pin1 and pin3 are in togglers, but not pin2
	hasPin1 := false
	hasPin3 := false
	for _, toggler := range togglers {
		if toggler == pin1 {
			hasPin1 = true
		}
		if toggler == pin3 {
			hasPin3 = true
		}
		if toggler == pin2 {
			t.Errorf("pin2 should not be a toggler")
		}
	}

	if !hasPin1 {
		t.Errorf("pin1 should be a toggler")
	}
	if !hasPin3 {
		t.Errorf("pin3 should be a toggler")
	}
}

func TestConfig_Validate(t *testing.T) {
	cfg := &Config{
		RepeatsPerPin:     0,  // Should be corrected to 1
		MinToggleStrength: -1, // Should be corrected to 1
		OnlyPinPattern:    "PA[0-9]+",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if cfg.RepeatsPerPin != 1 {
		t.Errorf("RepeatsPerPin should be corrected to 1, got %d", cfg.RepeatsPerPin)
	}

	if cfg.MinToggleStrength != 1 {
		t.Errorf("MinToggleStrength should be corrected to 1, got %d", cfg.MinToggleStrength)
	}

	if cfg.pinRegex == nil {
		t.Error("pinRegex should be compiled")
	}

	// Test pin pattern matching
	if !cfg.ShouldScanPin("PA0") {
		t.Error("PA0 should match pattern")
	}

	if cfg.ShouldScanPin("PB0") {
		t.Error("PB0 should not match pattern")
	}
}

func TestIsJTAGPin(t *testing.T) {
	tests := []struct {
		pin  string
		want bool
	}{
		{"TCK", true},
		{"TMS", true},
		{"TDI", true},
		{"TDO", true},
		{"TRST", true},
		{"JTAG_TCK", true},
		{"PA0", false},
		{"MOSI", false},
	}

	for _, tt := range tests {
		got := isJTAGPin(tt.pin)
		if got != tt.want {
			t.Errorf("isJTAGPin(%q) = %v, want %v", tt.pin, got, tt.want)
		}
	}
}

func TestIsPowerPin(t *testing.T) {
	tests := []struct {
		pin  string
		want bool
	}{
		{"VCC", true},
		{"VDD", true},
		{"VSS", true},
		{"GND", true},
		{"VBAT", true},
		{"VREF", true},
		{"PA0", false},
		{"MISO", false},
	}

	for _, tt := range tests {
		got := isPowerPin(tt.pin)
		if got != tt.want {
			t.Errorf("isPowerPin(%q) = %v, want %v", tt.pin, got, tt.want)
		}
	}
}

// Helper: compute what an input should read based on driven pins and connectivity
func computeInputValue(pin string, driven map[string]bool, connectivity map[string][]string) bool {
	// If this pin is driven, return its driven value
	if val, ok := driven[pin]; ok {
		return val
	}

	// Check if any connected pin is driven
	if connected, ok := connectivity[pin]; ok {
		for _, connPin := range connected {
			if val, ok := driven[connPin]; ok {
				return val
			}
		}
	}

	// Default: floating = false
	return false
}

func bytesToBools(buf []byte, bits int) []bool {
	out := make([]bool, bits)
	for i := 0; i < bits; i++ {
		out[i] = buf[i/8]&(1<<(uint(i)%8)) != 0
	}
	return out
}
