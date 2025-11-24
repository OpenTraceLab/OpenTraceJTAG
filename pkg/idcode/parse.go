package idcode

// ParseIDCode parses a raw 32-bit IDCODE into its component fields
func ParseIDCode(raw uint32) IDCode {
	return IDCode{
		Raw:              raw,
		Version:          uint8((raw >> 28) & 0xF),
		PartNumber:       uint16((raw >> 12) & 0xFFFF),
		ManufacturerCode: uint16((raw >> 1) & 0x7FF),
		HasIDCode:        (raw & 0x1) == 0x1,
	}
}
