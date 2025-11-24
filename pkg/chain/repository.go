package chain

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
)

// Repository knows how to look up BSDL files for a given device ID code.
type Repository interface {
	Lookup(id uint32) (*bsdl.BSDLFile, error)
}

// MemoryRepository is a simple in-memory implementation useful during tests or
// when the caller preloads a fixed set of devices.
type MemoryRepository struct {
	mu        sync.RWMutex
	devices   map[uint32]*bsdl.BSDLFile
	metadata  map[uint32]*bsdl.DeviceInfo
	wildcards []repoEntry
}

type repoEntry struct {
	value uint32
	mask  uint32
	file  *bsdl.BSDLFile
	info  *bsdl.DeviceInfo
}

// NewMemoryRepository creates an empty repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		devices:  make(map[uint32]*bsdl.BSDLFile),
		metadata: make(map[uint32]*bsdl.DeviceInfo),
	}
}

// Add registers a BSDL file under the provided IDCODE.
func (r *MemoryRepository) Add(id uint32, file *bsdl.BSDLFile) {
	var info *bsdl.DeviceInfo
	if file != nil && file.Entity != nil {
		info = file.Entity.GetDeviceInfo()
	}
	r.addEntry(repoEntry{
		value: id,
		mask:  0xFFFFFFFF,
		file:  file,
		info:  info,
	})
}

// AddFile extracts the IDCODE from the BSDL entity (if possible) and registers
// the file. It supports wildcard bits within the IDCODE ("X" characters) and
// returns both the parsed value and mask.
func (r *MemoryRepository) AddFile(file *bsdl.BSDLFile) (uint32, uint32, error) {
	if file == nil || file.Entity == nil {
		return 0, 0, fmt.Errorf("chain: invalid BSDL file")
	}
	info := file.Entity.GetDeviceInfo()
	if info == nil || info.IDCode == "" {
		return 0, 0, fmt.Errorf("chain: BSDL missing IDCODE_REGISTER")
	}
	value, mask, err := parseIDCode(info.IDCode)
	if err != nil {
		return 0, 0, err
	}
	r.addEntry(repoEntry{
		value: value,
		mask:  mask,
		file:  file,
		info:  info,
	})
	return value, mask, nil
}

// Lookup implements the Repository interface.
func (r *MemoryRepository) Lookup(id uint32) (*bsdl.BSDLFile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if file, ok := r.devices[id]; ok {
		return file, nil
	}
	for _, entry := range r.wildcards {
		if entry.matches(id) {
			return entry.file, nil
		}
	}
	return nil, fmt.Errorf("chain: no BSDL for IDCODE 0x%08X", id)
}

// DeviceInfo returns the cached metadata for an ID (if already loaded via Add or
// AddFile). This is optional but helps avoid re-parsing attributes repeatedly.
func (r *MemoryRepository) DeviceInfo(id uint32) *bsdl.DeviceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if info, ok := r.metadata[id]; ok {
		return info
	}
	for _, entry := range r.wildcards {
		if entry.matches(id) {
			return entry.info
		}
	}
	return nil
}

// LoadFiles parses the provided file paths and adds each BSDL file to the
// repository.
func (r *MemoryRepository) LoadFiles(paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	parser, err := bsdl.NewParser()
	if err != nil {
		return err
	}
	for _, path := range paths {
		file, err := parser.ParseFile(path)
		if err != nil {
			return fmt.Errorf("chain: parse %s: %w", path, err)
		}
		if _, _, err := r.AddFile(file); err != nil {
			return fmt.Errorf("chain: add %s: %w", path, err)
		}
	}
	return nil
}

// LoadDir recursively loads all .bsd/.bsdl/.bsm files from the provided
// directory.
func (r *MemoryRepository) LoadDir(root string) error {
	parser, err := bsdl.NewParser()
	if err != nil {
		return err
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !isBSDLFile(path) {
			return nil
		}
		file, err := parser.ParseFile(path)
		if err != nil {
			return fmt.Errorf("chain: parse %s: %w", path, err)
		}
		if _, _, err := r.AddFile(file); err != nil {
			return fmt.Errorf("chain: add %s: %w", path, err)
		}
		return nil
	})
}

func isBSDLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".bsd", ".bsdl", ".bsm":
		return true
	default:
		return false
	}
}

func (r *MemoryRepository) addEntry(entry repoEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry.mask == 0xFFFFFFFF {
		r.devices[entry.value] = entry.file
		if entry.info != nil {
			r.metadata[entry.value] = entry.info
		}
		return
	}
	r.wildcards = append(r.wildcards, entry)
}

func (e repoEntry) matches(id uint32) bool {
	return (id & e.mask) == (e.value & e.mask)
}
