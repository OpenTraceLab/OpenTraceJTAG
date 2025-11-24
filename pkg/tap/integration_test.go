package tap

import (
	"testing"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
)

func TestStateMachineSequencesDriveSimAdapter(t *testing.T) {
	m := NewStateMachine()
	// Leave reset so the path is more interesting.
	m.Clock(false) // -> Run-Test/Idle

	seq, err := m.GoTo(StateShiftIR)
	if err != nil {
		t.Fatalf("GoTo returned error: %v", err)
	}

	sim := jtag.NewSimAdapter(jtag.AdapterInfo{Name: "sim"})
	tmsBytes := boolsToBytes(seq.TMS)
	tdi := make([]byte, len(tmsBytes))

	if _, err := sim.ShiftIR(tmsBytes, tdi, len(seq.TMS)); err != nil {
		t.Fatalf("ShiftIR returned error: %v", err)
	}

	last := sim.LastShift()
	if last.Bits != len(seq.TMS) {
		t.Fatalf("adapter bits = %d, want %d", last.Bits, len(seq.TMS))
	}
	gotTMS := bytesToBools(last.TMS, last.Bits)
	if len(gotTMS) != len(seq.TMS) {
		t.Fatalf("decoded bits = %d, want %d", len(gotTMS), len(seq.TMS))
	}
	for i := range gotTMS {
		if gotTMS[i] != seq.TMS[i] {
			t.Fatalf("tms bit %d = %v, want %v", i, gotTMS[i], seq.TMS[i])
		}
	}
}

func boolsToBytes(bits []bool) []byte {
	if len(bits) == 0 {
		return nil
	}
	buf := make([]byte, (len(bits)+7)/8)
	for i, bit := range bits {
		if bit {
			buf[i/8] |= 1 << (uint(i) % 8)
		}
	}
	return buf
}

func bytesToBools(buf []byte, bits int) []bool {
	if bits == 0 {
		return nil
	}
	out := make([]bool, bits)
	for i := 0; i < bits; i++ {
		out[i] = (buf[i/8]&(1<<(uint(i)%8)) != 0)
	}
	return out
}
