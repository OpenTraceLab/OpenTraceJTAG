package renderer

import "image/color"

// ColorTheme represents a KiCad color theme
type ColorTheme int

const (
	ThemeClassic ColorTheme = iota
	ThemeKiCad2020
	ThemeBlueTone
	ThemeEagle
	ThemeNord
)

// ThemeNames maps theme enum to display name
var ThemeNames = map[ColorTheme]string{
	ThemeClassic:   "Classic",
	ThemeKiCad2020: "KiCad 2020",
	ThemeBlueTone:  "Blue Tone",
	ThemeEagle:     "Eagle",
	ThemeNord:      "Nord",
}

// CurrentTheme is the active color theme (default: Classic)
var CurrentTheme = ThemeClassic

// KiCad Classic theme colors
var classicColors = map[string]color.NRGBA{
	// Copper layers
	"F.Cu":  {R: 200, G: 52, B: 52, A: 255},    // Front copper (red)
	"B.Cu":  {R: 77, G: 127, B: 196, A: 255},   // Back copper (blue)
	"In1.Cu": {R: 127, G: 200, B: 127, A: 255}, // Inner layer 1
	"In2.Cu": {R: 206, G: 125, B: 44, A: 255},  // Inner layer 2
	
	// Silkscreen
	"F.SilkS": {R: 242, G: 237, B: 161, A: 255}, // Front silkscreen (yellow)
	"B.SilkS": {R: 232, G: 178, B: 167, A: 255}, // Back silkscreen (pink)
	
	// Solder mask
	"F.Mask": {R: 216, G: 100, B: 255, A: 102}, // Front mask (purple, semi-transparent)
	"B.Mask": {R: 2, G: 255, B: 238, A: 102},   // Back mask (cyan, semi-transparent)
	
	// Paste
	"F.Paste": {R: 180, G: 160, B: 154, A: 230}, // Front paste
	"B.Paste": {R: 0, G: 194, B: 194, A: 230},   // Back paste
	
	// Fabrication
	"F.Fab": {R: 175, G: 175, B: 175, A: 255}, // Front fab (gray)
	"B.Fab": {R: 88, G: 93, B: 132, A: 255},   // Back fab (dark blue)
	
	// Courtyard
	"F.CrtYd": {R: 255, G: 38, B: 226, A: 255}, // Front courtyard (magenta)
	"B.CrtYd": {R: 38, G: 233, B: 255, A: 255}, // Back courtyard (cyan)
	
	// Adhesive
	"F.Adhes": {R: 132, G: 0, B: 132, A: 255}, // Front adhesive
	"B.Adhes": {R: 0, G: 0, B: 132, A: 255},   // Back adhesive
	
	// User layers
	"Dwgs.User":  {R: 194, G: 194, B: 194, A: 255}, // Drawings
	"Cmts.User":  {R: 89, G: 148, B: 220, A: 255},  // Comments
	"Eco1.User":  {R: 180, G: 219, B: 210, A: 255}, // ECO1
	"Eco2.User":  {R: 216, G: 200, B: 82, A: 255},  // ECO2
	"Edge.Cuts":  {R: 208, G: 210, B: 205, A: 255}, // Board edge
	"Margin":     {R: 255, G: 38, B: 226, A: 255},  // Margin
	
	// User drawing layers
	"User.1": {R: 194, G: 194, B: 194, A: 255},
	"User.2": {R: 89, G: 148, B: 220, A: 255},
	"User.3": {R: 180, G: 219, B: 210, A: 255},
	"User.4": {R: 216, G: 200, B: 82, A: 255},
	"User.5": {R: 194, G: 194, B: 194, A: 255},
	"User.6": {R: 89, G: 148, B: 220, A: 255},
	"User.7": {R: 180, G: 219, B: 210, A: 255},
	"User.8": {R: 216, G: 200, B: 82, A: 255},
	"User.9": {R: 232, G: 178, B: 167, A: 255},
}

// Special colors
var (
	ColorPadTH       = color.NRGBA{R: 227, G: 183, B: 46, A: 255}  // Through-hole pad (gold)
	ColorPadSMD      = color.NRGBA{R: 227, G: 183, B: 46, A: 255}  // SMD pad (gold)
	ColorDrill       = color.NRGBA{R: 227, G: 183, B: 46, A: 255}  // Drill hole (gold)
	ColorVia         = color.NRGBA{R: 236, G: 236, B: 236, A: 255} // Via (light gray)
	ColorViaDrill    = color.NRGBA{R: 227, G: 183, B: 46, A: 255}  // Via drill (gold)
	ColorTrack       = color.NRGBA{R: 200, G: 52, B: 52, A: 255}   // Track (red, same as F.Cu)
	ColorZone        = color.NRGBA{R: 200, G: 52, B: 52, A: 180}   // Zone fill (red, semi-transparent)
	ColorBackground  = color.NRGBA{R: 0, G: 16, B: 35, A: 255}     // Background (dark blue)
)

// GetSubstrateColor returns the PCB substrate color for the current theme
func GetSubstrateColor() color.NRGBA {
	switch CurrentTheme {
	case ThemeKiCad2020:
		return color.NRGBA{R: 25, G: 95, B: 55, A: 255} // Slightly brighter green
	case ThemeBlueTone:
		return color.NRGBA{R: 20, G: 60, B: 90, A: 255} // Blue substrate
	case ThemeEagle:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255} // Black substrate (Eagle style)
	case ThemeNord:
		return color.NRGBA{R: 46, G: 52, B: 64, A: 255} // Nord0 (dark gray)
	default: // ThemeClassic
		return color.NRGBA{R: 20, G: 90, B: 50, A: 255} // Dark green (classic PCB)
	}
}

// GetLayerColor returns the color for a given layer name using the current theme
func GetLayerColor(layer string) color.NRGBA {
	var colors map[string]color.NRGBA
	
	switch CurrentTheme {
	case ThemeKiCad2020:
		colors = kicad2020Colors
	case ThemeBlueTone:
		colors = blueToneColors
	case ThemeEagle:
		colors = eagleColors
	case ThemeNord:
		colors = nordColors
	default:
		colors = classicColors
	}
	
	if c, ok := colors[layer]; ok {
		return c
	}
	// Default to gray for unknown layers
	return color.NRGBA{R: 128, G: 128, B: 128, A: 255}
}

// SetTheme changes the active color theme
func SetTheme(theme ColorTheme) {
	CurrentTheme = theme
}

// KiCad 2020 theme colors (modern, higher contrast)
var kicad2020Colors = map[string]color.NRGBA{
	"F.Cu":       {R: 179, G: 31, B: 31, A: 255},
	"B.Cu":       {R: 12, G: 98, B: 179, A: 255},
	"In1.Cu":     {R: 194, G: 194, B: 0, A: 255},
	"In2.Cu":     {R: 194, G: 0, B: 194, A: 255},
	"F.SilkS":    {R: 242, G: 237, B: 161, A: 255},
	"B.SilkS":    {R: 232, G: 178, B: 167, A: 255},
	"F.Mask":     {R: 132, G: 0, B: 132, A: 102},
	"B.Mask":     {R: 2, G: 132, B: 132, A: 102},
	"Edge.Cuts":  {R: 255, G: 255, B: 0, A: 255},
	"F.CrtYd":    {R: 255, G: 0, B: 255, A: 255},
	"B.CrtYd":    {R: 0, G: 255, B: 255, A: 255},
	"F.Fab":      {R: 128, G: 128, B: 128, A: 255},
	"B.Fab":      {R: 64, G: 64, B: 128, A: 255},
	"Dwgs.User":  {R: 255, G: 255, B: 255, A: 255},
	"Cmts.User":  {R: 0, G: 150, B: 255, A: 255},
	"Eco1.User":  {R: 0, G: 255, B: 0, A: 255},
	"Eco2.User":  {R: 255, G: 255, B: 0, A: 255},
}

// Blue Tone theme
var blueToneColors = map[string]color.NRGBA{
	"F.Cu":       {R: 72, G: 72, B: 200, A: 255},
	"B.Cu":       {R: 0, G: 132, B: 132, A: 255},
	"In1.Cu":     {R: 127, G: 200, B: 200, A: 255},
	"In2.Cu":     {R: 91, G: 195, B: 235, A: 255},
	"F.SilkS":    {R: 242, G: 242, B: 255, A: 255},
	"B.SilkS":    {R: 178, G: 178, B: 232, A: 255},
	"F.Mask":     {R: 52, G: 52, B: 255, A: 102},
	"B.Mask":     {R: 2, G: 132, B: 255, A: 102},
	"Edge.Cuts":  {R: 208, G: 210, B: 255, A: 255},
	"F.CrtYd":    {R: 150, G: 150, B: 255, A: 255},
	"B.CrtYd":    {R: 38, G: 200, B: 255, A: 255},
	"F.Fab":      {R: 175, G: 175, B: 200, A: 255},
	"B.Fab":      {R: 88, G: 93, B: 180, A: 255},
	"Dwgs.User":  {R: 194, G: 194, B: 255, A: 255},
	"Cmts.User":  {R: 89, G: 148, B: 255, A: 255},
	"Eco1.User":  {R: 180, G: 219, B: 255, A: 255},
	"Eco2.User":  {R: 150, G: 200, B: 255, A: 255},
}

// Eagle theme (similar to Eagle CAD)
var eagleColors = map[string]color.NRGBA{
	"F.Cu":       {R: 204, G: 0, B: 0, A: 255},
	"B.Cu":       {R: 0, G: 0, B: 204, A: 255},
	"In1.Cu":     {R: 194, G: 194, B: 0, A: 255},
	"In2.Cu":     {R: 194, G: 0, B: 194, A: 255},
	"F.SilkS":    {R: 255, G: 255, B: 255, A: 255},
	"B.SilkS":    {R: 200, G: 200, B: 200, A: 255},
	"F.Mask":     {R: 200, G: 61, B: 217, A: 102},
	"B.Mask":     {R: 61, G: 217, B: 217, A: 102},
	"Edge.Cuts":  {R: 255, G: 255, B: 0, A: 255},
	"F.CrtYd":    {R: 255, G: 0, B: 255, A: 255},
	"B.CrtYd":    {R: 0, G: 255, B: 255, A: 255},
	"F.Fab":      {R: 200, G: 200, B: 200, A: 255},
	"B.Fab":      {R: 100, G: 100, B: 150, A: 255},
	"Dwgs.User":  {R: 255, G: 255, B: 255, A: 255},
	"Cmts.User":  {R: 132, G: 132, B: 132, A: 255},
	"Eco1.User":  {R: 0, G: 255, B: 0, A: 255},
	"Eco2.User":  {R: 255, G: 255, B: 0, A: 255},
}

// Nord theme (based on Nord color palette)
var nordColors = map[string]color.NRGBA{
	"F.Cu":       {R: 191, G: 97, B: 106, A: 255},   // Nord11 (red)
	"B.Cu":       {R: 129, G: 161, B: 193, A: 255},  // Nord9 (blue)
	"In1.Cu":     {R: 163, G: 190, B: 140, A: 255},  // Nord14 (green)
	"In2.Cu":     {R: 235, G: 203, B: 139, A: 255},  // Nord13 (yellow)
	"F.SilkS":    {R: 236, G: 239, B: 244, A: 255},  // Nord6 (light)
	"B.SilkS":    {R: 216, G: 222, B: 233, A: 255},  // Nord4
	"F.Mask":     {R: 180, G: 142, B: 173, A: 102},  // Nord15 (purple)
	"B.Mask":     {R: 136, G: 192, B: 208, A: 102},  // Nord8 (cyan)
	"Edge.Cuts":  {R: 229, G: 233, B: 240, A: 255},  // Nord5
	"F.CrtYd":    {R: 180, G: 142, B: 173, A: 255},  // Nord15
	"B.CrtYd":    {R: 136, G: 192, B: 208, A: 255},  // Nord8
	"F.Fab":      {R: 216, G: 222, B: 233, A: 255},  // Nord4
	"B.Fab":      {R: 143, G: 188, B: 187, A: 255},  // Nord7
	"Dwgs.User":  {R: 229, G: 233, B: 240, A: 255},  // Nord5
	"Cmts.User":  {R: 94, G: 129, B: 172, A: 255},   // Nord10
	"Eco1.User":  {R: 163, G: 190, B: 140, A: 255},  // Nord14
	"Eco2.User":  {R: 235, G: 203, B: 139, A: 255},  // Nord13
}

// GetLayerColor returns the color for a given layer name using the current theme (kept for backward compatibility)
var LayerColors = classicColors
