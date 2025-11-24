package jtag

import (
	"fmt"
	"sync"
)

// CMSISDAPAdapter implements the Adapter interface for CMSIS-DAP probes
type CMSISDAPAdapter struct {
	transport *USBTransport
	protocol  *CMSISDAPProtocol

	info      AdapterInfo
	speedHz   int
	connected bool

	mu sync.Mutex // Protect concurrent access
}

// NewCMSISDAPAdapter creates a new CMSIS-DAP adapter
func NewCMSISDAPAdapter(vid, pid uint16) (*CMSISDAPAdapter, error) {
	// Create USB transport
	transport, err := NewUSBTransport(vid, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to open USB device: %w", err)
	}

	protocol := NewCMSISDAPProtocol(transport.GetPacketSize())

	adapter := &CMSISDAPAdapter{
		transport: transport,
		protocol:  protocol,
		speedHz:   1_000_000, // Default 1 MHz
	}

	// Query device information
	if err := adapter.queryInfo(); err != nil {
		transport.Close()
		return nil, fmt.Errorf("failed to query device info: %w", err)
	}

	// Connect to JTAG
	if err := adapter.connect(); err != nil {
		transport.Close()
		return nil, fmt.Errorf("failed to connect to JTAG: %w", err)
	}

	// Set default clock speed
	if err := adapter.SetSpeed(adapter.speedHz); err != nil {
		transport.Close()
		return nil, fmt.Errorf("failed to set default speed: %w", err)
	}

	return adapter, nil
}

// queryInfo retrieves device information from the probe
func (a *CMSISDAPAdapter) queryInfo() error {
	// Get vendor ID
	cmd := a.protocol.EncodeInfo(InfoVendorID)
	resp, err := a.transport.WriteRead(cmd)
	if err != nil {
		return err
	}
	vendor, _ := a.protocol.DecodeInfo(resp)

	// Get product ID
	cmd = a.protocol.EncodeInfo(InfoProductID)
	resp, _ = a.transport.WriteRead(cmd)
	product, _ := a.protocol.DecodeInfo(resp)

	// Get serial number
	cmd = a.protocol.EncodeInfo(InfoSerialNum)
	resp, _ = a.transport.WriteRead(cmd)
	serial, _ := a.protocol.DecodeInfo(resp)

	// Get firmware version
	cmd = a.protocol.EncodeInfo(InfoFirmwareVer)
	resp, _ = a.transport.WriteRead(cmd)
	firmware, _ := a.protocol.DecodeInfo(resp)

	a.info = AdapterInfo{
		Name:         "CMSIS-DAP Probe",
		Vendor:       vendor,
		Model:        product,
		SerialNumber: serial,
		Firmware:     firmware,
		MinFrequency: 1000,       // 1 kHz
		MaxFrequency: 10_000_000, // 10 MHz (typical for CMSIS-DAP)
		SupportsSRST: true,
		SupportsTRST: true,
	}

	return nil
}

// connect establishes JTAG connection
func (a *CMSISDAPAdapter) connect() error {
	cmd := a.protocol.EncodeConnect(PortJTAG)
	resp, err := a.transport.WriteRead(cmd)
	if err != nil {
		return err
	}

	port, err := a.protocol.DecodeConnect(resp)
	if err != nil {
		return err
	}

	if port != PortJTAG {
		return fmt.Errorf("failed to connect to JTAG (got port %d)", port)
	}

	a.connected = true
	return nil
}

// Info returns adapter capabilities
func (a *CMSISDAPAdapter) Info() (AdapterInfo, error) {
	return a.info, nil
}

// ShiftIR shifts data into the instruction register
func (a *CMSISDAPAdapter) ShiftIR(tms, tdi []byte, bits int) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, err := ValidateShiftBuffers(tms, tdi, bits); err != nil {
		return nil, err
	}

	return a.shiftRegister(tms, tdi, bits)
}

// ShiftDR shifts data into the data register
func (a *CMSISDAPAdapter) ShiftDR(tms, tdi []byte, bits int) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, err := ValidateShiftBuffers(tms, tdi, bits); err != nil {
		return nil, err
	}

	return a.shiftRegister(tms, tdi, bits)
}

// shiftRegister performs the actual JTAG shift operation
// This handles the complexity of splitting per-bit TMS into CMSIS-DAP sequences
func (a *CMSISDAPAdapter) shiftRegister(tms, tdi []byte, bits int) ([]byte, error) {
	// Build sequences - split by TMS changes
	sequences := a.buildSequences(tms, tdi, bits)

	// Encode and send command
	cmd := a.protocol.EncodeJTAGSequence(sequences)
	resp, err := a.transport.WriteRead(cmd)
	if err != nil {
		return nil, fmt.Errorf("shift failed: %w", err)
	}

	// Decode response and extract TDO
	tdoSeqs, err := a.protocol.DecodeJTAGSequence(resp, sequences)
	if err != nil {
		return nil, err
	}

	// Combine TDO sequences into single byte array
	tdo := make([]byte, (bits+7)/8)
	bitPos := 0

	for _, seqTDO := range tdoSeqs {
		// Copy bits from this sequence's TDO
		for i := 0; i < len(seqTDO); i++ {
			byteIdx := bitPos / 8
			bitIdx := bitPos % 8

			for bit := 0; bit < 8 && bitPos < bits; bit++ {
				if (seqTDO[i] & (1 << bit)) != 0 {
					tdo[byteIdx] |= (1 << bitIdx)
				}
				bitPos++
				if bitPos%8 == 0 {
					byteIdx++
					bitIdx = 0
				} else {
					bitIdx++
				}
			}
		}
	}

	return tdo, nil
}

// buildSequences splits a shift operation into CMSIS-DAP sequences
// CMSIS-DAP uses a single TMS value per sequence, but our Adapter interface
// expects per-bit TMS control, so we need to split whenever TMS changes
func (a *CMSISDAPAdapter) buildSequences(tms, tdi []byte, bits int) []JTAGSequence {
	sequences := make([]JTAGSequence, 0)

	if len(tms) == 0 {
		// No TMS provided - use single sequence with TMS=0
		seqTDI := make([]byte, (bits+7)/8)
		copy(seqTDI, tdi)

		// Split into 64-bit chunks if needed
		for bitPos := 0; bitPos < bits; bitPos += 64 {
			seqBits := bits - bitPos
			if seqBits > 64 {
				seqBits = 64
			}
			seqBytes := (seqBits + 7) / 8
			chunkTDI := make([]byte, seqBytes)
			copy(chunkTDI, seqTDI[bitPos/8:])

			seq := NewJTAGSequence(seqBits, false, true, chunkTDI)
			sequences = append(sequences, seq)
		}
		return sequences
	}

	// Build sequences based on TMS transitions
	bitPos := 0
	for bitPos < bits {
		// Determine TMS value for this segment
		byteIdx := bitPos / 8
		bitIdx := bitPos % 8
		currentTMS := (tms[byteIdx] & (1 << bitIdx)) != 0

		// Find how many consecutive bits have the same TMS
		seqBits := 0
		for bitPos+seqBits < bits && seqBits < 64 {
			idx := (bitPos + seqBits) / 8
			bit := (bitPos + seqBits) % 8
			tmsVal := (tms[idx] & (1 << bit)) != 0
			if tmsVal != currentTMS {
				break
			}
			seqBits++
		}

		// Extract TDI data for this sequence
		seqBytes := (seqBits + 7) / 8
		seqTDI := make([]byte, seqBytes)

		startByte := bitPos / 8
		startBit := bitPos % 8

		if startBit == 0 {
			// Aligned - simple copy
			copy(seqTDI, tdi[startByte:])
		} else {
			// Misaligned - need to shift bits
			for i := 0; i < seqBytes; i++ {
				if startByte+i < len(tdi) {
					seqTDI[i] = tdi[startByte+i] >> startBit
					if startByte+i+1 < len(tdi) {
						seqTDI[i] |= tdi[startByte+i+1] << (8 - startBit)
					}
				}
			}

			// Mask off extra bits in last byte
			if seqBits%8 != 0 {
				lastByte := seqBytes - 1
				mask := byte((1 << (seqBits % 8)) - 1)
				seqTDI[lastByte] &= mask
			}
		}

		// Create sequence
		seq := NewJTAGSequence(seqBits, currentTMS, true, seqTDI)
		sequences = append(sequences, seq)

		bitPos += seqBits
	}

	return sequences
}

// ResetTAP resets the JTAG TAP state machine
func (a *CMSISDAPAdapter) ResetTAP(hard bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if hard {
		// Use DAP_ResetTarget command
		cmd := a.protocol.EncodeResetTarget()
		resp, err := a.transport.WriteRead(cmd)
		if err != nil {
			return fmt.Errorf("hard reset failed: %w", err)
		}
		return a.protocol.DecodeResetTarget(resp)
	}

	// Soft reset via TMS sequence: 5+ clocks with TMS=1
	tdi := []byte{0x00}
	seq := NewJTAGSequence(5, true, false, tdi)

	cmd := a.protocol.EncodeJTAGSequence([]JTAGSequence{seq})
	resp, err := a.transport.WriteRead(cmd)
	if err != nil {
		return fmt.Errorf("TAP reset failed: %w", err)
	}

	_, err = a.protocol.DecodeJTAGSequence(resp, []JTAGSequence{seq})
	return err
}

// SetSpeed sets the TCK frequency
func (a *CMSISDAPAdapter) SetSpeed(hz int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if hz < a.info.MinFrequency || hz > a.info.MaxFrequency {
		return fmt.Errorf("frequency %d Hz out of range [%d, %d]",
			hz, a.info.MinFrequency, a.info.MaxFrequency)
	}

	cmd := a.protocol.EncodeSetClock(uint32(hz))
	resp, err := a.transport.WriteRead(cmd)
	if err != nil {
		return fmt.Errorf("set speed failed: %w", err)
	}

	if err := a.protocol.DecodeSetClock(resp); err != nil {
		return err
	}

	a.speedHz = hz
	return nil
}

// Close disconnects and releases resources
func (a *CMSISDAPAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.connected {
		// Send disconnect command
		cmd := a.protocol.EncodeDisconnect()
		a.transport.WriteRead(cmd)
		a.connected = false
	}

	return a.transport.Close()
}

// ConfigureJTAGChain configures the JTAG chain with IR lengths
// This is a CMSIS-DAP specific extension not part of the Adapter interface
func (a *CMSISDAPAdapter) ConfigureJTAGChain(irLengths []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	cmd := a.protocol.EncodeJTAGConfigure(irLengths)
	resp, err := a.transport.WriteRead(cmd)
	if err != nil {
		return fmt.Errorf("configure chain failed: %w", err)
	}

	return a.protocol.DecodeJTAGConfigure(resp)
}

// ReadIDCODE reads the IDCODE from a specific device in the chain
// This is a CMSIS-DAP specific extension not part of the Adapter interface
func (a *CMSISDAPAdapter) ReadIDCODE(deviceIndex byte) (uint32, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	cmd := a.protocol.EncodeJTAGIDCODE(deviceIndex)
	resp, err := a.transport.WriteRead(cmd)
	if err != nil {
		return 0, fmt.Errorf("read IDCODE failed: %w", err)
	}

	return a.protocol.DecodeJTAGIDCODE(resp)
}
