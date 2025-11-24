package jtag

// NewPicoProbeAdapter is a placeholder for the PicoProbe-backed adapter. It
// currently returns ErrNotImplemented so the UI can detect the lack of hardware
// support while the backend is under construction.
func NewPicoProbeAdapter(path string) (Adapter, error) {
	return nil, ErrNotImplemented
}
