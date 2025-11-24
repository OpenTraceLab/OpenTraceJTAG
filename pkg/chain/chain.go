package chain

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/tap"
)

// Controller orchestrates JTAG chain discovery and high-level operations.
type Controller struct {
	adapter jtag.Adapter
	repo    Repository
}

// NewController wires a JTAG adapter with a BSDL repository.
func NewController(adapter jtag.Adapter, repo Repository) *Controller {
	return &Controller{
		adapter: adapter,
		repo:    repo,
	}
}

// Chain represents the discovered devices and provides helper queries.
type Chain struct {
	devices []*Device
	xport   *transport
}

// Devices returns a copy of the known devices.
func (c *Chain) Devices() []*Device {
	out := make([]*Device, len(c.devices))
	copy(out, c.devices)
	return out
}

// DeviceByName returns the first device with the provided entity name.
func (c *Chain) DeviceByName(name string) (*Device, bool) {
	for _, dev := range c.devices {
		if dev.Name() == name {
			return dev, true
		}
	}
	return nil, false
}

// ProgramInstructions loads the specified instruction into each device's IR.
// Devices not in the mapping are programmed with BYPASS.
func (c *Chain) ProgramInstructions(mapping map[*Device]string) error {
	return c.programInstructions(mapping)
}

// ShiftDRBits shifts the provided bit pattern through the DR chain and returns
// the captured TDO bits. The chain must already be in the correct instruction state.
func (c *Chain) ShiftDRBits(bits []bool) ([]bool, error) {
	return c.shiftDR(bits)
}

// Device aggregates useful BSDL-derived metadata.
type Device struct {
	Position int
	IDCode   uint32
	File     *bsdl.BSDLFile
	Info     *bsdl.DeviceInfo

	boundaryOnce  sync.Once
	boundaryCells []bsdl.BoundaryCell
	boundaryErr   error
	cellByNumber  map[int]*bsdl.BoundaryCell
	cellsByPort   map[string][]*bsdl.BoundaryCell
}

// Name returns the entity name.
func (d *Device) Name() string {
	if d.File != nil && d.File.Entity != nil {
		return d.File.Entity.Name
	}
	return ""
}

// Instructions exposes the decoded instruction table.
func (d *Device) Instructions() []bsdl.Instruction {
	if d.File == nil || d.File.Entity == nil {
		return nil
	}
	return d.File.Entity.GetInstructionOpcodes()
}

// PinMap returns the mapping from signal name to package pin.
func (d *Device) PinMap() map[string]string {
	if d.File == nil || d.File.Entity == nil {
		return nil
	}
	return d.File.Entity.GetPinMap()
}

// ExtestOpcode returns the EXTEST instruction bits for this device.
func (d *Device) ExtestOpcode() ([]bool, error) {
	return d.instructionBits("EXTEST")
}

// BypassOpcode returns the BYPASS instruction bits for this device.
func (d *Device) BypassOpcode() ([]bool, error) {
	return d.instructionBits("BYPASS")
}

// IOPins returns a list of IO pin names (package pins) for this device.
// This excludes power pins (VCC, GND), NC (no connect), and internal pins.
func (d *Device) IOPins() ([]string, error) {
	cells, err := d.boundaryData()
	if err != nil {
		return nil, err
	}

	pinMap := d.PinMap()
	seen := make(map[string]bool)
	var pins []string

	for i := range cells {
		cell := &cells[i]

		// Skip internal cells (port == "*")
		if cell.Port == "*" {
			continue
		}

		// Get the package pin name
		packagePin, ok := pinMap[cell.Port]
		if !ok {
			// If no pin map entry, use the port name directly
			packagePin = cell.Port
		}

		// Skip if we've already seen this pin
		if seen[packagePin] {
			continue
		}

		// Filter out power and NC pins by name heuristics
		upper := strings.ToUpper(packagePin)
		if strings.Contains(upper, "VCC") ||
			strings.Contains(upper, "VDD") ||
			strings.Contains(upper, "VSS") ||
			strings.Contains(upper, "GND") ||
			strings.Contains(upper, "NC") {
			continue
		}

		seen[packagePin] = true
		pins = append(pins, packagePin)
	}

	return pins, nil
}

// BoundaryCells returns the boundary scan cells for this device.
// This exposes the internal boundaryData for use by the BSR package.
func (d *Device) BoundaryCells() ([]bsdl.BoundaryCell, error) {
	return d.boundaryData()
}

func (d *Device) boundaryData() ([]bsdl.BoundaryCell, error) {
	if d.File == nil || d.File.Entity == nil {
		return nil, fmt.Errorf("chain: device %s missing BSDL data", d.Name())
	}
	d.boundaryOnce.Do(func() {
		cells, err := d.File.Entity.GetBoundaryCells()
		if err != nil {
			d.boundaryErr = err
			return
		}
		d.boundaryCells = cells
		d.cellByNumber = make(map[int]*bsdl.BoundaryCell, len(cells))
		d.cellsByPort = make(map[string][]*bsdl.BoundaryCell)
		for i := range d.boundaryCells {
			cell := &d.boundaryCells[i]
			d.cellByNumber[cell.Number] = cell
			key := strings.ToUpper(cell.Port)
			d.cellsByPort[key] = append(d.cellsByPort[key], cell)
		}
	})
	return d.boundaryCells, d.boundaryErr
}

func (d *Device) boundaryLength() (int, error) {
	if d.Info != nil && d.Info.BoundaryLength > 0 {
		return d.Info.BoundaryLength, nil
	}
	cells, err := d.boundaryData()
	if err != nil {
		return 0, err
	}
	return len(cells), nil
}

func (d *Device) boundaryBaseVector() ([]bool, error) {
	cells, err := d.boundaryData()
	if err != nil {
		return nil, err
	}
	length, err := d.boundaryLength()
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, fmt.Errorf("chain: device %s has zero-length boundary register", d.Name())
	}
	bits := make([]bool, length)
	for i := range cells {
		cell := &cells[i]
		if cell.Number >= length {
			return nil, fmt.Errorf("chain: boundary cell %d exceeds length %d", cell.Number, length)
		}
		switch strings.ToUpper(strings.TrimSpace(cell.Safe)) {
		case "1":
			bits[cell.Number] = true
		case "0":
			bits[cell.Number] = false
		}
	}
	return bits, nil
}

func (d *Device) applyPin(bits []bool, pin string, high bool) error {
	output, control, err := d.outputCell(pin)
	if err != nil {
		return err
	}
	if output.Number >= len(bits) {
		return fmt.Errorf("chain: pin %s exceeds boundary length", pin)
	}
	bits[output.Number] = high
	if control != nil {
		disable := output.Disable
		if disable == -1 {
			disable = control.Disable
		}
		enable := disable == 0
		if disable == -1 {
			enable = true
		}
		if control.Number >= len(bits) {
			return fmt.Errorf("chain: control cell %d exceeds vector length", control.Number)
		}
		bits[control.Number] = enable
	}
	return nil
}

func (d *Device) boundaryVectorForPin(pin string, high bool) ([]bool, error) {
	bits, err := d.boundaryBaseVector()
	if err != nil {
		return nil, err
	}
	if err := d.applyPin(bits, pin, high); err != nil {
		return nil, err
	}
	return bits, nil
}

func (d *Device) outputCell(pin string) (*bsdl.BoundaryCell, *bsdl.BoundaryCell, error) {
	if _, err := d.boundaryData(); err != nil {
		return nil, nil, err
	}
	key := strings.ToUpper(pin)
	if entries, ok := d.cellsByPort[key]; ok {
		for _, cell := range entries {
			if strings.HasPrefix(strings.ToUpper(cell.Function), "OUTPUT") {
				var control *bsdl.BoundaryCell
				if cell.Control >= 0 {
					control = d.cellByNumber[cell.Control]
				}
				return cell, control, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("chain: no output cell for pin %s on %s", pin, d.Name())
}

func (d *Device) instructionBits(name string) ([]bool, error) {
	if d.Info == nil {
		return nil, fmt.Errorf("chain: device %s missing device info", d.Name())
	}
	wanted := strings.ToUpper(name)
	for _, instr := range d.Instructions() {
		if strings.ToUpper(instr.Name) == wanted {
			return opcodeToBits(instr.Opcode, d.Info.InstructionLength)
		}
	}
	return nil, fmt.Errorf("chain: instruction %s not found on %s", name, d.Name())
}

// Discover chains the adapter, TAP FSM, and repository together to produce a
// fully described chain. For now the caller must pass the expected device count.
func (c *Controller) Discover(deviceCount int) (*Chain, error) {
	if deviceCount <= 0 {
		return nil, fmt.Errorf("chain: deviceCount must be positive")
	}
	if c.adapter == nil {
		return nil, fmt.Errorf("chain: adapter is nil")
	}
	if c.repo == nil {
		return nil, fmt.Errorf("chain: repository is nil")
	}

	xport := newTransport(c.adapter)
	session := &session{
		transport: xport,
		repo:      c.repo,
	}

	if err := xport.reset(); err != nil {
		return nil, err
	}
	if err := xport.gotoState(tap.StateShiftDR); err != nil {
		return nil, err
	}

	ids, err := session.readIDCodes(deviceCount)
	if err != nil {
		return nil, err
	}

	devices := make([]*Device, 0, deviceCount)
	for idx, id := range ids {
		file, err := c.repo.Lookup(id)
		if err != nil {
			return nil, err
		}
		var info *bsdl.DeviceInfo
		if mr, ok := c.repo.(*MemoryRepository); ok {
			info = mr.DeviceInfo(id)
		}
		if info == nil && file != nil && file.Entity != nil {
			info = file.Entity.GetDeviceInfo()
		}

		devices = append(devices, &Device{
			Position: idx,
			IDCode:   id,
			File:     file,
			Info:     info,
		})
	}

	return &Chain{
		devices: devices,
		xport:   xport,
	}, nil
}

type session struct {
	transport *transport
	repo      Repository
}

func (s *session) readIDCodes(deviceCount int) ([]uint32, error) {
	bits := deviceCount * 32
	if bits == 0 {
		return nil, fmt.Errorf("chain: invalid device count")
	}

	tms := make([]bool, bits)
	if bits > 0 {
		tms[bits-1] = true // exit Shift-DR after final bit
	}

	tdo, err := s.transport.shiftDR(tms, nil)
	if err != nil {
		return nil, err
	}

	if err := s.transport.gotoState(tap.StateRunTestIdle); err != nil {
		return nil, err
	}

	bitsOut := bytesToBools(tdo, bits)
	out := make([]uint32, deviceCount)
	for i := 0; i < deviceCount; i++ {
		start := i * 32
		out[i] = bitsToUint32(bitsOut[start : start+32])
	}
	return out, nil
}

type transport struct {
	adapter jtag.Adapter
	tap     *tap.StateMachine
}

func newTransport(adapter jtag.Adapter) *transport {
	return &transport{adapter: adapter, tap: tap.NewStateMachine()}
}

func (t *transport) reset() error {
	if err := t.adapter.ResetTAP(true); err != nil && !errors.Is(err, jtag.ErrNotImplemented) {
		return err
	}
	seq := t.tap.Reset()
	return t.applySequence(seq, domainDR)
}

func (t *transport) gotoState(target tap.State) error {
	seq, err := t.tap.GoTo(target)
	if err != nil {
		return err
	}
	if len(seq.TMS) == 0 {
		return nil
	}
	return t.applySequence(seq, domainFromState(seq.States[0]))
}

func (t *transport) applySequence(seq tap.Sequence, domain shiftDomain) error {
	if len(seq.TMS) == 0 {
		return nil
	}
	_, err := t.dispatch(domain, seq.TMS, nil)
	return err
}

func (t *transport) shiftDR(tms, tdi []bool) ([]byte, error) {
	for _, bit := range tms {
		t.tap.Clock(bit)
	}
	return t.dispatch(domainDR, tms, tdi)
}

func (t *transport) shiftIR(tms, tdi []bool) ([]byte, error) {
	for _, bit := range tms {
		t.tap.Clock(bit)
	}
	return t.dispatch(domainIR, tms, tdi)
}

func (t *transport) dispatch(domain shiftDomain, tms []bool, tdi []bool) ([]byte, error) {
	if len(tms) == 0 {
		return nil, nil
	}
	bits := len(tms)
	tmsBytes := boolsToBytes(tms)
	var tdiBytes []byte
	if len(tdi) == 0 {
		tdiBytes = make([]byte, len(tmsBytes))
	} else {
		tdiBytes = boolsToBytes(tdi)
	}
	switch domain {
	case domainIR:
		return t.adapter.ShiftIR(tmsBytes, tdiBytes, bits)
	default:
		return t.adapter.ShiftDR(tmsBytes, tdiBytes, bits)
	}
}

type shiftDomain uint8

const (
	domainDR shiftDomain = iota
	domainIR
)

func domainFromState(state tap.State) shiftDomain {
	switch state {
	case tap.StateSelectIRScan,
		tap.StateCaptureIR,
		tap.StateShiftIR,
		tap.StateExit1IR,
		tap.StatePauseIR,
		tap.StateExit2IR,
		tap.StateUpdateIR:
		return domainIR
	default:
		return domainDR
	}
}
