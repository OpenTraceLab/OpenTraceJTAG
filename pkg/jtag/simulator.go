package jtag

import "fmt"

// ShiftRegion identifies whether a shift operation targets the instruction or
// data register.
type ShiftRegion uint8

const (
	ShiftRegionIR ShiftRegion = iota
	ShiftRegionDR
)

// ShiftHook allows the simulator to emulate device-specific TDO behavior.
type ShiftHook func(region ShiftRegion, tms, tdi []byte, bits int) ([]byte, error)

// ShiftOp captures the last shift invocation for inspection within tests.
type ShiftOp struct {
	Region ShiftRegion
	TMS    []byte
	TDI    []byte
	Bits   int
}

// SimAdapter is an in-memory adapter useful for unit tests. It records the last
// shift request and can optionally provide deterministic TDO data via OnShift.
type SimAdapter struct {
	InfoData AdapterInfo
	SpeedHz  int

	OnShift ShiftHook

	lastShift ShiftOp
	resets    int
	hardReset int
}

// NewSimAdapter constructs a simulator configured with the provided AdapterInfo.
func NewSimAdapter(info AdapterInfo) *SimAdapter {
	return &SimAdapter{InfoData: info}
}

// LastShift returns a copy of the most recent shift request.
func (s *SimAdapter) LastShift() ShiftOp {
	return ShiftOp{
		Region: s.lastShift.Region,
		TMS:    append([]byte(nil), s.lastShift.TMS...),
		TDI:    append([]byte(nil), s.lastShift.TDI...),
		Bits:   s.lastShift.Bits,
	}
}

// ResetCounts reports how many resets have been requested (soft as total,
// hardReset as subset).
func (s *SimAdapter) ResetCounts() (soft, hard int) {
	return s.resets, s.hardReset
}

func (s *SimAdapter) Info() (AdapterInfo, error) {
	return s.InfoData, nil
}

func (s *SimAdapter) ShiftIR(tms, tdi []byte, bits int) ([]byte, error) {
	return s.shift(ShiftRegionIR, tms, tdi, bits)
}

func (s *SimAdapter) ShiftDR(tms, tdi []byte, bits int) ([]byte, error) {
	return s.shift(ShiftRegionDR, tms, tdi, bits)
}

func (s *SimAdapter) ResetTAP(hard bool) error {
	s.resets++
	if hard {
		s.hardReset++
	}
	return nil
}

func (s *SimAdapter) SetSpeed(hz int) error {
	if hz <= 0 {
		return fmt.Errorf("jtag: invalid speed %dHz", hz)
	}
	s.SpeedHz = hz
	return nil
}

func (s *SimAdapter) shift(region ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
	if _, err := ValidateShiftBuffers(tms, tdi, bits); err != nil {
		return nil, err
	}

	s.lastShift = ShiftOp{
		Region: region,
		TMS:    append([]byte(nil), tms...),
		TDI:    append([]byte(nil), tdi...),
		Bits:   bits,
	}

	if s.OnShift != nil {
		return s.OnShift(region, tms, tdi, bits)
	}

	// Default: echo TDI to TDO to keep tests predictable.
	required := (bits + 7) / 8
	tdo := make([]byte, required)
	copy(tdo, tdi)
	return tdo, nil
}
