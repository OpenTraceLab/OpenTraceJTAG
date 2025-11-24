package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"sort"
	"time"

	"gioui.org/app"
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
)

type appView int

const (
	viewDrivers appView = iota
	viewReverse
	viewFootprint
)

type navEntry struct {
	view  appView
	name  string
	icon  *widget.Icon
	click widget.Clickable
}

type driverKindOption struct {
	Kind  jtag.InterfaceKind
	Label string
}

var driverKindOptions = []driverKindOption{
	{Kind: jtag.InterfaceKindUnknown, Label: "All Drivers"},
	{Kind: jtag.InterfaceKindSim, Label: "Simulator"},
	{Kind: jtag.InterfaceKindCMSISDAP, Label: "CMSIS-DAP"},
	{Kind: jtag.InterfaceKindPico, Label: "PicoProbe"},
}

// App drives the Gio-based boundary scan UI.
type App struct {
	Window *app.Window
	Theme  *material.Theme
	State  *AppState

	ops op.Ops

	scanChainBtn        widget.Clickable
	scanInterfacesBtn   widget.Clickable
	connectInterfaceBtn widget.Clickable

	interfaceList layout.List
	chainList     layout.List
	logList       layout.List

	deviceClicks    map[int]*widget.Clickable
	interfaceClicks map[int]*widget.Clickable

	driverKindButtons map[jtag.InterfaceKind]*widget.Clickable

	toggleLeftPanelBtn  widget.Clickable
	toggleRightPanelBtn widget.Clickable

	logPaneHeight float32
	logSplitter   gesture.Drag
	logSplitLastY float32
	logSplitDrag  bool

	currentView appView
	navEntries  []navEntry
}

// New wires the Gio window, theme, and shared state together.
func New(window *app.Window, state *AppState) *App {
	if state == nil {
		state = NewState()
	}
	baseTheme := material.NewTheme()
	baseTheme.Palette = material.Palette{
		Bg:         color.NRGBA{R: 245, G: 246, B: 252, A: 255},
		Fg:         color.NRGBA{R: 34, G: 37, B: 49, A: 255},
		ContrastBg: color.NRGBA{R: 80, G: 120, B: 255, A: 255},
		ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
	}
	app := &App{
		Window:          window,
		Theme:           baseTheme,
		State:           state,
		interfaceList:   layout.List{Axis: layout.Vertical},
		chainList:       layout.List{Axis: layout.Vertical},
		logList:         layout.List{Axis: layout.Vertical},
		deviceClicks:    make(map[int]*widget.Clickable),
		interfaceClicks: make(map[int]*widget.Clickable),
		driverKindButtons: func() map[jtag.InterfaceKind]*widget.Clickable {
			m := make(map[jtag.InterfaceKind]*widget.Clickable, len(driverKindOptions))
			for _, opt := range driverKindOptions {
				m[opt.Kind] = new(widget.Clickable)
			}
			return m
		}(),
		currentView:   state.SelectedView(),
		logPaneHeight: 0,
	}
	app.initNavigation()
	return app
}

// Run processes Gio events until the window is closed.
func (a *App) Run() error {
	for {
		e := a.Window.Event()
		switch ev := e.(type) {
		case app.DestroyEvent:
			return ev.Err
		case app.FrameEvent:
			gtx := app.NewContext(&a.ops, ev)
			a.layout(gtx)
			ev.Frame(gtx.Ops)
		}
	}
}

func (a *App) initNavigation() {
	makeIcon := func(data []byte, name string) *widget.Icon {
		icon, err := widget.NewIcon(data)
		if err != nil {
			log.Printf("ui: failed to load %s icon: %v", name, err)
			return nil
		}
		return icon
	}
	a.navEntries = []navEntry{
		{
			view: viewDrivers,
			name: "Drivers",
			icon: makeIcon(icons.ActionSettingsInputComponent, "drivers"),
		},
		{
			view: viewReverse,
			name: "Rev. Eng",
			icon: makeIcon(icons.ActionAutorenew, "reverse"),
		},
		{
			view: viewFootprint,
			name: "Footprint",
			icon: makeIcon(icons.HardwareDeveloperBoard, "footprint"),
		},
	}
	a.selectNav(a.State.SelectedView(), false)
}

func (a *App) selectNav(view appView, updateState bool) {
	a.currentView = view
	if updateState {
		a.State.SetView(view)
	}
	a.invalidate()
}

func (a *App) layout(gtx layout.Context) layout.Dimensions {
	state := a.State.Snapshot()
	if state.SelectedView != a.currentView {
		a.selectNav(state.SelectedView, false)
	}

	paint.FillShape(gtx.Ops, color.NRGBA{R: 238, G: 241, B: 251, A: 255}, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			width := gtx.Dp(unit.Dp(80))
			gtx.Constraints.Min.X = width
			gtx.Constraints.Max.X = width
			return a.layoutNavigation(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutTopBar(gtx, &state)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.layoutMainPanels(gtx, state)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutStatus(gtx, state)
				}),
			)
		}),
	)
}

func (a *App) layoutNavigation(gtx layout.Context) layout.Dimensions {
	width := gtx.Dp(unit.Dp(160))
	gtx.Constraints.Min.X = width
	gtx.Constraints.Max.X = width
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, color.NRGBA{R: 45, G: 50, B: 68, A: 255}, clip.Rect{Max: gtx.Constraints.Max}.Op())
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(24), Bottom: unit.Dp(24), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				children := make([]layout.FlexChild, 0, len(a.navEntries)*2)
				for i := range a.navEntries {
					entry := &a.navEntries[i]
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutNavEntry(gtx, entry)
					}))
					children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			})
		}),
	)
}

func (a *App) layoutTopBar(gtx layout.Context, state *StateSnapshot) layout.Dimensions {
	return layout.Inset{
		Top: unit.Dp(12), Bottom: unit.Dp(4), Left: unit.Dp(16), Right: unit.Dp(16),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(material.H6(a.Theme, "JTAG Control Center").Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				for a.toggleLeftPanelBtn.Clicked(gtx) {
					newVal := !state.LeftPanelVisible
					a.State.SetLeftPanelVisible(newVal)
					state.LeftPanelVisible = newVal
					a.invalidate()
				}
				label := "Hide Left Panel"
				if !state.LeftPanelVisible {
					label = "Show Left Panel"
				}
				btn := material.Button(a.Theme, &a.toggleLeftPanelBtn, label)
				btn.Inset = layout.UniformInset(unit.Dp(6))
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				for a.toggleRightPanelBtn.Clicked(gtx) {
					newVal := !state.RightPanelVisible
					a.State.SetRightPanelVisible(newVal)
					state.RightPanelVisible = newVal
					a.invalidate()
				}
				label := "Hide Right Panel"
				if !state.RightPanelVisible {
					label = "Show Right Panel"
				}
				btn := material.Button(a.Theme, &a.toggleRightPanelBtn, label)
				btn.Inset = layout.UniformInset(unit.Dp(6))
				return btn.Layout(gtx)
			}),
		)
	})
}

func (a *App) layoutMainPanels(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.layoutWorkspace(gtx, state.SelectedView, state)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutLogSplitter(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutLogPane(gtx, state)
		}),
	)
}

func (a *App) layoutWorkspace(gtx layout.Context, view appView, state StateSnapshot) layout.Dimensions {
	left := func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }
	right := func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }
	center := func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }

	switch view {
	case viewReverse:
		center = func(gtx layout.Context) layout.Dimensions { return a.layoutReverse(gtx, state) }
		left = func(gtx layout.Context) layout.Dimensions { return a.layoutReverseSidebar(gtx) }
		right = func(gtx layout.Context) layout.Dimensions { return a.layoutReverseDetails(gtx) }
	case viewFootprint:
		center = func(gtx layout.Context) layout.Dimensions { return a.layoutFootprint(gtx, state) }
		left = func(gtx layout.Context) layout.Dimensions { return a.layoutFootprintSidebar(gtx) }
		right = func(gtx layout.Context) layout.Dimensions { return a.layoutFootprintDetails(gtx) }
	default:
		center = func(gtx layout.Context) layout.Dimensions { return a.layoutDrivers(gtx, state) }
		left = func(gtx layout.Context) layout.Dimensions { return a.layoutChainPanel(gtx, state) }
		right = func(gtx layout.Context) layout.Dimensions { return a.layoutDetailsPanel(gtx, state) }
	}

	children := make([]layout.FlexChild, 0, 3)
	if state.LeftPanelVisible {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			width := gtx.Dp(unit.Dp(260))
			gtx.Constraints.Max.X = width
			gtx.Constraints.Min.X = width
			return a.layoutPanelSurface(gtx, left)
		}))
	}
	children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.layoutCenteredCard(gtx, center)
		})
	}))
	if state.RightPanelVisible {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			width := gtx.Dp(unit.Dp(320))
			gtx.Constraints.Max.X = width
			gtx.Constraints.Min.X = width
			return a.layoutPanelSurface(gtx, right)
		}))
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

func (a *App) layoutPanelSurface(gtx layout.Context, body layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			col := color.NRGBA{R: 238, G: 240, B: 247, A: 255}
			rr := gtx.Dp(unit.Dp(10))
			paint.FillShape(gtx.Ops, col, clip.RRect{
				Rect: image.Rectangle{Max: gtx.Constraints.Max},
				NW:   rr, NE: rr, SW: rr, SE: rr,
			}.Op(gtx.Ops))
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Left: unit.Dp(10), Right: unit.Dp(10), Top: unit.Dp(10), Bottom: unit.Dp(10),
			}.Layout(gtx, body)
		}),
	)
}

func (a *App) layoutCenteredCard(gtx layout.Context, body layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			col := color.NRGBA{R: 248, G: 248, B: 253, A: 255}
			paint.FillShape(gtx.Ops, col, clip.RRect{
				Rect: image.Rectangle{Max: gtx.Constraints.Max},
				NW:   gtx.Dp(unit.Dp(12)),
				NE:   gtx.Dp(unit.Dp(12)),
				SW:   gtx.Dp(unit.Dp(12)),
				SE:   gtx.Dp(unit.Dp(12)),
			}.Op(gtx.Ops))
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(16), Bottom: unit.Dp(16)}.Layout(gtx, body)
		}),
	)
}

func (a *App) layoutNavEntry(gtx layout.Context, entry *navEntry) layout.Dimensions {
	for entry.click.Clicked(gtx) {
		a.selectNav(entry.view, true)
	}

	width := gtx.Constraints.Max.X
	if width <= 0 {
		width = gtx.Dp(unit.Dp(140))
	}
	height := gtx.Dp(unit.Dp(52))
	size := image.Pt(width, height)
	gtx.Constraints.Min = size
	gtx.Constraints.Max = size

	selected := a.currentView == entry.view
	bg := color.NRGBA{R: 45, G: 50, B: 68, A: 255}
	if entry.click.Hovered() {
		bg = color.NRGBA{R: 60, G: 66, B: 88, A: 255}
	}
	if selected {
		bg = viewAccentColor(entry.view)
	}
	textColor := color.NRGBA{R: 240, G: 244, B: 255, A: 255}

	return entry.click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				inset := gtx.Dp(unit.Dp(2))
				rect := image.Rectangle{Max: size}.Inset(inset)
				paint.FillShape(gtx.Ops, bg, clip.RRect{
					Rect: rect,
					NW:   gtx.Dp(unit.Dp(8)),
					NE:   gtx.Dp(unit.Dp(8)),
					SW:   gtx.Dp(unit.Dp(8)),
					SE:   gtx.Dp(unit.Dp(8)),
				}.Op(gtx.Ops))
				return layout.Dimensions{Size: rect.Size()}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							size := gtx.Dp(unit.Dp(28))
							gtx.Constraints.Min = image.Pt(size, size)
							gtx.Constraints.Max = gtx.Constraints.Min
							if entry.icon != nil {
								return entry.icon.Layout(gtx, textColor)
							}
							return layout.Dimensions{Size: image.Pt(size, size)}
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body2(a.Theme, entry.name)
							lbl.Color = textColor
							lbl.Alignment = text.Start
							return lbl.Layout(gtx)
						}),
					)
				})
			}),
		)
	})
}

func (a *App) layoutDrivers(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H5(a.Theme, "JTAG Adapter").Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(material.Body2(a.Theme, "Choose a driver, scan interfaces, and test the connection.").Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutDriverKindSelector(gtx, state)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.layoutInterfaces(gtx, state)
		}),
	)
}

func (a *App) layoutDriverKindSelector(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
		func() []layout.FlexChild {
			children := make([]layout.FlexChild, 0, len(driverKindOptions))
			for _, opt := range driverKindOptions {
				option := opt
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(a.Theme, a.driverKindButtons[option.Kind], option.Label)
					if option.Kind == state.DriverKind {
						btn.Background = color.NRGBA{R: 98, G: 146, B: 255, A: 255}
					} else {
						btn.Background = color.NRGBA{R: 60, G: 64, B: 76, A: 255}
					}
					btn.Inset = layout.UniformInset(unit.Dp(6))
					dims := btn.Layout(gtx)
					for a.driverKindButtons[option.Kind].Clicked(gtx) {
						a.State.SetDriverKind(option.Kind)
						a.invalidate()
					}
					return dims
				}))
			}
			return children
		}()...,
	)
}

type interfaceDisplay struct {
	Info  jtag.InterfaceInfo
	Index int
}

func filterInterfaces(infos []jtag.InterfaceInfo, kind jtag.InterfaceKind) []interfaceDisplay {
	if len(infos) == 0 {
		return nil
	}
	results := make([]interfaceDisplay, 0, len(infos))
	for idx, info := range infos {
		if kind != jtag.InterfaceKindUnknown && kind != "" && info.Kind != kind {
			continue
		}
		results = append(results, interfaceDisplay{Info: info, Index: idx})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Info.Label() < results[j].Info.Label()
	})
	return results
}

func driverKindLabel(kind jtag.InterfaceKind) string {
	switch kind {
	case jtag.InterfaceKindSim:
		return "Simulator"
	case jtag.InterfaceKindCMSISDAP:
		return "CMSIS-DAP"
	case jtag.InterfaceKindPico:
		return "PicoProbe"
	case jtag.InterfaceKindUnknown, "":
		return "All Drivers"
	default:
		return string(kind)
	}
}

func (a *App) layoutChainPanel(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H6(a.Theme, "Scan Chain").Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.layoutChain(gtx, state)
		}),
	)
}

func (a *App) layoutDetailsPanel(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H6(a.Theme, "Device Details").Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.layoutDetails(gtx, state)
		}),
	)
}

func (a *App) layoutReverseSidebar(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.Body1(a.Theme, "Reverse Sidebar").Layout),
		layout.Rigid(material.Caption(a.Theme, "Explain workflow, scans, etc.").Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(material.Caption(a.Theme, "Scan chain → Configure pins → Export netlists").Layout),
	)
}

func (a *App) layoutReverseDetails(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.Body1(a.Theme, "Reverse Details").Layout),
		layout.Rigid(material.Caption(a.Theme, "Active pin, nets, progress.").Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(material.Caption(a.Theme, "Netlist stats, active pin, and progress.").Layout),
	)
}

func (a *App) layoutFootprintSidebar(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.Body1(a.Theme, "Footprint Sidebar").Layout),
		layout.Rigid(material.Caption(a.Theme, "Package presets and controls.").Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(material.Caption(a.Theme, "Select package families and pin counts.").Layout),
	)
}

func (a *App) layoutFootprintDetails(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.Body1(a.Theme, "Footprint Details").Layout),
		layout.Rigid(material.Caption(a.Theme, "Pad inspector and pin metadata.").Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(material.Caption(a.Theme, "Pin metadata and color legend.").Layout),
	)
}

func viewAccentColor(view appView) color.NRGBA {
	switch view {
	case viewReverse:
		return color.NRGBA{R: 64, G: 170, B: 110, A: 255}
	case viewFootprint:
		return color.NRGBA{R: 220, G: 90, B: 100, A: 255}
	default:
		return color.NRGBA{R: 0, G: 180, B: 200, A: 255}
	}
}

func (a *App) layoutLogPane(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	a.ensureLogPaneHeight(gtx)
	height := int(a.logPaneHeight)
	if h := gtx.Constraints.Max.Y; h > 0 && height > h {
		height = h
	}
	gtx.Constraints.Min.Y = height
	gtx.Constraints.Max.Y = height
	return layout.Inset{
		Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(6), Bottom: unit.Dp(6),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return a.layoutLogs(gtx, state)
	})
}

func (a *App) layoutLogSplitter(gtx layout.Context) layout.Dimensions {
	height := gtx.Dp(unit.Dp(10))
	if height < 4 {
		height = 4
	}
	size := image.Pt(gtx.Constraints.Max.X, height)
	if size.X == 0 {
		size.X = gtx.Dp(unit.Dp(400))
	}
	rect := clip.Rect{Max: size}
	paint.FillShape(gtx.Ops, color.NRGBA{R: 210, G: 214, B: 228, A: 255}, rect.Op())

	stack := rect.Push(gtx.Ops)
	a.logSplitter.Add(gtx.Ops)
	stack.Pop()

	if ev, ok := a.logSplitter.Update(gtx.Metric, gtx.Source, gesture.Vertical); ok {
		switch ev.Kind {
		case pointer.Press:
			a.logSplitDrag = true
			a.logSplitLastY = ev.Position.Y
		case pointer.Drag:
			if a.logSplitDrag {
				dy := ev.Position.Y - a.logSplitLastY
				a.logSplitLastY = ev.Position.Y
				a.logPaneHeight -= dy
				a.clampLogPaneHeight(gtx)
				a.invalidate()
			}
		case pointer.Release, pointer.Cancel:
			a.logSplitDrag = false
		}
	}
	return layout.Dimensions{Size: size}
}

func (a *App) ensureLogPaneHeight(gtx layout.Context) {
	if a.logPaneHeight > 0 {
		return
	}
	a.logPaneHeight = float32(gtx.Dp(unit.Dp(180)))
	a.clampLogPaneHeight(gtx)
}

func (a *App) clampLogPaneHeight(gtx layout.Context) {
	min := float32(gtx.Dp(unit.Dp(80)))
	max := float32(gtx.Dp(unit.Dp(400)))
	if a.logPaneHeight < min {
		a.logPaneHeight = min
	}
	if a.logPaneHeight > max {
		a.logPaneHeight = max
	}
}

func (a *App) layoutReverse(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	return layout.Inset{
		Top: unit.Dp(32), Left: unit.Dp(32), Right: unit.Dp(32),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.H4(a.Theme, "Reverse Engineering Center Panel").Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(material.Body1(a.Theme, "Workspace for orchestrating boundary-scan reverse-engineering sessions.").Layout),
			layout.Rigid(material.Body2(a.Theme, "Integrate reveng workspace component here.").Layout),
		)
	})
}

func (a *App) layoutFootprint(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	return layout.Inset{
		Top: unit.Dp(32), Left: unit.Dp(32), Right: unit.Dp(32),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.H4(a.Theme, "Footprint Center Panel").Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(material.Body1(a.Theme, "Package visualization workspace coming soon.").Layout),
			layout.Rigid(material.Body2(a.Theme, "Use this panel to explore package outlines, pad states, and pin metadata.").Layout),
		)
	})
}

func (a *App) layoutInterfaces(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	filtered := filterInterfaces(state.Interfaces, state.DriverKind)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(a.Theme, &a.scanInterfacesBtn, "Scan Interfaces")
			if state.Busy {
				btn.Text = "Scanning..."
			}
			dims := btn.Layout(gtx)
			for a.scanInterfacesBtn.Clicked(gtx) {
				go a.ScanInterfaces()
			}
			return dims
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.interfaceList.Layout(gtx, len(filtered), func(gtx layout.Context, idx int) layout.Dimensions {
				if idx >= len(filtered) {
					return layout.Dimensions{}
				}
				item := filtered[idx]
				clk := a.interfaceClickable(item.Index)
				card := material.Button(a.Theme, clk, item.Info.Label())
				if item.Index == state.SelectedInterface {
					card.Text = "▶ " + card.Text
				}
				dims := card.Layout(gtx)
				for clk.Clicked(gtx) {
					a.State.SelectInterface(item.Index)
					a.invalidate()
				}
				return dims
			})
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(a.Theme, &a.connectInterfaceBtn, "Test Connection")
			dims := btn.Layout(gtx)
			for a.connectInterfaceBtn.Clicked(gtx) {
				go a.ConnectSelectedInterface()
			}
			return dims
		}),
	)
}

func (a *App) layoutChain(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(a.Theme, &a.scanChainBtn, "Scan Chain")
			if state.Busy {
				btn.Text = "Scanning..."
			}
			dims := btn.Layout(gtx)
			for a.scanChainBtn.Clicked(gtx) {
				go a.ScanChain()
			}
			return dims
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(state.Chain) == 0 {
				lbl := material.Body2(a.Theme, "No devices discovered yet.")
				return lbl.Layout(gtx)
			}
			return a.chainList.Layout(gtx, len(state.Chain), func(gtx layout.Context, idx int) layout.Dimensions {
				dev := state.Chain[idx]
				clk := a.deviceClickable(idx)
				btn := material.Button(a.Theme, clk, fmt.Sprintf("#%d %s", dev.Index, dev.Name))
				if idx == state.SelectedIdx {
					btn.Text = fmt.Sprintf("▶ #%d %s", dev.Index, dev.Name)
				}
				dims := btn.Layout(gtx)
				for clk.Clicked(gtx) {
					a.State.SelectDevice(idx)
					a.invalidate()
				}
				return dims
			})
		}),
	)
}

func (a *App) layoutDetails(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	if state.SelectedIdx < 0 || state.SelectedIdx >= len(state.Chain) {
		lbl := material.Body1(a.Theme, "Select a device to view details.")
		return lbl.Layout(gtx)
	}
	dev := state.Chain[state.SelectedIdx]
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H5(a.Theme, dev.Name).Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(material.Body2(a.Theme, fmt.Sprintf("Index: %d", dev.Index)).Layout),
		layout.Rigid(material.Body2(a.Theme, fmt.Sprintf("IDCODE: 0x%08X", dev.IDCode)).Layout),
		layout.Rigid(material.Body2(a.Theme, fmt.Sprintf("Package: %s", dev.Package)).Layout),
		layout.Rigid(material.Body2(a.Theme, fmt.Sprintf("BSDL: %s", dev.BSDLFile)).Layout),
	)
}

func (a *App) layoutLogs(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	if len(state.Logs) == 0 {
		lbl := material.Caption(a.Theme, "Logs will appear here.")
		return lbl.Layout(gtx)
	}
	return a.logList.Layout(gtx, len(state.Logs), func(gtx layout.Context, idx int) layout.Dimensions {
		if idx >= len(state.Logs) {
			return layout.Dimensions{}
		}
		lbl := material.Caption(a.Theme, state.Logs[idx])
		lbl.Color = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
		return lbl.Layout(gtx)
	})
}

func (a *App) layoutStatus(gtx layout.Context, state StateSnapshot) layout.Dimensions {
	driverFilter := fmt.Sprintf("Driver Filter: %s", driverKindLabel(state.DriverKind))
	interfaceLabel := "Interface: none"
	if state.SelectedInterface >= 0 && state.SelectedInterface < len(state.Interfaces) {
		interfaceLabel = fmt.Sprintf("Interface: %s", state.Interfaces[state.SelectedInterface].Label())
	}
	adapterLabel := "Adapter: none"
	if state.AdapterInfo != nil {
		name := state.AdapterInfo.Name
		if state.AdapterInfo.SerialNumber != "" {
			name = fmt.Sprintf("%s (%s)", name, state.AdapterInfo.SerialNumber)
		}
		adapterLabel = fmt.Sprintf("Adapter: %s", name)
	}
	connLabel := "Disconnected"
	if state.Connected {
		connLabel = "Connected"
	}
	statusLabel := fmt.Sprintf("Status: %s", state.Status)
	versionLabel := fmt.Sprintf("Version: %s", state.AppVersion)

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, color.NRGBA{R: 230, G: 234, B: 244, A: 255}, clip.Rect{Max: gtx.Constraints.Max}.Op())
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			inset := layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(8), Bottom: unit.Dp(8)}
			return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(material.Body2(a.Theme, versionLabel).Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(18)}.Layout),
					layout.Rigid(material.Body2(a.Theme, driverFilter).Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(18)}.Layout),
					layout.Rigid(material.Body2(a.Theme, interfaceLabel).Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(18)}.Layout),
					layout.Rigid(material.Body2(a.Theme, adapterLabel).Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(18)}.Layout),
					layout.Rigid(material.Body2(a.Theme, connLabel).Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{}
					}),
					layout.Rigid(material.Body2(a.Theme, statusLabel).Layout),
				)
			})
		}),
	)
}

func (a *App) interfaceClickable(idx int) *widget.Clickable {
	if clk, ok := a.interfaceClicks[idx]; ok {
		return clk
	}
	clk := &widget.Clickable{}
	a.interfaceClicks[idx] = clk
	return clk
}

func (a *App) deviceClickable(idx int) *widget.Clickable {
	if clk, ok := a.deviceClicks[idx]; ok {
		return clk
	}
	clk := &widget.Clickable{}
	a.deviceClicks[idx] = clk
	return clk
}

// invalidate requests a new frame.
func (a *App) invalidate() {
	if a.Window != nil {
		a.Window.Invalidate()
	}
}

// ScanChain triggers chain discovery.
func (a *App) ScanChain() {
	if a.State.Busy() {
		return
	}
	a.State.SetBusy(true)
	a.State.SetStatus("Scanning boundary scan chain...")
	a.State.AppendLog("Starting chain scan")
	a.invalidate()

	adapter := a.State.Adapter()
	go func() {
		defer func() {
			a.State.SetBusy(false)
			a.invalidate()
		}()

		if adapter == nil {
			a.State.SetError(errors.New("no adapter connected"))
			a.State.AppendLog("Scan aborted: no adapter connected")
			a.State.SetStatus("Connect an adapter to scan the chain")
			return
		}

		time.Sleep(750 * time.Millisecond) // TODO: integrate with real discovery.
		devices := []JTAGDevice{
			{
				Index:    0,
				IDCode:   0x0BA00477,
				Name:     "Demo MCU",
				Package:  "QFP-64",
				BSDLFile: "demo_mcu.bsdl",
			},
			{
				Index:    1,
				IDCode:   0x59656093,
				Name:     "Demo FPGA",
				Package:  "BGA-256",
				BSDLFile: "demo_fpga.bsdl",
			},
		}

		a.State.SetChain(devices)
		a.State.SetError(nil)
		a.State.SetStatus(fmt.Sprintf("Discovered %d device(s)", len(devices)))
		a.State.AppendLog(fmt.Sprintf("Scan finished: %d device(s) found", len(devices)))
		a.invalidate()
	}()
}

// ScanInterfaces enumerates available adapters.
func (a *App) ScanInterfaces() {
	a.State.SetStatus("Scanning interfaces...")
	a.State.AppendLog("Scanning interfaces...")
	a.invalidate()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		infos, err := jtag.DiscoverInterfaces(ctx)
		if err != nil {
			a.State.SetError(err)
			a.State.AppendLog(fmt.Sprintf("Interface scan failed: %v", err))
			a.State.SetStatus("Interface scan failed")
		} else {
			a.State.SetInterfaces(infos)
			a.State.SetError(nil)
			if len(infos) == 0 {
				a.State.SetStatus("No interfaces found")
				a.State.AppendLog("No interfaces detected")
			} else {
				a.State.SetStatus(fmt.Sprintf("Found %d interface(s)", len(infos)))
				a.State.AppendLog(fmt.Sprintf("Found %d interface(s)", len(infos)))
			}
		}
		a.invalidate()
	}()
}

// ConnectSelectedInterface logs the selection.
func (a *App) ConnectSelectedInterface() {
	iface := a.State.SelectedInterface()
	if iface == nil {
		return
	}
	a.State.AppendLog(fmt.Sprintf("Selected interface: %s", iface.Label()))
	a.State.SetStatus(fmt.Sprintf("Interface selected: %s", iface.Label()))
	a.invalidate()
}
