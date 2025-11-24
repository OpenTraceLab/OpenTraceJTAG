package reveng

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsr"
)

func TestNewNetlist(t *testing.T) {
	pins := []bsr.PinRef{
		{ChainIndex: 0, DeviceName: "DEV0", PinName: "A0"},
		{ChainIndex: 0, DeviceName: "DEV0", PinName: "A1"},
		{ChainIndex: 1, DeviceName: "DEV1", PinName: "B0"},
	}

	nl := NewNetlist(pins)

	if len(nl.allPins) != 3 {
		t.Errorf("expected 3 pins, got %d", len(nl.allPins))
	}

	// Initially, each pin should be its own parent (isolated)
	for _, pin := range pins {
		root := nl.Find(pin)
		if root != pin {
			t.Errorf("pin %s should be its own root initially", pinKey(pin))
		}
	}
}

func TestConnect(t *testing.T) {
	pins := []bsr.PinRef{
		{ChainIndex: 0, DeviceName: "DEV0", PinName: "A0"},
		{ChainIndex: 0, DeviceName: "DEV0", PinName: "A1"},
		{ChainIndex: 0, DeviceName: "DEV0", PinName: "A2"},
	}

	nl := NewNetlist(pins)

	// Connect A0 and A1
	nl.Connect(pins[0], pins[1])

	// They should now have the same root
	root0 := nl.Find(pins[0])
	root1 := nl.Find(pins[1])
	if root0 != root1 {
		t.Errorf("A0 and A1 should have same root after Connect")
	}

	// A2 should still be separate
	root2 := nl.Find(pins[2])
	if root2 == root0 {
		t.Errorf("A2 should have different root from A0/A1")
	}

	// Connect A1 and A2 (transitive: A0-A1-A2)
	nl.Connect(pins[1], pins[2])

	// Now all three should have the same root
	root0 = nl.Find(pins[0])
	root1 = nl.Find(pins[1])
	root2 = nl.Find(pins[2])

	if root0 != root1 || root1 != root2 {
		t.Errorf("all pins should have same root after transitive connection")
	}
}

func TestFinalize(t *testing.T) {
	pins := []bsr.PinRef{
		{ChainIndex: 0, DeviceName: "DEV0", PinName: "A0"},
		{ChainIndex: 0, DeviceName: "DEV0", PinName: "A1"},
		{ChainIndex: 0, DeviceName: "DEV0", PinName: "A2"},
		{ChainIndex: 1, DeviceName: "DEV1", PinName: "B0"},
	}

	nl := NewNetlist(pins)

	// Create net: A0-A1
	nl.Connect(pins[0], pins[1])

	// A2 and B0 remain isolated (single-pin nets are not included)

	nl.Finalize()

	// Should have 1 net (only multi-pin nets are included, isolated pins are skipped)
	// The net is {A0, A1}
	if nl.NetCount() != 1 {
		t.Errorf("expected 1 net, got %d", nl.NetCount())
	}

	// Should have 1 multi-pin net
	if nl.MultiPinNetCount() != 1 {
		t.Errorf("expected 1 multi-pin net, got %d", nl.MultiPinNetCount())
	}

	// Find the multi-pin net
	var multiNet *Net
	for _, net := range nl.Nets {
		if len(net.Pins) > 1 {
			multiNet = net
			break
		}
	}

	if multiNet == nil {
		t.Fatalf("multi-pin net not found")
	}

	if len(multiNet.Pins) != 2 {
		t.Errorf("multi-pin net should have 2 pins, got %d", len(multiNet.Pins))
	}

	// Verify pins are sorted
	if multiNet.Pins[0].PinName != "A0" || multiNet.Pins[1].PinName != "A1" {
		t.Errorf("pins not sorted correctly: got %v", multiNet.Pins)
	}
}

func TestExportJSON(t *testing.T) {
	pins := []bsr.PinRef{
		{ChainIndex: 0, DeviceName: "DEV0", PinName: "A0"},
		{ChainIndex: 0, DeviceName: "DEV0", PinName: "A1"},
	}

	nl := NewNetlist(pins)
	nl.Connect(pins[0], pins[1])
	nl.Finalize()

	jsonData, err := nl.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	// Parse JSON to verify structure
	var output struct {
		Version    string `json:"version"`
		NetCount   int    `json:"net_count"`
		MultiNets  int    `json:"multi_pin_nets"`
		Nets       []Net  `json:"nets"`
		GeneratedBy string `json:"generated_by"`
	}

	if err := json.Unmarshal(jsonData, &output); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if output.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", output.Version)
	}

	if output.NetCount != 1 {
		t.Errorf("expected 1 net, got %d", output.NetCount)
	}

	if output.MultiNets != 1 {
		t.Errorf("expected 1 multi-pin net, got %d", output.MultiNets)
	}

	if len(output.Nets) != 1 {
		t.Fatalf("expected 1 net in array, got %d", len(output.Nets))
	}

	if len(output.Nets[0].Pins) != 2 {
		t.Errorf("expected 2 pins in net, got %d", len(output.Nets[0].Pins))
	}
}

func TestExportKiCad(t *testing.T) {
	pins := []bsr.PinRef{
		{ChainIndex: 0, DeviceName: "U1", PinName: "PA0"},
		{ChainIndex: 1, DeviceName: "U2", PinName: "PB0"},
	}

	nl := NewNetlist(pins)
	nl.Connect(pins[0], pins[1])
	nl.Finalize()

	kicad, err := nl.ExportKiCad()
	if err != nil {
		t.Fatalf("ExportKiCad failed: %v", err)
	}

	// Verify KiCad format structure
	if !strings.Contains(kicad, "(export") {
		t.Errorf("KiCad export missing (export header")
	}

	if !strings.Contains(kicad, "(components") {
		t.Errorf("KiCad export missing (components section")
	}

	if !strings.Contains(kicad, "(nets") {
		t.Errorf("KiCad export missing (nets section")
	}

	// Should contain component references
	if !strings.Contains(kicad, "U1_0") {
		t.Errorf("KiCad export missing component U1_0")
	}

	if !strings.Contains(kicad, "U2_1") {
		t.Errorf("KiCad export missing component U2_1")
	}

	// Should contain pin references
	if !strings.Contains(kicad, "PA0") {
		t.Errorf("KiCad export missing pin PA0")
	}

	if !strings.Contains(kicad, "PB0") {
		t.Errorf("KiCad export missing pin PB0")
	}
}

func TestPinKey(t *testing.T) {
	pin := bsr.PinRef{
		ChainIndex: 2,
		DeviceName: "STM32F103",
		PinName:    "PA0",
	}

	key := pinKey(pin)
	expected := "2:STM32F103:PA0"

	if key != expected {
		t.Errorf("pinKey() = %s, want %s", key, expected)
	}
}

func TestUnionFindPerformance(t *testing.T) {
	// Create a large set of pins
	numPins := 1000
	pins := make([]bsr.PinRef, numPins)
	for i := 0; i < numPins; i++ {
		pins[i] = bsr.PinRef{
			ChainIndex: i / 100,
			DeviceName: "DEV",
			PinName:    string(rune('A' + (i % 26))),
		}
	}

	nl := NewNetlist(pins)

	// Connect in a chain: 0-1, 1-2, 2-3, ..., 998-999
	for i := 0; i < numPins-1; i++ {
		nl.Connect(pins[i], pins[i+1])
	}

	// All pins should now be in the same net
	root := nl.Find(pins[0])
	for i := 1; i < numPins; i++ {
		if nl.Find(pins[i]) != root {
			t.Errorf("pin %d not in the same net as pin 0", i)
		}
	}

	nl.Finalize()

	if nl.NetCount() != 1 {
		t.Errorf("expected 1 net after connecting all pins, got %d", nl.NetCount())
	}

	if len(nl.Nets[0].Pins) != numPins {
		t.Errorf("expected %d pins in net, got %d", numPins, len(nl.Nets[0].Pins))
	}
}
