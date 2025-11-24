package chain

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
)

func TestMemoryRepositoryWildcardLookup(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("parser init failed: %v", err)
	}
	repo := NewMemoryRepository()
	bText := simpleBSDL("WILD", "0000000000000000000000000000XXXX")
	file, err := parser.ParseString(bText)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	_, mask, err := repo.AddFile(file)
	if err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}
	if mask == 0xFFFFFFFF {
		t.Fatalf("expected wildcard mask, got full mask")
	}
	for _, id := range []uint32{0x0, 0x5, 0xA} {
		if _, err := repo.Lookup(id); err != nil {
			t.Fatalf("Lookup failed for id 0x%X: %v", id, err)
		}
		if repo.DeviceInfo(id) == nil {
			t.Fatalf("DeviceInfo missing for id 0x%X", id)
		}
	}
}

func TestMemoryRepositoryLoadDir(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("parser init failed: %v", err)
	}
	bText := simpleBSDL("DIRDEV", "00000000000000000000000000000001")
	file, err := parser.ParseString(bText)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "device.bsdl")
	if err := os.WriteFile(path, []byte(bText), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	repo := NewMemoryRepository()
	if err := repo.LoadDir(tmpDir); err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}
	if _, err := repo.Lookup(1); err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if repo.DeviceInfo(1) == nil {
		t.Fatalf("DeviceInfo missing after LoadDir")
	}

	// Ensure loading via parser result also works to guard against unused variables.
	if _, _, err := repo.AddFile(file); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}
}

func simpleBSDL(entity, id string) string {
	return fmt.Sprintf(`
entity %s is
	attribute INSTRUCTION_LENGTH of %s : entity is 4;
	attribute BOUNDARY_LENGTH of %s : entity is 1;
	attribute INSTRUCTION_OPCODE of %s : entity is "BYPASS (1111)";
	attribute IDCODE_REGISTER of %s : entity is "%s";
	attribute BOUNDARY_REGISTER of %s : entity is
		"0 (BC_1, PIN, INPUT, X)";
end %s;
`, entity, entity, entity, entity, entity, id, entity, entity)
}
