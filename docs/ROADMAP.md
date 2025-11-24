# OpenTraceJTAG Roadmap

**Last Updated**: 2025-11-24
**Current Version**: 0.9.0
**Target Version**: 1.0.0

## Status Overview

- **Core Functionality**: ✅ Complete (95%)
- **Hardware Support**: 🔄 Partial (50%)
- **UI/Visualization**: 🔄 In Progress (70%)
- **Documentation**: ✅ Complete (90%)

## ✅ Completed Features

### JTAG Stack
- BSDL parser with full IEEE 1149.1 support (all cell types)
- TAP state machine with optimal path planning
- JTAG chain discovery and device enumeration
- Boundary scan runtime (BSR) with pin control
- Hardware adapter abstraction layer
- CMSIS-DAP adapter support (Raspberry Pi Pico, DAPLink)
- Simulator adapter for testing
- BSDL repository with wildcard IDCODE matching
- Batch operations with IR/DR optimization

### Reverse Engineering
- Connectivity detection algorithm (drive-one-watch-all)
- Netlist builder with union-find algorithm
- JSON and KiCad netlist export

### KiCad Integration
- KiCad 6.0+ PCB parser (.kicad_pcb files)
- S-expression parser with 100k element support
- Board renderer with hardware-accelerated 2D graphics (Gio)
- Interactive viewer (pan, zoom, rotate, flip)
- Net highlighting with dimming
- Layer visibility controls with per-layer rendering
- 5 color themes (Classic, KiCad 2020, Blue Tone, Eagle, Nord)
- Multi-layer zone support (ground planes)
- Arc rendering for board outlines
- PCB substrate rendering

### UI Features
- Gio-based interactive UI
- Multi-workspace architecture
- Debug Board workspace with:
  - Board file picker
  - Camera controls (mouse + keyboard)
  - Layer visibility panel with bulk controls
  - BSDL and component mapping UI
  - Scan chain device list
- Configuration persistence (platform-specific paths)
- Theme selection and persistence

### CLI Tools
- `gio-viewer` - Interactive PCB board viewer
- `net-info` - Net connectivity query tool
- `bsdl-parser` - BSDL file parser and analyzer
- `jtag` - JTAG chain control and boundary scan operations

## 🔄 In Progress

### Hardware Support
- Real hardware testing and validation
- USB transport optimization

### UI Polish
- Pin number rendering on pads
- Pin state visualization with color coding

## 📋 Planned Features

### Phase 1: Pin Visualization (High Priority)
- Render pin numbers on all package types
- Font scaling with zoom level
- BGA grid coordinates (A1, B2, etc.)
- BSDL pin name integration
- Hover tooltips with pin information
- State-based color coding (Hi/Lo/Hi-Z)
- Interactive pin selection and control
- Pin search and filtering

**Estimated Time**: 4-6 weeks

### Phase 2: Reverse Engineering UI (High Priority)
- Discovery workspace with real-time visualization
- Progress indicators and statistics
- Connection graph rendering
- Netlist comparison with PCB design
- Export to various formats

**Estimated Time**: 6-8 weeks

### Phase 3: Advanced Features (Medium Priority)
- SVF/XSVF playback support
- Automated test pattern generation
- Pin timing analysis
- Multi-board support
- Scripting interface (Lua/Python)

**Estimated Time**: 8-12 weeks

### Phase 4: Hardware Expansion (Low Priority)
- Bus Pirate adapter
- FTDI adapter support
- Custom USB adapter designs
- Network-based adapters

**Estimated Time**: 4-6 weeks

## Version 1.0 Goals

- ✅ Stable JTAG chain control
- ✅ Complete BSDL parsing
- ✅ KiCad PCB visualization
- 🔄 Pin state visualization
- 🔄 Real hardware validation
- ✅ Comprehensive documentation
- 📋 Reverse engineering UI

## Future Considerations

- JTAG programming support (flash, CPLD, FPGA)
- IEEE 1149.6 (AC-coupled) support
- IEEE 1149.7 (compact JTAG) support
- Integration with OpenOCD
- Web-based UI option
- Cloud-based netlist database
