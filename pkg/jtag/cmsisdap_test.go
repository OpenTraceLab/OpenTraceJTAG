package jtag

import (
	"testing"
)

func TestCMSISDAPAdapter_buildSequences(t *testing.T) {
	adapter := &CMSISDAPAdapter{
		protocol: NewCMSISDAPProtocol(64),
	}

	tests := []struct {
		name      string
		tms       []byte
		tdi       []byte
		bits      int
		wantSeqs  int
		checkFunc func(*testing.T, []JTAGSequence)
	}{
		{
			name:     "no TMS, 8 bits",
			tms:      nil,
			tdi:      []byte{0xAA},
			bits:     8,
			wantSeqs: 1,
			checkFunc: func(t *testing.T, seqs []JTAGSequence) {
				if seqs[0].TCKCount() != 8 {
					t.Errorf("Expected 8 TCK, got %d", seqs[0].TCKCount())
				}
				if seqs[0].TMS() != false {
					t.Errorf("Expected TMS=false")
				}
				if !seqs[0].CaptureTDO() {
					t.Errorf("Expected TDO capture")
				}
			},
		},
		{
			name:     "constant TMS=0, 16 bits",
			tms:      []byte{0x00, 0x00},
			tdi:      []byte{0xAA, 0x55},
			bits:     16,
			wantSeqs: 1,
			checkFunc: func(t *testing.T, seqs []JTAGSequence) {
				if seqs[0].TCKCount() != 16 {
					t.Errorf("Expected 16 TCK, got %d", seqs[0].TCKCount())
				}
				if seqs[0].TMS() != false {
					t.Errorf("Expected TMS=false")
				}
			},
		},
		{
			name:     "constant TMS=1, 8 bits",
			tms:      []byte{0xFF},
			tdi:      []byte{0xAA},
			bits:     8,
			wantSeqs: 1,
			checkFunc: func(t *testing.T, seqs []JTAGSequence) {
				if seqs[0].TCKCount() != 8 {
					t.Errorf("Expected 8 TCK, got %d", seqs[0].TCKCount())
				}
				if seqs[0].TMS() != true {
					t.Errorf("Expected TMS=true")
				}
			},
		},
		{
			name:     "TMS changes, 16 bits",
			tms:      []byte{0x0F, 0x00}, // First 4 bits TMS=1, rest TMS=0
			tdi:      []byte{0xAA, 0x55},
			bits:     16,
			wantSeqs: 2, // Should split into two sequences
			checkFunc: func(t *testing.T, seqs []JTAGSequence) {
				// First sequence: 4 bits with TMS=1
				if seqs[0].TCKCount() != 4 {
					t.Errorf("Seq 0: Expected 4 TCK, got %d", seqs[0].TCKCount())
				}
				if seqs[0].TMS() != true {
					t.Errorf("Seq 0: Expected TMS=true")
				}

				// Second sequence: 12 bits with TMS=0
				if seqs[1].TCKCount() != 12 {
					t.Errorf("Seq 1: Expected 12 TCK, got %d", seqs[1].TCKCount())
				}
				if seqs[1].TMS() != false {
					t.Errorf("Seq 1: Expected TMS=false")
				}
			},
		},
		{
			name:     "70 bits, no TMS (should split at 64)",
			tms:      nil,
			tdi:      make([]byte, 9),
			bits:     70,
			wantSeqs: 2, // 64 + 6
			checkFunc: func(t *testing.T, seqs []JTAGSequence) {
				if seqs[0].TCKCount() != 64 {
					t.Errorf("Seq 0: Expected 64 TCK, got %d", seqs[0].TCKCount())
				}
				if seqs[1].TCKCount() != 6 {
					t.Errorf("Seq 1: Expected 6 TCK, got %d", seqs[1].TCKCount())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seqs := adapter.buildSequences(tt.tms, tt.tdi, tt.bits)

			if len(seqs) != tt.wantSeqs {
				t.Errorf("Expected %d sequences, got %d", tt.wantSeqs, len(seqs))
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, seqs)
			}
		})
	}
}

func TestCMSISDAPAdapter_ValidateInterface(t *testing.T) {
	// Compile-time check that CMSISDAPAdapter implements Adapter interface
	var _ Adapter = (*CMSISDAPAdapter)(nil)
}

// Integration test - requires real CMSIS-DAP hardware
func TestCMSISDAPAdapter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	adapter, err := NewCMSISDAPAdapter(VendorIDRaspberryPi, ProductIDCMSISDAP)
	if err != nil {
		t.Skipf("No CMSIS-DAP hardware found: %v", err)
	}
	defer adapter.Close()

	t.Run("Info", func(t *testing.T) {
		info, err := adapter.Info()
		if err != nil {
			t.Fatalf("Info() failed: %v", err)
		}

		t.Logf("Adapter: %s", info.Name)
		t.Logf("Vendor: %s", info.Vendor)
		t.Logf("Model: %s", info.Model)
		t.Logf("Serial: %s", info.SerialNumber)
		t.Logf("Firmware: %s", info.Firmware)
		t.Logf("Frequency: %d - %d Hz", info.MinFrequency, info.MaxFrequency)

		if info.Vendor == "" {
			t.Error("Vendor should not be empty")
		}
	})

	t.Run("SetSpeed", func(t *testing.T) {
		// Test setting to 1 MHz
		err := adapter.SetSpeed(1_000_000)
		if err != nil {
			t.Errorf("SetSpeed(1MHz) failed: %v", err)
		}

		// Test invalid speed (too low)
		err = adapter.SetSpeed(100)
		if err == nil {
			t.Error("SetSpeed(100Hz) should have failed")
		}

		// Test invalid speed (too high)
		err = adapter.SetSpeed(100_000_000)
		if err == nil {
			t.Error("SetSpeed(100MHz) should have failed")
		}
	})

	t.Run("ResetTAP", func(t *testing.T) {
		// Soft reset
		err := adapter.ResetTAP(false)
		if err != nil {
			t.Errorf("ResetTAP(soft) failed: %v", err)
		}

		// Hard reset (may not be supported on all probes)
		err = adapter.ResetTAP(true)
		if err != nil && err != ErrNotImplemented {
			t.Logf("ResetTAP(hard) failed: %v (may not be supported)", err)
		}
	})

	t.Run("ShiftIR", func(t *testing.T) {
		// Simple 5-bit IR shift with TMS=0
		tms := []byte{0x00}
		tdi := []byte{0x1F} // All ones
		bits := 5

		tdo, err := adapter.ShiftIR(tms, tdi, bits)
		if err != nil {
			t.Errorf("ShiftIR() failed: %v", err)
		}

		if len(tdo) != 1 {
			t.Errorf("Expected 1 byte TDO, got %d", len(tdo))
		}

		t.Logf("ShiftIR TDO: 0x%02X", tdo[0])
	})

	t.Run("ShiftDR", func(t *testing.T) {
		// Simple 8-bit DR shift
		tms := []byte{0x00}
		tdi := []byte{0xAA}
		bits := 8

		tdo, err := adapter.ShiftDR(tms, tdi, bits)
		if err != nil {
			t.Errorf("ShiftDR() failed: %v", err)
		}

		if len(tdo) != 1 {
			t.Errorf("Expected 1 byte TDO, got %d", len(tdo))
		}

		t.Logf("ShiftDR TDO: 0x%02X", tdo[0])
	})

	t.Run("ConfigureJTAGChain", func(t *testing.T) {
		// Configure for single device with 5-bit IR
		irLengths := []byte{5}

		err := adapter.ConfigureJTAGChain(irLengths)
		if err != nil {
			t.Errorf("ConfigureJTAGChain() failed: %v", err)
		}
	})

	t.Run("ReadIDCODE", func(t *testing.T) {
		// Configure chain first
		adapter.ConfigureJTAGChain([]byte{5})

		// Read IDCODE from device 0
		idcode, err := adapter.ReadIDCODE(0)
		if err != nil {
			t.Errorf("ReadIDCODE() failed: %v", err)
		}

		t.Logf("IDCODE: 0x%08X", idcode)

		// IDCODE should have bit 0 = 1 (per IEEE 1149.1)
		if idcode&0x01 != 1 {
			t.Errorf("IDCODE bit 0 should be 1, got 0x%08X", idcode)
		}
	})
}

// Benchmark sequence building
func BenchmarkBuildSequences(b *testing.B) {
	adapter := &CMSISDAPAdapter{
		protocol: NewCMSISDAPProtocol(64),
	}

	tms := make([]byte, 100)
	tdi := make([]byte, 100)
	// Create some TMS transitions
	tms[10] = 0xFF
	tms[50] = 0xFF

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.buildSequences(tms, tdi, 800)
	}
}
