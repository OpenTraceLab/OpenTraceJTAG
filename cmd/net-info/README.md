# net-info

Command-line utility to query net information from KiCad PCB files.

## Usage

```bash
# List all nets with connection counts
./net-info board.kicad_pcb

# Show detailed information for a specific net
./net-info board.kicad_pcb GND
./net-info board.kicad_pcb /SCL
```

## Examples

### List All Nets

```bash
$ ./net-info board.kicad_pcb
Board: 42 nets

Net Name                         Pads Tracks   Vias
─────────────────────────────────────────────────────────
                                    0      0      0
+3V3                               12     45      3
+5V                                 8     23      2
GND                                67    156     18
/SCL                                4     12      1
/SDA                                4     11      1
...
```

### Show Net Details

```bash
$ ./net-info board.kicad_pcb GND
Net: GND (number 1)

Pads (67):
  Pad 1   : circle 1.60×1.60 mm at (150.00, 100.00)
  Pad 2   : rect 1.20×1.20 mm at (152.00, 102.00)
  ...

Tracks (156):
  Track 1: 0.25 mm wide on F.Cu from (100.00, 50.00) to (120.00, 50.00)
  Track 2: 0.50 mm wide on B.Cu from (120.00, 50.00) to (120.00, 70.00)
  ...

Vias (18):
  Via 1: 0.80 mm diameter, 0.40 mm drill at (120.00, 70.00)
  Via 2: 0.80 mm diameter, 0.40 mm drill at (135.00, 85.00)
  ...
```

## Use Cases

- Verify net connectivity
- Find all components connected to a power rail
- Debug routing issues
- Generate net statistics
- Identify unconnected nets
