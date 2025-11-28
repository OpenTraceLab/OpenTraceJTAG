# Schematic Viewer - Remaining Tasks

## Current Status
The schematic viewer is functional with basic rendering capabilities:
- ✅ Symbol shapes (rectangles, circles, arcs, polylines) with correct orientation
- ✅ Wires, buses, junctions
- ✅ Power symbols
- ✅ No-connect markers
- ✅ Visible outlines (red) around symbol bodies (yellow fill)
- ✅ Pan/zoom/theme support
- ✅ Text rendering for labels (local, global, hierarchical)
- ✅ Component reference and value text

## Priority Issues

### 1. Text Rendering (PARTIALLY COMPLETE)
**Status:** Basic text rendering implemented

**Completed:**
- ✅ Component references (U?, J?, R?, etc.)
- ✅ Component values
- ✅ Net labels (VBUS, USB_D+, GND, etc.)
- ✅ Global labels
- ✅ Hierarchical labels
- ✅ Text rotation and positioning
- ✅ Font size handling

**Still needed:**
- ⏸️ Pin numbers
- ⏸️ Pin names

**Files modified:**
- `pkg/kicad/schematic/renderer/labels.go` - ✅ Implemented renderLabelText with Gio widget.Label
- `pkg/kicad/schematic/renderer/symbols.go` - ✅ Implemented renderPropertyText for references and values

### 2. Positioning Refinements (MEDIUM PRIORITY)
**Issues:**
- No-connect X marks appear offset from wire endpoints
- Some junction circles may need position adjustment

**Root cause analysis needed:**
- Verify coordinate transformations
- Check if there's a systematic offset
- Test with multiple schematics

**Files to check:**
- `pkg/kicad/schematic/renderer/wires.go:111-141` - RenderNoConnects
- `pkg/kicad/schematic/renderer/wires.go:87-108` - RenderJunctions

### 3. Complex Symbol Graphics (MEDIUM PRIORITY)
**Issues:**
- Some internal symbol details render faintly
- Possible missing graphic types

**Investigation needed:**
- Verify all graphics from complex symbols (USB logo, detailed shapes) are parsed
- Check if any graphic primitives are not yet supported
- Ensure fills are rendering with correct opacity

**Files to review:**
- `pkg/kicad/schematic/parser.go` - Check if all graphic types are parsed
- `pkg/kicad/schematic/renderer/symbols.go` - Verify all graphic types have renderers

### 4. Missing Features (LOWER PRIORITY)

#### Pin Connection Indicators
- Small diamond/circle shapes where wires connect to pins
- May be rendered automatically by KiCad or may be explicit graphic elements

#### Multi-Sheet Support
- Hierarchical sheet rendering
- Sheet pins
- Sheet navigation

#### Images
- Embedded images in schematics (logos, diagrams)

#### Advanced Graphics
- Bezier curves
- Text boxes with backgrounds
- Filled arcs

## Testing Plan

### Test Cases Needed:
1. **Simple schematic** - Few components, basic connections (✓ partially tested with usb.kicad_sch)
2. **Complex schematic** - Many components, dense routing
3. **Hierarchical schematic** - Multiple sheets
4. **Symbol-heavy schematic** - Tests all symbol types
5. **Text-heavy schematic** - Tests label rendering

### Validation Checklist:
- [ ] All symbols render completely
- [ ] All text is readable
- [ ] Wire connections are accurate
- [ ] Junctions appear at correct locations
- [ ] No-connects align with wire endpoints
- [ ] Colors match KiCad theme
- [ ] Pan/zoom works smoothly
- [ ] File picker works on all platforms
- [ ] Command-line loading works

## Known Bugs

1. **No-connect positioning** - X marks offset from intended wire endpoints
   - Location: `pkg/kicad/schematic/renderer/wires.go:121-141`

2. **Pin numbers/names not rendered** - Pin text still needs implementation
   - Location: `pkg/kicad/schematic/renderer/symbols.go:389` (TODO comment in renderPin)

## Performance Notes

- Current implementation renders all elements every frame
- May need optimization for large schematics (1000+ components)
- Consider viewport culling for off-screen elements
- Symbol graphics could be cached

## Documentation Needed

- [ ] User guide for schematic viewer
- [ ] Keyboard shortcuts reference
- [ ] Architecture documentation
- [ ] Parser documentation for schematic format

## Future Enhancements

- Export to PNG/SVG
- Component search/highlight
- Net highlighting
- Cross-probing with PCB viewer
- BOM generation from schematic
- ERC (Electrical Rule Check) visualization
- Annotation/notes support
