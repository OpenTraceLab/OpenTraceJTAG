package bsdl

import (
	"path/filepath"
	"testing"
)

// TestParseAllBSDLFiles tests parsing of all BSDL files in testdata directory
func TestParseAllBSDLFiles(t *testing.T) {
	files := []struct {
		name         string
		manufacturer string
		expectPorts  int // minimum expected ports
	}{
		{"adsp-21562_adsp-21563_adsp-21565_lqfp_bsdl.bsdl", "Analog Devices", 60},
		{"STM32F303_F334_LQFP64.bsd", "STMicroelectronics", 50},
		{"STM32F405_LQFP100.bsd", "STMicroelectronics", 90},
		{"STM32F373_LQFP100.bsd", "STMicroelectronics", 90},
		{"STM32F405_LQFP176.bsd", "STMicroelectronics", 150},
		{"STM32F301_F302_LQFP48.bsd", "STMicroelectronics", 40},
		{"STM32F358_LQFP64.bsd", "STMicroelectronics", 50},
		{"STM32F378_LQFP100.bsd", "STMicroelectronics", 90},
		{"LFE5U_25F_CABGA381.bsm", "Lattice", 200},
		{"LFE5U_85F_CABGA756.bsm", "Lattice", 350},
	}

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	successCount := 0
	failCount := 0

	for _, tc := range files {
		t.Run(tc.name, func(t *testing.T) {
			filename := filepath.Join("../../testdata", tc.name)

			bsdl, err := parser.ParseFile(filename)
			if err != nil {
				t.Errorf("[%s] Failed to parse: %v", tc.manufacturer, err)
				failCount++
				return
			}

			if bsdl.Entity == nil {
				t.Error("Entity is nil")
				failCount++
				return
			}

			t.Logf("[%s] Successfully parsed entity '%s'", tc.manufacturer, bsdl.Entity.Name)

			// Check entity name is not empty
			if bsdl.Entity.Name == "" {
				t.Error("Entity name is empty")
				failCount++
				return
			}

			// Check ports if present
			if bsdl.Entity.Port != nil {
				portCount := len(bsdl.Entity.Port.Ports)
				t.Logf("  Ports: %d", portCount)

				if portCount < tc.expectPorts {
					t.Errorf("Expected at least %d ports, got %d", tc.expectPorts, portCount)
				}
			} else {
				t.Log("  No port clause")
			}

			// Check generic if present
			if bsdl.Entity.Generic != nil {
				t.Logf("  Generics: %d", len(bsdl.Entity.Generic.Generics))
			}

			// Check attributes
			attrs := bsdl.Entity.GetAttributes()
			t.Logf("  Attributes: %d", len(attrs))

			// Check use clause
			if use := bsdl.Entity.GetUseClause(); use != nil {
				t.Logf("  Use: %s.%s", use.Package, use.Dot)
			}

			successCount++
		})
	}

	// Summary
	t.Logf("\n=== SUMMARY ===")
	t.Logf("Total files: %d", len(files))
	t.Logf("Successful: %d", successCount)
	t.Logf("Failed: %d", failCount)

	if failCount > 0 {
		t.Errorf("%d out of %d files failed to parse", failCount, len(files))
	}
}

// TestQuickParseCheck does a quick sanity check on each file
func TestQuickParseCheck(t *testing.T) {
	testdataFiles := []string{
		"../../testdata/adsp-21562_adsp-21563_adsp-21565_lqfp_bsdl.bsdl",
		"../../testdata/STM32F303_F334_LQFP64.bsd",
		"../../testdata/STM32F405_LQFP100.bsd",
		"../../testdata/LFE5U_25F_CABGA381.bsm",
	}

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	for _, filename := range testdataFiles {
		bsdl, err := parser.ParseFile(filename)
		if err != nil {
			t.Errorf("Failed to parse %s: %v", filepath.Base(filename), err)
			continue
		}

		if bsdl.Entity == nil || bsdl.Entity.Name == "" {
			t.Errorf("Invalid entity in %s", filepath.Base(filename))
		}
	}
}
