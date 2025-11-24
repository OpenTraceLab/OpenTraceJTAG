package components

import (
	"fmt"
	"image/color"
)

const (
	qfnBodySize      = float32(260)
	qfnPadLength     = float32(18)
	qfnPadWidth      = float32(8)
	qfnExposedMargin = float32(40)
	qfnEdgeMargin    = float32(18)
	qfnMinSpacing    = float32(12)
)

// NewQFNPackage renders a quad-flat no-lead package. Pins are distributed evenly
// across four edges (total pins = pinsPerSide*4). The center exposed pad is drawn
// with partial transparency to help visualize ground connections.
func NewQFNPackage(pinsPerSide int, pins []PackagePin, opts *RenderOptions) PackageRender {
	if pinsPerSide <= 0 {
		pinsPerSide = 12
	}
	total := pinsPerSide * 4
	pins = EnsurePins(pins, total)
	cfg := normalizeOptions(opts)

	bodySize := scaledSize(qfnBodySize, cfg)
	padLength := scaledSize(qfnPadLength, cfg)
	padWidth := scaledSize(qfnPadWidth, cfg)
	exposedMargin := scaledSize(qfnExposedMargin, cfg)

	render := PackageRender{
		Size:  Vec{X: bodySize, Y: bodySize},
		Title: fmt.Sprintf("QFN-%d", total),
	}
	render.Rectangles = append(render.Rectangles, RectShape{
		Position:    Vec{},
		Size:        Vec{X: bodySize, Y: bodySize},
		Fill:        color.NRGBA{R: 30, G: 35, B: 40, A: 230},
		Stroke:      color.NRGBA{R: 210, G: 210, B: 220, A: 255},
		StrokeWidth: 2,
	})
	render.Rectangles = append(render.Rectangles, RectShape{
		Position: Vec{X: exposedMargin, Y: exposedMargin},
		Size:     Vec{X: bodySize - 2*exposedMargin, Y: bodySize - 2*exposedMargin},
		Fill:     color.NRGBA{R: 120, G: 130, B: 160, A: 140},
	})
	render.Circles = append(render.Circles, CircleShape{
		Center: Vec{X: scaledSize(8, cfg) + scaledSize(6, cfg), Y: scaledSize(8, cfg) + scaledSize(6, cfg)},
		Radius: scaledSize(6, cfg),
		Fill:   color.NRGBA{R: 220, G: 220, B: 230, A: 255},
	})
	edgeMargin := scaledSize(qfnEdgeMargin, cfg)
	baseSpacing := scaledSize(qfnMinSpacing, cfg)
	var spacing float32
	if pinsPerSide > 1 {
		requiredSpan := baseSpacing * float32(pinsPerSide-1)
		actualSpan := bodySize - 2*edgeMargin - padWidth
		if actualSpan < requiredSpan {
			bodySize += requiredSpan - actualSpan
			actualSpan = requiredSpan
			render.Rectangles[1].Size = Vec{X: bodySize - 2*exposedMargin, Y: bodySize - 2*exposedMargin}
		}
		spacing = actualSpan / float32(pinsPerSide-1)
	}

	for i := 0; i < pinsPerSide; i++ {
		offset := edgeMargin + spacing*float32(i)

		// Left side pins
		leftIdx := i
		leftPos := Vec{X: -padLength, Y: offset}
		leftColor := colorForState(pins[leftIdx].State)
		render.Rectangles = append(render.Rectangles, RectShape{
			Position: leftPos,
			Size:     Vec{X: padLength, Y: padWidth},
			Fill:     leftColor,
		})
		addPadArea(&render.Pads, leftIdx, pins[leftIdx], leftPos, Vec{X: padLength, Y: padWidth}, fmt.Sprintf("%d", pins[leftIdx].Number))
		if cfg.ShowLabels {
			addLabel(&render.Labels, fmt.Sprintf("%d", pins[leftIdx].Number), AlignCenter, scaledTextSize(8, cfg), 
				Vec{X: -padLength/2, Y: offset + padWidth/2}, contrastColor(leftColor))
		}

		// Bottom side pins
		bottomIdx := pinsPerSide + i
		bottomPos := Vec{X: offset, Y: bodySize}
		bottomColor := colorForState(pins[bottomIdx].State)
		render.Rectangles = append(render.Rectangles, RectShape{
			Position: bottomPos,
			Size:     Vec{X: padWidth, Y: padLength},
			Fill:     bottomColor,
		})
		addPadArea(&render.Pads, bottomIdx, pins[bottomIdx], bottomPos, Vec{X: padWidth, Y: padLength}, fmt.Sprintf("%d", pins[bottomIdx].Number))
		if cfg.ShowLabels {
			addLabel(&render.Labels, fmt.Sprintf("%d", pins[bottomIdx].Number), AlignCenter, scaledTextSize(8, cfg), 
				Vec{X: offset + padWidth/2, Y: bodySize + padLength/2}, contrastColor(bottomColor))
		}

		// Right side pins
		rightIdx := pinsPerSide*2 + (pinsPerSide - 1 - i)
		rightPos := Vec{X: bodySize, Y: offset}
		rightColor := colorForState(pins[rightIdx].State)
		render.Rectangles = append(render.Rectangles, RectShape{
			Position: rightPos,
			Size:     Vec{X: padLength, Y: padWidth},
			Fill:     rightColor,
		})
		addPadArea(&render.Pads, rightIdx, pins[rightIdx], rightPos, Vec{X: padLength, Y: padWidth}, fmt.Sprintf("%d", pins[rightIdx].Number))
		if cfg.ShowLabels {
			addLabel(&render.Labels, fmt.Sprintf("%d", pins[rightIdx].Number), AlignCenter, scaledTextSize(8, cfg), 
				Vec{X: bodySize + padLength/2, Y: offset + padWidth/2}, contrastColor(rightColor))
		}

		// Top side pins
		topIdx := pinsPerSide*3 + (pinsPerSide - 1 - i)
		topPos := Vec{X: offset, Y: -padLength}
		topColor := colorForState(pins[topIdx].State)
		render.Rectangles = append(render.Rectangles, RectShape{
			Position: topPos,
			Size:     Vec{X: padWidth, Y: padLength},
			Fill:     topColor,
		})
		addPadArea(&render.Pads, topIdx, pins[topIdx], topPos, Vec{X: padWidth, Y: padLength}, fmt.Sprintf("%d", pins[topIdx].Number))
		if cfg.ShowLabels {
			addLabel(&render.Labels, fmt.Sprintf("%d", pins[topIdx].Number), AlignCenter, scaledTextSize(8, cfg), 
				Vec{X: offset + padWidth/2, Y: -padLength/2}, contrastColor(topColor))
		}
	}

	return render
}
