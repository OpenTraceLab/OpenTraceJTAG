package renderer

// LayerConfig controls which layers are visible during rendering
type LayerConfig struct {
	visible map[string]bool
}

// NewLayerConfig creates a new layer configuration with all layers visible by default
func NewLayerConfig() *LayerConfig {
	return &LayerConfig{
		visible: make(map[string]bool),
	}
}

// SetVisible sets the visibility of a specific layer
func (lc *LayerConfig) SetVisible(layer string, visible bool) {
	lc.visible[layer] = visible
}

// IsVisible returns whether a layer is visible (default: false if not set)
func (lc *LayerConfig) IsVisible(layer string) bool {
	if visible, exists := lc.visible[layer]; exists {
		return visible
	}
	return false // Default to hidden
}

// HideAll hides all layers
func (lc *LayerConfig) HideAll() {
	// Set a special marker to indicate all layers should be hidden by default
	lc.visible["*"] = false
}

// ShowAll shows all layers
func (lc *LayerConfig) ShowAll() {
	// Clear the map to return to default (all visible)
	lc.visible = make(map[string]bool)
}

// ShowOnly shows only the specified layers, hiding all others
func (lc *LayerConfig) ShowOnly(layers ...string) {
	lc.HideAll()
	for _, layer := range layers {
		lc.SetVisible(layer, true)
	}
}

// Common layer groups for convenience
func (lc *LayerConfig) ShowCopperOnly() {
	lc.ShowOnly("F.Cu", "B.Cu", "In1.Cu", "In2.Cu", "In3.Cu", "In4.Cu")
}

func (lc *LayerConfig) ShowSilkscreenOnly() {
	lc.ShowOnly("F.SilkS", "B.SilkS")
}

func (lc *LayerConfig) ShowFabOnly() {
	lc.ShowOnly("F.Fab", "B.Fab")
}

func (lc *LayerConfig) HideCopper() {
	lc.SetVisible("F.Cu", false)
	lc.SetVisible("B.Cu", false)
	lc.SetVisible("In1.Cu", false)
	lc.SetVisible("In2.Cu", false)
	lc.SetVisible("In3.Cu", false)
	lc.SetVisible("In4.Cu", false)
}

func (lc *LayerConfig) HideSilkscreen() {
	lc.SetVisible("F.SilkS", false)
	lc.SetVisible("B.SilkS", false)
}
