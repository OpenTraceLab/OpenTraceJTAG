package deviceinfo

import "github.com/OpenTraceLab/OpenTraceJTAG/pkg/idcode"

// key is used for device database lookups
type key struct {
	ManufacturerCode uint16
	PartNumber       uint16
}

// db is the in-memory device database
var db = make(map[key]DeviceInfo)

// register adds a device entry to the database
func register(k key, info DeviceInfo) {
	db[k] = info
}

// Lookup returns device information for a given IDCODE
// Falls back to generic info if device is not in database
func Lookup(rawID uint32) DeviceInfo {
	id := idcode.ParseIDCode(rawID)
	m, _ := idcode.LookupManufacturer(id.ManufacturerCode)

	k := key{ManufacturerCode: id.ManufacturerCode, PartNumber: id.PartNumber}
	if info, ok := db[k]; ok {
		// Enrich with parsed ID and manufacturer
		info.IDCode = id
		info.Manufacturer = m
		return info
	}

	// Unknown device – return minimal info
	return DeviceInfo{
		IDCode:       id,
		Manufacturer: m,
		Name:         "Unknown device",
		Description:  "No entry in device database",
	}
}
