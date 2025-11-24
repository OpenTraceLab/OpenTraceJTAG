package jtag

import (
	"encoding/binary"
	"fmt"
)

// CMSIS-DAP Command IDs
const (
	CmdInfo          = 0x00
	CmdHostStatus    = 0x01
	CmdConnect       = 0x02
	CmdDisconnect    = 0x03
	CmdResetTarget   = 0x0A
	CmdSWJClock      = 0x11
	CmdSWJSequence   = 0x12
	CmdJTAGSequence  = 0x14
	CmdJTAGConfigure = 0x15
	CmdJTAGIDCODE    = 0x16
)

// DAP_Info Info IDs
const (
	InfoVendorID     = 0x01
	InfoProductID    = 0x02
	InfoSerialNum    = 0x03
	InfoFirmwareVer  = 0x04
	InfoCapabilities = 0xF0
	InfoPacketCount  = 0xFE
	InfoPacketSize   = 0xFF
)

// Connection ports
const (
	PortDefault = 0
	PortSWD     = 1
	PortJTAG    = 2
)

// Status codes
const (
	StatusOK    = 0x00
	StatusError = 0xFF
)

// JTAG Sequence info flags
const (
	JTAGSeqTCKMask = 0x3F // Bits [5:0] = TCK count (0-63, where 0 means 64)
	JTAGSeqTMS     = 0x40 // Bit [6] = TMS value
	JTAGSeqTDO     = 0x80 // Bit [7] = Capture TDO
)

// CMSISDAPProtocol handles encoding/decoding of CMSIS-DAP commands
type CMSISDAPProtocol struct {
	PacketSize int
}

// NewCMSISDAPProtocol creates a new protocol handler
func NewCMSISDAPProtocol(packetSize int) *CMSISDAPProtocol {
	return &CMSISDAPProtocol{
		PacketSize: packetSize,
	}
}

// EncodeInfo builds a DAP_Info command
func (p *CMSISDAPProtocol) EncodeInfo(infoID byte) []byte {
	return []byte{CmdInfo, infoID}
}

// DecodeInfo parses a DAP_Info response
func (p *CMSISDAPProtocol) DecodeInfo(resp []byte) (string, error) {
	if len(resp) < 2 {
		return "", fmt.Errorf("response too short")
	}
	if resp[0] != CmdInfo {
		return "", fmt.Errorf("invalid command ID: 0x%02X", resp[0])
	}

	length := int(resp[1])
	if len(resp) < 2+length {
		return "", fmt.Errorf("incomplete info string")
	}

	return string(resp[2 : 2+length]), nil
}

// EncodeConnect builds a DAP_Connect command
func (p *CMSISDAPProtocol) EncodeConnect(port byte) []byte {
	return []byte{CmdConnect, port}
}

// DecodeConnect parses a DAP_Connect response
func (p *CMSISDAPProtocol) DecodeConnect(resp []byte) (byte, error) {
	if len(resp) < 2 {
		return 0, fmt.Errorf("response too short")
	}
	if resp[0] != CmdConnect {
		return 0, fmt.Errorf("invalid command ID")
	}
	if resp[1] == 0 {
		return 0, fmt.Errorf("connection failed")
	}
	return resp[1], nil
}

// EncodeDisconnect builds a DAP_Disconnect command
func (p *CMSISDAPProtocol) EncodeDisconnect() []byte {
	return []byte{CmdDisconnect}
}

// DecodeDisconnect parses a DAP_Disconnect response
func (p *CMSISDAPProtocol) DecodeDisconnect(resp []byte) error {
	if len(resp) < 2 {
		return fmt.Errorf("response too short")
	}
	if resp[0] != CmdDisconnect {
		return fmt.Errorf("invalid command ID")
	}
	if resp[1] != StatusOK {
		return fmt.Errorf("disconnect failed")
	}
	return nil
}

// EncodeJTAGConfigure builds a DAP_JTAG_Configure command
func (p *CMSISDAPProtocol) EncodeJTAGConfigure(irLengths []byte) []byte {
	cmd := make([]byte, 2+len(irLengths))
	cmd[0] = CmdJTAGConfigure
	cmd[1] = byte(len(irLengths))
	copy(cmd[2:], irLengths)
	return cmd
}

// DecodeJTAGConfigure parses response
func (p *CMSISDAPProtocol) DecodeJTAGConfigure(resp []byte) error {
	if len(resp) < 2 {
		return fmt.Errorf("response too short")
	}
	if resp[0] != CmdJTAGConfigure {
		return fmt.Errorf("invalid command ID")
	}
	if resp[1] != StatusOK {
		return fmt.Errorf("configure failed")
	}
	return nil
}

// EncodeJTAGIDCODE builds a DAP_JTAG_IDCODE command
func (p *CMSISDAPProtocol) EncodeJTAGIDCODE(deviceIndex byte) []byte {
	return []byte{CmdJTAGIDCODE, deviceIndex}
}

// DecodeJTAGIDCODE parses response and extracts IDCODE
func (p *CMSISDAPProtocol) DecodeJTAGIDCODE(resp []byte) (uint32, error) {
	if len(resp) < 6 {
		return 0, fmt.Errorf("response too short")
	}
	if resp[0] != CmdJTAGIDCODE {
		return 0, fmt.Errorf("invalid command ID")
	}
	if resp[1] != StatusOK {
		return 0, fmt.Errorf("IDCODE read failed")
	}

	// IDCODE is 32-bit little-endian
	idcode := binary.LittleEndian.Uint32(resp[2:6])
	return idcode, nil
}

// JTAGSequence represents one JTAG shift operation
type JTAGSequence struct {
	Info byte   // Sequence info byte (TCK count, TMS, TDO capture)
	TDI  []byte // TDI data to shift
}

// NewJTAGSequence creates a sequence descriptor
func NewJTAGSequence(tckCount int, tms bool, captureTDO bool, tdi []byte) JTAGSequence {
	// Build info byte
	info := byte(tckCount & JTAGSeqTCKMask)
	if tms {
		info |= JTAGSeqTMS
	}
	if captureTDO {
		info |= JTAGSeqTDO
	}

	return JTAGSequence{
		Info: info,
		TDI:  tdi,
	}
}

// TCKCount returns the number of TCK clocks in this sequence
func (seq *JTAGSequence) TCKCount() int {
	count := int(seq.Info & JTAGSeqTCKMask)
	if count == 0 {
		return 64 // 0 means 64
	}
	return count
}

// TMS returns the TMS value for this sequence
func (seq *JTAGSequence) TMS() bool {
	return (seq.Info & JTAGSeqTMS) != 0
}

// CaptureTDO returns whether TDO should be captured
func (seq *JTAGSequence) CaptureTDO() bool {
	return (seq.Info & JTAGSeqTDO) != 0
}

// EncodeJTAGSequence builds a DAP_JTAG_Sequence command
// Each sequence is: [info_byte][tdi_data...]
func (p *CMSISDAPProtocol) EncodeJTAGSequence(sequences []JTAGSequence) []byte {
	// Calculate total size
	size := 2 // cmd + count
	for _, seq := range sequences {
		size += 1 + len(seq.TDI) // info + data
	}

	cmd := make([]byte, size)
	cmd[0] = CmdJTAGSequence
	cmd[1] = byte(len(sequences))

	offset := 2
	for _, seq := range sequences {
		cmd[offset] = seq.Info
		offset++
		copy(cmd[offset:], seq.TDI)
		offset += len(seq.TDI)
	}

	return cmd
}

// DecodeJTAGSequence parses response and extracts TDO data
func (p *CMSISDAPProtocol) DecodeJTAGSequence(resp []byte, sequences []JTAGSequence) ([][]byte, error) {
	if len(resp) < 2 {
		return nil, fmt.Errorf("response too short")
	}
	if resp[0] != CmdJTAGSequence {
		return nil, fmt.Errorf("invalid command ID")
	}
	if resp[1] != StatusOK {
		return nil, fmt.Errorf("sequence failed")
	}

	// Extract TDO data for sequences that requested capture
	result := make([][]byte, 0)
	offset := 2

	for _, seq := range sequences {
		if seq.CaptureTDO() {
			// This sequence captured TDO
			tdo := make([]byte, len(seq.TDI))
			if offset+len(tdo) > len(resp) {
				return nil, fmt.Errorf("incomplete TDO data")
			}
			copy(tdo, resp[offset:offset+len(tdo)])
			result = append(result, tdo)
			offset += len(tdo)
		}
	}

	return result, nil
}

// EncodeSetClock builds a DAP_SWJ_Clock command
func (p *CMSISDAPProtocol) EncodeSetClock(hz uint32) []byte {
	cmd := make([]byte, 5)
	cmd[0] = CmdSWJClock
	binary.LittleEndian.PutUint32(cmd[1:], hz)
	return cmd
}

// DecodeSetClock parses response
func (p *CMSISDAPProtocol) DecodeSetClock(resp []byte) error {
	if len(resp) < 2 {
		return fmt.Errorf("response too short")
	}
	if resp[0] != CmdSWJClock {
		return fmt.Errorf("invalid command ID")
	}
	if resp[1] != StatusOK {
		return fmt.Errorf("set clock failed")
	}
	return nil
}

// EncodeResetTarget builds a DAP_ResetTarget command
func (p *CMSISDAPProtocol) EncodeResetTarget() []byte {
	return []byte{CmdResetTarget}
}

// DecodeResetTarget parses response
func (p *CMSISDAPProtocol) DecodeResetTarget(resp []byte) error {
	if len(resp) < 2 {
		return fmt.Errorf("response too short")
	}
	if resp[0] != CmdResetTarget {
		return fmt.Errorf("invalid command ID")
	}
	if resp[1] != StatusOK {
		return fmt.Errorf("reset target failed")
	}
	return nil
}
