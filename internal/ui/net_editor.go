package newui

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsr"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/reveng"
)

// NetEditor manages the net editing UI state
type NetEditor struct {
	list          widget.List
	exportBtn     widget.Clickable
	addNetBtn     widget.Clickable
	
	netDeleteBtn  map[int]*widget.Clickable
	netAddPinBtn  map[int]*widget.Clickable
	pinDeleteBtn  map[string]*widget.Clickable
	
	// Pin selection dialog
	pinSelectVisible bool
	pinSelectNetID   int
	pinSelectList    widget.List
	pinSelectBtns    map[string]*widget.Clickable
	pinSelectCancel  widget.Clickable
	availablePins    []bsr.PinRef
}

func NewNetEditor() *NetEditor {
	ne := &NetEditor{
		netDeleteBtn:  make(map[int]*widget.Clickable),
		netAddPinBtn:  make(map[int]*widget.Clickable),
		pinDeleteBtn:  make(map[string]*widget.Clickable),
		pinSelectBtns: make(map[string]*widget.Clickable),
	}
	ne.list.Axis = layout.Vertical
	ne.pinSelectList.Axis = layout.Vertical
	return ne
}

func (ne *NetEditor) Open(netlist *reveng.Netlist) {
	for _, net := range netlist.Nets {
		if _, ok := ne.netDeleteBtn[net.ID]; !ok {
			ne.netDeleteBtn[net.ID] = &widget.Clickable{}
		}
		if _, ok := ne.netAddPinBtn[net.ID]; !ok {
			ne.netAddPinBtn[net.ID] = &widget.Clickable{}
		}
		
		for _, pin := range net.Pins {
			key := fmt.Sprintf("%d:%d:%s", net.ID, pin.ChainIndex, pin.PinName)
			if _, ok := ne.pinDeleteBtn[key]; !ok {
				ne.pinDeleteBtn[key] = &widget.Clickable{}
			}
		}
	}
}

func (ne *NetEditor) LayoutWorkspace(gtx layout.Context, th *material.Theme, app *App) layout.Dimensions {
	// Ensure widgets exist for new nets (only creates missing ones)
	for _, net := range app.discoveredNetlist.Nets {
		if _, ok := ne.netDeleteBtn[net.ID]; !ok {
			ne.netDeleteBtn[net.ID] = &widget.Clickable{}
		}
		if _, ok := ne.netAddPinBtn[net.ID]; !ok {
			ne.netAddPinBtn[net.ID] = &widget.Clickable{}
		}
		
		for _, pin := range net.Pins {
			key := fmt.Sprintf("%d:%d:%s", net.ID, pin.ChainIndex, pin.PinName)
			if _, ok := ne.pinDeleteBtn[key]; !ok {
				ne.pinDeleteBtn[key] = &widget.Clickable{}
			}
		}
	}
	
	if ne.exportBtn.Clicked(gtx) {
		app.exportNetlistKiCad()
	}
	
	if ne.addNetBtn.Clicked(gtx) {
		ne.addNewNet(app)
		app.buildRatsnest()
	}
	
	for netID, btn := range ne.netDeleteBtn {
		if btn.Clicked(gtx) {
			ne.deleteNet(netID, app)
			app.buildRatsnest()
		}
	}
	
	for netID, btn := range ne.netAddPinBtn {
		if btn.Clicked(gtx) {
			ne.openPinSelector(netID, app)
		}
	}
	
	for key, btn := range ne.pinDeleteBtn {
		if btn.Clicked(gtx) {
			ne.deletePinByKey(key, app)
			app.buildRatsnest()
		}
	}
	
	if ne.pinSelectVisible {
		return ne.layoutPinSelector(gtx, th, app)
	}
	
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return material.H6(th, fmt.Sprintf("Netlist (%d nets)", app.discoveredNetlist.NetCount())).Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(th, &ne.addNetBtn, "+ Add Net")
					btn.Background = color.NRGBA{R: 76, G: 175, B: 80, A: 255}
					return btn.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(th, &ne.exportBtn, "Export KiCad")
					btn.Background = color.NRGBA{R: 33, G: 150, B: 243, A: 255}
					return btn.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(th, &ne.list).Layout(gtx, len(app.discoveredNetlist.Nets), func(gtx layout.Context, i int) layout.Dimensions {
				return ne.layoutNet(gtx, th, app, app.discoveredNetlist.Nets[i])
			})
		}),
	)
}

func (ne *NetEditor) layoutNet(gtx layout.Context, th *material.Theme, app *App, net *reveng.Net) layout.Dimensions {
	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Draw background
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							label := material.Body1(th, fmt.Sprintf("Net %d (%d pins)", net.ID, len(net.Pins)))
							label.Font.Weight = font.Bold
							return label.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(th, ne.netDeleteBtn[net.ID], "Delete")
							btn.Background = color.NRGBA{R: 200, G: 50, B: 50, A: 255}
							return btn.Layout(gtx)
						}),
					)
				}),
				
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, func() []layout.FlexChild {
						var children []layout.FlexChild
						
						for _, pin := range net.Pins {
							pin := pin
							children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ne.layoutPin(gtx, th, net.ID, pin, app)
							}))
						}
						
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(th, ne.netAddPinBtn[net.ID], "+ Add Pin")
							btn.Background = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
							return btn.Layout(gtx)
						}))
						
						return children
					}()...)
				}),
			)
		})
		call := macro.Stop()
		
		paint.FillShape(gtx.Ops, color.NRGBA{R: 245, G: 245, B: 245, A: 255},
			clip.RRect{Rect: image.Rectangle{Max: dims.Size}, SE: 4, SW: 4, NE: 4, NW: 4}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		
		return dims
	})
}

func (ne *NetEditor) layoutPin(gtx layout.Context, th *material.Theme, netID int, pin bsr.PinRef, app *App) layout.Dimensions {
	key := fmt.Sprintf("%d:%d:%s", netID, pin.ChainIndex, pin.PinName)
	
	// Get pin number from device
	pinNum := 0
	if pin.ChainIndex < len(app.chainDevices) {
		device := &app.chainDevices[pin.ChainIndex]
		pinNum = app.findPinNumber(device, pin.PinName)
	}
	
	return layout.Inset{Top: unit.Dp(4), Left: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				text := fmt.Sprintf("• Device %d: %s (pin %d) - %s", pin.ChainIndex, pin.PinName, pinNum, pin.DeviceName)
				return material.Body2(th, text).Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(th, ne.pinDeleteBtn[key], "×")
				btn.Background = color.NRGBA{R: 150, G: 50, B: 50, A: 255}
				return btn.Layout(gtx)
			}),
		)
	})
}

func (ne *NetEditor) addNewNet(app *App) {
	maxID := 0
	for _, net := range app.discoveredNetlist.Nets {
		if net.ID > maxID {
			maxID = net.ID
		}
	}
	
	newNet := &reveng.Net{
		ID:   maxID + 1,
		Pins: []bsr.PinRef{},
	}
	
	app.discoveredNetlist.Nets = append(app.discoveredNetlist.Nets, newNet)
	ne.netDeleteBtn[newNet.ID] = &widget.Clickable{}
	ne.netAddPinBtn[newNet.ID] = &widget.Clickable{}
}

func (ne *NetEditor) deleteNet(netID int, app *App) {
	filtered := make([]*reveng.Net, 0, len(app.discoveredNetlist.Nets))
	for _, net := range app.discoveredNetlist.Nets {
		if net.ID != netID {
			filtered = append(filtered, net)
		}
	}
	app.discoveredNetlist.Nets = filtered
}

func (ne *NetEditor) deletePinByKey(key string, app *App) {
	var netID, chainIdx int
	var pinName string
	fmt.Sscanf(key, "%d:%d:%s", &netID, &chainIdx, &pinName)
	
	for _, net := range app.discoveredNetlist.Nets {
		if net.ID == netID {
			filtered := make([]bsr.PinRef, 0, len(net.Pins))
			for _, pin := range net.Pins {
				if !(pin.ChainIndex == chainIdx && pin.PinName == pinName) {
					filtered = append(filtered, pin)
				}
			}
			net.Pins = filtered
			break
		}
	}
}

func (ne *NetEditor) openPinSelector(netID int, app *App) {
	ne.pinSelectVisible = true
	ne.pinSelectNetID = netID
	ne.availablePins = ne.collectAvailablePins(app)
	
	for _, pin := range ne.availablePins {
		key := fmt.Sprintf("%d:%s", pin.ChainIndex, pin.PinName)
		if _, ok := ne.pinSelectBtns[key]; !ok {
			ne.pinSelectBtns[key] = &widget.Clickable{}
		}
	}
}

func (ne *NetEditor) collectAvailablePins(app *App) []bsr.PinRef {
	var pins []bsr.PinRef
	
	if len(app.chainDevices) == 0 {
		return pins
	}
	
	usedPins := make(map[string]bool)
	for _, net := range app.discoveredNetlist.Nets {
		for _, pin := range net.Pins {
			key := fmt.Sprintf("%d:%s", pin.ChainIndex, pin.PinName)
			usedPins[key] = true
		}
	}
	
	for devIdx, device := range app.chainDevices {
		if device.PinMapping == nil {
			continue
		}
		for _, pinName := range device.PinMapping.BSRIndexToPin {
			key := fmt.Sprintf("%d:%s", devIdx, pinName)
			if !usedPins[key] {
				// Only include pins that have valid pin numbers
				pinNum := app.findPinNumber(&device, pinName)
				if pinNum > 0 {
					pins = append(pins, bsr.PinRef{
						ChainIndex: devIdx,
						PinName:    pinName,
						DeviceName: device.Name,
					})
				}
			}
		}
	}
	
	return pins
}

func (ne *NetEditor) layoutPinSelector(gtx layout.Context, th *material.Theme, app *App) layout.Dimensions {
	if ne.pinSelectCancel.Clicked(gtx) {
		ne.pinSelectVisible = false
	}
	
	for key, btn := range ne.pinSelectBtns {
		if btn.Clicked(gtx) {
			ne.addPinToNet(key, app)
			app.buildRatsnest()
			ne.pinSelectVisible = false
		}
	}
	
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, color.NRGBA{A: 180}, clip.Rect{Max: gtx.Constraints.Max}.Op())
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(500))
				gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(500))
				gtx.Constraints.Min = image.Point{}
				
				return layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255},
							clip.RRect{Rect: image.Rectangle{Max: gtx.Constraints.Max}, SE: 8, SW: 8, NE: 8, NW: 8}.Op(gtx.Ops))
						return layout.Dimensions{Size: gtx.Constraints.Max}
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									title := material.H6(th, fmt.Sprintf("Select Pin for Net %d", ne.pinSelectNetID))
									return title.Layout(gtx)
								})
							}),
							
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return material.List(th, &ne.pinSelectList).Layout(gtx, len(ne.availablePins), func(gtx layout.Context, i int) layout.Dimensions {
									pin := ne.availablePins[i]
									key := fmt.Sprintf("%d:%s", pin.ChainIndex, pin.PinName)
									
									// Get pin number
									pinNum := 0
									if pin.ChainIndex < len(app.chainDevices) {
										device := &app.chainDevices[pin.ChainIndex]
										pinNum = app.findPinNumber(device, pin.PinName)
									}
									
									return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										// Force button to full width
										gtx.Constraints.Min.X = gtx.Constraints.Max.X
										btn := material.Button(th, ne.pinSelectBtns[key], fmt.Sprintf("Device %d: %s (pin %d) - %s", pin.ChainIndex, pin.PinName, pinNum, pin.DeviceName))
										btn.Background = color.NRGBA{R: 63, G: 81, B: 181, A: 255}
										return btn.Layout(gtx)
									})
								})
							}),
							
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(th, &ne.pinSelectCancel, "Cancel")
									return btn.Layout(gtx)
								})
							}),
						)
					}),
				)
			})
		}),
	)
}

func (ne *NetEditor) addPinToNet(pinKey string, app *App) {
	var chainIdx int
	var pinName string
	fmt.Sscanf(pinKey, "%d:%s", &chainIdx, &pinName)
	
	for _, pin := range ne.availablePins {
		if pin.ChainIndex == chainIdx && pin.PinName == pinName {
			for _, net := range app.discoveredNetlist.Nets {
				if net.ID == ne.pinSelectNetID {
					net.Pins = append(net.Pins, pin)
					
					key := fmt.Sprintf("%d:%d:%s", net.ID, pin.ChainIndex, pin.PinName)
					ne.pinDeleteBtn[key] = &widget.Clickable{}
					break
				}
			}
			break
		}
	}
}
