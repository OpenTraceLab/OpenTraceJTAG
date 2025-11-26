package pcb

import "math"

// GetBoundingBox calculates the bounding box of the entire board
// Includes tracks, pads, graphics, and vias
func (b *Board) GetBoundingBox() BoundingBox {
	bbox := NewBoundingBox()

	// Include all tracks
	for _, track := range b.Tracks {
		bbox.Expand(track.Start)
		bbox.Expand(track.End)
	}

	// Include all vias
	for _, via := range b.Vias {
		// Vias have a size, so expand by radius
		radius := via.Size / 2.0
		bbox.Expand(Position{X: via.Position.X - radius, Y: via.Position.Y - radius})
		bbox.Expand(Position{X: via.Position.X + radius, Y: via.Position.Y + radius})
	}

	// Include all footprint pads (with position transformation)
	for _, fp := range b.Footprints {
		fpBBox := fp.GetBoundingBox()
		bbox.ExpandBox(fpBBox)
	}

	// Include all graphics
	for _, line := range b.Graphics.Lines {
		bbox.Expand(line.Start)
		bbox.Expand(line.End)
	}

	for _, circle := range b.Graphics.Circles {
		// Calculate radius from center to end point
		dx := circle.End.X - circle.Center.X
		dy := circle.End.Y - circle.Center.Y
		radius := math.Sqrt(dx*dx + dy*dy)
		bbox.Expand(Position{X: circle.Center.X - radius, Y: circle.Center.Y - radius})
		bbox.Expand(Position{X: circle.Center.X + radius, Y: circle.Center.Y + radius})
	}

	for _, arc := range b.Graphics.Arcs {
		// For arcs, include start, mid, and end points
		// This is approximate but good enough for bounding box
		bbox.Expand(arc.Start)
		bbox.Expand(arc.Mid)
		bbox.Expand(arc.End)
	}

	for _, rect := range b.Graphics.Rects {
		bbox.Expand(rect.Start)
		bbox.Expand(rect.End)
	}

	for _, poly := range b.Graphics.Polys {
		for _, point := range poly.Points {
			bbox.Expand(point)
		}
	}

	for _, text := range b.Graphics.Texts {
		// For text, just include the position
		// A more accurate implementation would calculate text bounds
		bbox.Expand(text.Position)
	}

	return bbox
}

// GetBoundingBox calculates the bounding box of a footprint
// Includes all pads with their positions relative to footprint position
func (fp *Footprint) GetBoundingBox() BoundingBox {
	bbox := NewBoundingBox()

	// Transform pad positions by footprint position and rotation
	for _, pad := range fp.Pads {
		// Get absolute pad position
		absPos := fp.TransformPosition(pad.Position)

		// Expand by pad size (approximate as rectangle)
		halfWidth := pad.Size.Width / 2.0
		halfHeight := pad.Size.Height / 2.0

		bbox.Expand(Position{X: absPos.X - halfWidth, Y: absPos.Y - halfHeight})
		bbox.Expand(Position{X: absPos.X + halfWidth, Y: absPos.Y + halfHeight})
	}

	// If no pads, at least include footprint position
	if len(fp.Pads) == 0 {
		bbox.Expand(Position{X: fp.Position.X, Y: fp.Position.Y})
	}

	return bbox
}

// TransformPosition transforms a relative position by footprint position and rotation
func (fp *Footprint) TransformPosition(relPos PositionAngle) Position {
	x, y := relPos.X, relPos.Y

	// Apply footprint rotation (negate to match silkscreen coordinate system)
	if fp.Position.Angle != 0 {
		angleRad := -float64(fp.Position.Angle) * math.Pi / 180.0
		cos := math.Cos(angleRad)
		sin := math.Sin(angleRad)
		newX := x*cos - y*sin
		newY := x*sin + y*cos
		x = newX
		y = newY
	}

	// Apply translation
	x += fp.Position.X
	y += fp.Position.Y

	return Position{X: x, Y: y}
}
