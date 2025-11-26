package schematic

import (
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp/kicadsexp"
)

// Schematic-specific parsing functions that handle coordinates in millimeters
// (not nanometers like PCB files)

// getPosition extracts position and angle from an (at X Y [angle]) node
// In schematics, X and Y are already in millimeters (not nanometers)
func getPosition(s kicadsexp.Sexp) (sexp.PositionAngle, error) {
	// Expected format: (at X Y) or (at X Y angle)
	if s.IsLeaf() {
		return sexp.PositionAngle{}, fmt.Errorf("expected (at X Y [angle]) list")
	}

	// Verify first symbol is "at"
	key, err := sexp.GetString(s, 0)
	if err != nil {
		return sexp.PositionAngle{}, err
	}
	if key != "at" {
		return sexp.PositionAngle{}, fmt.Errorf("expected 'at', got %q", key)
	}

	// Extract X and Y - already in millimeters, no conversion needed
	x, err := sexp.GetFloat(s, 1)
	if err != nil {
		return sexp.PositionAngle{}, fmt.Errorf("failed to parse X coordinate: %w", err)
	}

	y, err := sexp.GetFloat(s, 2)
	if err != nil {
		return sexp.PositionAngle{}, fmt.Errorf("failed to parse Y coordinate: %w", err)
	}

	result := sexp.PositionAngle{
		Position: sexp.Position{
			X: x, // Already in mm
			Y: y, // Already in mm
		},
	}

	// Angle is optional
	if s.LeafCount() > 3 {
		angle, err := sexp.GetAngle(s, 3)
		if err == nil {
			result.Angle = angle
		}
	}

	return result, nil
}

// getPositionXY extracts just X,Y position (for nodes without angle)
// In schematics, coordinates are already in millimeters
func getPositionXY(s kicadsexp.Sexp) (sexp.Position, error) {
	if s.IsLeaf() {
		return sexp.Position{}, fmt.Errorf("expected position list")
	}

	// Format: (keyword X Y)
	x, err := sexp.GetFloat(s, 1)
	if err != nil {
		return sexp.Position{}, fmt.Errorf("failed to parse X: %w", err)
	}

	y, err := sexp.GetFloat(s, 2)
	if err != nil {
		return sexp.Position{}, fmt.Errorf("failed to parse Y: %w", err)
	}

	return sexp.Position{
		X: x, // Already in mm
		Y: y, // Already in mm
	}, nil
}

// getSize extracts width and height from a (size W H) node
// In schematics, dimensions are already in millimeters
func getSize(s kicadsexp.Sexp) (sexp.Size, error) {
	if s.IsLeaf() {
		return sexp.Size{}, fmt.Errorf("expected size list")
	}

	w, err := sexp.GetFloat(s, 1)
	if err != nil {
		return sexp.Size{}, fmt.Errorf("failed to parse width: %w", err)
	}

	h, err := sexp.GetFloat(s, 2)
	if err != nil {
		return sexp.Size{}, fmt.Errorf("failed to parse height: %w", err)
	}

	return sexp.Size{
		Width:  w, // Already in mm
		Height: h, // Already in mm
	}, nil
}
