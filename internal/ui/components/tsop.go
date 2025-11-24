package components

import (
	"fmt"
	"image/color"
)

const (
	tsopNarrowBodyWidth = float32(240) // TSOP-I narrow
	tsopWideBodyWidth   = float32(360) // TSOP-II wide
	tsopDefaultBodyHeight = float32(180)
	tsopPadWidth          = float32(18)
	tsopPadHeight         = float32(18)
	tsopPadGap            = float32(4)
	tsopMargin            = float32(48)
)

// NewTSOPPackage renders a top-down view for an arbitrary TSOP package (pins
// along the two long edges). pinCount must be even. Pins are labelled using the
// provided metadata or default sequential names.
// isWide determines if it's TSOP-II (wide) or TSOP-I (narrow).
func NewTSOPPackage(pinCount int, pins []PackagePin, isWide bool, opts *RenderOptions) PackageRender {
	if pinCount%2 != 0 || pinCount <= 0 {
		pinCount = 48
	}

	sidePins := pinCount / 2
	pins = EnsurePins(pins, pinCount)
	cfg := normalizeOptions(opts)

	// Select body width based on variant
	baseBodyWidth := tsopNarrowBodyWidth
	if isWide {
		baseBodyWidth = tsopWideBodyWidth
	}
	
	bodyWidth := scaledSize(baseBodyWidth, cfg)
	bodyHeight := scaledSize(tsopDefaultBodyHeight, cfg)
	padWidth := scaledSize(tsopPadWidth, cfg)
	padHeight := scaledSize(tsopPadHeight, cfg)
	padGap := scaledSize(tsopPadGap, cfg)
	margin := scaledSize(tsopMargin, cfg)
	minBodyHeight := float32(sidePins) * (padHeight + padGap)
	if minBodyHeight > bodyHeight {
		bodyHeight = minBodyHeight
	}
	var padSpacing float32
	if sidePins > 1 {
		padSpacing = (bodyHeight - padHeight) / float32(sidePins-1)
	} else {
		padSpacing = 0
	}
	totalWidth := bodyWidth + margin*2 + padWidth*2
	totalHeight := bodyHeight + margin*2
	topOffset := margin
	leftBody := margin + padWidth + padGap

	render := PackageRender{
		Size:  Vec{X: totalWidth, Y: totalHeight},
		Title: fmt.Sprintf("TSOP-%d", pinCount),
	}

	bodyColor := color.NRGBA{R: 40, G: 45, B: 62, A: 220}
	render.Rectangles = append(render.Rectangles, RectShape{
		Position:    Vec{X: leftBody, Y: topOffset},
		Size:        Vec{X: bodyWidth, Y: bodyHeight},
		Fill:        bodyColor,
		Stroke:      color.NRGBA{R: 230, G: 233, B: 242, A: 255},
		StrokeWidth: 2,
	})
	// Position marker circle closer to corner for better symmetry
	markerRadius := padHeight / 2
	render.Circles = append(render.Circles, CircleShape{
		Center: Vec{X: leftBody + scaledSize(28, cfg), Y: topOffset + scaledSize(28, cfg)},
		Radius: markerRadius,
		Fill:   color.NRGBA{R: 230, G: 233, B: 242, A: 255},
	})

	for i := 0; i < sidePins; i++ {
		y := topOffset + padSpacing*float32(i)

		// Left side pins
		leftPin := pins[i]
		leftPos := Vec{X: leftBody - padWidth - padGap, Y: y}
		leftColor := colorForState(leftPin.State)
		render.Rectangles = append(render.Rectangles, RectShape{
			Position: leftPos,
			Size:     Vec{X: padWidth, Y: padHeight},
			Fill:     leftColor,
		})
		addPadArea(&render.Pads, i, leftPin, leftPos, Vec{X: padWidth, Y: padHeight}, fmt.Sprintf("%d", leftPin.Number))
		if cfg.ShowLabels {
			addLabel(&render.Labels, fmt.Sprintf("%d", leftPin.Number), AlignCenter, scaledTextSize(8, cfg), 
				Vec{X: leftBody - padWidth - padGap + padWidth/2, Y: y + padHeight/2}, contrastColor(leftColor))
		}

		// Right side pins
		rightIndex := pinCount - 1 - i
		rightPin := pins[rightIndex]
		rightPadX := leftBody + bodyWidth + padGap
		rightPos := Vec{X: rightPadX, Y: y}
		rightColor := colorForState(rightPin.State)
		render.Rectangles = append(render.Rectangles, RectShape{
			Position: rightPos,
			Size:     Vec{X: padWidth, Y: padHeight},
			Fill:     rightColor,
		})
		addPadArea(&render.Pads, rightIndex, rightPin, rightPos, Vec{X: padWidth, Y: padHeight}, fmt.Sprintf("%d", rightPin.Number))
		if cfg.ShowLabels {
			addLabel(&render.Labels, fmt.Sprintf("%d", rightPin.Number), AlignCenter, scaledTextSize(8, cfg), 
				Vec{X: rightPadX + padWidth/2, Y: y + padHeight/2}, contrastColor(rightColor))
		}
	}

	return render
}
