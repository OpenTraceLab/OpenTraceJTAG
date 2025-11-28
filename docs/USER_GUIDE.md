# OpenTraceJTAG User Guide

## Installation

```bash
git clone https://github.com/OpenTraceLab/OpenTraceJTAG.git
cd OpenTraceJTAG
make build
```

Binaries will be in `./bin/`

## KiCad PCB Viewer

### Launch Viewer
```bash
./bin/gio-viewer path/to/board.kicad_pcb

# On Wayland if keyboard doesn't work:
GIO_BACKEND=x11 ./bin/gio-viewer path/to/board.kicad_pcb
```

### Viewer Controls

**Mouse:**
- Left Click: Rotate board 90° clockwise
- Right Click: Flip board (top/bottom view)
- Scroll Wheel: Zoom in/out

**Keyboard:**
- `F`: Flip board
- `R`: Rotate 90° clockwise
- `Left Arrow`: Rotate 90° counter-clockwise
- `Space`: Fit board to view
- `+`: Zoom in
- `-`: Zoom out
- `Q` / `Escape`: Quit

**Layer Panel:**
- Click checkboxes to toggle layer visibility
- "All" button: Show all layers
- "None" button: Hide all layers

**Color Themes:**
- Classic (default)
- KiCad 2020
- Blue Tone
- Eagle
- Nord

## Net Information Tool

```bash
# List all nets
./bin/net-info board.kicad_pcb

# Show details for specific net
./bin/net-info board.kicad_pcb GND
```

## BSDL Parser

```bash
# Parse a BSDL file
./bin/bsdl-parser parse device.bsd

# Show device information
./bin/bsdl-parser info device.bsd
```

## JTAG Operations

### Scan Chain
```bash
./bin/jtag scan
```
Discovers all devices in the JTAG chain and displays their IDCODEs.

### Boundary Scan Test
```bash
./bin/jtag test --device 0
```
Runs basic boundary scan operations on device 0.

### Reverse Engineer Connections
```bash
./bin/jtag reveng --output connections.json
```
Discovers board-level connectivity by driving pins and observing responses.

### Pin Control
```bash
# Set pin high
./bin/jtag set-pin --device 0 --pin PA0 --value 1

# Read pin
./bin/jtag read-pin --device 0 --pin PA1
```

## Hardware Setup

### Supported Adapters
- Raspberry Pi Pico (CMSIS-DAP firmware)
- DAPLink-compatible adapters
- Built-in simulator (for testing without hardware)

### JTAG Connections
```
Adapter    Target Board
------     ------------
TCK    →   TCK
TMS    →   TMS
TDI    →   TDI
TDO    ←   TDO
GND    →   GND
```

### BSDL Files
Place BSDL files in `testdata/` or specify path with `--bsdl` flag.

## Configuration

Settings are automatically saved to:
- Linux/macOS: `~/.config/opentracejtag/config.json`
- Windows: `%APPDATA%\OpenTraceJTAG\config.json`

Persisted settings:
- Color theme preference
- Layer visibility defaults
- Window size and position

## Troubleshooting

**Keyboard not working in viewer (Wayland):**
```bash
GIO_BACKEND=x11 ./bin/gio-viewer board.kicad_pcb
```

**USB permission errors (Linux):**
```bash
sudo usermod -a -G plugdev $USER
# Log out and back in
```

**JTAG chain not detected:**
- Check physical connections
- Verify target board is powered
- Try lower clock speed: `./bin/jtag scan --speed 100000`

**Large board files slow to load:**
- Parser supports up to 100,000 elements
- Zones with complex fills may take time to render
- Consider hiding unused layers

## Library Usage

### KiCad Parsing
```go
import "github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/pcb"

board, err := parser.ParseFile("board.kicad_pcb")
if err != nil {
    log.Fatal(err)
}

// Query nets
info := board.GetNetInfo("GND")
fmt.Printf("GND: %d pads, %d tracks\n", len(info.Pads), len(info.Tracks))
```

### JTAG Operations
```go
import (
    "github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
    "github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
)

// Parse BSDL
device, err := bsdl.ParseFile("device.bsd")

// Create chain controller
controller := chain.NewController(adapter)
devices, err := controller.Scan()

// Control pins
controller.SetPin(0, "PA0", true)
value := controller.ReadPin(0, "PA1")
```

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
