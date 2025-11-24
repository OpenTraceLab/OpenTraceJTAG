package components

import (
	"fmt"
	"image/color"
	"strings"
)

// NewBGAPackage renders a ball-grid array with the provided column and row
// counts. Pins can be annotated with custom labels; otherwise default alphanumeric
// references (A1, B1, ...) are used for display.
func NewBGAPackage(cols, rows int, pins []PackagePin, opts *RenderOptions) PackageRender {
	if cols <= 0 {
		cols = 10
	}
	if rows <= 0 {
		rows = 10
	}
	total := cols * rows
	pins = EnsurePins(pins, total)
	cfg := normalizeOptions(opts)
	cellSize := scaledSize(28, cfg)
	margin := scaledSize(30, cfg)
	width := float32(cols)*cellSize + 2*margin
	height := float32(rows)*cellSize + 2*margin
	render := PackageRender{
		Size:  Vec{X: width, Y: height},
		Title: fmt.Sprintf("BGA %dx%d", rows, cols),
	}
	render.Rectangles = append(render.Rectangles, RectShape{
		Position:    Vec{},
		Size:        Vec{X: width, Y: height},
		Fill:        color.NRGBA{R: 25, G: 30, B: 40, A: 220},
		Stroke:      color.NRGBA{R: 215, G: 222, B: 233, A: 255},
		StrokeWidth: 2,
	})
	render.Circles = append(render.Circles, CircleShape{
		Center: Vec{X: margin - scaledSize(16, cfg) + scaledSize(6, cfg), Y: margin - scaledSize(16, cfg) + scaledSize(6, cfg)},
		Radius: scaledSize(6, cfg),
		Fill:   color.NRGBA{R: 250, G: 250, B: 250, A: 255},
	})
	ballRadius := scaledSize(10, cfg)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			idx := r*cols + c
			pin := pins[idx]
			x := float32(c)*cellSize + margin
			y := float32(r)*cellSize + margin
			coord := defaultBGABallName(r, c)
			ballColor := colorForState(pin.State)
			
			render.Circles = append(render.Circles, CircleShape{
				Center: Vec{X: x + ballRadius, Y: y + ballRadius},
				Radius: ballRadius,
				Fill:   ballColor,
			})
			
			// Always show coordinate on the ball itself
			render.Labels = append(render.Labels, LabelShape{
				Text:     coord,
				Color:    contrastColor(ballColor),
				Size:     scaledTextSize(7, cfg),
				Align:    AlignCenter,
				Position: Vec{X: x + ballRadius, Y: y + ballRadius},
			})
			
			addPadArea(&render.Pads, idx, pin, Vec{X: x, Y: y}, Vec{X: ballRadius * 2, Y: ballRadius * 2}, coord)
		}
	}
	return render
}

func defaultBGABallName(row, col int) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rowName := ""
	r := row
	for {
		rowName = string(alphabet[r%len(alphabet)]) + rowName
		r = r/len(alphabet) - 1
		if r < 0 {
			break
		}
	}
	return strings.ToUpper(fmt.Sprintf("%s%d", rowName, col+1))
}
