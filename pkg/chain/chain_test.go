package chain

import (
	"fmt"
	"strings"
	"testing"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
)

func TestDiscoverChainWithRepository(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("parser init failed: %v", err)
	}

	type deviceDef struct {
		name       string
		id         uint32
		irLength   int
		boundary   int
		extraAttrs string
	}

	defs := []deviceDef{
		{name: "DEV_A", id: 0x12345678, irLength: 5, boundary: 32},
		{name: "DEV_B", id: 0x87654321, irLength: 4, boundary: 16},
	}

	repo := NewMemoryRepository()
	var ids []uint32

	for _, def := range defs {
		text := fmt.Sprintf(`
entity %s is
	attribute INSTRUCTION_LENGTH of %s : entity is %d;
	attribute BOUNDARY_LENGTH of %s : entity is %d;
	attribute IDCODE_REGISTER of %s : entity is "%s";
	attribute INSTRUCTION_OPCODE of %s : entity is "BYPASS (11111), IDCODE (00001)";
end %s;
`, def.name, def.name, def.irLength,
			def.name, def.boundary,
			def.name, idToBinary(def.id),
			def.name,
			def.name)

		file, err := parser.ParseString(text)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		id, _, err := repo.AddFile(file)
		if err != nil {
			t.Fatalf("AddFile failed: %v", err)
		}
		ids = append(ids, id)
	}

	sim := jtag.NewSimAdapter(jtag.AdapterInfo{Name: "sim"})
	idBytes := encodeIDCodes(ids)

	sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
		if region == jtag.ShiftRegionDR && bits == len(ids)*32 {
			return append([]byte(nil), idBytes...), nil
		}
		return make([]byte, (bits+7)/8), nil
	}

	ctrl := NewController(sim, repo)
	chain, err := ctrl.Discover(len(ids))
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	devices := chain.Devices()
	if len(devices) != len(ids) {
		t.Fatalf("got %d devices, want %d", len(devices), len(ids))
	}

	for i, dev := range devices {
		if dev.IDCode != ids[i] {
			t.Fatalf("device %d ID = 0x%08X, want 0x%08X", i, dev.IDCode, ids[i])
		}
		if dev.Name() != defs[i].name {
			t.Fatalf("device %d name = %s, want %s", i, dev.Name(), defs[i].name)
		}
		if dev.Info == nil || dev.Info.InstructionLength != defs[i].irLength {
			t.Fatalf("device %d IR len mismatch", i)
		}
		instr := dev.Instructions()
		if len(instr) == 0 {
			t.Fatalf("device %d missing instructions", i)
		}
	}
}

func idToBinary(id uint32) string {
	var b strings.Builder
	for i := 31; i >= 0; i-- {
		if id&(1<<uint(i)) != 0 {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
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

func collectBits(ops []jtag.ShiftOp) []int {
	vals := make([]int, len(ops))
	for i, op := range ops {
		vals[i] = op.Bits
	}
	return vals
}

func TestTogglePinDrivesBoundary(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("parser init failed: %v", err)
	}

	bsdlText := `
entity DEV is
	attribute INSTRUCTION_LENGTH of DEV : entity is 4;
	attribute BOUNDARY_LENGTH of DEV : entity is 2;
	attribute INSTRUCTION_OPCODE of DEV : entity is
		"BYPASS (1111)," &
		"EXTEST (0000)";
	attribute IDCODE_REGISTER of DEV : entity is "00000000000000000000000000000001";
	attribute BOUNDARY_REGISTER of DEV : entity is
		"1 (BC_1, *, CONTROL, 1)," &
		"0 (BC_1, LED, OUTPUT3, X, 1, 1, Z)";
end DEV;
`

	file, err := parser.ParseString(bsdlText)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	repo := NewMemoryRepository()
	if _, _, err := repo.AddFile(file); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	sim := jtag.NewSimAdapter(jtag.AdapterInfo{Name: "sim"})
	var (
		irOps []jtag.ShiftOp
		drOps []jtag.ShiftOp
	)
	idVector := make([]bool, 32)
	idVector[0] = true
	idBytes := boolsToBytes(idVector)
	idServed := false
	sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
		op := jtag.ShiftOp{
			Region: region,
			TMS:    append([]byte(nil), tms...),
			TDI:    append([]byte(nil), tdi...),
			Bits:   bits,
		}
		switch region {
		case jtag.ShiftRegionIR:
			irOps = append(irOps, op)
		case jtag.ShiftRegionDR:
			drOps = append(drOps, op)
		}
		if region == jtag.ShiftRegionDR && bits == 32 && !idServed {
			idServed = true
			return append([]byte(nil), idBytes...), nil
		}
		return make([]byte, (bits+7)/8), nil
	}

	chain, err := NewController(sim, repo).Discover(1)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	dev := chain.Devices()[0]
	if _, err := dev.boundaryData(); err != nil {
		t.Fatalf("boundary data error: %v", err)
	}
	if outCell, ctrlCell, err := dev.outputCell("LED"); err != nil {
		t.Fatalf("outputCell error: %v", err)
	} else {
		if outCell.Disable != 1 {
			t.Fatalf("expected output disable 1, got %d", outCell.Disable)
		}
		if ctrlCell == nil {
			t.Fatalf("control cell missing")
		}
	}
	vec, err := dev.boundaryVectorForPin("LED", true)
	if err != nil {
		t.Fatalf("boundaryVectorForPin error: %v", err)
	}
	if len(vec) != 2 {
		t.Fatalf("expected vector len 2, got %d", len(vec))
	}
	if !vec[0] {
		t.Fatalf("boundary vector did not set LED bit")
	}
	if vec[1] {
		outCell, ctrlCell, err := dev.outputCell("LED")
		if err != nil {
			t.Fatalf("outputCell error: %v", err)
		}
		if ctrlCell == nil {
			t.Fatalf("no control cell for %v", outCell)
		}
		t.Fatalf("boundary vector control bit should be 0 (Disable=%d)", ctrlCell.Disable)
	}
	preDRCount := len(drOps)
	if err := chain.TogglePin("DEV", "LED", true); err != nil {
		t.Fatalf("TogglePin failed: %v", err)
	}

	var irOp *jtag.ShiftOp
	for i := range irOps {
		if irOps[i].Bits == 4 {
			irOp = &irOps[i]
			break
		}
	}
	if irOp == nil {
		t.Fatalf("no IR operation with 4 bits captured (ops: %v)", collectBits(irOps))
	}
	irBits := bytesToBools(irOp.TDI, irOp.Bits)
	if len(irBits) != 4 {
		t.Fatalf("expected 4 IR bits, got %d", len(irBits))
	}
	for i, bit := range irBits {
		if bit {
			t.Fatalf("expected EXTEST opcode bits to be 0, bit %d was 1", i)
		}
	}

	var drOp *jtag.ShiftOp
	for i := len(drOps) - 1; i >= preDRCount; i-- {
		if drOps[i].Bits == 2 {
			tms := bytesToBools(drOps[i].TMS, drOps[i].Bits)
			if len(tms) == 2 && !tms[0] && tms[1] {
				drOp = &drOps[i]
				break
			}
		}
	}
	if drOp == nil {
		t.Fatalf("no 2-bit DR op found after toggle (ops: %v)", collectBits(drOps[preDRCount:]))
	}
	drBits := bytesToBools(drOp.TDI, drOp.Bits)
	if len(drBits) != 2 {
		t.Fatalf("expected 2 DR bits, got %d", len(drBits))
	}
	if !drBits[0] {
		t.Fatalf("LED bit not driven high")
	}
	if drBits[1] {
		t.Fatalf("control bit should enable output (0), got 1")
	}
}

func TestBatchCaptureReturnsData(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("parser init failed: %v", err)
	}

	bsdlText := `
entity DEV is
	attribute INSTRUCTION_LENGTH of DEV : entity is 4;
	attribute BOUNDARY_LENGTH of DEV : entity is 2;
	attribute INSTRUCTION_OPCODE of DEV : entity is
		"BYPASS (1111)," &
		"EXTEST (0000)," &
		"SAMPLE (1010)";
	attribute IDCODE_REGISTER of DEV : entity is "00000000000000000000000000000001";
	attribute BOUNDARY_REGISTER of DEV : entity is
		"1 (BC_1, *, CONTROL, 1)," &
		"0 (BC_1, LED, OUTPUT3, X, 1, 1, Z)";
end DEV;
`

	file, err := parser.ParseString(bsdlText)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	repo := NewMemoryRepository()
	if _, _, err := repo.AddFile(file); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	sim := jtag.NewSimAdapter(jtag.AdapterInfo{Name: "sim"})
	idVector := make([]bool, 32)
	idVector[0] = true
	idBytes := boolsToBytes(idVector)
	idServed := false
	sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
		if region == jtag.ShiftRegionDR {
			if !idServed && bits == 32 {
				idServed = true
				return append([]byte(nil), idBytes...), nil
			}
			if idServed && bits == 2 {
				return []byte{0x02}, nil
			}
		}
		return make([]byte, (bits+7)/8), nil
	}

	chain, err := NewController(sim, repo).Discover(1)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	batch := chain.NewBatch()
	if err := batch.SetPin("DEV", "LED", true); err != nil {
		t.Fatalf("SetPin failed: %v", err)
	}
	if err := batch.Capture("DEV"); err != nil {
		t.Fatalf("Capture failed: %v", err)
	}
	result, err := batch.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	bits, ok := result["DEV"]
	if !ok {
		t.Fatalf("missing capture data for DEV")
	}
	if len(bits) != 2 {
		t.Fatalf("expected 2 bits, got %d", len(bits))
	}
	if bits[0] {
		t.Fatalf("expected LED capture bit low")
	}
	if !bits[1] {
		t.Fatalf("expected control capture bit high")
	}
}
