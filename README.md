# OpenTraceJTAG

A unified Go toolkit for PCB design analysis and JTAG boundary scan testing.

## Overview

OpenTraceJTAG combines two powerful capabilities:
1. **KiCad PCB Parser & Renderer** - Parse and visualize KiCad board files
2. **JTAG Boundary Scan System** - Parse BSDL files and control JTAG chains

## Features

### KiCad PCB Tools

#### Parser (`pkg/kicad/parser`)
- Parse KiCad 6.0+ board files (.kicad_pcb)
- S-expression format support
- Extract tracks, vias, pads, zones, graphics, footprints
- Net connectivity information
- Bounding box calculations

#### Renderer (`pkg/kicad/renderer`)
- Gio-based 2D vector rendering with hardware acceleration
- Interactive board viewer with pan, zoom, rotate, flip
- Net highlighting (dims other elements)
- Layer visibility control
- Camera transformations

#### Commands
- **gio-viewer** - Interactive PCB board viewer
- **net-info** - Query net connections and statistics

### JTAG Boundary Scan Tools

#### BSDL Parser (`pkg/bsdl`)
- Parse IEEE 1149.1 BSDL files into Go structs
- Extract instruction sets, device info, TAP configuration
- Parse boundary register definitions (all cell types)
- Support for wildcards in IDCODE
- Pin mapping extraction

#### TAP State Machine (`pkg/tap`)
- Complete IEEE 1149.1 TAP FSM implementation
- Path planning for optimal TMS sequences
- State validation and transitions

#### Hardware Abstraction (`pkg/jtag`)
- Adapter interface for any JTAG hardware
- Built-in simulator for testing
- CMSIS-DAP support (Raspberry Pi Pico, DAPLink, etc.)
- Pluggable transport layer

#### Chain Controller (`pkg/chain`)
- Automatic JTAG chain discovery via IDCODE
- Multi-device chain support
- Pin control via boundary scan
- Batch operations (minimize USB traffic)
- BSDL repository with wildcard matching

#### Boundary Scan Runtime (`pkg/bsr`)
- Pin-centric API for boundary-scan operations
- EXTEST mode support
- High-impedance (HiZ) control
- Pin drive and capture operations
- Automatic DR layout management

#### Reverse Engineering (`pkg/reveng`)
- Discover board-level connectivity via boundary-scan
- "Drive one pin, watch all" algorithm

#### Commands
- **bsdl-parser** - Parse and analyze BSDL files
- **jtag** - JTAG chain control and boundary scan operations

## Installation

```bash
# Clone the repository
git clone https://github.com/OpenTraceLab/OpenTraceJTAG.git
cd OpenTraceJTAG

# Build all tools
make build

# Or build specific tools
make build-kicad  # KiCad tools only
make build-jtag   # JTAG tools only
```

## Usage

### KiCad Board Viewer

```bash
# Run the interactive viewer
./bin/gio-viewer path/to/board.kicad_pcb

# On Wayland, if keyboard doesn't work:
GIO_BACKEND=x11 ./bin/gio-viewer path/to/board.kicad_pcb
```

**Viewer Controls:**
- Left Click: Rotate board 90° clockwise
- Right Click: Flip board (top/bottom view)
- Scroll Wheel: Zoom in/out
- F: Flip board
- R: Rotate 90° clockwise
- Left Arrow: Rotate 90° counter-clockwise
- Space: Fit board to view
- Q / Escape: Quit

### Net Information

```bash
# List all nets
./bin/net-info board.kicad_pcb

# Show details for specific net
./bin/net-info board.kicad_pcb GND
```

### BSDL Parser

```bash
# Parse a BSDL file
./bin/bsdl-parser parse testdata/STM32F405_LQFP100.bsd

# Show device information
./bin/bsdl-parser info testdata/STM32F405_LQFP100.bsd
```

### JTAG Operations

```bash
# Scan JTAG chain
./bin/jtag scan

# Run boundary scan test
./bin/jtag test --device 0

# Reverse engineer connections
./bin/jtag reveng --output connections.json
```

## Project Structure

```
OpenTraceJTAG/
├── pkg/
│   ├── kicad/          # KiCad parser and renderer
│   │   ├── parser/     # Board file parsing
│   │   └── renderer/   # Gio-based rendering
│   ├── bsdl/           # BSDL parser
│   ├── bsr/            # Boundary scan runtime
│   ├── chain/          # JTAG chain controller
│   ├── jtag/           # Hardware abstraction
│   ├── tap/            # TAP state machine
│   └── reveng/         # Reverse engineering
├── cmd/
│   ├── gio-viewer/     # KiCad board viewer
│   ├── net-info/       # Net query tool
│   ├── bsdl-parser/    # BSDL parser CLI
│   └── jtag/           # JTAG CLI
├── internal/           # Internal packages
├── assets/             # Fonts and resources
├── testdata/           # Test files (BSDL, KiCad boards)
└── docs/               # Documentation

```

## As a Library

### KiCad Parsing

```go
import (
    "github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/parser"
    "github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer"
)

// Parse board
board, _ := parser.ParseFile("board.kicad_pcb")

// Query nets
info := board.GetNetInfo("GND")
fmt.Printf("GND: %d pads, %d tracks\n", len(info.Pads), len(info.Tracks))

// Render in Gio app
camera := renderer.NewCamera(800, 600)
renderer.RenderBoard(gtx, camera, board)

// Highlight a net
renderer.RenderBoardWithHighlight(gtx, camera, board, "VCC")

// Control layers
config := renderer.NewLayerConfig()
config.SetVisible("B.Cu", false)
renderer.RenderBoardWithConfig(gtx, camera, board, config)
```

### JTAG Operations

```go
import (
    "github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
    "github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
)

// Parse BSDL
device, _ := bsdl.ParseFile("device.bsd")

// Create chain controller
controller := chain.NewController(adapter)
devices, _ := controller.Scan()

// Control pins
controller.SetPin(0, "PA0", true)
value := controller.ReadPin(0, "PA1")
```

## Documentation

- [User Guide](docs/USER_GUIDE.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Roadmap](docs/ROADMAP.md)

## Dependencies

- [gioui.org](https://gioui.org/) - Portable immediate mode GUI
- [participle](https://github.com/alecthomas/participle) - Parser library
- [gousb](https://github.com/google/gousb) - USB library
- [cobra](https://github.com/spf13/cobra) - CLI framework

## Development

```bash
# Run tests
make test

# Run with coverage
make coverage

# Format code
make fmt

# Run linter
make lint

# Clean build artifacts
make clean
```

## License

GPLv3 - See LICENSE file for details

## Contributing

Contributions welcome! Please open an issue or pull request.
