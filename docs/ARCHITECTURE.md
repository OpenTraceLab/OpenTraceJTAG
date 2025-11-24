# OpenTraceJTAG Architecture

OpenTraceJTAG combines KiCad PCB visualization with JTAG boundary scan capabilities in a unified Go toolkit.

## System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     User Interface Layer                     │
│  • Gio-based PCB Viewer  • JTAG CLI  • Debug Board UI       │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┴─────────────────────┐
        │                                           │
┌───────▼────────┐                        ┌────────▼────────┐
│  KiCad Stack   │                        │   JTAG Stack    │
└────────────────┘                        └─────────────────┘
```

## KiCad Stack

### Parser (`pkg/kicad/parser`)
- Parses KiCad 6.0+ `.kicad_pcb` files (S-expression format)
- Extracts: tracks, vias, pads, zones, graphics, footprints
- Provides net connectivity information and bounding box calculations

### Renderer (`pkg/kicad/renderer`)
- Hardware-accelerated 2D vector rendering using Gio
- Interactive camera with pan, zoom, rotate, flip
- Layer visibility control with 5 color themes
- Net highlighting with element dimming
- Configurable layer rendering

## JTAG Stack

### BSDL Parser (`pkg/bsdl`)
- Parses IEEE 1149.1 BSDL files into Go structs
- Extracts instruction sets, device info, TAP configuration
- Supports all boundary cell types (BC_1, BC_2, BC_4, BC_7)
- Wildcard IDCODE matching for device families

### TAP State Machine (`pkg/tap`)
- Complete IEEE 1149.1 TAP FSM implementation
- Optimal TMS sequence path planning
- State validation and transitions
- I/O-free pure state logic

### Hardware Abstraction (`pkg/jtag`)
```go
type Adapter interface {
    Info() (AdapterInfo, error)
    ShiftIR(tms, tdi []byte, bits int) (tdo []byte, err error)
    ShiftDR(tms, tdi []byte, bits int) (tdo []byte, err error)
    ResetTAP(hard bool) error
    SetSpeed(hz int) error
}
```
- Pluggable adapter interface
- CMSIS-DAP support (Raspberry Pi Pico, DAPLink)
- Built-in simulator for testing

### Chain Controller (`pkg/chain`)
- Automatic JTAG chain discovery via IDCODE
- Multi-device chain management
- Pin control via boundary scan
- Batch operations to minimize USB traffic
- BSDL repository with device matching

### Boundary Scan Runtime (`pkg/bsr`)
- Pin-centric API for boundary-scan operations
- EXTEST mode support
- High-impedance (HiZ) control
- Pin drive and capture operations
- Automatic DR layout management

### Reverse Engineering (`pkg/reveng`)
- Board-level connectivity discovery
- "Drive one pin, watch all" algorithm
- Netlist generation (JSON, KiCad formats)

## Data Flow

### KiCad Visualization
```
.kicad_pcb file → Parser → Board struct → Renderer → Screen
                                    ↓
                              Layer Config
                              Camera Transform
```

### JTAG Operations
```
BSDL file → Parser → Device struct → Chain Controller → Adapter → Hardware
                                           ↓
                                    BSR Runtime
                                    Pin Control
```

## Configuration

- User preferences stored in platform-specific locations:
  - Linux/macOS: `~/.config/opentracejtag/config.json`
  - Windows: `%APPDATA%\OpenTraceJTAG\config.json`
- Persists: color theme, layer visibility, window state

## Dependencies

- **gioui.org** - Portable immediate mode GUI with hardware acceleration
- **participle** - Parser library for BSDL
- **gousb** - USB library for JTAG adapters
- **cobra** - CLI framework
