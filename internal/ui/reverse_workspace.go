package newui

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/OpenTraceLab/OpenTraceJTAG/internal/ui/components"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsr"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/reveng"
)

// layoutReverse renders the reverse engineering workspace
func (a *App) layoutReverse(gtx layout.Context) layout.Dimensions {
	// Handle scan button click
	if a.scanChainBtn.Clicked(gtx) {
		go a.startChainScan()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Top action bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutReverseActionBar(gtx)
		}),
		// Main content area
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(a.chainDevices) == 0 {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Body1(a.gvTheme.Theme, "Click 'Scan Chain' to discover devices").Layout(gtx)
				})
			}
			
			// Check if all devices are ready (have footprints assigned)
			allReady := a.allDevicesReady()
			if !allReady {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Body1(a.gvTheme.Theme, "Assign BSDL and footprints to all devices").Layout(gtx)
				})
			}
			
			// Render footprints in zoomable viewport
			return a.layoutFootprintViewport(gtx)
		}),
	)
}

// layoutReverseActionBar renders the action bar with buttons and progress
func (a *App) layoutReverseActionBar(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// Scan Chain button
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(a.gvTheme.Theme, &a.scanChainBtn, "Scan Chain")
				btn.Background = color.NRGBA{R: 63, G: 81, B: 181, A: 255} // Material blue
				if a.isScanning {
					btn.Text = "Scanning..."
					btn.Background = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
				}
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
			// Reverse Engineer button
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				// Check if all devices have BSDL and footprint assigned
				canStart := len(a.chainDevices) > 0 && a.allDevicesReady()
				
				// Check for click first
				if a.reverseEngineerBtn.Clicked(gtx) && canStart && !a.isReverseEngineering {
					go a.startReverseEngineering()
				}
				
				btn := material.Button(a.gvTheme.Theme, &a.reverseEngineerBtn, "Start Reverse Engineering")
				if !canStart {
					btn.Background = color.NRGBA{R: 150, G: 150, B: 150, A: 255} // Grayed out
				} else {
					btn.Background = color.NRGBA{R: 76, G: 175, B: 80, A: 255} // Green
				}
				if a.isReverseEngineering {
					btn.Text = "Reverse Engineering..."
					btn.Background = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
				}
				
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
		)
	})
}

// startChainScan initiates JTAG chain scanning
func (a *App) startChainScan() {
	a.isScanning = true
	a.scanProgress = "Initializing"
	a.chainDevices = nil // Clear old devices
	
	// Auto-expand right panel when scan starts
	if !a.rightPanelVisible {
		a.rightPanelVisible = true
	}
	
	a.window.Invalidate()

	// Animate dots during scan
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				a.window.Invalidate()
			}
		}
	}()

	// Simulate progressive device discovery
	time.Sleep(300 * time.Millisecond)
	
	// Always create fresh simulator to avoid stale data
	testdataPath := "testdata"
	sim, err := jtag.BuildSimple2DeviceScenario(testdataPath)
	if err != nil {
		a.scanProgress = fmt.Sprintf("Error: %v", err)
		a.isScanning = false
		done <- true
		a.window.Invalidate()
		return
	}
	a.chainSimulator = sim

	// Create chain controller with repository
	a.chainRepository = chain.NewMemoryRepository()
	a.chainController = chain.NewController(a.chainSimulator.Adapter(), a.chainRepository)

	// Discover devices
	a.scanProgress = "Discovering devices"
	a.window.Invalidate()
	time.Sleep(200 * time.Millisecond)

	deviceCount := a.chainSimulator.GetDeviceCount()

	// Add devices progressively with animation
	for i := 0; i < deviceCount; i++ {
		a.scanProgress = fmt.Sprintf("Found device %d/%d", i+1, deviceCount)
		
		dev, err := a.chainSimulator.GetDevice(i)
		if err != nil {
			continue
		}

		idInfo := jtag.DecodeIDCode(dev.IDCode)
		chainDev := ChainDevice{
			Index:      i,
			IDCode:     dev.IDCode,
			IDCodeInfo: idInfo,
			Name:       "", // Don't show name until BSDL is assigned
			IRLength:   dev.IRLength,
			State:      "discovered",
		}
		
		a.chainDevices = append(a.chainDevices, chainDev)
		a.window.Invalidate()
		time.Sleep(300 * time.Millisecond) // Animate card appearance
	}

	a.scanProgress = fmt.Sprintf("Scan complete - %d devices found", deviceCount)
	a.isScanning = false
	done <- true
	a.window.Invalidate()

	// Clear progress after 2 seconds
	time.Sleep(2 * time.Second)
	a.scanProgress = ""
	a.window.Invalidate()
}

// startReverseEngineering initiates the reverse engineering process
func (a *App) startReverseEngineering() {
	a.isReverseEngineering = true
	a.reverseProgress = "Initializing"
	a.ratsnestLines = nil
	a.window.Invalidate()

	// Reset simulator state
	if a.chainSimulator != nil {
		a.chainSimulator.Reset()
	}

	// Discover chain with BSDL files now that they're assigned
	deviceCount := len(a.chainDevices)
	ch, err := a.chainController.Discover(deviceCount)
	if err != nil {
		a.Logf("[REVENG] Failed to discover chain: %v", err)
		a.reverseProgress = fmt.Sprintf("Error: %v", err)
		a.isReverseEngineering = false
		a.window.Invalidate()
		return
	}
	a.chainDiscovered = ch

	// Create BSR controller
	ctl, err := bsr.NewController(a.chainDiscovered)
	if err != nil {
		a.Logf("[REVENG] Failed to create BSR controller: %v", err)
		a.reverseProgress = fmt.Sprintf("Error: %v", err)
		a.isReverseEngineering = false
		a.window.Invalidate()
		return
	}

	// Configure reverse engineering
	cfg := reveng.DefaultConfig()
	cfg.SkipKnownJTAGPins = true
	cfg.SkipPowerPins = true

	// Create progress channel
	progressCh := make(chan reveng.Progress, 10)
	
	// Start discovery in goroutine
	go func() {
		ctx := context.Background()
		netlist, err := reveng.DiscoverNetlist(ctx, ctl, cfg, progressCh)
		if err != nil {
			a.Logf("[REVENG] Discovery failed: %v", err)
			a.reverseProgress = fmt.Sprintf("Error: %v", err)
			a.isReverseEngineering = false
			a.window.Invalidate()
			return
		}
		
		a.discoveredNetlist = netlist
		a.Logf("[REVENG] Discovery complete: %d nets, %d multi-pin nets", 
			netlist.NetCount(), netlist.MultiPinNetCount())
		a.reverseProgress = fmt.Sprintf("Complete: %d nets found", netlist.MultiPinNetCount())
		a.isReverseEngineering = false
		a.window.Invalidate()
		
		// Clear progress after 3 seconds
		time.Sleep(3 * time.Second)
		a.reverseProgress = ""
		a.window.Invalidate()
	}()

	// Process progress updates
	go a.processReverseProgress(progressCh)
}

// processReverseProgress handles progress updates from reverse engineering
func (a *App) processReverseProgress(progressCh <-chan reveng.Progress) {
	for progress := range progressCh {
		switch progress.Phase {
		case "init":
			a.reverseProgress = "Initializing EXTEST mode"
			// Set all non-power pins to Hi-Z
			for i := range a.chainDevices {
				device := &a.chainDevices[i]
				if device.PinStates == nil {
					device.PinStates = make(map[int]string)
				}
				for pinNum := 1; pinNum <= device.PinCount; pinNum++ {
					pinName := device.GetPinName(pinNum)
					// Skip power/ground pins
					if pinName != "" && device.IsPowerPin(pinName) {
						continue
					}
					device.SetPinState(pinNum, string(components.PinStateHighZ))
				}
			}
			
		case "scanning":
			// Update progress text
			a.reverseProgress = fmt.Sprintf("Scanning %s (%d/%d)", 
				progress.Driver.PinName, progress.Index+1, progress.Total)
			a.currentScanPin = fmt.Sprintf("%s.%s", progress.Driver.DeviceName, progress.Driver.PinName)
			
			// Reset all non-power pins to Hi-Z (matches BSR controller behavior)
			for i := range a.chainDevices {
				dev := &a.chainDevices[i]
				if dev.PinStates != nil {
					for pinNum := range dev.PinStates {
						pinName := dev.GetPinName(pinNum)
						if pinName != "" && dev.IsPowerPin(pinName) {
							continue
						}
						dev.SetPinState(pinNum, string(components.PinStateHighZ))
					}
				}
			}
			
			// Find device and pin
			deviceIdx := progress.Driver.ChainIndex
			if deviceIdx >= 0 && deviceIdx < len(a.chainDevices) {
				device := &a.chainDevices[deviceIdx]
				
				// Find pin number from name
				pinNum := a.findPinNumber(device, progress.Driver.PinName)
				if pinNum > 0 {
					// Animate: Hi-Z -> Low -> High -> Low
					device.SetPinState(pinNum, string(components.PinStateLow))
					a.window.Invalidate()
					time.Sleep(50 * time.Millisecond)
					
					device.SetPinState(pinNum, string(components.PinStateHigh))
					a.window.Invalidate()
					time.Sleep(50 * time.Millisecond)
					
					device.SetPinState(pinNum, string(components.PinStateLow))
					a.window.Invalidate()
				}
			}
			
		case "finalizing":
			a.reverseProgress = "Building ratsnest"
			// Build ratsnest from discovered netlist
			if a.discoveredNetlist != nil {
				a.buildRatsnest()
			}
			
			// Reset all non-power pins to Hi-Z when scan complete
			for i := range a.chainDevices {
				device := &a.chainDevices[i]
				if device.PinStates != nil {
					for pinNum := range device.PinStates {
						pinName := device.GetPinName(pinNum)
						// Skip power/ground pins
						if pinName != "" && device.IsPowerPin(pinName) {
							continue
						}
						device.SetPinState(pinNum, string(components.PinStateHighZ))
					}
				}
			}
		}
		
		a.window.Invalidate()
	}
}

// findPinNumber finds the pin number for a given pin name in a device
func (a *App) findPinNumber(device *ChainDevice, pinName string) int {
	// First try to parse as a number directly (package pin number)
	var pinNum int
	if n, err := fmt.Sscanf(pinName, "%d", &pinNum); err == nil && n == 1 {
		return pinNum
	}
	
	// Otherwise look up in pin map (port name -> package pin number)
	if device.BSDLFile == nil || device.BSDLFile.Entity == nil {
		a.Logf("[REVENG] findPinNumber: device has no BSDL file for pin %s", pinName)
		return 0
	}
	
	pinMap := device.BSDLFile.Entity.GetPinMap()
	for name, pinStr := range pinMap {
		if name == pinName {
			fmt.Sscanf(pinStr, "%d", &pinNum)
			return pinNum
		}
	}
	a.Logf("[REVENG] Pin %s not found in pin map (have %d entries)", pinName, len(pinMap))
	return 0
}

// buildRatsnest converts the discovered netlist into visual ratsnest lines
func (a *App) buildRatsnest() {
	if a.discoveredNetlist == nil {
		return
	}
	
	a.ratsnestLines = nil
	
	// Generate colors for nets
	colors := []color.NRGBA{
		{R: 255, G: 100, B: 100, A: 200}, // Red
		{R: 100, G: 255, B: 100, A: 200}, // Green
		{R: 100, G: 100, B: 255, A: 200}, // Blue
		{R: 255, G: 255, B: 100, A: 200}, // Yellow
		{R: 255, G: 100, B: 255, A: 200}, // Magenta
		{R: 100, G: 255, B: 255, A: 200}, // Cyan
	}
	
	for _, net := range a.discoveredNetlist.Nets {
		if len(net.Pins) < 2 {
			a.Logf("[REVENG] Skipping net %d - only %d pin(s)", net.ID, len(net.Pins))
			continue // Skip single-pin nets
		}
		
		a.Logf("[REVENG] Building ratsnest for net %d with %d pins", net.ID, len(net.Pins))
		netColor := colors[net.ID % len(colors)]
		
		linesCreated := 0
		
		// Create star topology: connect first pin to all others
		// This is clearer than full mesh for multi-pin nets
		if len(net.Pins) > 0 {
			pinA := net.Pins[0]
			
			if pinA.ChainIndex >= len(a.chainDevices) {
				a.Logf("[REVENG] Invalid device index in net %d", net.ID)
				continue
			}
			
			deviceA := &a.chainDevices[pinA.ChainIndex]
			pinNumA := a.findPinNumber(deviceA, pinA.PinName)
			
			if pinNumA == 0 {
				a.Logf("[REVENG] Net %d: Could not find pin number for %s (dev%d)", 
					net.ID, pinA.PinName, pinA.ChainIndex)
				continue
			}
			
			// Connect first pin to all others
			for j := 1; j < len(net.Pins); j++ {
				pinB := net.Pins[j]
				
				if pinB.ChainIndex >= len(a.chainDevices) {
					a.Logf("[REVENG] Invalid device index in net %d", net.ID)
					continue
				}
				
				deviceB := &a.chainDevices[pinB.ChainIndex]
				pinNumB := a.findPinNumber(deviceB, pinB.PinName)
				
				if pinNumB > 0 {
					a.ratsnestLines = append(a.ratsnestLines, RatsnestLine{
						DeviceA: pinA.ChainIndex,
						PinA:    pinNumA,
						DeviceB: pinB.ChainIndex,
						PinB:    pinNumB,
						NetID:   net.ID,
						Color:   netColor,
					})
					linesCreated++
				} else {
					a.Logf("[REVENG] Net %d: Could not find pin number for %s (dev%d)", 
						net.ID, pinB.PinName, pinB.ChainIndex)
				}
			}
		}
		
		a.Logf("[REVENG] Net %d: Created %d ratsnest lines", net.ID, linesCreated)
	}
	
	a.Logf("[REVENG] Built %d ratsnest lines from %d nets", len(a.ratsnestLines), len(a.discoveredNetlist.Nets))
}

// exportNetlistKiCad exports the discovered netlist to KiCad format
func (a *App) exportNetlistKiCad() {
	if a.discoveredNetlist == nil {
		a.Logf("[REVENG] No netlist to export")
		return
	}
	
	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("netlist_%s.net", timestamp)
	
	a.Logf("[REVENG] Exporting netlist to %s...", filename)
	
	// Export to KiCad format
	kicadData, err := a.discoveredNetlist.ExportKiCad()
	if err != nil {
		a.Logf("[REVENG] Export failed: %v", err)
		return
	}
	
	// Write to file
	err = os.WriteFile(filename, []byte(kicadData), 0644)
	if err != nil {
		a.Logf("[REVENG] Failed to write file: %v", err)
		return
	}
	
	a.Logf("[REVENG] ✓ Netlist exported to %s (%d nets, %d multi-pin)", 
		filename, a.discoveredNetlist.NetCount(), a.discoveredNetlist.MultiPinNetCount())
}


// layoutDeviceFootprint renders a single device footprint
func (a *App) layoutDeviceFootprint(gtx layout.Context, deviceIndex int) layout.Dimensions {
	device := &a.chainDevices[deviceIndex]
	
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		// Device label
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := fmt.Sprintf("U%d: %s", deviceIndex+1, device.Name)
			if device.Name == "" {
				label = fmt.Sprintf("U%d", deviceIndex+1)
			}
			return material.Body2(a.gvTheme.Theme, label).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		// Footprint rendering
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// TODO: Actually render the footprint using components package
			// For now, show a placeholder
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				info := fmt.Sprintf("%s\n%d pins", device.FootprintType, device.PinCount)
				return material.Caption(a.gvTheme.Theme, info).Layout(gtx)
			})
		}),
	)
}
