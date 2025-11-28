package newui

import (
	"fmt"
	"image"
	"image/color"

	gfont "gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// layoutChains renders the detected devices panel on the right side
func (a *App) layoutChains(gtx layout.Context) layout.Dimensions {
	title := fmt.Sprintf("Detected Devices (%d)", len(a.chainDevices))
	
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// Larger bold title (30% larger than body text)
			label := material.H6(a.gvTheme.Theme, title)
			label.Font.Weight = gfont.Bold
			label.TextSize = unit.Sp(16) // ~30% larger than body (12sp)
			return label.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if a.isScanning {
				return material.Body2(a.gvTheme.Theme, "Scanning for devices...").Layout(gtx)
			}
			if len(a.chainDevices) == 0 {
				return material.Body2(a.gvTheme.Theme, "No devices detected").Layout(gtx)
			}
			// Add horizontal padding for cards
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				a.deviceCardList.Axis = layout.Vertical
				return material.List(a.gvTheme.Theme, &a.deviceCardList).Layout(gtx, len(a.chainDevices), func(gtx layout.Context, index int) layout.Dimensions {
					return a.layoutDeviceCard(gtx, index)
				})
			})
		}),
	)
}

// layoutDeviceCard renders a single device card
func (a *App) layoutDeviceCard(gtx layout.Context, index int) layout.Dimensions {
	device := &a.chainDevices[index]
	
	return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Force card to use full width
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				// Card background with rounded corners using theme colors
				rr := gtx.Dp(unit.Dp(8))
				rect := clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, rr).Push(gtx.Ops)
				// Use theme surface color for card background
				cardBg := a.gvTheme.Bg
				if !a.darkMode {
					// Light mode: use a slightly darker surface
					cardBg = color.NRGBA{R: 245, G: 245, B: 245, A: 255}
				}
				paint.Fill(gtx.Ops, cardBg)
				rect.Pop()
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				// Ensure content uses full width
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						// Device title
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							title := fmt.Sprintf("Device %d", index)
							if device.Name != "" {
								title = fmt.Sprintf("Device %d: %s", index, device.Name)
							}
							label := material.Body1(a.gvTheme.Theme, title)
							label.MaxLines = 1 // Prevent wrapping
							return label.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						// IDCODE with decoded info
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									idcode := fmt.Sprintf("IDCODE: 0x%08X", device.IDCode)
									return material.Caption(a.gvTheme.Theme, idcode).Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									mfg := fmt.Sprintf("Mfg: %s", device.IDCodeInfo.ManufName)
									return material.Caption(a.gvTheme.Theme, mfg).Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									part := fmt.Sprintf("Part: 0x%04X, Ver: %d", 
										device.IDCodeInfo.PartNumber, device.IDCodeInfo.Version)
									return material.Caption(a.gvTheme.Theme, part).Layout(gtx)
								}),
							)
						}),
						// IR Length (if known)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if device.IRLength > 0 {
								info := fmt.Sprintf("IR Length: %d bits", device.IRLength)
								return material.Caption(a.gvTheme.Theme, info).Layout(gtx)
							}
							return layout.Dimensions{}
						}),
						// BSR Length (if BSDL assigned)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if device.BSRLength > 0 {
								info := fmt.Sprintf("BSR Length: %d bits", device.BSRLength)
								return material.Caption(a.gvTheme.Theme, info).Layout(gtx)
							}
							return layout.Dimensions{}
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						// BSDL assignment
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if device.BSDLPath == "" {
								btn := material.Button(a.gvTheme.Theme, &device.bsdlBtn, "Assign BSDL...")
								if device.bsdlBtn.Clicked(gtx) {
									a.openBSDLPicker(index)
								}
								return btn.Layout(gtx)
							}
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return material.Body2(a.gvTheme.Theme, "✓ BSDL: "+device.Name).Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(a.gvTheme.Theme, &device.bsdlBtn, "Change...")
									btn.TextSize = unit.Sp(12)
									if device.bsdlBtn.Clicked(gtx) {
										a.openBSDLPicker(index)
									}
									return btn.Layout(gtx)
								}),
							)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
						// Footprint assignment
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if device.FootprintType == "" {
								btn := material.Button(a.gvTheme.Theme, &device.footprintBtn, "Assign Footprint...")
								if device.footprintBtn.Clicked(gtx) {
									a.openFootprintPicker(index)
								}
								return btn.Layout(gtx)
							}
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return material.Body2(a.gvTheme.Theme, "✓ Footprint: "+device.FootprintType).Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(a.gvTheme.Theme, &device.footprintBtn, "Change...")
									btn.TextSize = unit.Sp(12)
									if device.footprintBtn.Clicked(gtx) {
										a.openFootprintPicker(index)
									}
									return btn.Layout(gtx)
								}),
							)
						}),
					)
				})
			}),
		)
	})
}

// allDevicesReady checks if all devices have BSDL and footprint assigned
func (a *App) allDevicesReady() bool {
	for _, dev := range a.chainDevices {
		if dev.BSDLPath == "" || dev.FootprintType == "" {
			return false
		}
	}
	return true
}
