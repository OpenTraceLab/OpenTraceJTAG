package renderer

import (
	"math"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp"
)

// Camera represents a viewport onto a KiCad file (PCB or schematic)
// This is a generic camera that works with any KiCad coordinate system.
type Camera struct {
	// Center position in world coordinates (mm)
	CenterX float64
	CenterY float64

	// Zoom level (pixels per mm)
	// Higher values = more zoomed in
	Zoom float64

	// Screen dimensions (pixels)
	ScreenWidth  int
	ScreenHeight int

	// View controls
	FlipView bool    // true = mirrored view
	Rotation float64 // rotation in degrees (0, 90, 180, 270)
	InvertY  bool    // true = flip Y axis (for PCB files), false = no flip (for schematics)

	// Rotation center (world coordinates in mm)
	// View will rotate/flip around this point
	RotationCenterX float64
	RotationCenterY float64
}

// NewCamera creates a camera with default settings
// For PCB viewing, set InvertY=true. For schematic viewing, set InvertY=false.
func NewCamera(screenWidth, screenHeight int) *Camera {
	return &Camera{
		Zoom:         10.0, // 10 pixels per mm is a reasonable default
		ScreenWidth:  screenWidth,
		ScreenHeight: screenHeight,
		InvertY:      true, // Default to true for PCB backward compatibility
	}
}

// WorldToScreen converts world coordinates (mm) to screen coordinates (pixels)
func (c *Camera) WorldToScreen(pos sexp.Position) (float64, float64) {
	// Apply view transformations (rotation and flip)
	pos = c.applyViewTransform(pos)

	// Translate so camera center is at origin
	x := pos.X - c.CenterX
	y := pos.Y - c.CenterY

	// Apply zoom
	x *= c.Zoom
	y *= c.Zoom

	// Translate to screen center
	x += float64(c.ScreenWidth) / 2.0
	y += float64(c.ScreenHeight) / 2.0

	// Flip Y axis if needed (PCB files have Y increasing upward, schematics have Y increasing downward)
	if c.InvertY {
		y = float64(c.ScreenHeight) - y
	}

	return x, y
}

// ScreenToWorld converts screen coordinates (pixels) to world coordinates (mm)
func (c *Camera) ScreenToWorld(screenX, screenY float64) sexp.Position {
	// Flip Y axis if needed
	y := screenY
	if c.InvertY {
		y = float64(c.ScreenHeight) - screenY
	}

	// Translate from screen center
	x := screenX - float64(c.ScreenWidth)/2.0
	y = y - float64(c.ScreenHeight)/2.0

	// Apply inverse zoom
	x /= c.Zoom
	y /= c.Zoom

	// Translate by camera position
	x += c.CenterX
	y += c.CenterY

	pos := sexp.Position{X: x, Y: y}

	// Apply inverse view transform
	return c.applyInverseViewTransform(pos)
}

// Pan moves the camera by screen pixel offsets
func (c *Camera) Pan(deltaX, deltaY float64) {
	// Convert screen delta to world delta
	c.CenterX -= deltaX / c.Zoom
	if c.InvertY {
		c.CenterY += deltaY / c.Zoom // Flip Y for PCB files
	} else {
		c.CenterY -= deltaY / c.Zoom // No flip for schematics
	}
}

// ZoomAt zooms in/out at a specific screen position
// factor > 1 zooms in, factor < 1 zooms out
func (c *Camera) ZoomAt(screenX, screenY, factor float64) {
	// Get world position before zoom
	worldPos := c.ScreenToWorld(screenX, screenY)

	// Apply zoom
	c.Zoom *= factor

	// Clamp zoom to reasonable limits
	if c.Zoom < 0.1 {
		c.Zoom = 0.1
	}
	if c.Zoom > 1000.0 {
		c.Zoom = 1000.0
	}

	// Get world position after zoom
	newWorldPos := c.ScreenToWorld(screenX, screenY)

	// Adjust center to keep the point under cursor stationary
	c.CenterX += worldPos.X - newWorldPos.X
	c.CenterY += worldPos.Y - newWorldPos.Y
}

// Fit adjusts camera to fit the entire content in view
func (c *Camera) Fit(bbox sexp.BoundingBox) {
	// Calculate content dimensions
	width := bbox.Max.X - bbox.Min.X
	height := bbox.Max.Y - bbox.Min.Y

	if width <= 0 || height <= 0 {
		return
	}

	// Center camera on content center
	c.CenterX = (bbox.Min.X + bbox.Max.X) / 2.0
	c.CenterY = (bbox.Min.Y + bbox.Max.Y) / 2.0

	// Set rotation center to content center
	c.RotationCenterX = c.CenterX
	c.RotationCenterY = c.CenterY

	// Calculate zoom to fit content with some padding (90% of screen)
	zoomX := float64(c.ScreenWidth) * 0.9 / width
	zoomY := float64(c.ScreenHeight) * 0.9 / height

	// Use the smaller zoom to ensure everything fits
	if zoomX < zoomY {
		c.Zoom = zoomX
	} else {
		c.Zoom = zoomY
	}
}

// UpdateScreenSize updates camera when window is resized
func (c *Camera) UpdateScreenSize(width, height int) {
	c.ScreenWidth = width
	c.ScreenHeight = height
}

// Flip toggles the view flip state (mirrored/normal)
func (c *Camera) Flip() {
	c.FlipView = !c.FlipView
}

// Rotate rotates the view by the given degrees
func (c *Camera) Rotate(degrees float64) {
	c.Rotation = c.Rotation + degrees
	// Normalize to 0-360 range
	for c.Rotation >= 360 {
		c.Rotation -= 360
	}
	for c.Rotation < 0 {
		c.Rotation += 360
	}
}

// applyViewTransform applies flip and rotation to a world position
func (c *Camera) applyViewTransform(pos sexp.Position) sexp.Position {
	x, y := pos.X, pos.Y

	// Translate to rotation center
	x -= c.RotationCenterX
	y -= c.RotationCenterY

	// Apply rotation
	if c.Rotation != 0 {
		rad := c.Rotation * math.Pi / 180.0
		cos := math.Cos(rad)
		sin := math.Sin(rad)
		newX := x*cos - y*sin
		newY := x*sin + y*cos
		x = newX
		y = newY
	}

	// Apply flip (mirror X axis)
	if c.FlipView {
		x = -x
	}

	// Translate back from rotation center
	x += c.RotationCenterX
	y += c.RotationCenterY

	return sexp.Position{X: x, Y: y}
}

// applyInverseViewTransform applies inverse flip and rotation
func (c *Camera) applyInverseViewTransform(pos sexp.Position) sexp.Position {
	x, y := pos.X, pos.Y

	// Translate to rotation center
	x -= c.RotationCenterX
	y -= c.RotationCenterY

	// Apply inverse flip first (before rotation)
	if c.FlipView {
		x = -x
	}

	// Apply inverse rotation
	if c.Rotation != 0 {
		rad := -c.Rotation * math.Pi / 180.0 // Negative for inverse
		cos := math.Cos(rad)
		sin := math.Sin(rad)
		newX := x*cos - y*sin
		newY := x*sin + y*cos
		x = newX
		y = newY
	}

	// Translate back from rotation center
	x += c.RotationCenterX
	y += c.RotationCenterY

	return sexp.Position{X: x, Y: y}
}

// GetVisibleBounds returns the bounding box of the visible area in world coordinates
// Useful for culling off-screen elements
func (c *Camera) GetVisibleBounds() sexp.BoundingBox {
	// Get corners of screen in world coordinates
	topLeft := c.ScreenToWorld(0, 0)
	topRight := c.ScreenToWorld(float64(c.ScreenWidth), 0)
	bottomLeft := c.ScreenToWorld(0, float64(c.ScreenHeight))
	bottomRight := c.ScreenToWorld(float64(c.ScreenWidth), float64(c.ScreenHeight))

	// Find min/max of all corners (needed because rotation might change which corner is which)
	minX := math.Min(math.Min(topLeft.X, topRight.X), math.Min(bottomLeft.X, bottomRight.X))
	maxX := math.Max(math.Max(topLeft.X, topRight.X), math.Max(bottomLeft.X, bottomRight.X))
	minY := math.Min(math.Min(topLeft.Y, topRight.Y), math.Min(bottomLeft.Y, bottomRight.Y))
	maxY := math.Max(math.Max(topLeft.Y, topRight.Y), math.Max(bottomLeft.Y, bottomRight.Y))

	return sexp.BoundingBox{
		Min: sexp.Position{X: minX, Y: minY},
		Max: sexp.Position{X: maxX, Y: maxY},
	}
}

// ========================
// Backward compatibility aliases for PCB renderer
// These methods provide compatibility with existing PCB rendering code
// ========================

// BoardToScreen is an alias for WorldToScreen (for PCB backward compatibility)
func (c *Camera) BoardToScreen(pos sexp.Position) (float64, float64) {
	return c.WorldToScreen(pos)
}

// FitBoard is an alias for Fit (for PCB backward compatibility)
func (c *Camera) FitBoard(bbox sexp.BoundingBox) {
	c.Fit(bbox)
}

// FlipBoard is a compatibility field accessor for PCB code
// Use FlipView for new code
func (c *Camera) SetFlipBoard(flip bool) {
	c.FlipView = flip
}

// GetFlipBoard returns the flip state (for PCB backward compatibility)
func (c *Camera) GetFlipBoard() bool {
	return c.FlipView
}

// BoardRotation is a compatibility field accessor for PCB code
// Use Rotation for new code
func (c *Camera) SetBoardRotation(rotation float64) {
	c.Rotation = rotation
}

// GetBoardRotation returns the rotation angle (for PCB backward compatibility)
func (c *Camera) GetBoardRotation() float64 {
	return c.Rotation
}
