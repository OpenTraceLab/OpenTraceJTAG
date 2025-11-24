package jtag

import (
	"fmt"
	"time"

	"github.com/google/gousb"
)

const (
	// JTAGProbe USB identifiers
	VendorIDRaspberryPi = 0x2E8A
	ProductIDCMSISDAP   = 0x000C

	// Endpoint addresses (standard for CMSIS-DAP)
	// Note: Actual endpoint addresses should be discovered from descriptors
	// These are typical values
	EndpointOUT = 0x01 // Bulk OUT for commands
	EndpointIN  = 0x81 // Bulk IN for responses

	// Default packet size for CMSIS-DAP v1/v2
	DefaultPacketSize = 64
	DefaultTimeout    = 5 * time.Second
)

// USBTransport handles USB communication with CMSIS-DAP probe
type USBTransport struct {
	ctx  *gousb.Context
	dev  *gousb.Device
	intf *gousb.Interface

	epOut *gousb.OutEndpoint
	epIn  *gousb.InEndpoint

	packetSize int
	timeout    time.Duration

	vid uint16
	pid uint16
}

// NewUSBTransport creates a USB transport for CMSIS-DAP
func NewUSBTransport(vid, pid uint16) (*USBTransport, error) {
	ctx := gousb.NewContext()

	// Find and open device
	dev, err := ctx.OpenDeviceWithVIDPID(gousb.ID(vid), gousb.ID(pid))
	if err != nil {
		ctx.Close()
		return nil, fmt.Errorf("USB error: %w", err)
	}
	if dev == nil {
		ctx.Close()
		return nil, fmt.Errorf("device not found (VID:0x%04X PID:0x%04X)", vid, pid)
	}

	// Set auto-detach kernel driver (important for Linux)
	if err := dev.SetAutoDetach(true); err != nil {
		// Not fatal on all platforms
		// Continue anyway
	}

	transport := &USBTransport{
		ctx:        ctx,
		dev:        dev,
		packetSize: DefaultPacketSize,
		timeout:    DefaultTimeout,
		vid:        vid,
		pid:        pid,
	}

	// Claim interface and find endpoints
	if err := transport.claimInterface(); err != nil {
		dev.Close()
		ctx.Close()
		return nil, err
	}

	return transport, nil
}

// claimInterface finds and claims the CMSIS-DAP vendor interface
func (t *USBTransport) claimInterface() error {
	// CMSIS-DAP typically uses interface 0 (vendor class)
	// But we should search for the vendor interface to be safe
	cfg, err := t.dev.Config(1)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Find vendor interface (class 0xFF)
	var vendorIntfNum int = -1
	for _, intf := range cfg.Desc.Interfaces {
		if len(intf.AltSettings) > 0 {
			alt := intf.AltSettings[0]
			// CMSIS-DAP uses vendor-specific class (0xFF)
			if alt.Class == gousb.ClassVendorSpec {
				vendorIntfNum = intf.Number
				break
			}
		}
	}

	if vendorIntfNum == -1 {
		// Fall back to interface 0
		vendorIntfNum = 0
	}

	// Claim the interface
	intf, err := cfg.Interface(vendorIntfNum, 0)
	if err != nil {
		return fmt.Errorf("failed to claim interface %d: %w", vendorIntfNum, err)
	}
	t.intf = intf

	// Find bulk endpoints
	if err := t.findEndpoints(); err != nil {
		intf.Close()
		return err
	}

	return nil
}

// findEndpoints discovers the bulk IN and OUT endpoints
func (t *USBTransport) findEndpoints() error {
	// Get interface setting
	setting := t.intf.Setting

	// Find bulk OUT endpoint (host -> device)
	var outAddr int
	for _, ep := range setting.Endpoints {
		if ep.TransferType == gousb.TransferTypeBulk {
			if ep.Direction == gousb.EndpointDirectionOut {
				outAddr = ep.Number
				break
			}
		}
	}

	if outAddr == 0 {
		return fmt.Errorf("bulk OUT endpoint not found")
	}

	// Find bulk IN endpoint (device -> host)
	var inAddr int
	for _, ep := range setting.Endpoints {
		if ep.TransferType == gousb.TransferTypeBulk {
			if ep.Direction == gousb.EndpointDirectionIn {
				inAddr = ep.Number
				t.packetSize = ep.MaxPacketSize
				break
			}
		}
	}

	if inAddr == 0 {
		return fmt.Errorf("bulk IN endpoint not found")
	}

	// Open endpoints
	epOut, err := t.intf.OutEndpoint(outAddr)
	if err != nil {
		return fmt.Errorf("failed to open OUT endpoint: %w", err)
	}
	t.epOut = epOut

	epIn, err := t.intf.InEndpoint(inAddr)
	if err != nil {
		return fmt.Errorf("failed to open IN endpoint: %w", err)
	}
	t.epIn = epIn

	return nil
}

// Write sends a command packet to the probe
func (t *USBTransport) Write(data []byte) (int, error) {
	// CMSIS-DAP packets are fixed size, pad if necessary
	packet := make([]byte, t.packetSize)
	copy(packet, data)

	n, err := t.epOut.Write(packet)
	if err != nil {
		return 0, fmt.Errorf("USB write failed: %w", err)
	}

	return n, nil
}

// Read receives a response packet from the probe
func (t *USBTransport) Read(data []byte) (int, error) {
	n, err := t.epIn.Read(data)
	if err != nil {
		return 0, fmt.Errorf("USB read failed: %w", err)
	}
	return n, nil
}

// WriteRead performs a command/response transaction
func (t *USBTransport) WriteRead(cmd []byte) ([]byte, error) {
	// Write command
	if _, err := t.Write(cmd); err != nil {
		return nil, err
	}

	// Read response with same packet size
	resp := make([]byte, t.packetSize)
	n, err := t.Read(resp)
	if err != nil {
		return nil, err
	}

	return resp[:n], nil
}

// GetPacketSize returns the current packet size
func (t *USBTransport) GetPacketSize() int {
	return t.packetSize
}

// SetTimeout sets the read/write timeout
func (t *USBTransport) SetTimeout(timeout time.Duration) {
	t.timeout = timeout
}

// Close releases USB resources
func (t *USBTransport) Close() error {
	if t.intf != nil {
		t.intf.Close()
		t.intf = nil
	}
	if t.dev != nil {
		t.dev.Close()
		t.dev = nil
	}
	if t.ctx != nil {
		t.ctx.Close()
		t.ctx = nil
	}
	return nil
}

// DeviceInfo represents a discovered USB device
type DeviceInfo struct {
	VID          uint16
	PID          uint16
	SerialNumber string
	Path         string
	Description  string
}

// EnumerateCMSISDAPProbes finds all connected CMSIS-DAP devices
func EnumerateCMSISDAPProbes() ([]DeviceInfo, error) {
	ctx := gousb.NewContext()
	defer ctx.Close()

	devices := make([]DeviceInfo, 0)

	// List all USB devices
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		// Match CMSIS-DAP VID:PID
		return desc.Vendor == VendorIDRaspberryPi && desc.Product == ProductIDCMSISDAP
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate devices: %w", err)
	}

	// Get info from each device
	for _, dev := range devs {
		serial, _ := dev.SerialNumber()
		manufacturer, _ := dev.Manufacturer()
		product, _ := dev.Product()

		info := DeviceInfo{
			VID:          uint16(dev.Desc.Vendor),
			PID:          uint16(dev.Desc.Product),
			SerialNumber: serial,
			Description:  fmt.Sprintf("%s %s", manufacturer, product),
		}

		devices = append(devices, info)
		dev.Close()
	}

	return devices, nil
}
