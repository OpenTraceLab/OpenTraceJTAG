package reverse

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsr"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/reveng"
)

type chainDeviceInfo struct {
	Position int
	Name     string
	IDCode   uint32
	IRLength int
	BRLength int
}

// App hosts the Gio reverse-engineering window/workspace.
type App struct {
	window *app.Window
	theme  *material.Theme

	ops op.Ops

	adapterTypes []string
	adapterIndex int

	deviceCount widget.Editor
	bsdlPath    widget.Editor

	skipJTAG  widget.Bool
	skipPower widget.Bool
	pattern   widget.Editor

	discovered []chainDeviceInfo
	chainList  layout.List
	chainInfo  string

	progress   float32
	status     string
	currentPin string

	netStats string
	netLines []string
	netList  layout.List

	discoverBtn    widget.Clickable
	startScanBtn   widget.Clickable
	stopScanBtn    widget.Clickable
	exportJSONBtn  widget.Clickable
	exportKiCadBtn widget.Clickable

	chain       *chain.Chain
	bsrCtrl     *bsr.Controller
	repo        *chain.MemoryRepository
	adapter     jtag.Adapter
	cancel      context.CancelFunc
	netlist     *reveng.Netlist
	scanRunning bool
}

// NewApp builds a ready-to-run reverse engineering window.
func NewApp() *App {
	return NewAppWithWindow(new(app.Window))
}

// NewAppWithWindow wires the supplied Gio window to the reverse workspace.
func NewAppWithWindow(win *app.Window) *App {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	appInstance := &App{
		window:       win,
		theme:        th,
		adapterTypes: []string{"simulator", "cmsisdap", "pico", "buspirate"},
		chainList:    layout.List{Axis: layout.Vertical},
		netList:      layout.List{Axis: layout.Vertical},
		repo:         chain.NewMemoryRepository(),
		status:       "Ready",
	}
	appInstance.deviceCount.SingleLine = true
	appInstance.deviceCount.SetText("2")

	appInstance.bsdlPath.SingleLine = true
	appInstance.bsdlPath.SetText("testdata")

	appInstance.pattern.SingleLine = true

	appInstance.skipJTAG.Value = true
	appInstance.skipPower.Value = true

	if win != nil {
		win.Option(app.Title("JTAG Reverse Engineering"), app.Size(unit.Dp(1400), unit.Dp(900)))
	}

	return appInstance
}

// Run spins Gio's event loop until the window closes.
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
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(0.32, func(gtx layout.Context) layout.Dimensions {
			return a.layoutLeftPanel(gtx)
		}),
		layout.Flexed(0.68, func(gtx layout.Context) layout.Dimensions {
			return a.layoutRightPanel(gtx)
		}),
	)
}

func (a *App) layoutLeftPanel(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(16), Left: unit.Dp(16), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutCard(gtx, "Connection Settings", func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(material.Body2(a.theme, "Adapter Type").Layout),
						layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
						layout.Rigid(a.layoutAdapterSelect),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(material.Body2(a.theme, "Expected Devices").Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Editor(a.theme, &a.deviceCount, "").Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(material.Body2(a.theme, "BSDL Directory").Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Editor(a.theme, &a.bsdlPath, "").Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.button(gtx, &a.discoverBtn, "Discover Chain", false, func() {
								go a.discoverChain()
							})
						}),
					)
				})
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutCard(gtx, "Scan Settings", func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(material.CheckBox(a.theme, &a.skipJTAG, "Skip JTAG pins").Layout),
						layout.Rigid(material.CheckBox(a.theme, &a.skipPower, "Skip power/ground pins").Layout),
						layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
						layout.Rigid(material.Body2(a.theme, "Pin Pattern (regex)").Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Editor(a.theme, &a.pattern, "leave empty for all").Layout(gtx)
						}),
					)
				})
			}),
		)
	})
}

func (a *App) layoutRightPanel(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutCard(gtx, "Discovered Chain", func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(material.Body2(a.theme, a.chainInfo).Layout),
						layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.chainList.Layout(gtx, len(a.discovered), func(gtx layout.Context, idx int) layout.Dimensions {
								if idx >= len(a.discovered) {
									return layout.Dimensions{}
								}
								dev := a.discovered[idx]
								return layout.UniformInset(unit.Dp(3)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(material.Body1(a.theme, fmt.Sprintf("[%d] %s", dev.Position, dev.Name)).Layout),
										layout.Rigid(material.Caption(a.theme, fmt.Sprintf("IDCODE: 0x%08X", dev.IDCode)).Layout),
										layout.Rigid(material.Caption(a.theme, fmt.Sprintf("IR: %d bits, BR: %d bits", dev.IRLength, dev.BRLength)).Layout),
									)
								})
							})
						}),
					)
				})
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutCard(gtx, "Progress", func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							bar := material.ProgressBar(a.theme, a.progress)
							bar.Color = color.NRGBA{R: 110, G: 140, B: 255, A: 255}
							return bar.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
						layout.Rigid(material.Body2(a.theme, a.status).Layout),
						layout.Rigid(material.Caption(a.theme, a.currentPin).Layout),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.button(gtx, &a.startScanBtn, "Start Reverse Engineering", a.bsrCtrl == nil || a.scanRunning, func() {
										go a.startScan()
									})
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.button(gtx, &a.stopScanBtn, "Stop", !a.scanRunning, func() {
										a.stopScan()
									})
								}),
							)
						}),
					)
				})
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return a.layoutCard(gtx, "Discovered Netlist", func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(material.Body2(a.theme, a.netStats).Layout),
						layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.button(gtx, &a.exportJSONBtn, "Export JSON", a.netlist == nil, func() {
										a.exportJSON()
									})
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.button(gtx, &a.exportKiCadBtn, "Export KiCad", a.netlist == nil, func() {
										a.exportKiCad()
									})
								}),
							)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.netList.Layout(gtx, len(a.netLines), func(gtx layout.Context, idx int) layout.Dimensions {
								if idx >= len(a.netLines) {
									return layout.Dimensions{}
								}
								return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return material.Caption(a.theme, a.netLines[idx]).Layout(gtx)
								})
							})
						}),
					)
				})
			}),
		)
	})
}

func (a *App) layoutCard(gtx layout.Context, title string, body layout.Widget) layout.Dimensions {
	return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.H6(a.theme, title).Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				thickness := gtx.Dp(unit.Dp(1))
				paint.FillShape(gtx.Ops, color.NRGBA{A: 40}, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, thickness)}.Op())
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, thickness)}
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(body),
		)
	})
}

func (a *App) layoutAdapterSelect(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.Body1(a.theme, a.adapterTypes[a.adapterIndex]).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(a.theme, new(widget.Clickable), "Cycle")
			dims := btn.Layout(gtx)
			for btn.Button.Clicked(gtx) {
				a.adapterIndex = (a.adapterIndex + 1) % len(a.adapterTypes)
				a.invalidate()
			}
			return dims
		}),
	)
}

func (a *App) button(gtx layout.Context, clk *widget.Clickable, label string, disabled bool, action func()) layout.Dimensions {
	btn := material.Button(a.theme, clk, label)
	if disabled {
		btn.Background = color.NRGBA{R: 70, G: 70, B: 90, A: 180}
	}
	dims := btn.Layout(gtx)
	if !disabled {
		for clk.Clicked(gtx) {
			action()
		}
	} else {
		clk.Clicked(gtx)
	}
	return dims
}

func (a *App) discoverChain() {
	a.status = "Discovering chain..."
	a.invalidate()

	path := strings.TrimSpace(a.bsdlPath.Text())
	if path == "" {
		path = "testdata"
	}
	a.repo = chain.NewMemoryRepository()
	if err := a.repo.LoadDir(path); err != nil {
		a.status = fmt.Sprintf("BSDL load failed: %v", err)
		a.invalidate()
		return
	}

	switch a.adapterTypes[a.adapterIndex] {
	case "simulator":
		a.adapter = jtag.NewSimAdapter(jtag.AdapterInfo{Name: "Simulator"})
	case "cmsisdap":
		adapter, err := jtag.NewCMSISDAPAdapter(0, 0)
		if err != nil {
			a.status = fmt.Sprintf("Adapter error: %v", err)
			a.invalidate()
			return
		}
		a.adapter = adapter
	default:
		a.status = fmt.Sprintf("Adapter %s not implemented", a.adapterTypes[a.adapterIndex])
		a.invalidate()
		return
	}

	count := 2
	fmt.Sscanf(strings.TrimSpace(a.deviceCount.Text()), "%d", &count)
	ctrl := chain.NewController(a.adapter, a.repo)
	ch, err := ctrl.Discover(count)
	if err != nil {
		a.status = fmt.Sprintf("Chain discovery failed: %v", err)
		a.invalidate()
		return
	}

	a.chain = ch
	devs := ch.Devices()
	a.discovered = make([]chainDeviceInfo, len(devs))
	for i, dev := range devs {
		a.discovered[i] = chainDeviceInfo{
			Position: i,
			Name:     dev.Name(),
			IDCode:   dev.IDCode,
			IRLength: dev.Info.InstructionLength,
			BRLength: dev.Info.BoundaryLength,
		}
	}
	a.chainInfo = fmt.Sprintf("Found %d device(s)", len(devs))

	ctrlBSR, err := bsr.NewController(ch)
	if err != nil {
		a.status = fmt.Sprintf("BSR controller failed: %v", err)
		a.invalidate()
		return
	}
	a.bsrCtrl = ctrlBSR
	a.status = fmt.Sprintf("Discovered %d device(s), %d pins", len(devs), len(ctrlBSR.AllPins()))
	a.invalidate()
}

func (a *App) startScan() {
	if a.bsrCtrl == nil {
		a.status = "Discover chain first"
		a.invalidate()
		return
	}
	if a.scanRunning {
		return
	}

	cfg := reveng.DefaultConfig()
	cfg.SkipKnownJTAGPins = a.skipJTAG.Value
	cfg.SkipPowerPins = a.skipPower.Value
	cfg.OnlyPinPattern = strings.TrimSpace(a.pattern.Text())

	if err := cfg.Validate(); err != nil {
		a.status = fmt.Sprintf("Invalid config: %v", err)
		a.invalidate()
		return
	}

	candidates := countCandidates(a.bsrCtrl, cfg)
	if candidates == 0 {
		a.status = "No pins to scan after filtering"
		a.invalidate()
		return
	}

	a.scanRunning = true
	a.status = fmt.Sprintf("Scanning %d pins...", candidates)
	a.progress = 0
	a.currentPin = ""
	a.netStats = ""
	a.netLines = nil
	a.netlist = nil
	a.invalidate()

	progressCh := make(chan reveng.Progress, 8)
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	go a.runScan(ctx, cfg, progressCh)
	go a.consumeProgress(progressCh)
}

func (a *App) stopScan() {
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *App) runScan(ctx context.Context, cfg *reveng.Config, progressCh chan<- reveng.Progress) {
	defer close(progressCh)

	start := time.Now()
	netlist, err := reveng.DiscoverNetlist(ctx, a.bsrCtrl, cfg, progressCh)
	if err != nil {
		if err == context.Canceled {
			a.status = "Scan cancelled"
		} else {
			a.status = fmt.Sprintf("Scan failed: %v", err)
		}
		a.scanRunning = false
		a.invalidate()
		return
	}

	a.netlist = netlist
	a.netStats = fmt.Sprintf("Nets: %d (multi-pin %d) — %s", netlist.NetCount(), netlist.MultiPinNetCount(), time.Since(start).Round(time.Second))
	a.netLines = nil
	for _, net := range netlist.Nets {
		if len(net.Pins) < 2 {
			continue
		}
		summary := fmt.Sprintf("Net %d (%d pins)", net.ID, len(net.Pins))
		a.netLines = append(a.netLines, summary)
	}
	a.scanRunning = false
	a.status = "Scan complete"
	a.invalidate()
}

func (a *App) consumeProgress(progressCh <-chan reveng.Progress) {
	for p := range progressCh {
		if p.Total <= 0 {
			continue
		}
		a.progress = float32(p.Index) / float32(p.Total)
		a.currentPin = fmt.Sprintf("Pin %d/%d: %s.%s — nets %d", p.Index, p.Total, p.Driver.DeviceName, p.Driver.PinName, p.NetsFound)
		a.invalidate()
	}
}

func (a *App) exportJSON() {
	if a.netlist == nil {
		return
	}
	path := filepath.Join(os.TempDir(), "netlist.json")
	data, err := a.netlist.ExportJSON()
	if err != nil {
		a.status = fmt.Sprintf("Export failed: %v", err)
		a.invalidate()
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		a.status = fmt.Sprintf("Write failed: %v", err)
		a.invalidate()
		return
	}
	a.status = fmt.Sprintf("Exported to %s", path)
	a.invalidate()
}

func (a *App) exportKiCad() {
	if a.netlist == nil {
		return
	}
	path := filepath.Join(os.TempDir(), "netlist.kicad")
	data, err := a.netlist.ExportKiCad()
	if err != nil {
		a.status = fmt.Sprintf("Export failed: %v", err)
		a.invalidate()
		return
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		a.status = fmt.Sprintf("Write failed: %v", err)
		a.invalidate()
		return
	}
	a.status = fmt.Sprintf("Exported to %s", path)
	a.invalidate()
}

func (a *App) invalidate() {
	if a.window != nil {
		a.window.Invalidate()
	}
}

// SetTheme allows embedding contexts to override the material theme.
func (a *App) SetTheme(th *material.Theme) {
	if th != nil {
		a.theme = th
	}
}

// LayoutEmbedded renders the reverse UI inside another Gio application.
func (a *App) LayoutEmbedded(gtx layout.Context) layout.Dimensions {
	if a.theme == nil {
		th := material.NewTheme()
		th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
		a.theme = th
	}
	return a.layout(gtx)
}

// countCandidates estimates how many pins will be scanned given the config.
func countCandidates(ctl *bsr.Controller, cfg *reveng.Config) int {
	if ctl == nil || cfg == nil {
		return 0
	}
	count := 0
	for _, dev := range ctl.Devices {
		if !cfg.ShouldScanDevice(dev.ChainDev.Name()) {
			continue
		}
		for pinName := range dev.Pins {
			if !cfg.ShouldScanPin(pinName) {
				continue
			}
			if cfg.SkipKnownJTAGPins && isJTAGPin(pinName) {
				continue
			}
			if cfg.SkipPowerPins && isPowerPin(pinName) {
				continue
			}
			count++
		}
	}
	return count
}

func isJTAGPin(pinName string) bool {
	upper := strings.ToUpper(pinName)
	return strings.Contains(upper, "TCK") ||
		strings.Contains(upper, "TMS") ||
		strings.Contains(upper, "TDI") ||
		strings.Contains(upper, "TDO") ||
		strings.Contains(upper, "TRST") ||
		strings.Contains(upper, "JTAG")
}

func isPowerPin(pinName string) bool {
	upper := strings.ToUpper(pinName)
	return strings.Contains(upper, "VCC") ||
		strings.Contains(upper, "VDD") ||
		strings.Contains(upper, "VSS") ||
		strings.Contains(upper, "GND") ||
		strings.Contains(upper, "VBAT") ||
		strings.Contains(upper, "VREF")
}
