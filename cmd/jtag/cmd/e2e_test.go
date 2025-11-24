package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDiscoverE2E tests the discover command end-to-end
func TestDiscoverE2E(t *testing.T) {
	// Find testdata directory
	testdata := "../../testdata"
	if _, err := os.Stat(testdata); os.IsNotExist(err) {
		testdata = "../../../testdata"
	}

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		wantContain []string
	}{
		{
			name: "single device",
			args: []string{"discover", "--count", "1", "--bsdl", testdata, "--sim-ids", "0x06438041"},
			wantContain: []string{
				"JTAG Chain Discovery Results",
				"Found 1 device(s)",
				"0x06438041",
				"STM32F303",
				"IR Length:",
				"Boundary Length:",
			},
		},
		{
			name: "two devices",
			args: []string{"discover", "--count", "2", "--bsdl", testdata, "--sim-ids", "0x06438041", "--sim-ids", "0x41111043"},
			wantContain: []string{
				"Found 2 device(s)",
				"0x06438041",
				"0x41111043",
				"STM32F303",
				"LFE5U",
				"Total IR Length:",
			},
		},
		{
			name: "three devices",
			args: []string{"discover", "--count", "3", "--bsdl", testdata, "--sim-ids", "0x06438041", "--sim-ids", "0x41111043", "--sim-ids", "0x028200CB"},
			wantContain: []string{
				"Found 3 device(s)",
				"0x028200CB",
				"ADSP",
			},
		},
		{
			name:    "missing count flag",
			args:    []string{"discover", "--bsdl", testdata},
			wantErr: true,
		},
		{
			name:    "mismatched count and IDs",
			args:    []string{"discover", "--count", "2", "--bsdl", testdata, "--sim-ids", "0x06438041"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Read in background to prevent pipe buffer from blocking on Windows
			var buf bytes.Buffer
			done := make(chan struct{})
			go func() {
				buf.ReadFrom(r)
				close(done)
			}()

			// Reset flags to prevent accumulation between tests
			simIDCodes = nil
			deviceCount = 1
			bsdlDir = "testdata"
			adapterType = "simulator"
			adapterSerial = ""

			// Reset root command for each test
			rootCmd.SetArgs(tt.args)

			// Execute
			err := rootCmd.Execute()

			// Restore stdout and wait for reader
			w.Close()
			os.Stdout = old
			<-done

			output := buf.String()

			// Check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
				return
			}

			// Check output contains expected strings
			for _, want := range tt.wantContain {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing expected string: %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

// TestParseE2E tests the parse command end-to-end
func TestParseE2E(t *testing.T) {
	// Find testdata directory
	testdata := "../../testdata"
	if _, err := os.Stat(testdata); os.IsNotExist(err) {
		testdata = "../../../testdata"
	}

	stm32File := filepath.Join(testdata, "STM32F303_F334_LQFP64.bsd")
	latticeFile := filepath.Join(testdata, "LFE5U_25F_CABGA381.bsm")

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		wantContain []string
	}{
		{
			name: "basic parse STM32",
			args: []string{"parse", stm32File},
			wantContain: []string{
				"STM32F303",
				"IR Length:",
				"Boundary Length:",
				"Instructions:",
				"BYPASS",
				"EXTEST",
			},
		},
		{
			name: "parse with instructions flag",
			args: []string{"parse", "--instructions", stm32File},
			wantContain: []string{
				"BYPASS",
				"EXTEST",
				"SAMPLE",
				"PRELOAD",
				"IDCODE",
			},
		},
		{
			name: "parse with pins flag",
			args: []string{"parse", "--pins", stm32File},
			wantContain: []string{
				"Pin Mappings:",
				"->",
				"Pin",
			},
		},
		{
			name: "parse Lattice",
			args: []string{"parse", latticeFile},
			wantContain: []string{
				"LFE5U",
				"IR Length:",
				"8 bits",
				"Boundary Length:",
			},
		},
		{
			name:    "parse non-existent file",
			args:    []string{"parse", "/nonexistent/file.bsd"},
			wantErr: true,
		},
		{
			name:    "parse missing argument",
			args:    []string{"parse"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Read in background to prevent pipe buffer from blocking on Windows
			var buf bytes.Buffer
			done := make(chan struct{})
			go func() {
				buf.ReadFrom(r)
				close(done)
			}()

			// Reset parse flags to prevent accumulation between tests
			showInstructions = false
			showBoundary = false
			showPins = false

			// Reset root command
			rootCmd.SetArgs(tt.args)

			// Execute
			err := rootCmd.Execute()

			// Restore stdout and wait for reader
			w.Close()
			os.Stdout = old
			<-done

			output := buf.String()

			// Check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
				return
			}

			// Check output
			for _, want := range tt.wantContain {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing: %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

// TestVerboseFlag tests that verbose flag works across commands
func TestVerboseFlag(t *testing.T) {
	// Find testdata directory
	testdata := "../../testdata"
	if _, err := os.Stat(testdata); os.IsNotExist(err) {
		testdata = "../../../testdata"
	}

	tests := []struct {
		name        string
		args        []string
		wantContain []string
	}{
		{
			name: "discover verbose",
			args: []string{"discover", "-v", "--count", "1", "--bsdl", testdata, "--sim-ids", "0x06438041"},
			wantContain: []string{
				"Creating simulator adapter",
				"Loading BSDL files",
				"Adapter Information:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Read in background to prevent pipe buffer from blocking on Windows
			var buf bytes.Buffer
			done := make(chan struct{})
			go func() {
				buf.ReadFrom(r)
				close(done)
			}()

			// Reset flags to prevent accumulation between tests
			simIDCodes = nil
			deviceCount = 1
			bsdlDir = "testdata"
			adapterType = "simulator"
			adapterSerial = ""

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			w.Close()
			os.Stdout = old
			<-done

			output := buf.String()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			for _, want := range tt.wantContain {
				if !strings.Contains(output, want) {
					t.Errorf("Verbose output missing: %q", want)
				}
			}
		})
	}
}

// TestInfoE2E tests the info command end-to-end
func TestInfoE2E(t *testing.T) {
	// Find testdata directory
	testdata := "../../testdata"
	if _, err := os.Stat(testdata); os.IsNotExist(err) {
		testdata = "../../../testdata"
	}

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		wantContain []string
	}{
		{
			name: "single device human-readable",
			args: []string{"info", "--count", "1", "--bsdl", testdata, "--sim-ids", "0x06438041"},
			wantContain: []string{
				"JTAG Chain Information",
				"Devices: 1",
				"STM32F303",
				"IDCODE:",
				"0x06438041",
				"STMicroelectronics",
				"LQFP-64",
			},
		},
		{
			name: "two devices JSON",
			args: []string{"info", "--json", "--count", "2", "--bsdl", testdata, "--sim-ids", "0x06438041", "--sim-ids", "0x41111043"},
			wantContain: []string{
				"\"device_count\": 2",
				"\"idcode\": \"0x06438041\"",
				"\"idcode\": \"0x41111043\"",
				"\"manufacturer\": \"STMicroelectronics\"",
				"\"manufacturer\": \"Lattice Semiconductor\"",
				"\"package\": \"LQFP-64\"",
			},
		},
		{
			name: "verbose mode",
			args: []string{"info", "-v", "--count", "1", "--bsdl", testdata, "--sim-ids", "0x06438041"},
			wantContain: []string{
				"STM32F303",
				"Instructions:",
				"BYPASS",
				"EXTEST",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Read in background to prevent pipe buffer from blocking on Windows
			var buf bytes.Buffer
			done := make(chan struct{})
			go func() {
				buf.ReadFrom(r)
				close(done)
			}()

			simIDCodes = nil
			deviceCount = 1
			bsdlDir = "testdata"
			adapterType = "simulator"
			adapterSerial = ""
			outputJSON = false

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			w.Close()
			os.Stdout = old
			<-done

			output := buf.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
				return
			}

			for _, want := range tt.wantContain {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing: %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

// TestPinE2E tests the pin command end-to-end
func TestPinE2E(t *testing.T) {
	testdata := "../../testdata"
	if _, err := os.Stat(testdata); os.IsNotExist(err) {
		testdata = "../../../testdata"
	}

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		wantContain []string
	}{
		{
			name: "drive pin high",
			args: []string{"pin", "--count", "1", "--bsdl", testdata, "--sim-ids", "0x06438041",
				"--device", "STM32F303_F334_LQFP64", "--pin", "PA0", "--high"},
			wantContain: []string{
				"Setting pin PA0",
				"STM32F303_F334_LQFP64",
				"high",
				"successfully",
			},
		},
		{
			name: "drive pin low",
			args: []string{"pin", "--count", "1", "--bsdl", testdata, "--sim-ids", "0x06438041",
				"--device", "STM32F303_F334_LQFP64", "--pin", "PA1", "--low"},
			wantContain: []string{
				"Setting pin PA1",
				"low",
				"successfully",
			},
		},
		{
			name: "verbose mode",
			args: []string{"pin", "-v", "--count", "1", "--bsdl", testdata, "--sim-ids", "0x06438041",
				"--device", "STM32F303_F334_LQFP64", "--pin", "PA0", "--high"},
			wantContain: []string{
				"Creating simulator adapter",
				"Loading BSDL files",
				"Discovering chain",
				"Target device:",
				"Boundary scan operation completed",
			},
		},
		{
			name: "device not found",
			args: []string{"pin", "--count", "1", "--bsdl", testdata, "--sim-ids", "0x06438041",
				"--device", "NONEXISTENT", "--pin", "PA0", "--high"},
			wantErr: true,
		},
		{
			name: "missing high/low flag",
			args: []string{"pin", "--count", "1", "--bsdl", testdata, "--sim-ids", "0x06438041",
				"--device", "STM32F303_F334_LQFP64", "--pin", "PA0"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Read in background to prevent pipe buffer from blocking on Windows
			var buf bytes.Buffer
			done := make(chan struct{})
			go func() {
				buf.ReadFrom(r)
				close(done)
			}()

			simIDCodes = nil
			deviceCount = 1
			bsdlDir = "testdata"
			adapterType = "simulator"
			adapterSerial = ""
			pinDeviceName = ""
			pinName = ""
			pinHigh = false
			pinLow = false

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			w.Close()
			os.Stdout = old
			<-done

			output := buf.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
				return
			}

			for _, want := range tt.wantContain {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing: %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}
