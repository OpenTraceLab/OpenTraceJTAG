package jtag

import (
	"testing"
)

func TestUSBTransportConstants(t *testing.T) {
	// Verify VID/PID constants
	if VendorIDRaspberryPi != 0x2E8A {
		t.Errorf("Expected VID 0x2E8A, got 0x%04X", VendorIDRaspberryPi)
	}

	if ProductIDCMSISDAP != 0x000C {
		t.Errorf("Expected PID 0x000C, got 0x%04X", ProductIDCMSISDAP)
	}

	// Verify packet size
	if DefaultPacketSize != 64 {
		t.Errorf("Expected packet size 64, got %d", DefaultPacketSize)
	}
}

func TestEnumerateCMSISDAPProbes(t *testing.T) {
	// This will work even if no hardware is connected
	devices, err := EnumerateCMSISDAPProbes()
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}

	// Just log what we found (may be empty if no hardware)
	t.Logf("Found %d CMSIS-DAP device(s)", len(devices))
	for i, dev := range devices {
		t.Logf("  Device %d: VID:0x%04X PID:0x%04X Serial:%s Desc:%s",
			i, dev.VID, dev.PID, dev.SerialNumber, dev.Description)
	}
}

// Integration test - only runs with real hardware
func TestUSBTransportIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Try to open a CMSIS-DAP probe
	transport, err := NewUSBTransport(VendorIDRaspberryPi, ProductIDCMSISDAP)
	if err != nil {
		t.Skipf("No CMSIS-DAP hardware found: %v", err)
	}
	defer transport.Close()

	// Verify packet size was discovered
	packetSize := transport.GetPacketSize()
	if packetSize < 64 {
		t.Errorf("Packet size too small: %d", packetSize)
	}
	t.Logf("Packet size: %d bytes", packetSize)

	// Test basic write/read (DAP_Info Vendor ID command)
	cmd := []byte{0x00, 0x01} // DAP_Info, Vendor ID
	resp, err := transport.WriteRead(cmd)
	if err != nil {
		t.Fatalf("WriteRead failed: %v", err)
	}

	// Response should start with command ID
	if len(resp) < 2 {
		t.Fatalf("Response too short: %d bytes", len(resp))
	}

	if resp[0] != 0x00 {
		t.Errorf("Expected response command ID 0x00, got 0x%02X", resp[0])
	}

	// Byte 1 is the string length
	strLen := int(resp[1])
	if strLen > 0 && len(resp) >= 2+strLen {
		vendor := string(resp[2 : 2+strLen])
		t.Logf("Probe vendor: %s", vendor)
	}
}
