package parser

import (
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/parser/kicadsexp"
)

// parseZone extracts a zone (copper fill) definition
// Returns a slice because multi-layer zones create one zone per layer
func parseZone(node kicadsexp.Sexp, netMap *NetMap) ([]Zone, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected zone list, got leaf")
	}

	baseZone := &Zone{}

	// Parse net number
	if netNode, found := findNode(node, "net"); found {
		netNum, err := getInt(netNode, 1)
		if err == nil && netMap != nil {
			if net, ok := netMap.GetByNumber(netNum); ok {
				baseZone.Net = net
			}
		}
	}

	// Parse outline polygon
	if polyNode, found := findNode(node, "polygon"); found {
		if ptsNode, found := findNode(polyNode, "pts"); found {
			points, err := parsePoints(ptsNode)
			if err == nil {
				baseZone.Outline = points
			}
		}
	}

	// Parse layer(s) - determine if single or multi-layer zone
	var zoneLayers []string
	isMultiLayer := false
	
	// Try single layer first
	if layerNode, found := findNode(node, "layer"); found {
		layer, err := getQuotedString(layerNode, 1)
		if err == nil {
			zoneLayers = append(zoneLayers, layer)
		}
	}
	
	// Try multi-layer (overrides single layer if present)
	if layersNode, found := findNode(node, "layers"); found {
		zoneLayers = nil // Clear single layer
		isMultiLayer = true
		layerItems := getListItems(layersNode)
		for _, item := range layerItems {
			if sym, ok := item.(kicadsexp.Symbol); ok {
				zoneLayers = append(zoneLayers, string(sym))
			}
		}
	}

	// Parse filled polygons
	// For multi-layer zones, each filled_polygon has its own layer attribute
	filledPolyNodes := findAllNodes(node, "filled_polygon")
	
	if isMultiLayer {
		// Multi-layer zone: group fills by layer
		fillsByLayer := make(map[string][][]Position)
		
		for _, fpNode := range filledPolyNodes {
			// Get the layer for this filled_polygon
			var fillLayer string
			if layerNode, found := findNode(fpNode, "layer"); found {
				layer, err := getQuotedString(layerNode, 1)
				if err == nil {
					fillLayer = layer
				}
			}
			
			// Parse points
			if ptsNode, found := findNode(fpNode, "pts"); found {
				points, err := parsePoints(ptsNode)
				if err == nil && fillLayer != "" {
					fillsByLayer[fillLayer] = append(fillsByLayer[fillLayer], points)
				}
			}
		}
		
		// Create one zone per layer that has fills (not all declared layers)
		zones := make([]Zone, 0)
		for layer, fills := range fillsByLayer {
			zone := *baseZone // Copy base zone
			zone.Layer = layer
			zone.Fills = fills
			zones = append(zones, zone)
		}
		return zones, nil
		
	} else {
		// Single-layer zone: all fills go to the same layer
		for _, fpNode := range filledPolyNodes {
			if ptsNode, found := findNode(fpNode, "pts"); found {
				points, err := parsePoints(ptsNode)
				if err == nil {
					baseZone.Fills = append(baseZone.Fills, points)
				}
			}
		}
		
		// Create single zone
		zones := make([]Zone, 0, len(zoneLayers))
		for _, layer := range zoneLayers {
			zone := *baseZone
			zone.Layer = layer
			zones = append(zones, zone)
		}
		return zones, nil
	}
}

// parsePoints extracts xy coordinate pairs from a pts node
func parsePoints(ptsNode kicadsexp.Sexp) ([]Position, error) {
	var points []Position

	items := getListItems(ptsNode)
	for _, item := range items {
		if item.IsLeaf() {
			continue
		}

		// Check if this is an (xy x y) node
		first, err := getString(item, 0)
		if err == nil && first == "xy" {
			x, errX := getFloat(item, 1)
			y, errY := getFloat(item, 2)
			if errX == nil && errY == nil {
				points = append(points, Position{X: x, Y: y})
			}
		}
	}

	return points, nil
}
