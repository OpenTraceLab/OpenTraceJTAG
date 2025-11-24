# KiCad Board Viewer Integration

The JTAG newui now includes an integrated KiCad board viewer in the "Debug Board" workspace.

## Usage

1. **Launch JTAG UI:**
   ```bash
   ./bin/jtag
   # or: make run-jtag
   ```

2. **Navigate to Debug Board:**
   - Click "Debug Board" in the left navigation panel

3. **Load a KiCad Board:**
   - Click "Open KiCad Board..." button
   - Select a `.kicad_pcb` file from the file picker
   - Board will render in the main view

## Features

- Full PCB visualization with all layers
- Automatic fit-to-view on load
- Renders tracks, vias, pads, zones, graphics
- Integrated with JTAG boundary scan context

## File Picker

The file picker uses `zenity` on Linux. Make sure it's installed:

```bash
# Ubuntu/Debian
sudo apt install zenity

# Fedora
sudo dnf install zenity

# Arch
sudo pacman -S zenity
```

## Future Enhancements

Planned features:
- Click on pads to highlight connected nets
- Overlay JTAG chain device positions
- Show boundary scan pin mappings on board
- Interactive net tracing with JTAG data
- Export board connectivity for reverse engineering

## Alternative: Manual Path

If zenity is not available, you can modify the code to use a hardcoded path:

```go
// In openBoardFilePicker(), replace with:
filepath := "/path/to/your/board.kicad_pcb"
a.loadBoardFile(filepath)
```

Or set an environment variable:

```bash
export KICAD_BOARD="/path/to/board.kicad_pcb"
./bin/jtag
```
