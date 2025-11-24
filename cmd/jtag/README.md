# JTAG CLI Tool

A command-line interface for JTAG boundary scan operations including BSDL parsing and chain discovery.

## Installation

```bash
go build -o jtag ./cmd/jtag
```

## Commands

### discover - Discover JTAG Chain

Discover and identify devices in a JTAG chain by reading their IDCODEs and matching them to BSDL files.

```bash
# Discover with simulator (requires --sim-ids for testing)
jtag discover --count 2 --sim-ids 0x06438041,0x41111043

# With verbose output
jtag discover -v --count 1 --sim-ids 0x41113043

# Specify BSDL directory
jtag discover --count 2 --bsdl /path/to/bsdl --sim-ids 0x06438041,0x41111043
```

**Options:**
- `-a, --adapter` - Adapter type (simulator, pico, buspirate) [default: simulator]
- `-c, --count` - Number of devices in chain **[required]**
- `-b, --bsdl` - Directory containing BSDL files [default: testdata]
- `--sim-ids` - Simulator only: Comma-separated hex IDCODEs (e.g., 0x12345678,0x87654321)
- `--speed` - TCK frequency in Hz [default: 1000000]
- `-v, --verbose` - Verbose output

**Example Output:**
```
╔════════════════════════════════════════════════════════════════╗
║ JTAG Chain Discovery Results                                   ║
╠════════════════════════════════════════════════════════════════╣
║ Found 2 device(s)                                              ║
╚════════════════════════════════════════════════════════════════╝

┌─ Device 1 (Position 0) ─────────────────────────────────────┐
│ IDCODE: 0x06438041                                          │
│ Name:   STM32F303_F334_LQFP64                              │
│                                                              │
│ Device Information:                                          │
│   IR Length:       5 bits                                  │
│   Boundary Length: 139 bits                                  │
│                                                              │
│ Instructions (5 total):                                     │
│   BYPASS        11111 (0x1F)                              │
│   EXTEST        00000 (0x0)                              │
│   ...                                                     │
└──────────────────────────────────────────────────────────────┘

Chain Summary:
  Total IR Length:       13 bits
  Total Boundary Length: 548 bits
```

### parse - Parse BSDL File

Parse and display information from a single BSDL file.

```bash
# Basic parse
jtag parse testdata/STM32F303_F334_LQFP64.bsd

# Show all instructions
jtag parse --instructions testdata/STM32F303_F334_LQFP64.bsd

# Show boundary cells
jtag parse --boundary testdata/STM32F303_F334_LQFP64.bsd

# Show pin mappings
jtag parse --pins testdata/STM32F303_F334_LQFP64.bsd

# Verbose with all details
jtag parse -v --instructions --boundary --pins testdata/STM32F303_F334_LQFP64.bsd
```

**Options:**
- `-i, --instructions` - Show all instructions
- `-b, --boundary` - Show boundary scan cells
- `-p, --pins` - Show pin mappings
- `-v, --verbose` - Verbose output

**Example Output:**
```
╔════════════════════════════════════════════════════════════════╗
║ BSDL File Information                                          ║
╠════════════════════════════════════════════════════════════════╣
║ Entity: STM32F303_F334_LQFP64                                  ║
╚════════════════════════════════════════════════════════════════╝

Device Information:
  IR Length:       5 bits
  Boundary Length: 139 bits
  IDCODE:          0x06438041 (mask: 0x0FFFFFFF, has wildcards)

Instructions: 5 total
  BYPASS          11111 (0x1F)
  EXTEST          00000 (0x0)
  SAMPLE          00010 (0x2)
  ...

TAP Configuration:
  TDI (Scan In):    JTDI
  TDO (Scan Out):   JTDO
  Max Frequency:    10000000 Hz (10.00 MHz)
```

## Global Flags

- `-v, --verbose` - Enable verbose output
- `--version` - Show version information
- `-h, --help` - Show help for any command

## Simulator Mode

The simulator adapter is useful for testing and development without physical hardware. Configure it using `--sim-ids`:

```bash
# Single device (STM32F303)
jtag discover --count 1 --sim-ids 0x06438041

# Two devices (STM32 + Lattice FPGA)
jtag discover --count 2 --sim-ids 0x06438041,0x41111043

# Three devices
jtag discover --count 3 --sim-ids 0x06438041,0x41111043,0x028200CB
```

### Common IDCODEs

From the test suite:

| Device | IDCODE | Description |
|--------|--------|-------------|
| STM32F303 | 0x06438041 | ARM Cortex-M4, LQFP64 |
| STM32F405 | 0x06413041 | ARM Cortex-M4, LQFP100 |
| LFE5U-25F | 0x41111043 | Lattice ECP5 FPGA, CABGA381 |
| LFE5U-85F | 0x41113043 | Lattice ECP5 FPGA, CABGA756 |
| ADSP-21562 | 0x028200CB | Analog Devices DSP |

## Examples

### Discover a single STM32 device

```bash
jtag discover -v --count 1 --sim-ids 0x06438041
```

### Discover a mixed chain (MCU + FPGA)

```bash
jtag discover --count 2 --sim-ids 0x06438041,0x41111043
```

### Parse a BSDL file with full details

```bash
jtag parse -v --instructions --boundary --pins testdata/LFE5U_25F_CABGA381.bsm
```

### Discover with custom BSDL directory

```bash
jtag discover --count 1 --bsdl ~/bsdl-files --sim-ids 0x06438041
```

## Hardware Adapters

Currently supported adapters:

- **simulator** - In-memory simulator (for testing)
- **pico** - Raspberry Pi Pico (USB transport pending)
- **buspirate** - Bus Pirate (not implemented)

### Future: Using with Real Hardware

When physical adapters are implemented:

```bash
# Raspberry Pi Pico
jtag discover --adapter pico --count 2

# Bus Pirate
jtag discover --adapter buspirate --serial /dev/ttyUSB0 --count 3
```

## Exit Codes

- `0` - Success
- `1` - Error (see error message)

## Tips

1. **Use verbose mode** (`-v`) to see detailed adapter information and loading progress
2. **Test with simulator** before connecting to real hardware
3. **Verify BSDL directory** contains files matching your device IDCODEs
4. **Check IDCODE format** - must be hex with or without `0x` prefix

## See Also

- [Main README](../../README.md) - Project overview
- [Architecture](../../docs/architecture.md) - System design
- [API Documentation](../../docs/api/) - Package references
