package jtag

import "fmt"

// PicoAdapter is a placeholder for the upcoming Pico/JTAG-over-CDC backend. It
// satisfies the Adapter interface so higher level code can compile and be
// wired up later once USB transport details are nailed down.
type PicoAdapter struct {
	PortPath string
	speedHz  int
	info     AdapterInfo
}

// NewPicoAdapter creates a stub instance associated with the provided serial
// device path. The actual USB protocol is not implemented yet.
func NewPicoAdapter(port string) *PicoAdapter {
	return &PicoAdapter{
		PortPath: port,
		info: AdapterInfo{
			Name:         "Pico JTAG Stub",
			Vendor:       "Pico",
			Model:        "PiCi-2",
			SerialNumber: "N/A",
			Firmware:     "unknown",
			MinFrequency: 1_000,
			MaxFrequency: 25_000_000,
			SupportsSRST: true,
			SupportsTRST: true,
			Notes:        "transport not yet implemented",
		},
	}
}

func (p *PicoAdapter) Info() (AdapterInfo, error) {
	return p.info, nil
}

func (p *PicoAdapter) ShiftIR(_, _ []byte, _ int) ([]byte, error) {
	return nil, ErrNotImplemented
}

func (p *PicoAdapter) ShiftDR(_, _ []byte, _ int) ([]byte, error) {
	return nil, ErrNotImplemented
}

func (p *PicoAdapter) ResetTAP(_ bool) error {
	return ErrNotImplemented
}

func (p *PicoAdapter) SetSpeed(hz int) error {
	if hz <= 0 {
		return fmt.Errorf("jtag: invalid speed %dHz", hz)
	}
	p.speedHz = hz
	return ErrNotImplemented
}
