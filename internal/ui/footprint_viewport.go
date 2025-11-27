package newui

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/OpenTraceLab/OpenTraceJTAG/internal/ui/components"
)

// layoutFootprintViewport renders footprints in a zoomable/pannable viewport
func (a *App) layoutFootprintViewport(gtx layout.Context) layout.Dimensions {
	// Handle keyboard shortcuts for zoom and reset
	for {
		ev, ok := gtx.Event(key.Filter{Name: "+", Optional: key.ModShift})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			a.Logf("[VIEWPORT] Zoom in (+ key)")
			a.viewportZoom *= 1.2 // Multiply by 1.2
			if a.viewportZoom > 3.0 {
				a.viewportZoom = 3.0
			}
			gtx.Execute(op.InvalidateCmd{})
		}
	}
	
	for {
		ev, ok := gtx.Event(key.Filter{Name: "-"})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			a.Logf("[VIEWPORT] Zoom out (- key)")
			a.viewportZoom /= 1.2 // Divide by 1.2 (inverse of multiply)
			if a.viewportZoom < 0.5 {
				a.viewportZoom = 0.5
			}
			// Snap to 1.0 if very close
			if a.viewportZoom > 0.999 && a.viewportZoom < 1.001 {
				a.viewportZoom = 1.0
			}
			gtx.Execute(op.InvalidateCmd{})
		}
	}
	
	for {
		ev, ok := gtx.Event(key.Filter{Name: "R"})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			a.Logf("[VIEWPORT] Reset view (R key)")
			a.resetViewport()
			gtx.Execute(op.InvalidateCmd{})
		}
	}
	
	// Get current viewport size for dynamic pan limits
	viewportWidth := float32(gtx.Constraints.Max.X)
	viewportHeight := float32(gtx.Constraints.Max.Y)
	
	// Handle click for right-click detection
	for {
		_, ok := a.viewportClick.Update(gtx.Source)
		if !ok {
			break
		}
		// Click gesture doesn't distinguish buttons well, skip this approach
	}
	
	// Handle scroll for zoom
	dist := a.viewportScroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical, pointer.ScrollRange{}, pointer.ScrollRange{})
	if dist != 0 {
		a.viewportZoom *= 1.0 + float32(dist)*0.1
		if a.viewportZoom < 0.5 {
			a.viewportZoom = 0.5
		} else if a.viewportZoom > 3.0 {
			a.viewportZoom = 3.0
		}
		gtx.Execute(op.InvalidateCmd{})
	}
	
	// Handle raw pointer events for right-click
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: &a.viewportDrag,
			Kinds:  pointer.Press | pointer.Release | pointer.Drag | pointer.Scroll,
		})
		if !ok {
			break
		}
		
		if pev, ok := ev.(pointer.Event); ok {
			// Handle left-click press to start drag
			if pev.Kind == pointer.Press && pev.Buttons == pointer.ButtonPrimary {
				// Check if clicking on a pad first
				centerX := float32(gtx.Constraints.Max.X) / 2
				centerY := float32(gtx.Constraints.Max.Y) / 2
				clickX := pev.Position.X
				clickY := pev.Position.Y
				worldX := ((clickX - centerX - a.viewportPanX) / a.viewportZoom) + centerX
				worldY := ((clickY - centerY - a.viewportPanY) / a.viewportZoom) + centerY
				transformedPos := image.Pt(int(worldX), int(worldY))
				
				clickedPad := false
				for _, pad := range a.renderedPads {
					if transformedPos.In(pad.Bounds) {
						// Find net containing this pad
						a.selectedNetID = -1
						if a.discoveredNetlist != nil {
							for _, net := range a.discoveredNetlist.Nets {
								for _, pin := range net.Pins {
									if pin.ChainIndex == pad.DeviceIndex && a.findPinNumber(&a.chainDevices[pad.DeviceIndex], pin.PinName) == pad.PinNumber {
										a.selectedNetID = net.ID
										a.Logf("[VIEWPORT] Selected net %d", net.ID)
										break
									}
								}
								if a.selectedNetID != -1 {
									break
								}
							}
						}
						clickedPad = true
						gtx.Execute(op.InvalidateCmd{})
						break
					}
				}
				
				if !clickedPad {
					a.selectedNetID = -1 // Deselect if clicking background
					a.contextMenuVisible = false // Dismiss context menu
					a.viewportDragging = true
					a.viewportDragStart = pev.Position
				}
			}
			
			// Handle right-click for context menu
			if pev.Kind == pointer.Press && pev.Buttons == pointer.ButtonSecondary {
				// Get viewport center
				centerX := float32(gtx.Constraints.Max.X) / 2
				centerY := float32(gtx.Constraints.Max.Y) / 2
				
				// Reverse the transform: click -> world coordinates
				// Transform is: Offset(-center) -> Scale(zoom) -> Offset(center+pan)
				// Reverse: subtract (center+pan), divide by zoom, add center
				clickX := pev.Position.X
				clickY := pev.Position.Y
				
				worldX := ((clickX - centerX - a.viewportPanX) / a.viewportZoom) + centerX
				worldY := ((clickY - centerY - a.viewportPanY) / a.viewportZoom) + centerY
				
				transformedPos := image.Pt(int(worldX), int(worldY))
				
				// Check if clicking on any pad
				foundPad := false
				for _, pad := range a.renderedPads {
					if transformedPos.In(pad.Bounds) {
						a.contextMenuVisible = true
						a.contextMenuPos = pev.Position
						a.contextMenuDevice = pad.DeviceIndex
						a.contextMenuPin = pad.PinNumber
						a.Logf("[VIEWPORT] Right-click on pin %d (%s)", pad.PinNumber, pad.PinName)
						foundPad = true
						break
					}
				}
				
				if !foundPad {
					a.contextMenuVisible = false
				}
				
				gtx.Execute(op.InvalidateCmd{})
			}
			
			// Handle release to stop drag
			if pev.Kind == pointer.Release {
				a.viewportDragging = false
			}
			
			// Handle X11/Wayland scroll wheel as button 4/5 events
			if pev.Kind == pointer.Press {
				if pev.Buttons == pointer.Buttons(4) { // Scroll up
					a.Logf("[VIEWPORT] Scroll up (button 4)")
					a.viewportZoom *= 1.1
					if a.viewportZoom > 3.0 {
						a.viewportZoom = 3.0
					}
					gtx.Execute(op.InvalidateCmd{})
				} else if pev.Buttons == pointer.Buttons(5) { // Scroll down
					a.Logf("[VIEWPORT] Scroll down (button 5)")
					a.viewportZoom *= 0.9
					if a.viewportZoom < 0.5 {
						a.viewportZoom = 0.5
					}
					gtx.Execute(op.InvalidateCmd{})
				}
			}
			
			// Handle drag movement
			if pev.Kind == pointer.Drag && a.viewportDragging && pev.Buttons == pointer.ButtonPrimary {
				deltaX := pev.Position.X - a.viewportDragStart.X
				deltaY := pev.Position.Y - a.viewportDragStart.Y
				
				newPanX := a.viewportPanX + deltaX
				newPanY := a.viewportPanY + deltaY
				
				// Dynamic pan limits
				maxPanX := viewportWidth * 1.5
				minPanX := -viewportWidth * 2.0
				maxPanY := viewportHeight * 1.5
				minPanY := -viewportHeight * 2.0
				
				if newPanX > maxPanX {
					newPanX = maxPanX
				} else if newPanX < minPanX {
					newPanX = minPanX
				}
				
				if newPanY > maxPanY {
					newPanY = maxPanY
				} else if newPanY < minPanY {
					newPanY = minPanY
				}
				
				a.viewportPanX = newPanX
				a.viewportPanY = newPanY
				a.viewportDragStart = pev.Position
				gtx.Execute(op.InvalidateCmd{})
			}
			
			// Right-click is handled separately for context menu (see below)
			
			// Handle scroll wheel zoom (works on native Windows/Mac, not WSL2/X11)
			if pev.Kind == pointer.Scroll && pev.Scroll.Y != 0 {
				a.viewportZoom *= 1.0 + pev.Scroll.Y*0.1
				if a.viewportZoom < 0.5 {
					a.viewportZoom = 0.5
				} else if a.viewportZoom > 3.0 {
					a.viewportZoom = 3.0
				}
				gtx.Execute(op.InvalidateCmd{})
			}
		}
	}
	
	// Handle drag for pan - process all pointer events
	for {
		ev, ok := a.viewportDrag.Update(gtx.Metric, gtx.Source, gesture.Axis(3)) // Both axes
		if !ok {
			break
		}
		
		// Process event without logging every frame
		
		switch ev.Kind {
		case pointer.Press:
			if ev.Buttons == pointer.ButtonPrimary {
				// Start drag
				a.viewportDragging = true
				a.viewportDragStart = ev.Position
			} else if ev.Buttons == pointer.ButtonSecondary {
				// Right-click: check if clicking on a pad
				clickPos := image.Pt(int(ev.Position.X), int(ev.Position.Y))
				
				// Transform click position through viewport transform
				// Reverse the pan and zoom
				transformedX := (clickPos.X - int(a.viewportPanX)) / int(a.viewportZoom)
				transformedY := (clickPos.Y - int(a.viewportPanY)) / int(a.viewportZoom)
				transformedPos := image.Pt(transformedX, transformedY)
				
				// Check if clicking on any pad
				foundPad := false
				for _, pad := range a.renderedPads {
					if transformedPos.In(pad.Bounds) {
						// Show context menu for this pad
						a.contextMenuVisible = true
						a.contextMenuPos = ev.Position
						a.contextMenuDevice = pad.DeviceIndex
						a.contextMenuPin = pad.PinNumber
						a.Logf("[VIEWPORT] Right-click on pin %d (%s)", pad.PinNumber, pad.PinName)
						foundPad = true
						break
					}
				}
				
				if !foundPad {
					// Click on background - hide menu
					a.contextMenuVisible = false
				}
				
				gtx.Execute(op.InvalidateCmd{})
			}
			
		case pointer.Release:
			a.viewportDragging = false
			
		case pointer.Drag:
			if a.viewportDragging && ev.Buttons == pointer.ButtonPrimary {
				// Calculate delta from last position
				deltaX := ev.Position.X - a.viewportDragStart.X
				deltaY := ev.Position.Y - a.viewportDragStart.Y
				
				// Update pan
				newPanX := a.viewportPanX + deltaX
				newPanY := a.viewportPanY + deltaY
				
				// Very relaxed pan limits - allow exploring the full canvas
				maxPanX := viewportWidth * 1.5
				minPanX := -viewportWidth * 2.0
				maxPanY := viewportHeight * 1.5
				minPanY := -viewportHeight * 2.0
				
				if newPanX > maxPanX {
					newPanX = maxPanX
				} else if newPanX < minPanX {
					newPanX = minPanX
				}
				
				if newPanY > maxPanY {
					newPanY = maxPanY
				} else if newPanY < minPanY {
					newPanY = minPanY
				}
				
				a.viewportPanX = newPanX
				a.viewportPanY = newPanY
				a.viewportDragStart = ev.Position
				gtx.Execute(op.InvalidateCmd{})
			}
		}
	}
	
	// Render viewport with theme background
	// Constrain to allocated space
	maxSize := gtx.Constraints.Max
	
	return layout.Stack{}.Layout(gtx,
		// Content layer - footprints and ratsnest with transform
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			// Only apply transform if zoom != 1.0 or pan != 0 to avoid rendering artifacts
			applyTransform := a.viewportZoom != 1.0 || a.viewportPanX != 0 || a.viewportPanY != 0
			
			if applyTransform {
				// Scale from center of viewport
				centerX := float32(maxSize.X) / 2
				centerY := float32(maxSize.Y) / 2
				
				defer op.Affine(f32.Affine2D{}.
					Offset(f32.Pt(-centerX, -centerY)).
					Scale(f32.Point{}, f32.Pt(a.viewportZoom, a.viewportZoom)).
					Offset(f32.Pt(centerX+a.viewportPanX, centerY+a.viewportPanY))).Push(gtx.Ops).Pop()
			}
			
			a.renderFootprints(gtx)
			a.renderRatsnest(gtx)
			return layout.Dimensions{}
		}),
		// Input layer - transparent, on top for events
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			area := clip.Rect{Max: maxSize}.Push(gtx.Ops)
			event.Op(gtx.Ops, &a.viewportDrag)
			a.viewportScroll.Add(gtx.Ops)
			a.viewportClick.Add(gtx.Ops)
			a.viewportDrag.Add(gtx.Ops)
			area.Pop()
			return layout.Dimensions{Size: maxSize}
		}),
		// Hint overlay
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 8, Left: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Caption(a.gvTheme.Theme, "Drag to pan | Scroll/+/- to zoom | Right-click pin for menu")
				label.Color = a.gvTheme.Palette.Fg
				label.Color.A = 128
				return label.Layout(gtx)
			})
		}),
		// Context menu overlay
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if !a.contextMenuVisible {
				return layout.Dimensions{}
			}
			return a.layoutPinContextMenu(gtx)
		}),
	)
}

// resetViewport resets zoom and pan to default values
func (a *App) resetViewport() {
	a.viewportZoom = 1.0
	a.viewportPanX = 0
	a.viewportPanY = 0
}

// layoutPinContextMenu renders the context menu for pin control
func (a *App) layoutPinContextMenu(gtx layout.Context) layout.Dimensions {
	// Get pin info
	device := &a.chainDevices[a.contextMenuDevice]
	pinName := device.GetPinName(a.contextMenuPin)
	isPower := pinName != "" && device.IsPowerPin(pinName)
	
	// Build menu title
	title := fmt.Sprintf("Pin %d", a.contextMenuPin)
	if pinName != "" {
		title = fmt.Sprintf("Pin %d: %s", a.contextMenuPin, pinName)
	}
	
	// Position menu at click location
	return layout.Inset{
		Top:  unit.Dp(a.contextMenuPos.Y),
		Left: unit.Dp(a.contextMenuPos.X),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Don't constrain - let content determine size
		gtx.Constraints.Min = image.Point{}
		
		// Render content first to get size, then draw background
		macro := op.Record(gtx.Ops)
		contentDims := layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			// First pass: measure all children to find widest (use dummy widgets)
			var maxWidth int
			measureOps := new(op.Ops)
			
			// Measure title
			titleGtx := gtx
			titleGtx.Ops = measureOps
			titleGtx.Constraints.Min = image.Point{}
			lbl := material.Body2(a.gvTheme.Theme, title)
			titleDims := lbl.Layout(titleGtx)
			if titleDims.Size.X > maxWidth {
				maxWidth = titleDims.Size.X
			}
			
			// Measure buttons if not power pin (use dummy clickables)
			if !isPower {
				for i := 0; i < 3; i++ {
					btnGtx := gtx
					btnGtx.Ops = measureOps
					btnGtx.Constraints.Min = image.Point{}
					var btn material.ButtonStyle
					dummyClick := &widget.Clickable{}
					switch i {
					case 0:
						btn = material.Button(a.gvTheme.Theme, dummyClick, "Set Hi")
					case 1:
						btn = material.Button(a.gvTheme.Theme, dummyClick, "Set Lo")
					case 2:
						btn = material.Button(a.gvTheme.Theme, dummyClick, "Set Hi-Z")
					}
					btnDims := btn.Layout(btnGtx)
					if btnDims.Size.X > maxWidth {
						maxWidth = btnDims.Size.X
					}
				}
			}
			
			// Second pass: render with uniform width
			children := []layout.FlexChild{
				// Title
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = maxWidth
					lbl := material.Body2(a.gvTheme.Theme, title)
					lbl.Color = a.gvTheme.Palette.Fg
					return lbl.Layout(gtx)
				}),
			}
			
			// Only show state buttons for non-power pins
			if !isPower {
				children = append(children,
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					// Set Hi option
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if a.contextMenuOptions[0].Clicked(gtx) {
							a.setPinState(a.contextMenuDevice, a.contextMenuPin, components.PinStateHigh)
							a.contextMenuVisible = false
						}
						gtx.Constraints.Min.X = maxWidth
						btn := material.Button(a.gvTheme.Theme, &a.contextMenuOptions[0], "Set Hi")
						btn.Background = color.NRGBA{R: 220, G: 68, B: 68, A: 255}
						return btn.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
					// Set Lo option
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if a.contextMenuOptions[1].Clicked(gtx) {
							a.setPinState(a.contextMenuDevice, a.contextMenuPin, components.PinStateLow)
							a.contextMenuVisible = false
						}
						gtx.Constraints.Min.X = maxWidth
						btn := material.Button(a.gvTheme.Theme, &a.contextMenuOptions[1], "Set Lo")
						btn.Background = color.NRGBA{R: 66, G: 135, B: 245, A: 255}
						return btn.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
					// Set Hi-Z option
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if a.contextMenuOptions[2].Clicked(gtx) {
							a.setPinState(a.contextMenuDevice, a.contextMenuPin, components.PinStateHighZ)
							a.contextMenuVisible = false
						}
						gtx.Constraints.Min.X = maxWidth
						btn := material.Button(a.gvTheme.Theme, &a.contextMenuOptions[2], "Set Hi-Z")
						btn.Background = color.NRGBA{R: 235, G: 138, B: 52, A: 255}
						return btn.Layout(gtx)
					}),
				)
			}
			
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
		call := macro.Stop()
		
		// Draw background with actual content size
		paint.FillShape(gtx.Ops, a.gvTheme.Palette.Bg,
			clip.RRect{
				Rect: image.Rectangle{Max: contentDims.Size},
				NW: 4, NE: 4, SW: 4, SE: 4,
			}.Op(gtx.Ops))
		
		// Draw content on top
		call.Add(gtx.Ops)
		
		return contentDims
	})
}

// setPinState updates the state of a pin
func (a *App) setPinState(deviceIndex, pinNumber int, state components.PinState) {
	a.Logf("[PIN] Set device %d pin %d to state %s", deviceIndex, pinNumber, state)
	
	// Update pin state in device data
	device := &a.chainDevices[deviceIndex]
	device.SetPinState(pinNumber, string(state))
	
	// TODO: Apply to hardware via JTAG
	
	// Trigger re-render to show visual feedback
	if a.window != nil {
		a.window.Invalidate()
	}
}

// renderFootprints renders all device footprints with spacing
func (a *App) renderFootprints(gtx layout.Context) layout.Dimensions {
	if len(a.chainDevices) == 0 {
		return layout.Dimensions{}
	}
	
	// Clear previous pad bounds
	a.renderedPads = a.renderedPads[:0]
	
	// Simple horizontal layout with spacing
	spacing := 150 // pixels between footprints
	
	var maxHeight int
	xOffset := 100 // Start offset from left
	yOffset := 100 // Start offset from top
	
	for i := range a.chainDevices {
		device := &a.chainDevices[i]
		if device.FootprintType == "" {
			continue
		}
		
		// Render footprint at position
		dims := a.renderDeviceFootprint(gtx, i, xOffset, yOffset)
		xOffset += dims.Size.X + spacing
		if dims.Size.Y > maxHeight {
			maxHeight = dims.Size.Y
		}
	}
	
	return layout.Dimensions{
		Size: image.Pt(xOffset+100, maxHeight+yOffset+100),
	}
}

// renderDeviceFootprint renders a single device footprint at specified position
func (a *App) renderDeviceFootprint(gtx layout.Context, deviceIndex, x, y int) layout.Dimensions {
	device := &a.chainDevices[deviceIndex]
	
	// Offset to position
	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
	
	// Create pins from device with BSDL names
	pins := components.DefaultPins(device.PinCount)
	for i := range pins {
		// Get logical pin name from BSDL (if available)
		pinName := device.GetPinName(pins[i].Number)
		if pinName != "" {
			pins[i].Name = pinName
		}
		
		// Apply stored pin state (if any)
		if device.PinStates != nil {
			stateStr := device.GetPinState(pins[i].Number)
			if stateStr != "" {
				pins[i].State = components.PinState(stateStr)
			}
		}
		
		// Power/ground pins always stay gray (override any state)
		if pinName != "" && device.IsPowerPin(pinName) {
			pins[i].State = components.PinStatePower
		}
	}
	
	// Render options
	opts := &components.RenderOptions{
		Scale:      1.0,
		ShowLabels: true,
	}
	
	// Render based on package type
	var pkg components.PackageRender
	switch device.FootprintType {
	case "TSOP-I":
		pkg = components.NewTSOPPackage(device.PinCount, pins, false, opts)
	case "TSOP-II":
		pkg = components.NewTSOPPackage(device.PinCount, pins, true, opts)
	case "QFP":
		pinsPerSide := device.PinCount / 4
		pkg = components.NewQFPPackage(pinsPerSide, pins, opts)
	case "QFN":
		pinsPerSide := device.PinCount / 4
		pkg = components.NewQFNPackage(pinsPerSide, pins, opts)
	case "BGA":
		// Estimate grid size
		cols := 12
		rows := device.PinCount / cols
		pkg = components.NewBGAPackage(cols, rows, pins, opts)
	default:
		// Fallback
		return layout.Dimensions{}
	}
	
	// Store pad bounds for hit testing (before rendering)
	for _, pad := range pkg.Pads {
		// Calculate screen-space bounds (pad position + device offset)
		bounds := image.Rectangle{
			Min: image.Pt(x+int(pad.Position.X), y+int(pad.Position.Y)),
			Max: image.Pt(x+int(pad.Position.X+pad.Size.X), y+int(pad.Position.Y+pad.Size.Y)),
		}
		
		// Calculate exact center in float coordinates
		center := f32.Pt(
			float32(x)+pad.Position.X+pad.Size.X/2.0,
			float32(y)+pad.Position.Y+pad.Size.Y/2.0,
		)
		
		a.renderedPads = append(a.renderedPads, RenderedPad{
			DeviceIndex: deviceIndex,
			PinNumber:   pad.Pin.Number,
			PinName:     pad.Pin.Name,
			Bounds:      bounds,
			Center:      center,
		})
		
		// Debug: log pins 21-23 only
		if pad.Pin.Number >= 21 && pad.Pin.Number <= 23 {
			a.Logf("[PAD] Dev=%d Pin=%d Type=%s Pos=(%.1f,%.1f) Size=(%.1f,%.1f) Center=(%.1f,%.1f)",
				deviceIndex, pad.Pin.Number, device.FootprintType,
				pad.Position.X, pad.Position.Y, pad.Size.X, pad.Size.Y,
				center.X, center.Y)
		}
	}
	
	// Render the package using the components renderer
	return a.renderPackage(gtx, pkg)
}

// renderPackage renders a PackageRender using Gio operations
func (a *App) renderPackage(gtx layout.Context, pkg components.PackageRender) layout.Dimensions {
	// Draw rectangles
	for _, rect := range pkg.Rectangles {
		// Force full opacity
		col := rect.Fill
		col.A = 255
		
		paint.FillShape(gtx.Ops, col, clip.Rect{
			Min: image.Pt(int(rect.Position.X), int(rect.Position.Y)),
			Max: image.Pt(int(rect.Position.X+rect.Size.X), int(rect.Position.Y+rect.Size.Y)),
		}.Op())
	}
	
	// Draw circles
	for _, circ := range pkg.Circles {
		// Force full opacity
		col := circ.Fill
		col.A = 255
		
		paint.FillShape(gtx.Ops, col, clip.Ellipse{
			Min: image.Pt(int(circ.Center.X-circ.Radius), int(circ.Center.Y-circ.Radius)),
			Max: image.Pt(int(circ.Center.X+circ.Radius), int(circ.Center.Y+circ.Radius)),
		}.Op(gtx.Ops))
	}
	
	// Draw labels
	for _, label := range pkg.Labels {
		// Create text widget
		lbl := material.Label(a.gvTheme.Theme, unit.Sp(label.Size), label.Text)
		lbl.Color = label.Color
		lbl.Alignment = text.Middle
		
		// Measure text to center it properly
		macro := op.Record(gtx.Ops)
		dims := lbl.Layout(gtx)
		_ = macro.Stop()
		
		// Center the text at the label position
		offsetX := int(label.Position.X) - dims.Size.X/2
		offsetY := int(label.Position.Y) - dims.Size.Y/2
		
		stack := op.Offset(image.Pt(offsetX, offsetY)).Push(gtx.Ops)
		lbl.Layout(gtx)
		stack.Pop()
	}
	
	return layout.Dimensions{
		Size: image.Pt(int(pkg.Size.X), int(pkg.Size.Y)),
	}
}

// renderRatsnest draws connection lines between pads
func (a *App) renderRatsnest(gtx layout.Context) layout.Dimensions {
	// Collect all pads in selected net
	selectedPads := make(map[string]f32.Point) // key: "devIdx:pinNum"
	
	for _, line := range a.ratsnestLines {
		// Find pad centers
		var centerA, centerB f32.Point
		foundA, foundB := false, false
		
		for _, pad := range a.renderedPads {
			if pad.DeviceIndex == line.DeviceA && pad.PinNumber == line.PinA {
				centerA = pad.Center
				foundA = true
				if a.selectedNetID == line.NetID {
					key := fmt.Sprintf("%d:%d", line.DeviceA, line.PinA)
					selectedPads[key] = centerA
				}
			}
			if pad.DeviceIndex == line.DeviceB && pad.PinNumber == line.PinB {
				centerB = pad.Center
				foundB = true
				if a.selectedNetID == line.NetID {
					key := fmt.Sprintf("%d:%d", line.DeviceB, line.PinB)
					selectedPads[key] = centerB
				}
			}
			if foundA && foundB {
				break
			}
		}
		
		if !foundA || !foundB {
			continue
		}
		

		// Highlight selected net
		lineColor := line.Color
		lineWidth := float32(2.0)
		if a.selectedNetID != -1 && a.selectedNetID == line.NetID {
			// Brighten and thicken selected net - use bright orange (not in default palette)
			lineColor = color.NRGBA{R: 255, G: 128, B: 0, A: 255} // Bright orange
			lineWidth = 4.0
		}
		
		// Draw line with explicit macro to ensure correct coordinate space
		macro := op.Record(gtx.Ops)
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(centerA)
		p.LineTo(centerB)
		paint.FillShape(gtx.Ops, lineColor,
			clip.Stroke{
				Path: p.End(),
				Width: lineWidth,
			}.Op())
		call := macro.Stop()
		call.Add(gtx.Ops)
	}
	
	// Draw circles on all pads in selected net
	for _, center := range selectedPads {
		// Draw a filled circle using clip.Ellipse (5px radius)
		stack := clip.Ellipse{
			Min: image.Pt(int(center.X-5), int(center.Y-5)),
			Max: image.Pt(int(center.X+5), int(center.Y+5)),
		}.Push(gtx.Ops)
		
		paint.ColorOp{Color: color.NRGBA{R: 0, G: 0, B: 0, A: 255}}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		
		stack.Pop()
	}
	
	// Reset debug flag after logging all lines
	a.debugLogOnce = false
	
	return layout.Dimensions{}
}
