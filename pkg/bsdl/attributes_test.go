package bsdl

import (
	"testing"
)

// TestGetInstructionOpcodes tests instruction extraction from BSDL files
func TestGetInstructionOpcodes(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	t.Run("STM32F303", func(t *testing.T) {
		bsdl, err := parser.ParseFile("../../testdata/STM32F303_F334_LQFP64.bsd")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		instructions := bsdl.Entity.GetInstructionOpcodes()
		if instructions == nil {
			t.Fatal("No instructions found")
		}

		t.Logf("Found %d instructions", len(instructions))

		// Check for mandatory instructions
		foundBypass := false
		foundExtest := false
		foundIdcode := false

		for _, instr := range instructions {
			t.Logf("  %s: %s", instr.Name, instr.Opcode)

			switch instr.Name {
			case "BYPASS":
				foundBypass = true
				if len(instr.Opcode) == 0 {
					t.Error("BYPASS has empty opcode")
				}
			case "EXTEST":
				foundExtest = true
			case "IDCODE":
				foundIdcode = true
			}
		}

		if !foundBypass {
			t.Error("BYPASS instruction not found")
		}
		if !foundExtest {
			t.Error("EXTEST instruction not found")
		}
		if !foundIdcode {
			t.Error("IDCODE instruction not found")
		}
	})

	t.Run("Lattice LFE5U", func(t *testing.T) {
		bsdl, err := parser.ParseFile("../../testdata/LFE5U_25F_CABGA381.bsm")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		instructions := bsdl.Entity.GetInstructionOpcodes()
		if instructions == nil {
			t.Fatal("No instructions found")
		}

		t.Logf("Found %d Lattice instructions", len(instructions))

		for i, instr := range instructions {
			if i < 5 { // Log first 5
				t.Logf("  %s: %s", instr.Name, instr.Opcode)
			}
		}

		if len(instructions) == 0 {
			t.Error("Expected at least some instructions")
		}
	})
}

// TestGetDeviceInfo tests device information extraction
func TestGetDeviceInfo(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	t.Run("STM32F303", func(t *testing.T) {
		bsdl, err := parser.ParseFile("../../testdata/STM32F303_F334_LQFP64.bsd")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		info := bsdl.Entity.GetDeviceInfo()
		if info == nil {
			t.Fatal("DeviceInfo is nil")
		}

		t.Logf("Instruction Length: %d bits", info.InstructionLength)
		t.Logf("Boundary Length: %d bits", info.BoundaryLength)
		t.Logf("IDCODE: %s", info.IDCode)
		t.Logf("Instruction Capture: %s", info.InstructionCapture)

		// Validate expected values
		if info.InstructionLength == 0 {
			t.Error("InstructionLength is 0")
		}
		if info.BoundaryLength == 0 {
			t.Error("BoundaryLength is 0")
		}
		if info.IDCode == "" {
			t.Error("IDCode is empty")
		}

		// STM32 typically has 5-bit IR
		if info.InstructionLength < 4 || info.InstructionLength > 8 {
			t.Errorf("Unexpected IR length: %d", info.InstructionLength)
		}
	})

	t.Run("ADSP", func(t *testing.T) {
		bsdl, err := parser.ParseFile("../../testdata/adsp-21562_adsp-21563_adsp-21565_lqfp_bsdl.bsdl")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		info := bsdl.Entity.GetDeviceInfo()
		if info == nil {
			t.Fatal("DeviceInfo is nil")
		}

		t.Logf("ADSP IR Length: %d", info.InstructionLength)
		t.Logf("ADSP Boundary Length: %d", info.BoundaryLength)
		t.Logf("ADSP IDCODE: %s", info.IDCode)

		if info.InstructionLength == 0 {
			t.Error("InstructionLength is 0")
		}
	})

	t.Run("Lattice", func(t *testing.T) {
		bsdl, err := parser.ParseFile("../../testdata/LFE5U_25F_CABGA381.bsm")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		info := bsdl.Entity.GetDeviceInfo()
		if info == nil {
			t.Fatal("DeviceInfo is nil")
		}

		t.Logf("Lattice IR Length: %d", info.InstructionLength)
		t.Logf("Lattice Boundary Length: %d", info.BoundaryLength)
		t.Logf("Lattice IDCODE: %s", info.IDCode)
		t.Logf("Lattice USERCODE: %s", info.UserCode)

		// Lattice typically has 8-bit IR
		if info.InstructionLength != 8 {
			t.Logf("Note: Lattice IR length is %d (expected 8)", info.InstructionLength)
		}
	})
}

// TestGetTAPConfig tests TAP configuration extraction
func TestGetTAPConfig(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	t.Run("STM32F303", func(t *testing.T) {
		bsdl, err := parser.ParseFile("../../testdata/STM32F303_F334_LQFP64.bsd")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		config := bsdl.Entity.GetTAPConfig()
		if config == nil {
			t.Fatal("TAPConfig is nil")
		}

		t.Logf("TAP_SCAN_IN: %s", config.ScanIn)
		t.Logf("TAP_SCAN_OUT: %s", config.ScanOut)
		t.Logf("TAP_SCAN_MODE: %s", config.ScanMode)
		t.Logf("TAP_SCAN_RESET: %s", config.ScanReset)
		t.Logf("TAP_SCAN_CLOCK: %s", config.ScanClock)
		t.Logf("Max Frequency: %.0f Hz", config.MaxFreq)
		t.Logf("Clock Edge: %s", config.Edge)

		// Validate TAP signals are present
		if config.ScanIn == "" {
			t.Error("TAP_SCAN_IN is empty")
		}
		if config.ScanOut == "" {
			t.Error("TAP_SCAN_OUT is empty")
		}
		if config.ScanMode == "" {
			t.Error("TAP_SCAN_MODE is empty")
		}

		// Check frequency is reasonable (typically 1-50 MHz)
		if config.MaxFreq > 0 {
			if config.MaxFreq < 100000 || config.MaxFreq > 100e6 {
				t.Logf("Note: MaxFreq seems unusual: %.0f Hz", config.MaxFreq)
			}
		}
	})

	t.Run("Lattice", func(t *testing.T) {
		bsdl, err := parser.ParseFile("../../testdata/LFE5U_25F_CABGA381.bsm")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		config := bsdl.Entity.GetTAPConfig()
		if config == nil {
			t.Fatal("TAPConfig is nil")
		}

		t.Logf("Lattice TAP_SCAN_IN: %s", config.ScanIn)
		t.Logf("Lattice TAP_SCAN_OUT: %s", config.ScanOut)
		t.Logf("Lattice Max Frequency: %.0f Hz", config.MaxFreq)
	})
}

// TestGetPinMap tests pin mapping extraction
func TestGetPinMap(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	t.Run("STM32F303", func(t *testing.T) {
		bsdl, err := parser.ParseFile("../../testdata/STM32F303_F334_LQFP64.bsd")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		pinMap := bsdl.Entity.GetPinMap()
		if pinMap == nil {
			t.Fatal("PinMap is nil")
		}

		t.Logf("Found %d pin mappings", len(pinMap))

		if len(pinMap) == 0 {
			t.Error("Expected at least some pin mappings")
		}

		// Check for JTAG pins
		if tdi, ok := pinMap["TDI"]; ok {
			t.Logf("  TDI -> Pin %s", tdi)
		}
		if tdo, ok := pinMap["TDO"]; ok {
			t.Logf("  TDO -> Pin %s", tdo)
		}
		if tms, ok := pinMap["TMS"]; ok {
			t.Logf("  TMS -> Pin %s", tms)
		}
		if tck, ok := pinMap["TCK"]; ok {
			t.Logf("  TCK -> Pin %s", tck)
		}

		// Show first few mappings
		count := 0
		for signal, pin := range pinMap {
			if count < 5 {
				t.Logf("  %s -> Pin %s", signal, pin)
			}
			count++
		}
	})

	t.Run("ADSP", func(t *testing.T) {
		bsdl, err := parser.ParseFile("../../testdata/adsp-21562_adsp-21563_adsp-21565_lqfp_bsdl.bsdl")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		pinMap := bsdl.Entity.GetPinMap()
		t.Logf("ADSP found %d pin mappings", len(pinMap))

		if len(pinMap) > 0 {
			// Show a few examples
			count := 0
			for signal, pin := range pinMap {
				if count < 3 {
					t.Logf("  %s -> Pin %s", signal, pin)
				}
				count++
			}
		}
	})
}

// TestParseBinaryString tests binary string parsing with wildcards
func TestParseBinaryString(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedValue  uint32
		expectedMask   uint32
		hasWildcards   bool
	}{
		{
			name:          "Simple binary",
			input:         "1010",
			expectedValue: 0b1010,
			expectedMask:  0b1111,
			hasWildcards:  false,
		},
		{
			name:          "32-bit value",
			input:         "00000000000000000000000000001111",
			expectedValue: 0x0000000F,
			expectedMask:  0xFFFFFFFF,
			hasWildcards:  false,
		},
		{
			name:          "With wildcards lowercase",
			input:         "10x0",
			expectedValue: 0b1000,
			expectedMask:  0b1101,
			hasWildcards:  true,
		},
		{
			name:          "With wildcards uppercase",
			input:         "10X0",
			expectedValue: 0b1000,
			expectedMask:  0b1101,
			hasWildcards:  true,
		},
		{
			name:          "Multiple wildcards",
			input:         "1XX0",
			expectedValue: 0b1000,
			expectedMask:  0b1001,
			hasWildcards:  true,
		},
		{
			name:          "All wildcards",
			input:         "XXXX",
			expectedValue: 0b0000,
			expectedMask:  0b0000,
			hasWildcards:  true,
		},
		{
			name:          "With spaces",
			input:         " 1010 ",
			expectedValue: 0b1010,
			expectedMask:  0b1111,
			hasWildcards:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, mask, hasWildcards := ParseBinaryString(tt.input)

			if value != tt.expectedValue {
				t.Errorf("Value: expected 0x%08X, got 0x%08X", tt.expectedValue, value)
			}
			if mask != tt.expectedMask {
				t.Errorf("Mask: expected 0x%08X, got 0x%08X", tt.expectedMask, mask)
			}
			if hasWildcards != tt.hasWildcards {
				t.Errorf("HasWildcards: expected %v, got %v", tt.hasWildcards, hasWildcards)
			}
		})
	}
}

// TestOpcodeToUint tests opcode conversion
func TestOpcodeToUint(t *testing.T) {
	tests := []struct {
		name     string
		opcode   string
		expected uint
		wantErr  bool
	}{
		{"5-bit BYPASS", "11111", 31, false},
		{"5-bit EXTEST", "00000", 0, false},
		{"4-bit value", "1010", 10, false},
		{"8-bit value", "11110000", 240, false},
		{"Invalid char", "1012", 0, true},
		{"Empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := OpcodeToUint(tt.opcode)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestGetInstructions tests the GetInstructions helper function
func TestGetInstructions(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Parse a file and get the raw expression
	bsdl, err := parser.ParseFile("../../testdata/STM32F303_F334_LQFP64.bsd")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Get the INSTRUCTION_OPCODE attribute
	attrs := bsdl.Entity.GetAttributes()
	var opcodeExpr *Expression
	for _, attr := range attrs {
		if attr.Spec != nil && attr.Spec.Name == "INSTRUCTION_OPCODE" {
			opcodeExpr = attr.Spec.Is
			break
		}
	}

	if opcodeExpr == nil {
		t.Fatal("INSTRUCTION_OPCODE attribute not found")
	}

	instructions := GetInstructions(opcodeExpr)
	if instructions == nil {
		t.Fatal("GetInstructions returned nil")
	}

	t.Logf("Parsed %d instructions from expression", len(instructions))

	for _, instr := range instructions {
		// Validate format
		if instr.Name == "" {
			t.Error("Found instruction with empty name")
		}
		if instr.Opcode == "" {
			t.Errorf("Instruction %s has empty opcode", instr.Name)
		}

		// Validate opcode is binary
		for _, ch := range instr.Opcode {
			if ch != '0' && ch != '1' {
				t.Errorf("Instruction %s has invalid opcode character: %c", instr.Name, ch)
			}
		}
	}
}

// TestAllFilesDeviceInfo validates device info extraction across all test files
func TestAllFilesDeviceInfo(t *testing.T) {
	files := []string{
		"../../testdata/adsp-21562_adsp-21563_adsp-21565_lqfp_bsdl.bsdl",
		"../../testdata/STM32F303_F334_LQFP64.bsd",
		"../../testdata/STM32F405_LQFP100.bsd",
		"../../testdata/STM32F373_LQFP100.bsd",
		"../../testdata/STM32F405_LQFP176.bsd",
		"../../testdata/STM32F301_F302_LQFP48.bsd",
		"../../testdata/STM32F358_LQFP64.bsd",
		"../../testdata/STM32F378_LQFP100.bsd",
		"../../testdata/LFE5U_25F_CABGA381.bsm",
		"../../testdata/LFE5U_85F_CABGA756.bsm",
	}

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	for _, filename := range files {
		t.Run(filename, func(t *testing.T) {
			bsdl, err := parser.ParseFile(filename)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			info := bsdl.Entity.GetDeviceInfo()
			if info == nil {
				t.Fatal("DeviceInfo is nil")
			}

			t.Logf("IR: %d bits, Boundary: %d bits",
				info.InstructionLength, info.BoundaryLength)

			// All files should have these basic fields
			if info.InstructionLength == 0 {
				t.Error("Missing InstructionLength")
			}
			if info.BoundaryLength == 0 {
				t.Error("Missing BoundaryLength")
			}

			// Check for IDCODE
			if info.IDCode != "" {
				value, mask, hasWildcards := ParseBinaryString(info.IDCode)
				t.Logf("IDCODE: 0x%08X (mask: 0x%08X, wildcards: %v)",
					value, mask, hasWildcards)
			}
		})
	}
}
