package reveng

import "regexp"

// Config controls the behavior of the reverse engineering algorithm.
type Config struct {
	// Detection settings
	RepeatsPerPin          int  // Number of 0→1→0 cycles per pin (default: 1)
	RequireSymmetricToggle bool // Require both 0→1→0 AND 1→0→1 patterns (default: false)

	// Pin filtering
	SkipKnownJTAGPins bool     // Exclude TCK/TMS/TDI/TDO pins (default: true)
	SkipPowerPins     bool     // Exclude VCC/GND pins (default: true)
	OnlyDevices       []string // If set, only scan pins from these devices (by name)
	OnlyPinPattern    string   // If set, only scan pins matching this regex

	// Advanced heuristics (future enhancements)
	DetectPullResistors bool // Attempt to detect weak pull-up/down (default: false)
	MinToggleStrength   int  // Minimum number of successful toggles required (default: 1)

	// Internal compiled regex
	pinRegex *regexp.Regexp
}

// DefaultConfig returns a Config with sensible defaults for most use cases.
func DefaultConfig() *Config {
	return &Config{
		RepeatsPerPin:          1,
		RequireSymmetricToggle: false,
		SkipKnownJTAGPins:      true,
		SkipPowerPins:          true,
		OnlyDevices:            nil,
		OnlyPinPattern:         "",
		DetectPullResistors:    false,
		MinToggleStrength:      1,
	}
}

// Validate checks the configuration for errors and compiles any regex patterns.
func (c *Config) Validate() error {
	if c.RepeatsPerPin < 1 {
		c.RepeatsPerPin = 1
	}

	if c.MinToggleStrength < 1 {
		c.MinToggleStrength = 1
	}

	// Compile pin pattern regex if provided
	if c.OnlyPinPattern != "" {
		regex, err := regexp.Compile(c.OnlyPinPattern)
		if err != nil {
			return err
		}
		c.pinRegex = regex
	}

	return nil
}

// ShouldScanDevice returns true if the given device name should be scanned
// based on the OnlyDevices filter.
func (c *Config) ShouldScanDevice(deviceName string) bool {
	if len(c.OnlyDevices) == 0 {
		return true // No filter, scan all devices
	}

	for _, allowed := range c.OnlyDevices {
		if deviceName == allowed {
			return true
		}
	}
	return false
}

// ShouldScanPin returns true if the given pin name should be scanned
// based on the OnlyPinPattern filter.
func (c *Config) ShouldScanPin(pinName string) bool {
	if c.pinRegex == nil {
		return true // No filter, scan all pins
	}
	return c.pinRegex.MatchString(pinName)
}
