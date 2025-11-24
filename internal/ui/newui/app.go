package newui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	gfont "gioui.org/font"
	"gioui.org/font/gofont"
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
	"gioui.org/x/explorer"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/OpenTraceLab/OpenTraceJTAG/internal/ui/workspaces/footprint"
	"github.com/OpenTraceLab/OpenTraceJTAG/internal/ui/workspaces/reverse"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/reveng"
	kicadparser "github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/parser"
	kicadrenderer "github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer"
	"github.com/oligo/gioview/menu"
	"github.com/oligo/gioview/theme"
)

// App drives the experimental GioView-based UI.
type App struct {
	window *app.Window
	ops    op.Ops

	gvTheme  *theme.Theme
	darkMode bool

	settingsIcon     *widget.Icon
	revIcon          *widget.Icon
	debugIcon        *widget.Icon
	netlistIcon      *widget.Icon
	footprintIcon    *widget.Icon
	leftHandleClick  gesture.Click
	rightHandleClick gesture.Click

	leftPanelVisible   bool
	rightPanelVisible  bool
	rightPanelWidth    int // Dynamic width in dp
	rightPanelDrag     gesture.Drag
	rightPanelDragging bool
	rightPanelLastX    float32

	navItems    []string
	navClicks   []widget.Clickable
	selectedNav int

	chainItems []string
	chainList  widget.List

	logs          []string
	logText       string
	logSelectable widget.Selectable
	logPaneHeight float32
	logSplitter   gesture.Drag
	logSplitDrag  bool
	logSplitLastY float32
	logList       widget.List

	statusText string

	windowWidth int
	centerWidth int
	logWidth    int
	loggedSizes bool

	debugOverlay bool

	monoShaper *text.Shaper

	workspaces []workspaceView

	darkModeSwitch  widget.Bool
	debugModeSwitch widget.Bool

	driverOptions  []string
	selectedDriver int
	driverMenu     *menu.DropdownMenu
	driverMenuBtn  widget.Clickable

	footprintView *footprint.App
	reverseView   *reverse.App

	// Chain discovery state
	chainDevices       []ChainDevice
	deviceCardList     widget.List
	chainSimulator     *jtag.ChainSimulator
	chainRepository    *chain.MemoryRepository
	chainController    *chain.Controller
	chainDiscovered    *chain.Chain
	scanChainBtn       widget.Clickable
	isScanning         bool
	scanProgress       string
	reverseEngineerBtn widget.Clickable
	isReverseEngineering bool
	reverseProgress    string
	netEditor          *NetEditor
	
	// Reverse engineering state
	discoveredNetlist *reveng.Netlist
	ratsnestLines     []RatsnestLine
	currentScanPin    string // Currently scanned pin for visual feedback
	
	// Footprint viewport state
	viewportZoom      float32
	viewportPanX      float32
	viewportPanY      float32
	viewportDrag      gesture.Drag
	viewportClick     gesture.Click
	viewportScroll    gesture.Scroll
	viewportDragStart f32.Point
	debugLogOnce      bool // Flag to log line coordinates once after footprint change
	viewportDragging  bool
	
	// Tooltip state
	tooltipVisible    bool
	tooltipText       string
	tooltipPos        f32.Point
	hoveredDevice     int // Index of device being hovered (-1 if none)
	hoveredPin        int // Pin number being hovered (-1 if none)
	selectedNetID     int // Net ID to highlight (-1 if none)
	
	// Context menu state
	contextMenuVisible bool
	contextMenuPos     f32.Point
	contextMenuDevice  int // Device index for context menu
	contextMenuPin     int // Pin number for context menu
	contextMenuOptions []widget.Clickable // Menu option buttons
	renderedPads       []RenderedPad // Pad bounds for hit testing

	boardFilePath   string
	boardFileBtn    widget.Clickable
	boardExplorer   *explorer.Explorer
	loadedBoard     *kicadparser.Board
	boardCamera     *kicadrenderer.Camera
	showBoardViewer bool
	layerConfig     *kicadrenderer.LayerConfig
	layerToggles    map[string]*widget.Bool
	layerSelectAll  widget.Clickable
	layerDeselectAll widget.Clickable
	
	// Component selector state
	componentSelectorVisible bool
	componentSelectorDevice int
	componentButtons      map[string]*widget.Clickable
	
	// Board color theme
	boardColorTheme kicadrenderer.ColorTheme
	themeButtons    [5]widget.Clickable
}





// New creates the GioView UI app.
func New(w *app.Window) *App {
	if w == nil {
		w = new(app.Window)
	}
	w.Option(app.Title("JTAG New UI"), app.Size(unit.Dp(1360), unit.Dp(860)))

	gv := theme.NewTheme("", nil, true)
	app := &App{
		window:            w,
		gvTheme:           gv,
		leftPanelVisible:  true,
		rightPanelVisible: false,
		rightPanelWidth:   320, // Default width in dp
		viewportZoom:      1.0, // Default zoom level
		hoveredDevice:     -1,  // No device hovered initially
		hoveredPin:        -1,  // No pin hovered initially
		selectedNetID:     -1,  // No net selected initially
		contextMenuDevice: -1,  // No context menu initially
		contextMenuPin:    -1,
		contextMenuOptions: make([]widget.Clickable, 3), // Hi, Lo, Hi-Z
		navItems:          []string{"Rev Eng", "Debug Board", "Netlist", "Footprints", "Settings"},
		chainItems: []string{
			"Chain Device A", "Chain Device B", "Chain Device C",
			"Chain Device D", "Chain Device E",
		},
		statusText: "Disconnected",
		workspaces: []workspaceView{
			{
				Description: "Manage adapters and driver configuration from this view.",
				QuickActions: []string{
					"Scan for interfaces",
					"Select driver implementation",
					"Configure transport speed",
				},
				Metrics: []workspaceMetric{
					{Label: "Adapters", Value: "3", Sub: "CMSIS-DAP, Pico, Debug Stub"},
					{Label: "Drivers", Value: "5", Sub: "USB, SWD, JTAG, SPI, Dummy"},
					{Label: "Profiles", Value: "2", Sub: "Lab rig, Field probe"},
				},
				Sections: []workspaceSection{
					{
						Title: "Recent Activity",
						Items: []string{
							"Lattice ECP5 chain connected",
							"CMSIS-DAP driver updated",
							"Transport speed locked at 4 MHz",
						},
					},
				},
			},
			{
				Description: "Plan reverse-engineering scans and review captured data.",
				QuickActions: []string{
					"Open scan template",
					"Queue dry run",
					"Attach BSDL libraries",
				},
				Metrics: []workspaceMetric{
					{Label: "Boards queued", Value: "4", Sub: "2 pending review"},
					{Label: "Scan time", Value: "~08:30", Sub: "avg per board"},
					{Label: "Coverage", Value: "92%", Sub: "pins analyzed"},
				},
				Sections: []workspaceSection{
					{
						Title: "Checklist",
						Items: []string{
							"Verify reference voltage",
							"Capture baseline BSR snapshot",
							"Export delta report",
						},
					},
					{
						Title: "Recent Notes",
						Items: []string{
							"STM32 chain discovery pending",
							"Run dry scan without power",
							"Cross-check KiCad net labels",
						},
					},
				},
			},
		},
	}
	monoFaces := filterMonoFaces()
	if len(monoFaces) > 0 {
		app.monoShaper = text.NewShaper(text.WithCollection(monoFaces), text.NoSystemFonts())
	}
	if icon, err := widget.NewIcon(icons.ActionSettings); err == nil {
		app.settingsIcon = icon
	}
	if icon, err := widget.NewIcon(icons.ActionBuild); err == nil {
		app.revIcon = icon
	}
	if icon, err := widget.NewIcon(icons.ActionBugReport); err == nil {
		app.debugIcon = icon
	}
	if icon, err := widget.NewIcon(icons.ActionList); err == nil {
		app.netlistIcon = icon
	}
	if icon, err := widget.NewIcon(icons.HardwareMemory); err == nil {
		app.footprintIcon = icon
	}
	app.darkModeSwitch.Value = app.darkMode
	app.debugModeSwitch.Value = app.debugOverlay
	app.driverOptions = []string{"CMSIS-DAP", "Pico Debug", "Simulator"}
	app.driverMenu = app.buildDriverMenu()
	app.netEditor = NewNetEditor()
	app.footprintView = footprint.NewAppWithWindow(nil)
	app.footprintView.SetInvalidateCallback(func() { w.Invalidate() })
	app.reverseView = reverse.NewAppWithWindow(nil)
	app.boardExplorer = explorer.NewExplorer(w)
	app.navClicks = make([]widget.Clickable, len(app.navItems))
	app.logs = nil
	
	// Load config and apply theme
	if config, err := LoadConfig(); err == nil {
		app.boardColorTheme = kicadrenderer.ColorTheme(config.BoardColorTheme)
		kicadrenderer.SetTheme(app.boardColorTheme)
	}
	app.logSelectable.WrapPolicy = text.WrapGraphemes
	app.logList.Axis = layout.Vertical
	app.logList.ScrollToEnd = true
	app.chainList.Axis = layout.Vertical

	app.applyPalette()
	app.Logf("[BOOT] Experimental UI initialized")
	app.Logf("[INFO] Use the left buttons to switch apps")
	app.Logf("[INFO] Toggle panels to experiment with layout")
	return app
}

// Run blocks processing window events until the window closes.
func (a *App) Run() error {
	for {
		e := a.window.Event()
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

func (a *App) layout(gtx layout.Context) layout.Dimensions {
	a.windowWidth = gtx.Constraints.Max.X

	paint.FillShape(gtx.Ops, a.gvTheme.Palette.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(a.layoutHeader),
		layout.Flexed(1, a.layoutBody),
		layout.Rigid(a.layoutStatusBar),
	)
}

func (a *App) layoutHeader(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }

func (a *App) layoutBody(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(a.layoutHeader),
		layout.Flexed(1, a.layoutWorkspace),
		layout.Rigid(a.layoutLogSplitter),
		layout.Rigid(a.layoutLogPane),
	)
}

func (a *App) layoutWorkspace(gtx layout.Context) layout.Dimensions {
	bg := a.gvTheme.Bg2
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !a.leftPanelVisible {
				return layout.Dimensions{}
			}
			width := gtx.Dp(unit.Dp(220))
			gtx.Constraints.Min.X = width
			gtx.Constraints.Max.X = width
			return layout.Inset{Top: unit.Dp(16), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
				return a.layoutNav(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := "<"
			if !a.leftPanelVisible {
				label = ">"
			}
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(16))
			gtx.Constraints.Max.X = gtx.Constraints.Min.X
			return a.layoutPanelHandle(gtx, &a.leftHandleClick, label, func() {
				a.togglePanelVisibility("left", "handle")
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, a.layoutCenterCard)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := ">"
			if !a.rightPanelVisible {
				label = "<"
			}
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(16))
			gtx.Constraints.Max.X = gtx.Constraints.Min.X
			return a.layoutPanelHandle(gtx, &a.rightHandleClick, label, func() {
				a.togglePanelVisibility("right", "handle")
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !a.rightPanelVisible {
				return layout.Dimensions{}
			}
			
			// Handle drag events for resizing
			for {
				ev, ok := a.rightPanelDrag.Update(gtx.Metric, gtx.Source, gesture.Horizontal)
				if !ok {
					break
				}
				if ev.Kind == pointer.Press {
					a.rightPanelDragging = true
					a.rightPanelLastX = ev.Position.X
				} else if ev.Kind == pointer.Release {
					a.rightPanelDragging = false
				} else if ev.Kind == pointer.Drag && a.rightPanelDragging {
					// Adjust width (drag left = wider, drag right = narrower)
					deltaX := int(ev.Position.X - a.rightPanelLastX)
					a.rightPanelLastX = ev.Position.X
					newWidth := a.rightPanelWidth - deltaX
					// Clamp between 200 and 600 dp
					if newWidth < 200 {
						newWidth = 200
					} else if newWidth > 600 {
						newWidth = 600
					}
					a.rightPanelWidth = newWidth
				}
			}
			
			width := gtx.Dp(unit.Dp(float32(a.rightPanelWidth)))
			gtx.Constraints.Min.X = width
			gtx.Constraints.Max.X = width
			
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(12), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
						return a.layoutChains(gtx)
					})
				}),
				// Resize handle on left edge
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					handleWidth := gtx.Dp(unit.Dp(8))
					gtx.Constraints.Min.X = handleWidth
					gtx.Constraints.Max.X = handleWidth
					
					defer clip.Rect{Max: image.Pt(handleWidth, gtx.Constraints.Max.Y)}.Push(gtx.Ops).Pop()
					a.rightPanelDrag.Add(gtx.Ops)
					pointer.CursorColResize.Add(gtx.Ops)
					
					// Draw subtle handle indicator
					if a.rightPanelDragging {
						paint.Fill(gtx.Ops, color.NRGBA{R: 100, G: 100, B: 255, A: 100})
					}
					
					return layout.Dimensions{Size: image.Pt(handleWidth, gtx.Constraints.Max.Y)}
				}),
			)
		}),
	)
}

func (a *App) layoutNav(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		func() []layout.FlexChild {
			children := make([]layout.FlexChild, 0, len(a.navItems)*2)
			for i, item := range a.navItems {
				idx := i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.layoutNavItem(gtx, idx, item)
					})
				}))
			}
			return children
		}()...,
	)
}

func (a *App) layoutNavItem(gtx layout.Context, idx int, label string) layout.Dimensions {
	click := &a.navClicks[idx]
	for click.Clicked(gtx) {
		if a.selectedNav != idx {
			a.selectedNav = idx
			a.Logf("[INFO] Switched to %s view", label)
			
			// Rebuild ratsnest when switching to Rev Eng to show any manual edits
			if label == "Rev Eng" && a.discoveredNetlist != nil {
				a.Logf("[REVENG] Rebuilding ratsnest: %d nets in netlist", len(a.discoveredNetlist.Nets))
				a.buildRatsnest()
			}
			
			a.invalidate()
		}
	}
	height := gtx.Dp(unit.Dp(42))
	width := gtx.Constraints.Max.X
	if width == 0 {
		width = gtx.Dp(unit.Dp(180))
	}
	size := image.Pt(width, height)

	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = size
		gtx.Constraints.Max = size
		bg := a.gvTheme.Bg2
		fg := a.gvTheme.Palette.Fg
		if idx == a.selectedNav {
			bg = a.gvTheme.Palette.ContrastBg
			fg = a.gvTheme.Palette.ContrastFg
		}
		card := clip.RRect{
			Rect: image.Rectangle{Max: size},
			NW:   gtx.Dp(unit.Dp(6)),
			NE:   gtx.Dp(unit.Dp(6)),
			SW:   gtx.Dp(unit.Dp(6)),
			SE:   gtx.Dp(unit.Dp(6)),
		}
		paint.FillShape(gtx.Ops, bg, card.Op(gtx.Ops))
		return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{}
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				iconSize := gtx.Dp(unit.Dp(20))
				gtx.Constraints.Min = image.Pt(iconSize, iconSize)
				gtx.Constraints.Max = gtx.Constraints.Min
				var icon *widget.Icon
				switch label {
				case "Settings":
					icon = a.settingsIcon
				case "Rev Eng":
					icon = a.revIcon
				case "Debug Board":
					icon = a.debugIcon
				case "Netlist":
					icon = a.netlistIcon
				case "Footprints":
					icon = a.footprintIcon
				}
				if icon == nil {
					return layout.Dimensions{Size: gtx.Constraints.Min}
				}
				return icon.Layout(gtx, fg)
			}))
			children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout))
			children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(a.gvTheme.Theme, label)
				lbl.Color = fg
				return lbl.Layout(gtx)
			}))
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
			})
		})
	})
}

func (a *App) layoutCenterCard(gtx layout.Context) layout.Dimensions {
	a.centerWidth = gtx.Constraints.Max.X
	stackClip := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	defer stackClip.Pop()
	return a.withDebugOverlay(gtx, func(gtx layout.Context) layout.Dimensions {
		content := a.navItems[a.selectedNav]
		if a.selectedNav < len(a.workspaces) {
			content = fmt.Sprintf("%s Workspace", content)
		}
		cardColor := a.gvTheme.Palette.Bg
		if !a.darkMode {
			cardColor = color.NRGBA{R: 250, G: 250, B: 254, A: 255}
		}
		return layout.Stack{}.Layout(gtx,
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, cardColor, clip.RRect{
					Rect: image.Rectangle{Max: gtx.Constraints.Max},
					NW:   gtx.Dp(unit.Dp(12)),
					NE:   gtx.Dp(unit.Dp(12)),
					SW:   gtx.Dp(unit.Dp(12)),
					SE:   gtx.Dp(unit.Dp(12)),
				}.Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Max}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(24), Right: unit.Dp(24), Top: unit.Dp(24), Bottom: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(material.H4(a.gvTheme.Theme, content).Layout),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.layoutAppView(gtx)
						}),
					)
				})
			}),
		)
	})
}


func (a *App) layoutPanelHandle(gtx layout.Context, clk *gesture.Click, label string, toggle func()) layout.Dimensions {
	height := gtx.Constraints.Max.Y
	if height == 0 {
		height = gtx.Dp(unit.Dp(120))
	}
	width := gtx.Constraints.Max.X
	if width == 0 {
		width = gtx.Dp(unit.Dp(16))
	}
	size := image.Pt(width, height)

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			handleClip := clip.Rect{Max: size}.Push(gtx.Ops)
			pointer.CursorPointer.Add(gtx.Ops)
			clk.Add(gtx.Ops)
			for {
				ev, ok := clk.Update(gtx.Source)
				if !ok {
					break
				}
				if ev.Kind == gesture.KindClick {
					toggle()
				}
			}
			paint.FillShape(gtx.Ops, color.NRGBA{R: 176, G: 182, B: 206, A: 255}, clip.Rect{Max: size}.Op())
			handleClip.Pop()
			return layout.Dimensions{Size: size}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			top := gtx
			top.Constraints.Min = image.Point{}
			top.Constraints.Max = image.Point{X: size.X, Y: size.Y / 2}
			bottom := top
			bottom.Constraints.Min = image.Point{}
			bottom.Constraints.Max = image.Point{X: size.X, Y: size.Y / 2}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints = top.Constraints
					return layout.Dimensions{}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(a.gvTheme.Theme, label)
						lbl.Color = a.gvTheme.Palette.Fg
						return lbl.Layout(gtx)
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints = bottom.Constraints
					return layout.Dimensions{}
				}),
			)
		}),
	)
}

func (a *App) layoutLogSplitter(gtx layout.Context) layout.Dimensions {
	height := gtx.Dp(unit.Dp(8))
	if height < 4 {
		height = 4
	}
	size := image.Pt(gtx.Constraints.Max.X, height)
	paint.FillShape(gtx.Ops, color.NRGBA{R: 210, G: 214, B: 228, A: 255}, clip.Rect{Max: size}.Op())

	stack := clip.Rect{Max: size}.Push(gtx.Ops)
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

func (a *App) layoutLogPane(gtx layout.Context) layout.Dimensions {
	a.ensureLogPaneHeight(gtx)
	h := int(a.logPaneHeight)
	gtx.Constraints.Min.Y = h
	gtx.Constraints.Max.Y = h

	a.logWidth = gtx.Constraints.Max.X

	size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
	if size.X <= 0 {
		size.X = 1
	}
	if size.Y <= 0 {
		size.Y = 1
	}
	logClip := clip.Rect{Max: size}.Push(gtx.Ops)
	paint.FillShape(gtx.Ops, a.gvTheme.Bg2, clip.Rect{Max: size}.Op())
	logClip.Pop()

	return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return a.logList.Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
			label := material.Body2(a.gvTheme.Theme, a.logText)
			label.State = &a.logSelectable
			label.WrapPolicy = text.WrapGraphemes
			label.Alignment = text.Start
			label.Font.Typeface = gfont.Typeface("Go Mono")
			if a.monoShaper != nil {
				label.Shaper = a.monoShaper
			}
			label.Color = a.opaqueFg()
			label.SelectionColor = a.selectionColor()
			return label.Layout(gtx)
		})
	})
}

func (a *App) layoutStatusBar(gtx layout.Context) layout.Dimensions {
	inset := layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(8), Bottom: unit.Dp(8)}
	a.logSizesOnce()
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// Left: Operation status with animated dots
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				var msg string
				if a.scanProgress != "" {
					msg = a.scanProgress
				} else if a.reverseProgress != "" {
					msg = a.reverseProgress
				}
				
				if msg != "" {
					// Add animated dots if actively working
					if a.isScanning || a.isReverseEngineering {
						dots := int(time.Now().UnixMilli()/500) % 4
						msg += strings.Repeat(".", dots)
					}
					return material.Body2(a.gvTheme.Theme, msg).Layout(gtx)
				}
				return material.Body2(a.gvTheme.Theme, "Ready").Layout(gtx)
			}),
			// Spacer
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
			// Right: JTAG device status (fixed width for stability)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				// Reserve space for "JTAG: Disconnected" (longest text)
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(150))
				status := "Connected"
				if a.statusText != "" {
					status = a.statusText
				}
				return material.Body2(a.gvTheme.Theme, "JTAG: "+status).Layout(gtx)
			}),
		)
	})
}

func (a *App) layoutAppView(gtx layout.Context) layout.Dimensions {
	if len(a.navItems) == 0 {
		return material.Body1(a.gvTheme.Theme, "Select a workspace to begin.").Layout(gtx)
	}
	label := a.navItems[a.selectedNav]
	switch label {
	case "Settings":
		return a.layoutSettings(gtx)
	case "Debug Board":
		return a.layoutDebugBoard(gtx)
	case "Netlist":
		return a.layoutNetlist(gtx)
	case "Footprints":
		return a.layoutFootprints(gtx)
	case "Rev Eng":
		return a.layoutReverse(gtx)
	}
	if len(a.workspaces) == 0 {
		return material.Body1(a.gvTheme.Theme, "Select a workspace to begin.").Layout(gtx)
	}
	idx := a.selectedNav
	if idx >= len(a.workspaces) {
		idx = len(a.workspaces) - 1
	}
	if idx < 0 {
		idx = 0
	}
	ws := a.workspaces[idx]

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.Body1(a.gvTheme.Theme, ws.Description).Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(ws.Metrics) == 0 {
				return layout.Dimensions{}
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				func() []layout.FlexChild {
					children := make([]layout.FlexChild, 0, len(ws.Metrics))
					for _, metric := range ws.Metrics {
						m := metric
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							cardWidth := gtx.Dp(unit.Dp(180))
							gtx.Constraints.Min.X = cardWidth
							gtx.Constraints.Max.X = cardWidth
							return layout.Inset{Right: unit.Dp(12), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								cardColor := color.NRGBA{R: 255, G: 255, B: 255, A: 32}
								paint.FillShape(gtx.Ops, cardColor, clip.RRect{
									Rect: image.Rectangle{Max: gtx.Constraints.Max},
									NW:   gtx.Dp(unit.Dp(8)),
									NE:   gtx.Dp(unit.Dp(8)),
									SW:   gtx.Dp(unit.Dp(8)),
									SE:   gtx.Dp(unit.Dp(8)),
								}.Op(gtx.Ops))
								return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(material.Caption(a.gvTheme.Theme, m.Label).Layout),
										layout.Rigid(material.H6(a.gvTheme.Theme, m.Value).Layout),
										layout.Rigid(material.Body2(a.gvTheme.Theme, m.Sub).Layout),
									)
								})
							})
						}))
					}
					return children
				}()...,
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(ws.QuickActions) == 0 {
				return layout.Dimensions{}
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(material.Body2(a.gvTheme.Theme, "Quick actions:").Layout),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.layoutBulletList(gtx, ws.QuickActions)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				func() []layout.FlexChild {
					children := make([]layout.FlexChild, 0, len(ws.Sections)*2)
					for _, section := range ws.Sections {
						sec := section
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(material.Body2(a.gvTheme.Theme, sec.Title).Layout),
									layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.layoutBulletList(gtx, sec.Items)
									}),
								)
							})
						}))
					}
					return children
				}()...,
			)
		}),
	)
}

func (a *App) layoutDebugBoard(gtx layout.Context) layout.Dimensions {
	// Component selector overlay
	if a.componentSelectorVisible {
		return a.layoutComponentSelectorDialog(gtx)
	}
	
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Top action bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutDebugBoardActionBar(gtx)
		}),
		// Main content area
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if a.showBoardViewer && a.loadedBoard != nil {
				return a.layoutBoardViewer(gtx)
			}
			
			// Show prompt to load board
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(material.Body1(a.gvTheme.Theme, "No board loaded").Layout),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(material.Body2(a.gvTheme.Theme, "Click 'Open Board' to load a KiCad PCB file").Layout),
				)
			})
		}),
	)
}

func (a *App) layoutDebugBoardActionBar(gtx layout.Context) layout.Dimensions {
	// Handle file picker button
	if a.boardFileBtn.Clicked(gtx) {
		a.openBoardFilePicker()
	}

	// Handle scan chain button
	if a.scanChainBtn.Clicked(gtx) {
		go a.startChainScan()
	}

	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// Open Board button
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(a.gvTheme.Theme, &a.boardFileBtn, "Open Board")
				btn.Background = color.NRGBA{R: 63, G: 81, B: 181, A: 255} // Material blue
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
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
			// Show loaded board path
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if a.boardFilePath != "" {
					return material.Body2(a.gvTheme.Theme, "Loaded: "+a.boardFilePath).Layout(gtx)
				}
				return layout.Dimensions{}
			}),
		)
	})
}

func (a *App) layoutBoardViewer(gtx layout.Context) layout.Dimensions {
	if a.boardCamera == nil {
		a.boardCamera = kicadrenderer.NewCamera(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
		bbox := a.loadedBoard.GetBoundingBox()
		if !bbox.IsEmpty() {
			a.boardCamera.FitBoard(bbox)
		}
	}

	// Initialize layer config and toggles if needed
	if a.layerConfig == nil {
		a.layerConfig = kicadrenderer.NewLayerConfig()
		a.layerToggles = make(map[string]*widget.Bool)
		// Initialize toggles for all layers from the board (all enabled by default)
		for _, layer := range a.loadedBoard.Layers {
			a.layerToggles[layer.Name] = &widget.Bool{Value: true}
			a.layerConfig.SetVisible(layer.Name, true) // Actually set visibility in config
		}
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		// Device list panel on the left (if devices exist)
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(a.chainDevices) > 0 {
				return a.layoutDebugDevicePanel(gtx)
			}
			return layout.Dimensions{}
		}),
		// Board view (flexed to take remaining space)
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.layoutBoardViewport(gtx)
		}),
		// Layer panel on the right
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutLayerPanel(gtx)
		}),
	)
}

func (a *App) layoutBoardViewport(gtx layout.Context) layout.Dimensions {
	// Update camera size
	a.boardCamera.UpdateScreenSize(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)

	// Handle keyboard events
	for {
		ev, ok := gtx.Event(key.Filter{})
		if !ok {
			break
		}
		
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			switch ke.Name {
			case "F":
				a.boardCamera.Flip()
				gtx.Execute(op.InvalidateCmd{})
			case "R":
				a.boardCamera.Rotate(90)
				gtx.Execute(op.InvalidateCmd{})
			case key.NameLeftArrow:
				a.boardCamera.Rotate(-90)
				gtx.Execute(op.InvalidateCmd{})
			case key.NameSpace:
				bbox := a.loadedBoard.GetBoundingBox()
				if !bbox.IsEmpty() {
					a.boardCamera.FitBoard(bbox)
				}
				gtx.Execute(op.InvalidateCmd{})
			case "+", "=":
				// Zoom in at center
				centerX := float64(gtx.Constraints.Max.X) / 2
				centerY := float64(gtx.Constraints.Max.Y) / 2
				a.boardCamera.ZoomAt(centerX, centerY, 1.2)
				gtx.Execute(op.InvalidateCmd{})
			case "-":
				// Zoom out at center
				centerX := float64(gtx.Constraints.Max.X) / 2
				centerY := float64(gtx.Constraints.Max.Y) / 2
				a.boardCamera.ZoomAt(centerX, centerY, 0.8)
				gtx.Execute(op.InvalidateCmd{})
			}
		}
	}

	// Handle pointer events
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Kinds: pointer.Press | pointer.Scroll,
		})
		if !ok {
			break
		}
		
		if pe, ok := ev.(pointer.Event); ok {
			switch pe.Kind {
			case pointer.Press:
				if pe.Buttons == pointer.ButtonPrimary {
					a.boardCamera.Rotate(90)
					gtx.Execute(op.InvalidateCmd{})
				} else if pe.Buttons == pointer.ButtonSecondary {
					a.boardCamera.Flip()
					gtx.Execute(op.InvalidateCmd{})
				}
			case pointer.Scroll:
				if pe.Scroll.Y != 0 {
					zoomFactor := 1.0 + float64(pe.Scroll.Y)*0.1
					a.boardCamera.ZoomAt(float64(pe.Position.X), float64(pe.Position.Y), zoomFactor)
					gtx.Execute(op.InvalidateCmd{})
				}
			}
		}
	}

	// Register for input events
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, a)
	area.Pop()

	// Fill background with substrate color
	substrateColor := kicadrenderer.GetSubstrateColor()
	paint.Fill(gtx.Ops, substrateColor)

	// Render the board with layer config
	kicadrenderer.RenderBoardWithConfig(gtx, a.boardCamera, a.loadedBoard, a.layerConfig)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (a *App) layoutLayerPanel(gtx layout.Context) layout.Dimensions {
	// Handle select/deselect all buttons
	if a.layerSelectAll.Clicked(gtx) {
		for _, toggle := range a.layerToggles {
			toggle.Value = true
		}
		for _, layer := range a.loadedBoard.Layers {
			a.layerConfig.SetVisible(layer.Name, true)
		}
		gtx.Execute(op.InvalidateCmd{})
	}
	
	if a.layerDeselectAll.Clicked(gtx) {
		for _, toggle := range a.layerToggles {
			toggle.Value = false
		}
		for _, layer := range a.loadedBoard.Layers {
			a.layerConfig.SetVisible(layer.Name, false)
		}
		gtx.Execute(op.InvalidateCmd{})
	}
	
	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.H6(a.gvTheme.Theme, "Layers").Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			// Control buttons
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(a.gvTheme.Theme, &a.layerSelectAll, "All")
						return btn.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(a.gvTheme.Theme, &a.layerDeselectAll, "None")
						return btn.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			// Scrollable layer list
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return a.layoutLayerToggles(gtx)
			}),
		)
	})
}

func (a *App) layoutLayerToggles(gtx layout.Context) layout.Dimensions {
	// Get all layers from the loaded board
	if a.loadedBoard == nil {
		return layout.Dimensions{}
	}
	
	children := make([]layout.FlexChild, 0)
	for _, layer := range a.loadedBoard.Layers {
		layerName := layer.Name
		toggle, exists := a.layerToggles[layerName]
		if !exists {
			continue
		}
		
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// Check if toggle changed
			if toggle.Update(gtx) {
				a.layerConfig.SetVisible(layerName, toggle.Value)
				gtx.Execute(op.InvalidateCmd{})
			}
			
			// Get layer color and create colored checkbox
			layerColor := kicadrenderer.GetLayerColor(layerName)
			checkbox := material.CheckBox(a.gvTheme.Theme, toggle, layerName)
			checkbox.Color = layerColor
			return checkbox.Layout(gtx)
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout))
	}
	
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) layoutDebugDevicePanel(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.H6(a.gvTheme.Theme, "Chain Devices").Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutDebugDeviceList(gtx)
			}),
		)
	})
}

func (a *App) layoutDebugDeviceList(gtx layout.Context) layout.Dimensions {
	children := make([]layout.FlexChild, 0)
	
	for i := range a.chainDevices {
		deviceIdx := i
		device := &a.chainDevices[deviceIdx]
		
		children = append(children,
			// Device header
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Body1(a.gvTheme.Theme, fmt.Sprintf("Device %d", deviceIdx)).Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			// IDCODE
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Body2(a.gvTheme.Theme, fmt.Sprintf("IDCODE: 0x%08X", device.IDCode)).Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			// BSDL selector button
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if device.bsdlBtn.Clicked(gtx) {
					a.showBSDLSelector(deviceIdx)
				}
				bsdlText := "Select BSDL..."
				if device.BSDLPath != "" {
					bsdlText = "BSDL: " + device.BSDLPath
				}
				return material.Button(a.gvTheme.Theme, &device.bsdlBtn, bsdlText).Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			// Component selector button
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if device.componentBtn.Clicked(gtx) {
					a.showComponentSelector(deviceIdx)
				}
				componentText := "Select Component..."
				if device.ComponentRef != "" {
					componentText = "Component: " + device.ComponentRef
				}
				return material.Button(a.gvTheme.Theme, &device.componentBtn, componentText).Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		)
	}
	
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) showBSDLSelector(deviceIdx int) {
	// Open file picker for BSDL file
	go func() {
		file, err := a.boardExplorer.ChooseFile("bsd", "bsdl")
		if err != nil {
			if err != explorer.ErrUserDecline {
				a.Logf("[ERROR] BSDL file picker failed: %v", err)
			}
			return
		}
		defer file.Close()

		if f, ok := file.(*os.File); ok {
			a.assignBSDLToDevice(deviceIdx, f.Name())
		} else {
			a.Logf("[ERROR] Unable to get file path from picker")
		}
	}()
}

func (a *App) assignBSDLToDevice(deviceIdx int, bsdlPath string) {
	if deviceIdx >= len(a.chainDevices) {
		return
	}
	
	device := &a.chainDevices[deviceIdx]
	device.BSDLPath = bsdlPath
	
	// Parse BSDL file
	parser, err := bsdl.NewParser()
	if err != nil {
		a.Logf("[ERROR] Failed to create BSDL parser: %v", err)
		return
	}
	
	bsdlFile, err := parser.ParseFile(bsdlPath)
	if err != nil {
		a.Logf("[ERROR] Failed to parse BSDL: %v", err)
		return
	}
	
	device.BSDLFile = bsdlFile
	if bsdlFile.Entity != nil {
		device.Name = bsdlFile.Entity.Name
	}
	
	a.Logf("[INFO] Assigned BSDL to device %d: %s", deviceIdx, bsdlPath)
	a.invalidate()
}

func (a *App) showComponentSelector(deviceIdx int) {
	if a.loadedBoard == nil {
		a.Logf("[ERROR] No board loaded")
		return
	}
	
	a.componentSelectorVisible = true
	a.componentSelectorDevice = deviceIdx
	
	// Initialize component buttons if needed
	if a.componentButtons == nil {
		a.componentButtons = make(map[string]*widget.Clickable)
	}
	
	// Create buttons for all candidate components
	for _, fp := range a.loadedBoard.Footprints {
		if a.isJTAGCandidate(&fp) {
			if _, exists := a.componentButtons[fp.Reference]; !exists {
				a.componentButtons[fp.Reference] = &widget.Clickable{}
			}
		}
	}
	
	a.Logf("[DEBUG] Show component selector for device %d (board has %d footprints)", deviceIdx, len(a.loadedBoard.Footprints))
	a.invalidate()
}

func (a *App) isJTAGCandidate(fp *kicadparser.Footprint) bool {
	// Check if reference suggests it's an IC (U, IC, or similar)
	ref := fp.Reference
	if len(ref) == 0 {
		return false
	}
	
	firstChar := ref[0]
	if firstChar != 'U' && firstChar != 'I' {
		return false
	}
	
	// Check minimum pad count (need at least 4 for JTAG: TDI, TDO, TMS, TCK)
	if len(fp.Pads) < 4 {
		return false
	}
	
	return true
}

func (a *App) assignComponentToDevice(deviceIdx int, componentRef string) {
	if deviceIdx >= len(a.chainDevices) {
		return
	}
	
	device := &a.chainDevices[deviceIdx]
	device.ComponentRef = componentRef
	
	a.Logf("[INFO] Assigned component %s to device %d", componentRef, deviceIdx)
	a.componentSelectorVisible = false
	a.invalidate()
}

func (a *App) layoutComponentSelectorDialog(gtx layout.Context) layout.Dimensions {
	// Semi-transparent background
	paint.Fill(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 128})
	
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Dialog box
		return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			// White background for dialog
			paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255},
				clip.Rect{Max: gtx.Constraints.Max}.Op())
			
			return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.H6(a.gvTheme.Theme, "Select Component").Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutComponentList(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						cancelBtn := &widget.Clickable{}
						if cancelBtn.Clicked(gtx) {
							a.componentSelectorVisible = false
							a.invalidate()
						}
						return material.Button(a.gvTheme.Theme, cancelBtn, "Cancel").Layout(gtx)
					}),
				)
			})
		})
	})
}

func (a *App) layoutComponentList(gtx layout.Context) layout.Dimensions {
	children := make([]layout.FlexChild, 0)
	
	for i := range a.loadedBoard.Footprints {
		fp := &a.loadedBoard.Footprints[i]
		if !a.isJTAGCandidate(fp) {
			continue
		}
		
		ref := fp.Reference
		btn := a.componentButtons[ref]
		
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if btn.Clicked(gtx) {
				a.assignComponentToDevice(a.componentSelectorDevice, ref)
			}
			
			label := fmt.Sprintf("%s (%d pads)", ref, len(fp.Pads))
			return material.Button(a.gvTheme.Theme, btn, label).Layout(gtx)
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout))
	}
	
	if len(children) == 0 {
		return material.Body2(a.gvTheme.Theme, "No JTAG candidates found").Layout(gtx)
	}
	
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) openBoardFilePicker() {
	go func() {
		file, err := a.boardExplorer.ChooseFile("kicad_pcb")
		if err != nil {
			if err != explorer.ErrUserDecline {
				a.Logf("[ERROR] File picker failed: %v", err)
			}
			return
		}
		defer file.Close()

		// Get file path - explorer returns ReadCloser, need to get path differently
		// For now, read the file content and parse it
		if f, ok := file.(*os.File); ok {
			a.loadBoardFile(f.Name())
		} else {
			a.Logf("[ERROR] Unable to get file path from picker")
		}
	}()
}

func (a *App) loadBoardFile(filepath string) {
	board, err := kicadparser.ParseFile(filepath)
	if err != nil {
		a.Logf("[ERROR] Failed to load board: %v", err)
		return
	}

	a.boardFilePath = filepath
	a.loadedBoard = board
	a.showBoardViewer = true
	a.boardCamera = nil // Reset camera

	a.Logf("[INFO] Loaded KiCad board: %s", filepath)
	a.Logf("[INFO] Board: %d footprints, %d tracks, %d vias", 
		len(board.Footprints), len(board.Tracks), len(board.Vias))
	
	a.invalidate()
}

func (a *App) layoutSettings(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.Body1(a.gvTheme.Theme, "Tune the experimental UI from this panel.").Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutSettingsCard(gtx, "Appearance", func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutSettingsSwitch(gtx, "Dark mode", "Switch between light and dark palettes.", &a.darkModeSwitch, a.setDarkMode)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutSettingsSwitch(gtx, "Debug overlay", "Overlay layout bands for measurement.", &a.debugModeSwitch, a.setDebugOverlay)
					}),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutSettingsCard(gtx, "Board Color Theme", a.layoutBoardThemeSelector)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutSettingsCard(gtx, "Drivers", a.layoutDriverPicker)
		}),
	)
}

func (a *App) layoutBoardThemeSelector(gtx layout.Context) layout.Dimensions {
	themes := []kicadrenderer.ColorTheme{
		kicadrenderer.ThemeClassic,
		kicadrenderer.ThemeKiCad2020,
		kicadrenderer.ThemeBlueTone,
		kicadrenderer.ThemeEagle,
		kicadrenderer.ThemeNord,
	}
	
	children := make([]layout.FlexChild, 0)
	for i, theme := range themes {
		themeIdx := i
		themeName := kicadrenderer.ThemeNames[theme]
		
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if a.themeButtons[themeIdx].Clicked(gtx) {
				a.boardColorTheme = theme
				kicadrenderer.SetTheme(theme)
				
				// Save config
				config := &AppConfig{
					BoardColorTheme: int(theme),
				}
				if err := SaveConfig(config); err != nil {
					a.Logf("[ERROR] Failed to save config: %v", err)
				}
				
				a.invalidate()
			}
			
			btn := material.Button(a.gvTheme.Theme, &a.themeButtons[themeIdx], themeName)
			if a.boardColorTheme == theme {
				btn.Background = color.NRGBA{R: 63, G: 81, B: 181, A: 255} // Highlight selected
			}
			return btn.Layout(gtx)
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
	}
	
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) layoutSettingsCard(gtx layout.Context, title string, content layout.Widget) layout.Dimensions {
	return layout.Inset{Bottom: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		card := clip.RRect{
			Rect: image.Rectangle{Max: gtx.Constraints.Max},
			NW:   gtx.Dp(unit.Dp(10)),
			NE:   gtx.Dp(unit.Dp(10)),
			SW:   gtx.Dp(unit.Dp(10)),
			SE:   gtx.Dp(unit.Dp(10)),
		}
		cardColor := color.NRGBA{R: 242, G: 244, B: 251, A: 255}
		if a.darkMode {
			cardColor = color.NRGBA{R: 36, G: 40, B: 52, A: 255}
		}
		paint.FillShape(gtx.Ops, cardColor, card.Op(gtx.Ops))
		return layout.Inset{Top: unit.Dp(18), Bottom: unit.Dp(18), Left: unit.Dp(18), Right: unit.Dp(18)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(material.Body2(a.gvTheme.Theme, title).Layout),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(content),
			)
		})
	})
}

func (a *App) layoutSettingsSwitch(gtx layout.Context, title, subtitle string, control *widget.Bool, onChange func(bool)) layout.Dimensions {
	prev := control.Value
	return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(material.Body2(a.gvTheme.Theme, title).Layout),
					layout.Rigid(material.Caption(a.gvTheme.Theme, subtitle).Layout),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				sw := material.Switch(a.gvTheme.Theme, control, title)
				d := sw.Layout(gtx)
				if prev != control.Value {
					onChange(control.Value)
				}
				return d
			}),
		)
	})
}

func (a *App) layoutDriverPicker(gtx layout.Context) layout.Dimensions {
	current := "Select driver"
	if a.selectedDriver >= 0 && a.selectedDriver < len(a.driverOptions) {
		current = a.driverOptions[a.selectedDriver]
	}
	description := "Pick which driver implementation powers the experimental UI."
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(material.Body2(a.gvTheme.Theme, current).Layout),
				layout.Rigid(material.Caption(a.gvTheme.Theme, description).Layout),
			)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btnLabel := "Change"
			if a.driverMenu == nil {
				btnLabel = "Unavailable"
			}
			if a.driverMenu != nil && a.driverMenuBtn.Clicked(gtx) {
				a.driverMenu.ToggleVisibility(gtx)
			}
			dims := material.Button(a.gvTheme.Theme, &a.driverMenuBtn, btnLabel).Layout(gtx)
			if a.driverMenu != nil {
				a.driverMenu.Layout(gtx, a.gvTheme)
			}
			return dims
		}),
	)
}

func (a *App) layoutFootprints(gtx layout.Context) layout.Dimensions {
	if a.footprintView == nil {
		a.footprintView = footprint.NewAppWithWindow(nil)
		a.footprintView.SetInvalidateCallback(func() { a.window.Invalidate() })
	}
	if a.gvTheme != nil {
		a.footprintView.SetTheme(a.gvTheme.Theme)
	}
	return layout.Inset{Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return a.footprintView.LayoutEmbedded(gtx)
	})
}

func (a *App) layoutNetlist(gtx layout.Context) layout.Dimensions {
	if a.discoveredNetlist == nil || a.discoveredNetlist.NetCount() == 0 {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.Body1(a.gvTheme.Theme, "No netlist available.").Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(material.Body2(a.gvTheme.Theme, "Run reverse engineering to discover nets.").Layout),
		)
	}
	return a.netEditor.LayoutWorkspace(gtx, a.gvTheme.Theme, a)
}







func (a *App) initializeChain() {
	// Initialize with simulated chain (default to simple scenario)
	sim, err := jtag.BuildSimple2DeviceScenario("testdata")
	if err != nil {
		a.Logf("[ERROR] Failed to create simulator: %v", err)
		return
	}
	
	a.chainSimulator = sim
	
	// Create device entries from simulator
	a.chainDevices = make([]ChainDevice, sim.GetDeviceCount())
	for i := 0; i < sim.GetDeviceCount(); i++ {
		dev, err := sim.GetDevice(i)
		if err != nil {
			continue
		}
		
		a.chainDevices[i] = ChainDevice{
			Index:      i,
			IDCode:     dev.IDCode,
			IDCodeInfo: jtag.DecodeIDCode(dev.IDCode),
			IRLength:   dev.IRLength,
			BSRLength:  dev.Info.BoundaryLength,
			State:      "discovered",
		}
	}
	
	// Initialize device card list
	a.deviceCardList.Axis = layout.Vertical
	
	a.Logf("[INFO] Initialized chain with %d devices", len(a.chainDevices))
}

func (a *App) openBSDLPicker(deviceIndex int) {
	go func() {
		file, err := a.boardExplorer.ChooseFile("bsd", "bsdl")
		if err != nil {
			if err != explorer.ErrUserDecline {
				a.Logf("[ERROR] BSDL picker failed: %v", err)
			}
			return
		}
		defer file.Close()

		// Get file path
		if f, ok := file.(*os.File); ok {
			a.assignBSDL(deviceIndex, f.Name())
		} else {
			a.Logf("[ERROR] Unable to get file path from picker")
		}
	}()
}

func (a *App) assignBSDL(deviceIndex int, bsdlPath string) {
	if deviceIndex < 0 || deviceIndex >= len(a.chainDevices) {
		return
	}

	// Parse BSDL file
	parser, err := bsdl.NewParser()
	if err != nil {
		a.Logf("[ERROR] Failed to create BSDL parser: %v", err)
		return
	}

	bsdlFile, err := parser.ParseFile(bsdlPath)
	if err != nil {
		a.Logf("[ERROR] Failed to parse BSDL file: %v", err)
		return
	}

	// Extract device info
	info := bsdlFile.Entity.GetDeviceInfo()
	
	// Extract pin mapping
	pinMapping := bsdl.ExtractPinMapping(bsdlFile)
	
	// Update device
	device := &a.chainDevices[deviceIndex]
	device.BSDLPath = bsdlPath
	device.BSDLFile = bsdlFile
	device.PinMapping = pinMapping
	device.Name = bsdlFile.Entity.Name
	device.BSRLength = info.BoundaryLength
	if info.InstructionLength > 0 {
		device.IRLength = info.InstructionLength
	}
	
	// Add to chain repository
	if a.chainRepository != nil {
		a.chainRepository.Add(device.IDCode, bsdlFile)
	}
	
	// Update state
	if device.FootprintType != "" {
		device.State = "ready"
	} else {
		device.State = "bsdl_assigned"
	}
	
	a.Logf("[INFO] Assigned BSDL to device %d: %s (%d pins mapped)", 
		deviceIndex, device.Name, len(pinMapping.BSRIndexToPin))
	a.window.Invalidate()
}

func (a *App) openFootprintPicker(deviceIndex int) {
	// For now, cycle through footprint types with default parameters
	// TODO: Replace with proper picker dialog
	device := &a.chainDevices[deviceIndex]
	
	footprints := []string{"TSOP-I", "TSOP-II", "QFP", "QFN", "BGA"}
	currentIdx := -1
	for i, fp := range footprints {
		if fp == device.FootprintType {
			currentIdx = i
			break
		}
	}
	
	nextIdx := (currentIdx + 1) % len(footprints)
	device.FootprintType = footprints[nextIdx]
	
	// Set default parameters based on package type
	switch device.FootprintType {
	case "TSOP-I":
		device.PinCount = 48
		device.PackageWidth = 12.0  // 12mm narrow body
		device.PackageHeight = 18.4 // 18.4mm length
	case "TSOP-II":
		device.PinCount = 54
		device.PackageWidth = 22.2  // 22.2mm wide body
		device.PackageHeight = 10.2 // 10.2mm length
	case "QFP":
		device.PinCount = 100
		device.PackageWidth = 14.0  // 14mm x 14mm
		device.PackageHeight = 14.0
	case "QFN":
		device.PinCount = 48
		device.PackageWidth = 7.0 // 7mm x 7mm
		device.PackageHeight = 7.0
	case "BGA":
		device.PinCount = 144
		device.BallPitch = 0.8 // 0.8mm pitch
		device.PackageWidth = 10.0
		device.PackageHeight = 10.0
	}
	
	// Update state
	if device.BSDLPath != "" {
		device.State = "ready"
	} else {
		device.State = "footprint_assigned"
	}
	
	a.Logf("[INFO] Assigned footprint to device %d: %s (%d pins)", 
		deviceIndex, device.FootprintType, device.PinCount)
	a.debugLogOnce = true // Enable one-time logging for next render
	a.window.Invalidate()
}

func (a *App) applyPalette() {
	if a.gvTheme == nil {
		return
	}
	if a.darkMode {
		a.gvTheme.WithPalette(theme.Palette{
			Bg:         color.NRGBA{R: 18, G: 20, B: 26, A: 255},
			Fg:         color.NRGBA{R: 233, G: 236, B: 245, A: 255},
			ContrastBg: color.NRGBA{R: 120, G: 150, B: 255, A: 255},
			ContrastFg: color.NRGBA{R: 12, G: 16, B: 24, A: 255},
			Bg2:        color.NRGBA{R: 34, G: 40, B: 50, A: 255},
		})
	} else {
		a.gvTheme.WithPalette(theme.Palette{
			Bg:         color.NRGBA{R: 245, G: 247, B: 253, A: 255},
			Fg:         color.NRGBA{R: 34, G: 37, B: 49, A: 255},
			ContrastBg: color.NRGBA{R: 80, G: 120, B: 255, A: 255},
			ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
			Bg2:        color.NRGBA{R: 225, G: 230, B: 244, A: 255},
		})
	}
}

func (a *App) ensureLogPaneHeight(gtx layout.Context) {
	if a.logPaneHeight > 0 {
		return
	}
	a.logPaneHeight = float32(gtx.Dp(unit.Dp(160)))
	a.clampLogPaneHeight(gtx)
}

func (a *App) clampLogPaneHeight(gtx layout.Context) {
	min := float32(gtx.Dp(unit.Dp(80)))
	max := float32(gtx.Dp(unit.Dp(360)))
	if a.logPaneHeight < min {
		a.logPaneHeight = min
	}
	if a.logPaneHeight > max {
		a.logPaneHeight = max
	}
}

func (a *App) invalidate() {
	if a.window != nil {
		a.window.Invalidate()
	}
}

func (a *App) Logf(format string, args ...any) {
	prefix := time.Now().Format(time.Stamp)
	entry := fmt.Sprintf("[%s] %s", prefix, fmt.Sprintf(format, args...))
	a.logs = append(a.logs, entry)
	a.logText = strings.Join(a.logs, "\n")
	a.logSelectable.SetText(a.logText)
	a.invalidate()
}

func (a *App) opaqueFg() color.NRGBA {
	fg := a.gvTheme.Palette.Fg
	fg.A = 0xFF
	return fg
}

func (a *App) selectionColor() color.NRGBA {
	bg := a.gvTheme.Palette.ContrastBg
	if bg.A == 0 {
		bg.A = 0xFF
	}
	return color.NRGBA{R: bg.R, G: bg.G, B: bg.B, A: 0x88}
}

func (a *App) layoutBulletList(gtx layout.Context, items []string) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		func() []layout.FlexChild {
			children := make([]layout.FlexChild, 0, len(items))
			for _, item := range items {
				txt := item
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					content := fmt.Sprintf("• %s", txt)
					return material.Caption(a.gvTheme.Theme, content).Layout(gtx)
				}))
			}
			return children
		}()...,
	)
}

func (a *App) buildDriverMenu() *menu.DropdownMenu {
	if len(a.driverOptions) == 0 {
		return nil
	}
	opts := make([]menu.MenuOption, 0, len(a.driverOptions))
	for i, name := range a.driverOptions {
		idx := i
		label := name
		opts = append(opts, menu.MenuOption{
			OnClicked: func() error {
				a.setDriverSelection(idx)
				return nil
			},
			Layout: func(gtx menu.C, th *theme.Theme) menu.D {
				lbl := material.Body1(th.Theme, label)
				if idx == a.selectedDriver {
					lbl.Color = th.Palette.ContrastBg
				}
				return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, lbl.Layout)
			},
		})
	}
	drop := menu.NewDropdownMenu([][]menu.MenuOption{opts})
	drop.MaxWidth = unit.Dp(220)
	return drop
}

func (a *App) logSizesOnce() {
	if a.loggedSizes {
		return
	}
	if a.windowWidth == 0 || a.centerWidth == 0 || a.logWidth == 0 {
		return
	}
	a.Logf("[DEBUG] layout widths -> window=%dpx center=%dpx log=%dpx", a.windowWidth, a.centerWidth, a.logWidth)
	a.loggedSizes = true
}

func (a *App) togglePanelVisibility(side, source string) {
	switch side {
	case "left":
		a.leftPanelVisible = !a.leftPanelVisible
		if a.leftPanelVisible {
			a.Logf("[INFO] Left panel shown (%s)", source)
		} else {
			a.Logf("[INFO] Left panel hidden (%s)", source)
		}
	case "right":
		a.rightPanelVisible = !a.rightPanelVisible
		if a.rightPanelVisible {
			a.Logf("[INFO] Right panel shown (%s)", source)
		} else {
			a.Logf("[INFO] Right panel hidden (%s)", source)
		}
	}
	a.invalidate()
}

func (a *App) setDarkMode(enabled bool) {
	if a.darkMode == enabled {
		return
	}
	a.darkMode = enabled
	a.darkModeSwitch.Value = enabled
	a.applyPalette()
	if enabled {
		a.Logf("[INFO] Theme switched to dark mode (settings)")
	} else {
		a.Logf("[INFO] Theme switched to light mode (settings)")
	}
	a.invalidate()
}

func (a *App) setDebugOverlay(enabled bool) {
	if a.debugOverlay == enabled {
		return
	}
	a.debugOverlay = enabled
	a.debugModeSwitch.Value = enabled
	if enabled {
		a.Logf("[INFO] Debug overlay enabled (settings)")
	} else {
		a.Logf("[INFO] Debug overlay disabled (settings)")
	}
	a.invalidate()
}

func (a *App) setDriverSelection(idx int) {
	if idx < 0 || idx >= len(a.driverOptions) {
		return
	}
	if a.selectedDriver == idx {
		return
	}
	a.selectedDriver = idx
	a.Logf("[INFO] Driver switched to %s", a.driverOptions[idx])
	a.invalidate()
}

func (a *App) withDebugOverlay(gtx layout.Context, content layout.Widget) layout.Dimensions {
	if !a.debugOverlay {
		return content(gtx)
	}
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(content),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			a.drawDebugBands(gtx)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
	)
}

func (a *App) drawDebugBands(gtx layout.Context) {
	if !a.debugOverlay {
		return
	}
	width := gtx.Constraints.Max.X
	height := gtx.Constraints.Max.Y
	if width <= 0 || height <= 0 {
		return
	}
	segments := []color.NRGBA{
		{R: 255, A: 32},
		{G: 255, A: 32},
		{B: 255, A: 32},
		{R: 255, G: 255, A: 32},
	}
	segmentWidth := width / len(segments)
	for i, col := range segments {
		minX := i * segmentWidth
		maxX := minX + segmentWidth
		if i == len(segments)-1 {
			maxX = width
		}
		rect := clip.Rect{Min: image.Pt(minX, 0), Max: image.Pt(maxX, height)}
		paint.FillShape(gtx.Ops, col, rect.Op())
	}
}

func filterMonoFaces() []gfont.FontFace {
	var mono []gfont.FontFace
	for _, face := range gofont.Collection() {
		if face.Font.Typeface == gfont.Typeface("Go Mono") {
			mono = append(mono, face)
		}
	}
	return mono
}
