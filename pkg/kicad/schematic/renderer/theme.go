package renderer

import "image/color"

// Theme represents a color scheme for schematic rendering
type Theme int

const (
	// ThemeLight is a light background theme (white/light gray background)
	ThemeLight Theme = iota
	// ThemeDark is a dark background theme (dark gray/black background)
	ThemeDark
)

// SchematicColors defines the color scheme for rendering schematic elements
type SchematicColors struct {
	// Background and grid
	Background color.NRGBA
	Grid       color.NRGBA

	// Wires and connections
	Wire      color.NRGBA
	Bus       color.NRGBA
	Junction  color.NRGBA
	NoConnect color.NRGBA

	// Labels
	LocalLabel  color.NRGBA
	GlobalLabel color.NRGBA
	HierLabel   color.NRGBA

	// Symbols
	SymbolBody    color.NRGBA
	SymbolFill    color.NRGBA
	SymbolPin     color.NRGBA
	SymbolPinText color.NRGBA
	SymbolText    color.NRGBA

	// Hierarchical sheets
	Sheet     color.NRGBA
	SheetFill color.NRGBA
	SheetPin  color.NRGBA
	SheetText color.NRGBA

	// Text and annotations
	Text color.NRGBA

	// Selection and highlight
	Selection color.NRGBA
	Highlight color.NRGBA
}

// GetSchematicColors returns the color scheme for the given theme
func GetSchematicColors(theme Theme) *SchematicColors {
	switch theme {
	case ThemeDark:
		return getDarkTheme()
	case ThemeLight:
		return getLightTheme()
	default:
		return getLightTheme()
	}
}

// getLightTheme returns KiCad-style light theme colors
func getLightTheme() *SchematicColors {
	return &SchematicColors{
		// Background and grid
		Background: color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // White
		Grid:       color.NRGBA{R: 220, G: 220, B: 220, A: 255}, // Light gray

		// Wires and connections (KiCad green)
		Wire:      color.NRGBA{R: 0, G: 132, B: 0, A: 255},    // Dark green
		Bus:       color.NRGBA{R: 0, G: 0, B: 132, A: 255},    // Dark blue
		Junction:  color.NRGBA{R: 0, G: 132, B: 0, A: 255},    // Dark green (filled circle)
		NoConnect: color.NRGBA{R: 0, G: 0, B: 132, A: 255},    // Dark blue

		// Labels (various colors for visibility)
		LocalLabel:  color.NRGBA{R: 0, G: 0, B: 0, A: 255},       // Black
		GlobalLabel: color.NRGBA{R: 132, G: 0, B: 0, A: 255},     // Dark red
		HierLabel:   color.NRGBA{R: 132, G: 66, B: 0, A: 255},    // Brown

		// Symbols (dark gray/black)
		SymbolBody:    color.NRGBA{R: 132, G: 0, B: 0, A: 255},      // Dark red
		SymbolFill:    color.NRGBA{R: 255, G: 255, B: 194, A: 128},  // Light yellow (translucent)
		SymbolPin:     color.NRGBA{R: 132, G: 0, B: 0, A: 255},      // Dark red
		SymbolPinText: color.NRGBA{R: 0, G: 100, B: 100, A: 255},    // Teal
		SymbolText:    color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // Black

		// Hierarchical sheets
		Sheet:     color.NRGBA{R: 132, G: 0, B: 132, A: 255},    // Purple
		SheetFill: color.NRGBA{R: 255, G: 255, B: 255, A: 64},   // White (very translucent)
		SheetPin:  color.NRGBA{R: 132, G: 0, B: 132, A: 255},    // Purple
		SheetText: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // Black

		// Text and annotations
		Text: color.NRGBA{R: 0, G: 0, B: 0, A: 255}, // Black

		// Selection and highlight
		Selection: color.NRGBA{R: 255, G: 0, B: 0, A: 128},    // Red (translucent)
		Highlight: color.NRGBA{R: 255, G: 255, B: 0, A: 128},  // Yellow (translucent)
	}
}

// getDarkTheme returns KiCad-style dark theme colors
func getDarkTheme() *SchematicColors {
	return &SchematicColors{
		// Background and grid
		Background: color.NRGBA{R: 30, G: 30, B: 30, A: 255},    // Dark gray (almost black)
		Grid:       color.NRGBA{R: 60, G: 60, B: 60, A: 255},    // Medium gray

		// Wires and connections (bright for contrast)
		Wire:      color.NRGBA{R: 0, G: 255, B: 0, A: 255},      // Bright green
		Bus:       color.NRGBA{R: 0, G: 150, B: 255, A: 255},    // Bright blue
		Junction:  color.NRGBA{R: 0, G: 255, B: 0, A: 255},      // Bright green (filled circle)
		NoConnect: color.NRGBA{R: 0, G: 150, B: 255, A: 255},    // Bright blue

		// Labels (bright colors for visibility)
		LocalLabel:  color.NRGBA{R: 255, G: 255, B: 0, A: 255},     // Yellow
		GlobalLabel: color.NRGBA{R: 255, G: 100, B: 100, A: 255},   // Light red
		HierLabel:   color.NRGBA{R: 255, G: 150, B: 0, A: 255},     // Orange

		// Symbols (bright red/cyan)
		SymbolBody:    color.NRGBA{R: 255, G: 100, B: 100, A: 255},  // Light red
		SymbolFill:    color.NRGBA{R: 60, G: 60, B: 0, A: 128},      // Dark yellow (translucent)
		SymbolPin:     color.NRGBA{R: 255, G: 100, B: 100, A: 255},  // Light red
		SymbolPinText: color.NRGBA{R: 100, G: 255, B: 255, A: 255},  // Cyan
		SymbolText:    color.NRGBA{R: 255, G: 255, B: 255, A: 255},  // White

		// Hierarchical sheets
		Sheet:     color.NRGBA{R: 255, G: 100, B: 255, A: 255},  // Light purple
		SheetFill: color.NRGBA{R: 50, G: 40, B: 50, A: 64},      // Dark purple (very translucent)
		SheetPin:  color.NRGBA{R: 255, G: 100, B: 255, A: 255},  // Light purple
		SheetText: color.NRGBA{R: 255, G: 255, B: 255, A: 255},  // White

		// Text and annotations
		Text: color.NRGBA{R: 220, G: 220, B: 220, A: 255}, // Light gray

		// Selection and highlight
		Selection: color.NRGBA{R: 255, G: 100, B: 100, A: 128},  // Light red (translucent)
		Highlight: color.NRGBA{R: 255, G: 255, B: 100, A: 128},  // Light yellow (translucent)
	}
}

// String returns the theme name as a string
func (t Theme) String() string {
	switch t {
	case ThemeLight:
		return "Light"
	case ThemeDark:
		return "Dark"
	default:
		return "Unknown"
	}
}
