package jtag

import (
	"fmt"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/idcode"
)

// IDCodeInfo contains decoded IDCODE information (legacy compatibility).
type IDCodeInfo struct {
	Raw          uint32
	Version      uint8
	PartNumber   uint16
	Manufacturer uint16
	ManufName    string
}

// DecodeIDCode decodes a 32-bit IDCODE into its components.
// This is a legacy wrapper around the new idcode package.
func DecodeIDCode(raw uint32) IDCodeInfo {
	id := idcode.ParseIDCode(raw)
	m, _ := idcode.LookupManufacturer(id.ManufacturerCode)
	
	return IDCodeInfo{
		Raw:          id.Raw,
		Version:      id.Version,
		PartNumber:   id.PartNumber,
		Manufacturer: id.ManufacturerCode,
		ManufName:    m.Name,
	}
}

// String returns a formatted string representation of the IDCODE.
func (i IDCodeInfo) String() string {
	return fmt.Sprintf("0x%08X (Mfg: %s, Part: 0x%04X, Ver: %d)",
		i.Raw, i.ManufName, i.PartNumber, i.Version)
}

// GetManufacturerName returns the manufacturer name for a given JEDEC ID.
// This is a legacy wrapper around the new idcode package.
func GetManufacturerName(id uint16) string {
	m, _ := idcode.LookupManufacturer(id)
	return m.Name
}
