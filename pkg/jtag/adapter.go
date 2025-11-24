package jtag

import (
	"errors"
	"fmt"
)

// AdapterInfo describes capabilities reported by a JTAG adapter implementation.
type AdapterInfo struct {
	Name         string
	Vendor       string
	Model        string
	SerialNumber string
	Firmware     string
	MinFrequency int // Hertz
	MaxFrequency int // Hertz
	SupportsSRST bool
	SupportsTRST bool
	Notes        string
}

// Adapter abstracts a physical or virtual JTAG Test Access Port adapter.
type Adapter interface {
	Info() (AdapterInfo, error)
	ShiftIR(tms, tdi []byte, bits int) (tdo []byte, err error)
	ShiftDR(tms, tdi []byte, bits int) (tdo []byte, err error)
	ResetTAP(hard bool) error
	SetSpeed(hz int) error
}

// ErrNotImplemented lets backends signal that a requested capability is not yet
// available without relying on fmt.Errorf each time.
var ErrNotImplemented = errors.New("jtag: not implemented")

// ValidateShiftBuffers ensures TMS/TDIs are present when bits exceed their
// lengths and returns the number of bytes required to accommodate the bit
// length.
func ValidateShiftBuffers(tms, tdi []byte, bits int) (int, error) {
	if bits <= 0 {
		return 0, fmt.Errorf("jtag: bits must be positive, got %d", bits)
	}
	required := (bits + 7) / 8
	if len(tms) > 0 && len(tms) < required {
		return 0, fmt.Errorf("jtag: tms buffer too short, need %d bytes", required)
	}
	if len(tdi) > 0 && len(tdi) < required {
		return 0, fmt.Errorf("jtag: tdi buffer too short, need %d bytes", required)
	}
	return required, nil
}
