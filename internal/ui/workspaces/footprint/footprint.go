package footprint

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/OpenTraceLab/OpenTraceJTAG/internal/ui/components"
	"github.com/oligo/gioview/menu"
	"github.com/oligo/gioview/theme"
)

type packageType string

const (
	pkgTSOP packageType = "TSOP"
	pkgQFP  packageType = "QFP"
	pkgQFN  packageType = "QFN"
	pkgBGA  packageType = "BGA"
)

// App hosts the Gio footprint viewer window/workspace.
type App struct {
	window *app.Window
	theme  *material.Theme
	ops    op.Ops

	pkgType     packageType
	pkgOptions  []string
	pkgMenu     *menu.DropdownMenu
	pkgMenuBtn  widget.Clickable

	tsopPinsEditor   widget.Editor
	tsopWidthEditor  widget.Editor
	tsopHeightEditor widget.Editor
	tsopWideBtn      widget.Clickable
	tsopNarrowBtn    widget.Clickable
	tsopIsWide       bool
	
	qfpPinsEditor   widget.Editor
	qfpWidthEditor  widget.Editor
	qfpHeightEditor widget.Editor
	
	qfnPinsEditor   widget.Editor
	qfnWidthEditor  widget.Editor
	qfnHeightEditor widget.Editor
	
	bgaColsEditor  widget.Editor
	bgaRowsEditor  widget.Editor
	bgaPitchEditor widget.Editor

	zoom widget.Float

	pinData    map[packageType][]components.PackagePin
	stateButtons map[components.PinState]*widget.Clickable
	selectedPad  int

	padClicks map[int]*widget.Clickable
	
	onInvalidate func()
}

// NewApp creates a ready-to-run footprint viewer window.
func NewApp() *App {
	return NewAppWithWindow(new(app.Window))
}

// NewAppWithWindow wires the footprint viewer to a supplied window.
func NewAppWithWindow(win *app.Window) *App {
	appInstance := &App{
		window:  win,
		theme:   material.NewTheme(),
		pkgType: pkgTSOP,
		pkgOptions: []string{"TSOP", "QFP", "QFN", "BGA"},
		pinData: map[packageType][]components.PackagePin{
			pkgTSOP: components.DefaultPins(48),
			pkgQFP:  components.DefaultPins(64),
			pkgQFN:  components.DefaultPins(48),
			pkgBGA:  components.DefaultPins(100),
		},
		stateButtons: make(map[components.PinState]*widget.Clickable),
		padClicks:    make(map[int]*widget.Clickable),
		selectedPad:  -1,
	}
	appInstance.pkgMenu = appInstance.buildPackageMenu()
	appInstance.tsopIsWide = true // Default to TSOP-II (wide)
	appInstance.tsopPinsEditor.SingleLine = true
	appInstance.tsopPinsEditor.SetText("48")
	appInstance.tsopWidthEditor.SingleLine = true
	appInstance.tsopWidthEditor.SetText("10")
	appInstance.tsopHeightEditor.SingleLine = true
	appInstance.tsopHeightEditor.SetText("5")
	
	appInstance.qfpPinsEditor.SingleLine = true
	appInstance.qfpPinsEditor.SetText("16")
	appInstance.qfpWidthEditor.SingleLine = true
	appInstance.qfpWidthEditor.SetText("10")
	appInstance.qfpHeightEditor.SingleLine = true
	appInstance.qfpHeightEditor.SetText("10")
	
	appInstance.qfnPinsEditor.SingleLine = true
	appInstance.qfnPinsEditor.SetText("12")
	appInstance.qfnWidthEditor.SingleLine = true
	appInstance.qfnWidthEditor.SetText("5")
	appInstance.qfnHeightEditor.SingleLine = true
	appInstance.qfnHeightEditor.SetText("5")
	
	appInstance.bgaColsEditor.SingleLine = true
	appInstance.bgaColsEditor.SetText("10")
	appInstance.bgaRowsEditor.SingleLine = true
	appInstance.bgaRowsEditor.SetText("10")
	appInstance.bgaPitchEditor.SingleLine = true
	appInstance.bgaPitchEditor.SetText("0.8")
	appInstance.zoom.Value = 0.2

	for _, st := range []components.PinState{
		components.PinStateUnknown,
		components.PinStateHigh,
		components.PinStateLow,
		components.PinStateHighZ,
		components.PinStatePower,
	} {
		appInstance.stateButtons[st] = new(widget.Clickable)
	}
	return appInstance
}

// Run executes the Gio event loop until the window closes.
func (f *App) Run() error {
	f.window.Option(app.Title("Footprint Viewer"), app.Size(unit.Dp(1200), unit.Dp(800)))
	for {
		e := f.window.Event()
		switch ev := e.(type) {
		case app.DestroyEvent:
			return ev.Err
		case app.FrameEvent:
			gtx := app.NewContext(&f.ops, ev)
			f.layout(gtx)
			ev.Frame(gtx.Ops)
		}
	}
}

// SetTheme allows embedding contexts to override the material theme.
func (f *App) SetTheme(th *material.Theme) {
	if th != nil {
		f.theme = th
	}
}

// SetInvalidateCallback sets a callback to notify parent when redraw is needed.
func (f *App) SetInvalidateCallback(cb func()) {
	f.onInvalidate = cb
}

func (f *App) buildPackageMenu() *menu.DropdownMenu {
	opts := make([]menu.MenuOption, 0, len(f.pkgOptions))
	for i, name := range f.pkgOptions {
		idx := i
		label := name
		opts = append(opts, menu.MenuOption{
			OnClicked: func() error {
				f.pkgType = packageType(f.pkgOptions[idx])
				f.selectedPad = -1
				f.invalidate()
				return nil
			},
			Layout: func(gtx menu.C, th *theme.Theme) menu.D {
				lbl := material.Body1(th.Theme, label)
				if packageType(label) == f.pkgType {
					lbl.Color = th.Palette.ContrastBg
				}
				return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, lbl.Layout)
			},
		})
	}
	drop := menu.NewDropdownMenu([][]menu.MenuOption{opts})
	drop.MaxWidth = unit.Dp(180)
	return drop
}

// LayoutEmbedded renders the footprint UI within another Gio application.
func (f *App) LayoutEmbedded(gtx layout.Context) layout.Dimensions {
	if f.theme == nil {
		f.theme = material.NewTheme()
	}
	return f.layout(gtx)
}

func (f *App) invalidate() {
	if f.window != nil {
		f.window.Invalidate()
	}
	if f.onInvalidate != nil {
		f.onInvalidate()
	}
}

func (f *App) layout(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(16), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return f.layoutControls(gtx)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return f.layoutViewer(gtx)
			})
		}),
	)
}

func (f *App) layoutControls(gtx layout.Context) layout.Dimensions {
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(f.theme, "Package Type").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return f.layoutPackageDropdown(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
	}

	// Add package-specific inputs
	switch f.pkgType {
	case pkgTSOP:
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.numericField(gtx, "Pin Count (16-68)", &f.tsopPinsEditor)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Body2(f.theme, "Variant").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Spacing: layout.SpaceEvenly}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(f.theme, &f.tsopNarrowBtn, "TSOP-I")
						if !f.tsopIsWide {
							btn.Background = color.NRGBA{R: 120, G: 140, B: 250, A: 255}
						}
						if f.tsopNarrowBtn.Clicked(gtx) {
							f.tsopIsWide = false
							f.invalidate()
						}
						return btn.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(f.theme, &f.tsopWideBtn, "TSOP-II")
						if f.tsopIsWide {
							btn.Background = color.NRGBA{R: 120, G: 140, B: 250, A: 255}
						}
						if f.tsopWideBtn.Clicked(gtx) {
							f.tsopIsWide = true
							f.invalidate()
						}
						return btn.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		)
	case pkgQFP:
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.numericField(gtx, "Pins per Side", &f.qfpPinsEditor)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.numericField(gtx, "Width (mm)", &f.qfpWidthEditor)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.numericField(gtx, "Height (mm)", &f.qfpHeightEditor)
			}),
		)
	case pkgQFN:
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.numericField(gtx, "Pins per Side", &f.qfnPinsEditor)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.numericField(gtx, "Width (mm)", &f.qfnWidthEditor)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.numericField(gtx, "Height (mm)", &f.qfnHeightEditor)
			}),
		)
	case pkgBGA:
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.numericField(gtx, "Columns", &f.bgaColsEditor)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.numericField(gtx, "Rows", &f.bgaRowsEditor)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.numericField(gtx, "Pitch (mm)", &f.bgaPitchEditor)
			}),
		)
	}

	children = append(children,
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			title := material.Body1(f.theme, fmt.Sprintf("Zoom: %d%%", int(f.currentScale()*100)))
			return title.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(4), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return f.layoutSlider(gtx)
			})
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return f.layoutPadInspector(gtx)
		}),
	)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (f *App) layoutPackageDropdown(gtx layout.Context) layout.Dimensions {
	current := string(f.pkgType)
	if f.pkgMenuBtn.Clicked(gtx) {
		f.pkgMenu.ToggleVisibility(gtx)
	}
	btn := material.Button(f.theme, &f.pkgMenuBtn, current)
	dims := btn.Layout(gtx)
	
	// Layout menu after button so it appears on top
	if f.pkgMenu != nil {
		gvTheme := theme.NewTheme("", nil, true)
		f.pkgMenu.Layout(gtx, gvTheme)
	}
	
	return dims
}

func (f *App) numericField(gtx layout.Context, label string, editor *widget.Editor) layout.Dimensions {
	editor.SingleLine = true
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body2(f.theme, label).Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			ed := material.Editor(f.theme, editor, "")
			ed.TextSize = unit.Sp(14)
			return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(6)}.Layout(gtx, ed.Layout)
		}),
	)
}

func (f *App) layoutSlider(gtx layout.Context) layout.Dimensions {
	if f.zoom.Update(gtx) {
		f.invalidate()
	}
	return material.Slider(f.theme, &f.zoom).Layout(gtx)
}

func (f *App) layoutPadInspector(gtx layout.Context) layout.Dimensions {
	if f.selectedPad < 0 {
		return material.Body2(f.theme, "Select a pad to edit state").Layout(gtx)
	}
	pin := f.activePins()[f.selectedPad]
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.Body1(f.theme, fmt.Sprintf("Pad %d (%s)", pin.Number, pin.Name)).Layout),
		layout.Rigid(material.Caption(f.theme, fmt.Sprintf("State: %s", pin.State)).Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(f.stateButton(gtx, components.PinStateHigh, "High")),
				layout.Rigid(f.stateButton(gtx, components.PinStateLow, "Low")),
				layout.Rigid(f.stateButton(gtx, components.PinStateHighZ, "High-Z")),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(f.stateButton(gtx, components.PinStatePower, "Power")),
				layout.Rigid(f.stateButton(gtx, components.PinStateUnknown, "Unknown")),
			)
		}),
	)
}

func (f *App) stateButton(gtx layout.Context, state components.PinState, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		btn := material.Button(f.theme, f.stateButtons[state], label)
		if f.selectedPad >= 0 && f.activePins()[f.selectedPad].State == state {
			btn.Background = color.NRGBA{R: 140, G: 160, B: 255, A: 255}
		}
		dims := btn.Layout(gtx)
		for f.stateButtons[state].Clicked(gtx) {
			if f.selectedPad >= 0 {
				pins := f.activePins()
				pins[f.selectedPad].State = state
				f.invalidate()
			}
		}
		return dims
	}
}

func (f *App) layoutViewer(gtx layout.Context) layout.Dimensions {
	render := f.currentRender()
	if render == nil {
		return material.Body1(f.theme, "Invalid configuration").Layout(gtx)
	}
	scale := f.currentScale()
	width := int(render.Size.X*scale) + gtx.Dp(unit.Dp(40))
	height := int(render.Size.Y*scale) + gtx.Dp(unit.Dp(40))
	gtx.Constraints = layout.Exact(image.Pt(width, height))
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return f.drawShapes(gtx, render, scale)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return f.layoutPadHits(gtx, render, scale)
		}),
	)
}

func (f *App) drawShapes(gtx layout.Context, render *components.PackageRender, scale float32) layout.Dimensions {
	margin := float32(gtx.Dp(unit.Dp(20)))
	off := op.Offset(image.Pt(int(margin), int(margin))).Push(gtx.Ops)
	for _, rect := range render.Rectangles {
		drawRect(gtx, rect, scale)
	}
	for _, circle := range render.Circles {
		drawCircle(gtx, circle, scale)
	}
	for _, label := range render.Labels {
		drawLabel(gtx, f.theme, label, scale)
	}
	off.Pop()
	return layout.Dimensions{Size: image.Pt(int(render.Size.X*scale+margin*2), int(render.Size.Y*scale+margin*2))}
}

func drawRect(gtx layout.Context, rect components.RectShape, scale float32) {
	x := int(rect.Position.X * scale)
	y := int(rect.Position.Y * scale)
	offset := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	defer offset.Pop()
	sz := image.Pt(int(rect.Size.X*scale), int(rect.Size.Y*scale))
	cl := clip.Rect{Max: sz}.Push(gtx.Ops)
	paint.Fill(gtx.Ops, rect.Fill)
	cl.Pop()
}

func drawCircle(gtx layout.Context, circle components.CircleShape, scale float32) {
	size := int(circle.Radius * 2 * scale)
	x := int((circle.Center.X - circle.Radius) * scale)
	y := int((circle.Center.Y - circle.Radius) * scale)
	offset := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	defer offset.Pop()
	rect := image.Rect(0, 0, size, size)
	cl := clip.UniformRRect(rect, size/2).Push(gtx.Ops)
	paint.Fill(gtx.Ops, circle.Fill)
	cl.Pop()
}

func drawLabel(gtx layout.Context, th *material.Theme, label components.LabelShape, scale float32) {
	lbl := material.Label(th, unit.Sp(label.Size*scale/1.2), label.Text)
	lbl.Color = label.Color
	switch label.Align {
	case components.AlignStart:
		lbl.Alignment = text.Start
	case components.AlignCenter:
		lbl.Alignment = text.Middle
	case components.AlignEnd:
		lbl.Alignment = text.End
	}
	offset := op.Offset(image.Pt(int(label.Position.X*scale), int(label.Position.Y*scale))).Push(gtx.Ops)
	lbl.Layout(gtx)
	offset.Pop()
}

func (f *App) layoutPadHits(gtx layout.Context, render *components.PackageRender, scale float32) layout.Dimensions {
	margin := float32(gtx.Dp(unit.Dp(20)))
	for _, pad := range render.Pads {
		clk := f.padClickable(pad.Index)
		x := int(margin + pad.Position.X*scale)
		y := int(margin + pad.Position.Y*scale)
		offset := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		size := image.Pt(int(pad.Size.X*scale), int(pad.Size.Y*scale))
		clk.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: size}
		})
		offset.Pop()
		if clk.Clicked(gtx) {
			f.selectedPad = pad.Index
			f.invalidate()
		}
	}
	return layout.Dimensions{Size: image.Pt(int(render.Size.X*scale+margin*2), int(render.Size.Y*scale+margin*2))}
}

func (f *App) padClickable(idx int) *widget.Clickable {
	if clk, ok := f.padClicks[idx]; ok {
		return clk
	}
	clk := new(widget.Clickable)
	f.padClicks[idx] = clk
	return clk
}

func (f *App) currentScale() float32 {
	return 0.5 + f.zoom.Value*2.5
}

func (f *App) currentRender() *components.PackageRender {
	opts := &components.RenderOptions{Scale: 1, ShowLabels: true}
	switch f.pkgType {
	case pkgTSOP:
		count := clampInt(parseInt(f.tsopPinsEditor.Text(), 48), 16, 68)
		// Ensure even pin count
		if count%2 != 0 {
			count++
		}
		pins := f.ensurePins(pkgTSOP, count)
		render := components.NewTSOPPackage(count, pins, f.tsopIsWide, opts)
		return &render
	case pkgQFP:
		perSide := clampInt(parseInt(f.qfpPinsEditor.Text(), 16), 2, 64)
		pins := f.ensurePins(pkgQFP, perSide*4)
		render := components.NewQFPPackage(perSide, pins, opts)
		return &render
	case pkgQFN:
		perSide := clampInt(parseInt(f.qfnPinsEditor.Text(), 12), 2, 64)
		pins := f.ensurePins(pkgQFN, perSide*4)
		render := components.NewQFNPackage(perSide, pins, opts)
		return &render
	case pkgBGA:
		cols := clampInt(parseInt(f.bgaColsEditor.Text(), 10), 2, 32)
		rows := clampInt(parseInt(f.bgaRowsEditor.Text(), 10), 2, 32)
		pins := f.ensurePins(pkgBGA, cols*rows)
		render := components.NewBGAPackage(cols, rows, pins, opts)
		return &render
	}
	return nil
}

func (f *App) activePins() []components.PackagePin {
	return f.pinData[f.pkgType]
}

func (f *App) ensurePins(pkg packageType, count int) []components.PackagePin {
	pins := f.pinData[pkg]
	if len(pins) != count {
		pins = components.EnsurePins(pins, count)
		f.pinData[pkg] = pins
		if f.pkgType == pkg && f.selectedPad >= len(pins) {
			f.selectedPad = -1
		}
	}
	return pins
}

func parseInt(value string, fallback int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && v > 0 {
		return v
	}
	return fallback
}

func parseFloat(value string, fallback float32) float32 {
	if v, err := strconv.ParseFloat(strings.TrimSpace(value), 32); err == nil && v > 0 {
		return float32(v)
	}
	return fallback
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
