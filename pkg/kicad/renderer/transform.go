package renderer

import (
	"math"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/parser"
)

// Transform represents a 2D transformation (translate + rotate + scale)
type Transform struct {
	TranslateX float64 // Translation in X
	TranslateY float64 // Translation in Y
	Rotate     float64 // Rotation in degrees
	ScaleX     float64 // Scale factor in X
	ScaleY     float64 // Scale factor in Y
}

// NewTransform creates an identity transform
func NewTransform() Transform {
	return Transform{
		ScaleX: 1.0,
		ScaleY: 1.0,
	}
}

// Apply applies the transformation to a position
func (t Transform) Apply(pos parser.Position) parser.Position {
	x, y := pos.X, pos.Y

	// Apply scale
	x *= t.ScaleX
	y *= t.ScaleY

	// Apply rotation (convert to radians)
	if t.Rotate != 0 {
		rad := t.Rotate * math.Pi / 180.0
		cos := math.Cos(rad)
		sin := math.Sin(rad)
		newX := x*cos - y*sin
		newY := x*sin + y*cos
		x = newX
		y = newY
	}

	// Apply translation
	x += t.TranslateX
	y += t.TranslateY

	return parser.Position{X: x, Y: y}
}

// ApplyInverse applies the inverse transformation (for screen to world)
func (t Transform) ApplyInverse(pos parser.Position) parser.Position {
	x, y := pos.X, pos.Y

	// Inverse translation
	x -= t.TranslateX
	y -= t.TranslateY

	// Inverse rotation
	if t.Rotate != 0 {
		rad := -t.Rotate * math.Pi / 180.0 // Negative for inverse
		cos := math.Cos(rad)
		sin := math.Sin(rad)
		newX := x*cos - y*sin
		newY := x*sin + y*cos
		x = newX
		y = newY
	}

	// Inverse scale
	if t.ScaleX != 0 {
		x /= t.ScaleX
	}
	if t.ScaleY != 0 {
		y /= t.ScaleY
	}

	return parser.Position{X: x, Y: y}
}

// Camera represents a viewport onto the board
type Camera struct {
	// Center position in board coordinates (mm)
	CenterX float64
	CenterY float64

	// Zoom level (pixels per mm)
	// Higher values = more zoomed in
	Zoom float64

	// Screen dimensions (pixels)
	ScreenWidth  int
	ScreenHeight int

	// View controls
	FlipBoard     bool    // true = bottom view (mirrored)
	BoardRotation float64 // rotation in degrees (0, 90, 180, 270)

	// Rotation center (board coordinates in mm)
	// Board will rotate/flip around this point
	RotationCenterX float64
	RotationCenterY float64
}

// NewCamera creates a camera with default settings
func NewCamera(screenWidth, screenHeight int) *Camera {
	return &Camera{
		Zoom:         10.0, // 10 pixels per mm is a reasonable default
		ScreenWidth:  screenWidth,
		ScreenHeight: screenHeight,
	}
}

// BoardToScreen converts board coordinates (mm) to screen coordinates (pixels)
func (c *Camera) BoardToScreen(pos parser.Position) (float64, float64) {
	// Apply board transformations (rotation and flip)
	pos = c.applyBoardTransform(pos)

	// Translate so camera center is at origin
	x := pos.X - c.CenterX
	y := pos.Y - c.CenterY

	// Apply zoom
	x *= c.Zoom
	y *= c.Zoom

	// Translate to screen center
	x += float64(c.ScreenWidth) / 2.0
	y += float64(c.ScreenHeight) / 2.0

	// Flip Y axis (screen Y increases downward, board Y increases upward)
	y = float64(c.ScreenHeight) - y

	return x, y
}

// ScreenToBoard converts screen coordinates (pixels) to board coordinates (mm)
func (c *Camera) ScreenToBoard(screenX, screenY float64) parser.Position {
	// Flip Y axis
	y := float64(c.ScreenHeight) - screenY

	// Translate from screen center
	x := screenX - float64(c.ScreenWidth)/2.0
	y = y - float64(c.ScreenHeight)/2.0

	// Apply inverse zoom
	x /= c.Zoom
	y /= c.Zoom

	// Translate by camera position
	x += c.CenterX
	y += c.CenterY

	return parser.Position{X: x, Y: y}
}

// Pan moves the camera by screen pixel offsets
func (c *Camera) Pan(deltaX, deltaY float64) {
	// Convert screen delta to board delta
	c.CenterX -= deltaX / c.Zoom
	c.CenterY += deltaY / c.Zoom // Flip Y
}

// ZoomAt zooms in/out at a specific screen position
// factor > 1 zooms in, factor < 1 zooms out
func (c *Camera) ZoomAt(screenX, screenY, factor float64) {
	// Get board position before zoom
	boardPos := c.ScreenToBoard(screenX, screenY)

	// Apply zoom
	c.Zoom *= factor

	// Clamp zoom to reasonable limits
	if c.Zoom < 0.1 {
		c.Zoom = 0.1
	}
	if c.Zoom > 1000.0 {
		c.Zoom = 1000.0
	}

	// Get board position after zoom
	newBoardPos := c.ScreenToBoard(screenX, screenY)

	// Adjust center to keep the point under cursor stationary
	c.CenterX += boardPos.X - newBoardPos.X
	c.CenterY += boardPos.Y - newBoardPos.Y
}

// FitBoard adjusts camera to fit the entire board in view
func (c *Camera) FitBoard(bbox parser.BoundingBox) {
	// Calculate board dimensions
	boardWidth := bbox.Max.X - bbox.Min.X
	boardHeight := bbox.Max.Y - bbox.Min.Y

	if boardWidth <= 0 || boardHeight <= 0 {
		return
	}

	// Center camera on board center
	c.CenterX = (bbox.Min.X + bbox.Max.X) / 2.0
	c.CenterY = (bbox.Min.Y + bbox.Max.Y) / 2.0

	// Set rotation center to board center
	c.RotationCenterX = c.CenterX
	c.RotationCenterY = c.CenterY

	// Calculate zoom to fit board with some padding (90% of screen)
	zoomX := float64(c.ScreenWidth) * 0.9 / boardWidth
	zoomY := float64(c.ScreenHeight) * 0.9 / boardHeight

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

// Flip toggles the board flip state (top/bottom view)
func (c *Camera) Flip() {
	c.FlipBoard = !c.FlipBoard
}

// Rotate rotates the board view by the given degrees
func (c *Camera) Rotate(degrees float64) {
	c.BoardRotation = c.BoardRotation + degrees
	// Normalize to 0-360 range
	for c.BoardRotation >= 360 {
		c.BoardRotation -= 360
	}
	for c.BoardRotation < 0 {
		c.BoardRotation += 360
	}
}

// applyBoardTransform applies flip and rotation to a board position
func (c *Camera) applyBoardTransform(pos parser.Position) parser.Position {
	x, y := pos.X, pos.Y

	// Translate to rotation center
	x -= c.RotationCenterX
	y -= c.RotationCenterY

	// Apply rotation
	if c.BoardRotation != 0 {
		rad := c.BoardRotation * math.Pi / 180.0
		cos := math.Cos(rad)
		sin := math.Sin(rad)
		newX := x*cos - y*sin
		newY := x*sin + y*cos
		x = newX
		y = newY
	}

	// Apply flip (mirror X axis for bottom view)
	if c.FlipBoard {
		x = -x
	}

	// Translate back from rotation center
	x += c.RotationCenterX
	y += c.RotationCenterY

	return parser.Position{X: x, Y: y}
}
