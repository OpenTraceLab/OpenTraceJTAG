package chain

import (
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/tap"
)

// TogglePin now builds a single-operation batch so multiple callers can share
// the batching machinery.
func (c *Chain) TogglePin(deviceName, pin string, high bool) error {
	batch := c.NewBatch()
	if err := batch.SetPin(deviceName, pin, high); err != nil {
		return err
	}
	_, err := batch.Execute()
	return err
}

// BatchResult maps device names to the bits captured while executing a batch.
// Each slice is ordered LSB-first (cell 0 at index 0).
type BatchResult map[string][]bool

// Batch accumulates multiple pin toggles and capture requests before executing a
// single set of IR/DR operations on the chain.
type Batch struct {
	chain *Chain
	ops   map[*Device]*batchDevice
}

type batchDevice struct {
	pins    map[string]bool
	capture bool
}

// NewBatch returns a fresh batch bound to the chain instance.
func (c *Chain) NewBatch() *Batch {
	return &Batch{
		chain: c,
		ops:   make(map[*Device]*batchDevice),
	}
}

// SetPin queues a request to drive the given pin high or low when Execute is
// called. Multiple calls override the previous level for the same pin.
func (b *Batch) SetPin(deviceName, pin string, high bool) error {
	dev, err := b.lookup(deviceName)
	if err != nil {
		return err
	}
	op := b.ensureDevice(dev)
	if op.pins == nil {
		op.pins = make(map[string]bool)
	}
	op.pins[pin] = high
	return nil
}

// Capture registers that the batch should read back the boundary register for
// the specified device. The returned bits are available via Execute.
func (b *Batch) Capture(deviceName string) error {
	dev, err := b.lookup(deviceName)
	if err != nil {
		return err
	}
	b.ensureDevice(dev).capture = true
	return nil
}

// Execute programs the appropriate instructions, applies all queued pin changes,
// and returns any captured boundary data (if requested via Capture).
func (b *Batch) Execute() (BatchResult, error) {
	if len(b.ops) == 0 {
		return nil, fmt.Errorf("chain: batch has no operations")
	}

	instMap := make(map[*Device]string)
	vectors := make(map[*Device][]bool)
	captures := make(map[*Device]bool)

	for dev, op := range b.ops {
		if len(op.pins) > 0 {
			instMap[dev] = "EXTEST"
			base, err := dev.boundaryBaseVector()
			if err != nil {
				return nil, err
			}
			for pin, level := range op.pins {
				if err := dev.applyPin(base, pin, level); err != nil {
					return nil, err
				}
			}
			vectors[dev] = base
		} else if op.capture {
			instMap[dev] = "SAMPLE"
			length, err := dev.boundaryLength()
			if err != nil {
				return nil, err
			}
			vectors[dev] = make([]bool, length)
		}

		if op.capture {
			captures[dev] = true
		}
	}

	if err := b.chain.programInstructions(instMap); err != nil {
		return nil, err
	}

	stream, segments, err := b.chain.composeDRStreamMulti(vectors, captures)
	if err != nil {
		return nil, err
	}

	tdo, err := b.chain.shiftDR(stream)
	if err != nil {
		return nil, err
	}

	results := make(BatchResult)
	offset := 0
	for _, seg := range segments {
		if seg.capture {
			bits := append([]bool(nil), tdo[offset:offset+seg.length]...)
			results[seg.device.Name()] = bits
		}
		offset += seg.length
	}
	return results, nil
}

func (b *Batch) lookup(name string) (*Device, error) {
	dev, ok := b.chain.DeviceByName(name)
	if !ok {
		return nil, fmt.Errorf("chain: device %s not found", name)
	}
	return dev, nil
}

func (b *Batch) ensureDevice(dev *Device) *batchDevice {
	if entry, ok := b.ops[dev]; ok {
		return entry
	}
	entry := &batchDevice{}
	b.ops[dev] = entry
	return entry
}

func (c *Chain) programInstructions(mapping map[*Device]string) error {
	var stream []bool
	for _, dev := range c.devices {
		name := "BYPASS"
		if instr, ok := mapping[dev]; ok {
			name = instr
		}
		bits, err := dev.instructionBits(name)
		if err != nil {
			return err
		}
		stream = append(stream, bits...)
	}
	if len(stream) == 0 {
		return fmt.Errorf("chain: no devices to program")
	}
	tms := shiftPattern(len(stream))
	if err := c.xport.gotoState(tap.StateShiftIR); err != nil {
		return err
	}
	if _, err := c.xport.shiftIR(tms, stream); err != nil {
		return err
	}
	return c.xport.gotoState(tap.StateRunTestIdle)
}

type drSegment struct {
	device  *Device
	length  int
	capture bool
}

func (c *Chain) composeDRStreamMulti(vectors map[*Device][]bool, captures map[*Device]bool) ([]bool, []drSegment, error) {
	total := 0
	stream := make([]bool, 0)
	segments := make([]drSegment, 0, len(c.devices))

	for _, dev := range c.devices {
		vec, ok := vectors[dev]
		if !ok {
			vec = []bool{false}
		}
		stream = append(stream, vec...)
		total += len(vec)
		segments = append(segments, drSegment{
			device:  dev,
			length:  len(vec),
			capture: captures != nil && captures[dev],
		})
	}

	return stream, segments, nil
}

func (c *Chain) shiftDR(bits []bool) ([]bool, error) {
	if len(bits) == 0 {
		return nil, fmt.Errorf("chain: empty DR pattern")
	}
	tms := shiftPattern(len(bits))
	if err := c.xport.gotoState(tap.StateShiftDR); err != nil {
		return nil, err
	}
	tdo, err := c.xport.shiftDR(tms, bits)
	if err != nil {
		return nil, err
	}
	if err := c.xport.gotoState(tap.StateRunTestIdle); err != nil {
		return nil, err
	}
	return bytesToBools(tdo, len(bits)), nil
}

func shiftPattern(length int) []bool {
	pattern := make([]bool, length)
	if length > 0 {
		pattern[length-1] = true
	}
	return pattern
}
