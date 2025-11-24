# Quick Start Guide

## Project Setup Complete! ✓

Your KiBrd project is now initialized with the following structure:

```
kibrd/
├── README.md                      # Project overview
├── IMPLEMENTATION_PLAN.md         # Detailed 6-week implementation plan
├── go.mod                         # Go module definition
├── parser/                        # KiCad file parsing
│   ├── types.go                   # Core data types (Position, Layer, etc.)
│   ├── board.go                   # Board structure definitions
│   └── parser.go                  # Parser implementation (stubs)
├── renderer/                      # Fyne rendering engine
│   └── renderer.go                # Renderer implementation (stubs)
└── examples/
    └── viewer/                    # Example board viewer app
        └── main.go
```

## Dependencies Installed

- ✓ github.com/chewxy/sexp - S-expression parser
- ✓ fyne.io/fyne/v2 - GUI toolkit and 2D graphics

## Current Status

**Phase**: Initial Setup Complete
**Next Step**: Phase 1, Milestone 1.1 - Basic S-Expression Parsing

## Getting Started with Development

### 1. Review the Implementation Plan

Read `IMPLEMENTATION_PLAN.md` to understand the full scope:
- 6 phases over 6-7 weeks
- 20+ milestones with clear deliverables
- Incremental, testable progress

### 2. Start with Phase 1: S-Expression Parser

The foundation of everything. Begin here:

```bash
cd parser
# Create sexp_utils.go with helper functions
```

**First tasks**:
1. Create `sexp_utils.go` with helper functions
2. Write function to read and parse .kicad_pcb files
3. Create utilities to navigate s-expression trees
4. Write tests with sample data

### 3. Get a Test File

You'll need sample .kicad_pcb files for testing:

**Option A**: Create one in KiCad
- Open KiCad 6.0+
- Create a simple PCB with a few components
- Save as `test_simple.kicad_pcb`

**Option B**: Download examples
- Find open source KiCad projects on GitHub
- Look for .kicad_pcb files

**Place test files in**: `examples/testdata/`

```bash
mkdir -p examples/testdata
# Copy your .kicad_pcb files here
```

### 4. Run the Example (After Implementation)

Once parsing is implemented:

```bash
cd examples/viewer
go run main.go -file ../../testdata/your_board.kicad_pcb
```

## Development Workflow

### Incremental Development

Each milestone in the plan is designed to:
1. **Build on previous work** - No big bang integration
2. **Be independently testable** - Verify as you go
3. **Provide visible progress** - See results quickly

### Testing Strategy

For each milestone:

```bash
# 1. Write the implementation
# 2. Write tests
go test ./parser/...

# 3. Test with real files
go run examples/viewer/main.go -file testdata/sample.kicad_pcb
```

### Recommended Order

**Week 1: Parser Foundation (Phase 1)**
- Days 1-2: S-expression utilities and helpers
- Days 3-4: Header, general, layers, nets parsing
- Day 5: Integration testing with real files

**Week 2-3: Element Parsing (Phase 2)**
- Week 2: Graphics, tracks, vias, pads
- Week 3: Footprints and zones

**Week 4-5: Rendering (Phases 3-4)**
- Week 4: Coordinate systems, basic shapes
- Week 5: Advanced shapes, optimization

**Week 5-6: Interactive Features (Phase 5)**
- Navigation, layer controls, selection

**Week 6-7: Polish (Phase 6)**
- Testing, documentation, cleanup

## Key Implementation Notes

### Parser Design

The parser is **two-stage**:
1. **S-expression parsing** (using chewxy/sexp)
   - Handles the syntax
   - Returns tree of symbols and lists

2. **Semantic parsing** (our code)
   - Interprets the tree
   - Builds typed Go structures
   - Validates data

### Renderer Design

The renderer is **layered**:
1. **Data layer**: Board structures from parser
2. **Transform layer**: Coordinate conversion, scaling
3. **Canvas layer**: Fyne canvas objects
4. **UI layer**: Interactive controls

### Coordinate Systems

**Important**: KiCad uses different coordinates than screen:
- **KiCad**: Millimeters, Y-down, origin varies
- **Screen**: Pixels, Y-down (Fyne), origin top-left
- **Transformations needed**: Translation, scaling, rotation

## Useful Resources

### KiCad Documentation
- File format: https://dev-docs.kicad.org/en/file-formats/sexpr-pcb/
- S-expression spec: https://dev-docs.kicad.org/en/file-formats/sexpr-intro/

### S-Expression Parser
- Library docs: https://pkg.go.dev/github.com/chewxy/sexp
- Example usage: See tests in the repo

### Fyne Documentation
- Getting started: https://developer.fyne.io/started/
- Canvas objects: https://developer.fyne.io/canvas/
- Custom widgets: https://developer.fyne.io/extend/custom-widget

## Quick Reference Commands

```bash
# Build everything
go build ./...

# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Run benchmarks
go test -bench=. ./...

# View coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Format code
go fmt ./...

# Lint (if you have golangci-lint installed)
golangci-lint run

# Run the viewer
go run examples/viewer/main.go -file path/to/board.kicad_pcb
```

## Tips for Success

1. **Start small**: Get a minimal parser working first
2. **Test frequently**: Don't write too much without testing
3. **Use real data**: Test with actual .kicad_pcb files early
4. **Iterate**: Don't try to make it perfect the first time
5. **Visualize**: Even crude rendering helps debug parsing
6. **Commit often**: Small, focused commits

## Common Pitfalls to Avoid

1. **Don't parse everything at once** - Follow the milestones
2. **Don't skip tests** - They'll save you time later
3. **Don't hardcode coordinates** - Use proper transformations
4. **Don't ignore layer ordering** - It matters for rendering
5. **Don't forget performance** - Profile early with large boards

## Next Immediate Steps

1. ✓ Review this quickstart
2. ✓ Read IMPLEMENTATION_PLAN.md Phase 1
3. → Create `parser/sexp_utils.go`
4. → Write basic s-expression parsing helpers
5. → Get a sample .kicad_pcb file for testing
6. → Start Milestone 1.1!

---

**Ready to start coding!** Begin with Phase 1, Milestone 1.1 in IMPLEMENTATION_PLAN.md

Good luck! 🚀
