package reveng

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsr"
)

// Net represents a connected set of pins that share the same electrical net.
type Net struct {
	ID   int          `json:"id"`
	Pins []bsr.PinRef `json:"pins"`
}

// Netlist manages the discovered connectivity between pins using a union-find
// data structure for efficient connection tracking.
type Netlist struct {
	// Union-find data structures
	parent map[string]string // Maps pin key to parent pin key
	rank   map[string]int    // Rank for union-by-rank optimization

	// Final nets after calling Finalize()
	Nets []*Net

	// All pins in the netlist
	allPins []bsr.PinRef
	pinKeys map[string]bsr.PinRef // Maps pin key back to PinRef
}

// NewNetlist creates a new netlist from a list of pins.
// Initially, each pin is in its own isolated net.
func NewNetlist(pins []bsr.PinRef) *Netlist {
	nl := &Netlist{
		parent:  make(map[string]string),
		rank:    make(map[string]int),
		allPins: make([]bsr.PinRef, len(pins)),
		pinKeys: make(map[string]bsr.PinRef),
	}

	copy(nl.allPins, pins)

	// Initialize union-find: each pin is its own parent
	for _, pin := range pins {
		key := pinKey(pin)
		nl.parent[key] = key
		nl.rank[key] = 0
		nl.pinKeys[key] = pin
	}

	return nl
}

// Connect marks two pins as electrically connected.
// This merges their nets using the union-find algorithm.
func (nl *Netlist) Connect(a, b bsr.PinRef) {
	rootA := nl.Find(a)
	rootB := nl.Find(b)

	if rootA == rootB {
		return // Already in the same net
	}

	keyA := pinKey(rootA)
	keyB := pinKey(rootB)

	// Union by rank
	if nl.rank[keyA] < nl.rank[keyB] {
		nl.parent[keyA] = keyB
	} else if nl.rank[keyA] > nl.rank[keyB] {
		nl.parent[keyB] = keyA
	} else {
		nl.parent[keyB] = keyA
		nl.rank[keyA]++
	}
}

// Find returns the root (representative) pin for the net containing the given pin.
// Uses path compression for O(α(n)) amortized time complexity.
func (nl *Netlist) Find(pin bsr.PinRef) bsr.PinRef {
	key := pinKey(pin)

	// Find root
	root := key
	for nl.parent[root] != root {
		root = nl.parent[root]
	}

	// Path compression: make all nodes on the path point directly to root
	current := key
	for current != root {
		next := nl.parent[current]
		nl.parent[current] = root
		current = next
	}

	return nl.pinKeys[root]
}

// Finalize builds the final net list from the union-find structure.
// This should be called after all Connect() operations are complete.
// Only nets with 2+ pins are included; isolated single-pin "nets" are skipped.
func (nl *Netlist) Finalize() {
	// Group pins by their root
	netMap := make(map[string][]bsr.PinRef)
	for _, pin := range nl.allPins {
		root := nl.Find(pin)
		rootKey := pinKey(root)
		netMap[rootKey] = append(netMap[rootKey], pin)
	}

	// Convert to Net objects (only nets with 2+ pins)
	nl.Nets = make([]*Net, 0, len(netMap))
	netID := 0
	for _, pins := range netMap {
		// Skip single-pin "nets" - they're not actually nets
		if len(pins) < 2 {
			continue
		}
		
		// Sort pins for consistent ordering
		sort.Slice(pins, func(i, j int) bool {
			if pins[i].ChainIndex != pins[j].ChainIndex {
				return pins[i].ChainIndex < pins[j].ChainIndex
			}
			if pins[i].DeviceName != pins[j].DeviceName {
				return pins[i].DeviceName < pins[j].DeviceName
			}
			return pins[i].PinName < pins[j].PinName
		})

		nl.Nets = append(nl.Nets, &Net{
			ID:   netID,
			Pins: pins,
		})
		netID++
	}

	// Sort nets by ID for consistent output
	sort.Slice(nl.Nets, func(i, j int) bool {
		return nl.Nets[i].ID < nl.Nets[j].ID
	})
}

// NetCount returns the number of unique nets.
// Only valid after calling Finalize().
func (nl *Netlist) NetCount() int {
	return len(nl.Nets)
}

// MultiPinNetCount returns the number of nets with more than one pin.
// Only valid after calling Finalize().
func (nl *Netlist) MultiPinNetCount() int {
	count := 0
	for _, net := range nl.Nets {
		if len(net.Pins) > 1 {
			count++
		}
	}
	return count
}

// Clone creates a deep copy of the netlist
func (nl *Netlist) Clone() *Netlist {
	clone := &Netlist{
		parent:  make(map[string]string),
		rank:    make(map[string]int),
		allPins: make([]bsr.PinRef, len(nl.allPins)),
		pinKeys: make(map[string]bsr.PinRef),
		Nets:    make([]*Net, len(nl.Nets)),
	}
	
	// Copy maps
	for k, v := range nl.parent {
		clone.parent[k] = v
	}
	for k, v := range nl.rank {
		clone.rank[k] = v
	}
	for k, v := range nl.pinKeys {
		clone.pinKeys[k] = v
	}
	
	// Copy allPins
	copy(clone.allPins, nl.allPins)
	
	// Deep copy nets
	for i, net := range nl.Nets {
		clonedNet := &Net{
			ID:   net.ID,
			Pins: make([]bsr.PinRef, len(net.Pins)),
		}
		copy(clonedNet.Pins, net.Pins)
		clone.Nets[i] = clonedNet
	}
	
	return clone
}

// ExportJSON exports the netlist to JSON format.
func (nl *Netlist) ExportJSON() ([]byte, error) {
	if nl.Nets == nil {
		return nil, fmt.Errorf("reveng: netlist not finalized")
	}

	output := struct {
		Version    string `json:"version"`
		NetCount   int    `json:"net_count"`
		MultiNets  int    `json:"multi_pin_nets"`
		Nets       []*Net `json:"nets"`
		GeneratedBy string `json:"generated_by"`
	}{
		Version:    "1.0",
		NetCount:   nl.NetCount(),
		MultiNets:  nl.MultiPinNetCount(),
		Nets:       nl.Nets,
		GeneratedBy: "jtag boundary-scan reverse engineering",
	}

	return json.MarshalIndent(output, "", "  ")
}

// ExportKiCad exports the netlist to KiCad netlist format.
// This is a simplified format for basic connectivity.
func (nl *Netlist) ExportKiCad() (string, error) {
	if nl.Nets == nil {
		return "", fmt.Errorf("reveng: netlist not finalized")
	}

	var output string
	output += "(export (version D)\n"
	output += "  (design\n"
	output += "    (source \"JTAG Boundary Scan Reverse Engineering\")\n"
	output += "    (date \"" + "auto-generated" + "\")\n"
	output += "  )\n"
	output += "  (components\n"

	// Group pins by device to create components
	devicePins := make(map[string][]bsr.PinRef)
	for _, net := range nl.Nets {
		for _, pin := range net.Pins {
			key := fmt.Sprintf("%s_%d", pin.DeviceName, pin.ChainIndex)
			devicePins[key] = append(devicePins[key], pin)
		}
	}

	for deviceKey := range devicePins {
		output += fmt.Sprintf("    (comp (ref %s))\n", deviceKey)
	}
	output += "  )\n"

	output += "  (nets\n"
	for _, net := range nl.Nets {
		if len(net.Pins) < 2 {
			continue // Skip single-pin nets
		}
		output += fmt.Sprintf("    (net (code %d) (name Net-%d)\n", net.ID, net.ID)
		for _, pin := range net.Pins {
			compRef := fmt.Sprintf("%s_%d", pin.DeviceName, pin.ChainIndex)
			output += fmt.Sprintf("      (node (ref %s) (pin %s))\n", compRef, pin.PinName)
		}
		output += "    )\n"
	}
	output += "  )\n"
	output += ")\n"

	return output, nil
}

// pinKey generates a unique string key for a PinRef.
func pinKey(pin bsr.PinRef) string {
	return fmt.Sprintf("%d:%s:%s", pin.ChainIndex, pin.DeviceName, pin.PinName)
}
