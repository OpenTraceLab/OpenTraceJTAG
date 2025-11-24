# Test Data

This directory contains test data for parser and renderer testing.

## Directory Structure

### `boards/`
Real KiCad PCB files (.kicad_pcb) for integration testing:

- **minimal.kicad_pcb** - Minimal board with basic elements only (few components, simple routing)
- **simple.kicad_pcb** - Simple 2-layer board with through-hole components
- **complex.kicad_pcb** - Complex 4-layer board with SMD components, zones, and complex routing
- **large.kicad_pcb** - Large board with 500+ components for performance testing

### `golden/`
Reference renderings for regression testing:

- PNG images of expected rendering output
- Named as `{board}_{layer}.png` (e.g., `minimal_top.png`, `complex_bottom.png`)
- Used to detect visual regressions

### `sexpr/`
Sample s-expression snippets for unit testing parser functions:

- `position.txt` - (at X Y) and (at X Y angle) examples
- `footprint.txt` - Complete footprint s-expressions
- `track.txt` - Track and arc segment examples
- `via.txt` - Via examples
- `zone.txt` - Zone examples
- `graphics.txt` - Graphic element examples (lines, arcs, circles, polygons, text)

## Adding Test Boards

To add a new test board:

1. Create or obtain a .kicad_pcb file
2. Place in `boards/` directory
3. Document what features it tests
4. (Optional) Generate golden images by rendering in KiCad and exporting
5. Add integration test in `parser_test.go`

## Sample S-Expressions

Sample s-expression files should contain isolated examples for unit testing.
Extract these from real .kicad_pcb files or create minimal examples.

Example `testdata/sexpr/position.txt`:
```lisp
; Simple position
(at 100 50)

; Position with angle
(at 100 50 90)

; Position in footprint context
(fp_text reference "R1" (at 0 0 90) (layer "F.SilkS"))
```

## Test Coverage Goals

- **Parser**: Every s-expression node type has unit tests
- **Integration**: At least 4 complete boards (minimal, simple, complex, large)
- **Regression**: Golden images for all test boards
- **Performance**: Benchmark with large.kicad_pcb
