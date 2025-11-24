package idcode

// IDCode represents a parsed IEEE 1149.1 JTAG IDCODE
type IDCode struct {
	Raw              uint32 // full IDCODE
	Version          uint8  // [31:28]
	PartNumber       uint16 // [27:12]
	ManufacturerCode uint16 // [11:1] JEP106
	HasIDCode        bool   // bit 0 == 1
}

// Manufacturer represents a JEP106 manufacturer entry
type Manufacturer struct {
	Code         uint16 // JEP106 code
	Name         string // "NXP Semiconductors"
	Abbreviation string // "NXP"
	Country      string // optional
}
