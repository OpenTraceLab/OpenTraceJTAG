package components

import (
	"fmt"
	"image/color"
)

const (
	qfpBodySize   = float32(320)
	qfpPadLength  = float32(26)
	qfpPadWidth   = float32(10)
	qfpPadSpacing = float32(14)
)

// NewQFPPackage renders a quad-flat package with the provided pins-per-side
// (total pins = pinsPerSide*4). Pins are drawn along each edge with labels.
func NewQFPPackage(pinsPerSide int, pins []PackagePin, opts *RenderOptions) PackageRender {
	if pinsPerSide <= 0 {
		pinsPerSide = 20
	}
	total := pinsPerSide * 4
	pins = EnsurePins(pins, total)
	cfg := normalizeOptions(opts)

	bodySize := scaledSize(qfpBodySize, cfg)
	padLength := scaledSize(qfpPadLength, cfg)
	padWidth := scaledSize(qfpPadWidth, cfg)
	edgeMargin := scaledSize(24, cfg)
	baseSpacing := scaledSize(qfpPadSpacing, cfg)
	var spacing float32
	if pinsPerSide > 1 {
		requiredSpan := baseSpacing * float32(pinsPerSide-1)
		actualSpan := bodySize - 2*edgeMargin - padWidth
		if actualSpan < requiredSpan {
			bodySize += requiredSpan - actualSpan
			actualSpan = requiredSpan
		}
		spacing = actualSpan / float32(pinsPerSide-1)
	}

	render := PackageRender{
		Size:  Vec{X: bodySize, Y: bodySize},
		Title: fmt.Sprintf("QFP-%d", total),
	}
	render.Rectangles = append(render.Rectangles, RectShape{
		Position:    Vec{},
		Size:        Vec{X: bodySize, Y: bodySize},
		Fill:        color.NRGBA{R: 35, G: 39, B: 48, A: 220},
		Stroke:      color.NRGBA{R: 220, G: 220, B: 230, A: 255},
		StrokeWidth: 2,
	})
	render.Circles = append(render.Circles, CircleShape{
		Center: Vec{X: scaledSize(10, cfg) + scaledSize(8, cfg), Y: scaledSize(10, cfg) + scaledSize(8, cfg)},
		Radius: scaledSize(8, cfg),
		Fill:   color.NRGBA{R: 220, G: 220, B: 230, A: 255},
	})

	for i := 0; i < pinsPerSide; i++ {
		offsetScaled := edgeMargin + spacing*float32(i)

		// Left side pins
		leftIdx := i
		leftPos := Vec{X: -padLength, Y: offsetScaled}
		leftColor := colorForState(pins[leftIdx].State)
		render.Rectangles = append(render.Rectangles, RectShape{
			Position: leftPos,
			Size:     Vec{X: padLength, Y: padWidth},
			Fill:     leftColor,
		})
		addPadArea(&render.Pads, leftIdx, pins[leftIdx], leftPos, Vec{X: padLength, Y: padWidth}, fmt.Sprintf("%d", pins[leftIdx].Number))
		if cfg.ShowLabels {
			addLabel(&render.Labels, fmt.Sprintf("%d", pins[leftIdx].Number), AlignCenter, scaledTextSize(8, cfg), 
				Vec{X: -padLength/2, Y: offsetScaled + padWidth/2}, contrastColor(leftColor))
		}

		// Bottom side pins
		bottomIdx := pinsPerSide + i
		bottomPos := Vec{X: offsetScaled, Y: bodySize}
		bottomColor := colorForState(pins[bottomIdx].State)
		render.Rectangles = append(render.Rectangles, RectShape{
			Position: bottomPos,
			Size:     Vec{X: padWidth, Y: padLength},
			Fill:     bottomColor,
		})
		addPadArea(&render.Pads, bottomIdx, pins[bottomIdx], bottomPos, Vec{X: padWidth, Y: padLength}, fmt.Sprintf("%d", pins[bottomIdx].Number))
		if cfg.ShowLabels {
			addLabel(&render.Labels, fmt.Sprintf("%d", pins[bottomIdx].Number), AlignCenter, scaledTextSize(8, cfg), 
				Vec{X: offsetScaled + padWidth/2, Y: bodySize + padLength/2}, contrastColor(bottomColor))
		}

		// Right side pins
		rightIdx := pinsPerSide*2 + (pinsPerSide - 1 - i)
		rightPos := Vec{X: bodySize, Y: offsetScaled}
		rightColor := colorForState(pins[rightIdx].State)
		render.Rectangles = append(render.Rectangles, RectShape{
			Position: rightPos,
			Size:     Vec{X: padLength, Y: padWidth},
			Fill:     rightColor,
		})
		addPadArea(&render.Pads, rightIdx, pins[rightIdx], rightPos, Vec{X: padLength, Y: padWidth}, fmt.Sprintf("%d", pins[rightIdx].Number))
		if cfg.ShowLabels {
			addLabel(&render.Labels, fmt.Sprintf("%d", pins[rightIdx].Number), AlignCenter, scaledTextSize(8, cfg), 
				Vec{X: bodySize + padLength/2, Y: offsetScaled + padWidth/2}, contrastColor(rightColor))
		}

		// Top side pins
		topIdx := pinsPerSide*3 + (pinsPerSide - 1 - i)
		topPos := Vec{X: offsetScaled, Y: -padLength}
		topColor := colorForState(pins[topIdx].State)
		render.Rectangles = append(render.Rectangles, RectShape{
			Position: topPos,
			Size:     Vec{X: padWidth, Y: padLength},
			Fill:     topColor,
		})
		addPadArea(&render.Pads, topIdx, pins[topIdx], topPos, Vec{X: padWidth, Y: padLength}, fmt.Sprintf("%d", pins[topIdx].Number))
		if cfg.ShowLabels {
			addLabel(&render.Labels, fmt.Sprintf("%d", pins[topIdx].Number), AlignCenter, scaledTextSize(8, cfg), 
				Vec{X: offsetScaled + padWidth/2, Y: -padLength/2}, contrastColor(topColor))
		}
	}

	return render
}
