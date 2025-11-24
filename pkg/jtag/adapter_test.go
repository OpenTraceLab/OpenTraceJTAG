package jtag

import (
	"bytes"
	"testing"
)

func TestValidateShiftBuffers(t *testing.T) {
	if _, err := ValidateShiftBuffers(nil, nil, 0); err == nil {
		t.Fatalf("expected error for zero bits")
	}

	_, err := ValidateShiftBuffers([]byte{0x00}, nil, 16)
	if err == nil {
		t.Fatalf("expected error when TMS buffer too small")
	}

	if _, err := ValidateShiftBuffers(nil, []byte{0x01}, 8); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSimAdapterEchoShift(t *testing.T) {
	sim := NewSimAdapter(AdapterInfo{Name: "sim"})
	tdo, err := sim.ShiftDR([]byte{0xAA}, []byte{0xCC}, 8)
	if err != nil {
		t.Fatalf("ShiftDR returned error: %v", err)
	}
	if !bytes.Equal(tdo, []byte{0xCC}) {
		t.Fatalf("tdo = %X, want CC", tdo)
	}

	last := sim.LastShift()
	if last.Region != ShiftRegionDR || last.Bits != 8 {
		t.Fatalf("unexpected last shift metadata: %+v", last)
	}
}

func TestSimAdapterHook(t *testing.T) {
	sim := NewSimAdapter(AdapterInfo{Name: "sim"})
	sim.OnShift = func(region ShiftRegion, _, _ []byte, bits int) ([]byte, error) {
		if region != ShiftRegionIR || bits != 4 {
			t.Fatalf("unexpected hook args: region=%d bits=%d", region, bits)
		}
		return []byte{0x0F}, nil
	}

	tdo, err := sim.ShiftIR(nil, nil, 4)
	if err != nil {
		t.Fatalf("ShiftIR returned error: %v", err)
	}
	if !bytes.Equal(tdo, []byte{0x0F}) {
		t.Fatalf("tdo = %X, want 0F", tdo)
	}
}

func TestSimAdapterResetsAndSpeed(t *testing.T) {
	sim := NewSimAdapter(AdapterInfo{})
	if err := sim.SetSpeed(1_000_000); err != nil {
		t.Fatalf("SetSpeed returned error: %v", err)
	}
	if err := sim.SetSpeed(0); err == nil {
		t.Fatalf("expected error for zero speed")
	}

	if err := sim.ResetTAP(false); err != nil {
		t.Fatalf("ResetTAP returned error: %v", err)
	}
	if err := sim.ResetTAP(true); err != nil {
		t.Fatalf("ResetTAP hard returned error: %v", err)
	}
	if soft, hard := sim.ResetCounts(); soft != 2 || hard != 1 {
		t.Fatalf("ResetCounts = %d soft / %d hard, want 2/1", soft, hard)
	}
}
