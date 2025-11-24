package idcode

import "fmt"

// manufacturers is the JEP106 manufacturer database
var manufacturers = map[uint16]Manufacturer{
	0x001: {Code: 0x001, Name: "AMD", Abbreviation: "AMD"},
	0x002: {Code: 0x002, Name: "AMI", Abbreviation: "AMI"},
	0x003: {Code: 0x003, Name: "Fairchild", Abbreviation: "Fairchild"},
	0x004: {Code: 0x004, Name: "Fujitsu", Abbreviation: "Fujitsu"},
	0x005: {Code: 0x005, Name: "GTE", Abbreviation: "GTE"},
	0x006: {Code: 0x006, Name: "Harris", Abbreviation: "Harris"},
	0x007: {Code: 0x007, Name: "Hitachi", Abbreviation: "Hitachi"},
	0x008: {Code: 0x008, Name: "Inmos", Abbreviation: "Inmos"},
	0x009: {Code: 0x009, Name: "Intel", Abbreviation: "Intel"},
	0x00A: {Code: 0x00A, Name: "I.T.T.", Abbreviation: "ITT"},
	0x00B: {Code: 0x00B, Name: "Intersil", Abbreviation: "Intersil"},
	0x00C: {Code: 0x00C, Name: "Monolithic Memories", Abbreviation: "MMI"},
	0x00D: {Code: 0x00D, Name: "Mostek", Abbreviation: "Mostek"},
	0x00E: {Code: 0x00E, Name: "Freescale (Motorola)", Abbreviation: "Freescale"},
	0x00F: {Code: 0x00F, Name: "National", Abbreviation: "National"},
	0x010: {Code: 0x010, Name: "NEC", Abbreviation: "NEC"},
	0x011: {Code: 0x011, Name: "RCA", Abbreviation: "RCA"},
	0x012: {Code: 0x012, Name: "Raytheon", Abbreviation: "Raytheon"},
	0x013: {Code: 0x013, Name: "Conexant (Rockwell)", Abbreviation: "Conexant"},
	0x014: {Code: 0x014, Name: "Seeq", Abbreviation: "Seeq"},
	0x015: {Code: 0x015, Name: "Philips Semi. (Signetics)", Abbreviation: "Philips"},
	0x016: {Code: 0x016, Name: "Synertek", Abbreviation: "Synertek"},
	0x017: {Code: 0x017, Name: "Texas Instruments", Abbreviation: "TI"},
	0x018: {Code: 0x018, Name: "Toshiba", Abbreviation: "Toshiba"},
	0x019: {Code: 0x019, Name: "Xicor", Abbreviation: "Xicor"},
	0x01A: {Code: 0x01A, Name: "Zilog", Abbreviation: "Zilog"},
	0x01B: {Code: 0x01B, Name: "Eurotechnique", Abbreviation: "Eurotechnique"},
	0x01C: {Code: 0x01C, Name: "Mitsubishi", Abbreviation: "Mitsubishi"},
	0x01D: {Code: 0x01D, Name: "Lucent (AT&T)", Abbreviation: "Lucent"},
	0x01E: {Code: 0x01E, Name: "Exel", Abbreviation: "Exel"},
	0x01F: {Code: 0x01F, Name: "Atmel", Abbreviation: "Atmel"},
	0x020: {Code: 0x020, Name: "STMicroelectronics", Abbreviation: "STM"},
	0x025: {Code: 0x025, Name: "Analog Devices", Abbreviation: "ADI"},
	0x02E: {Code: 0x02E, Name: "Cypress", Abbreviation: "Cypress"},
	0x031: {Code: 0x031, Name: "Xilinx", Abbreviation: "Xilinx"},
	0x03D: {Code: 0x03D, Name: "Altera", Abbreviation: "Altera"},
	0x041: {Code: 0x041, Name: "Lattice", Abbreviation: "Lattice"},
	0x049: {Code: 0x049, Name: "Infineon", Abbreviation: "Infineon"},
	0x06E: {Code: 0x06E, Name: "Microchip", Abbreviation: "Microchip"},
	0x093: {Code: 0x093, Name: "ARM", Abbreviation: "ARM"},
	0x0B7: {Code: 0x0B7, Name: "Espressif", Abbreviation: "Espressif"},
	0x13B: {Code: 0x13B, Name: "Nordic Semiconductor", Abbreviation: "Nordic"},
	0x1F1: {Code: 0x1F1, Name: "Raspberry Pi", Abbreviation: "RPi"},
}

// LookupManufacturer returns manufacturer info for a JEP106 code
func LookupManufacturer(code uint16) (Manufacturer, bool) {
	m, ok := manufacturers[code]
	if !ok {
		// Return unknown manufacturer
		return Manufacturer{
			Code:         code,
			Name:         fmt.Sprintf("Unknown (0x%03X)", code),
			Abbreviation: "Unknown",
		}, false
	}
	return m, true
}
