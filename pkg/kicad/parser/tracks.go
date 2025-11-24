package parser

import (
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/parser/kicadsexp"
)

// parseSegment extracts a track segment (copper trace)
// Expected format: (segment (start x y) (end x y) (width w) (layer "layer") (net n) ...)
func parseSegment(node kicadsexp.Sexp, netMap *NetMap) (*Track, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected segment list, got leaf")
	}

	track := &Track{
		Width: 0.15, // Default width
	}

	// Parse start position
	if startNode, found := findNode(node, "start"); found {
		start, err := parsePosition(startNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start position: %w", err)
		}
		track.Start = start
	} else {
		return nil, fmt.Errorf("missing required 'start' position")
	}

	// Parse end position
	if endNode, found := findNode(node, "end"); found {
		end, err := parsePosition(endNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end position: %w", err)
		}
		track.End = end
	} else {
		return nil, fmt.Errorf("missing required 'end' position")
	}

	// Parse width
	if widthNode, found := findNode(node, "width"); found {
		width, err := getFloat(widthNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse width: %w", err)
		}
		track.Width = width
	}

	// Parse layer
	if layerNode, found := findNode(node, "layer"); found {
		layer, err := getQuotedString(layerNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layer: %w", err)
		}
		track.Layer = layer
	} else {
		return nil, fmt.Errorf("missing required 'layer' field")
	}

	// Parse net (optional - may be unconnected)
	if netNode, found := findNode(node, "net"); found {
		netNum, err := getInt(netNode, 1)
		if err == nil && netMap != nil {
			if net, ok := netMap.GetByNumber(netNum); ok {
				track.Net = net
			}
		}
	}

	// Parse locked flag (optional)
	if _, found := findNode(node, "locked"); found {
		track.Locked = true
	}

	return track, nil
}

// parseVia extracts a via definition
// Expected format: (via (at x y) (size diameter) (drill diameter) (layers "L1" "L2") (net n) ...)
func parseVia(node kicadsexp.Sexp, netMap *NetMap) (*Via, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected via list, got leaf")
	}

	via := &Via{}

	// Parse position (at x y)
	if atNode, found := findNode(node, "at"); found {
		pos, err := parsePosition(atNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse position: %w", err)
		}
		via.Position = pos
	} else {
		return nil, fmt.Errorf("missing required 'at' position")
	}

	// Parse size (via diameter)
	if sizeNode, found := findNode(node, "size"); found {
		size, err := getFloat(sizeNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse size: %w", err)
		}
		via.Size = size
	} else {
		return nil, fmt.Errorf("missing required 'size' field")
	}

	// Parse drill diameter
	if drillNode, found := findNode(node, "drill"); found {
		drill, err := getFloat(drillNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse drill: %w", err)
		}
		via.Drill = drill
	} else {
		return nil, fmt.Errorf("missing required 'drill' field")
	}

	// Parse layers (layer pair)
	if layersNode, found := findNode(node, "layers"); found {
		if layersNode.IsLeaf() {
			return nil, fmt.Errorf("expected layers list, got leaf")
		}

		// Get all layer names from the layers node
		items := getListItems(layersNode)
		var layers []string
		for _, item := range items {
			if item.IsLeaf() {
				layerName := item.String()
				// Remove quotes if present
				if len(layerName) > 0 && layerName[0] == '"' {
					layerName = layerName[1:]
				}
				if len(layerName) > 0 && layerName[len(layerName)-1] == '"' {
					layerName = layerName[:len(layerName)-1]
				}
				// Skip "layers" keyword itself
				if layerName != "layers" && layerName != "" {
					layers = append(layers, layerName)
				}
			}
		}
		via.Layers = LayerSet(layers)
	} else {
		return nil, fmt.Errorf("missing required 'layers' field")
	}

	// Parse net (optional - may be unconnected)
	if netNode, found := findNode(node, "net"); found {
		netNum, err := getInt(netNode, 1)
		if err == nil && netMap != nil {
			if net, ok := netMap.GetByNumber(netNum); ok {
				via.Net = net
			}
		}
	}

	// Parse locked flag (optional)
	if _, found := findNode(node, "locked"); found {
		via.Locked = true
	}

	return via, nil
}

// parseTracks extracts all track segments from the root node
// Finds and parses all (segment ...) nodes
func parseTracks(root kicadsexp.Sexp, netMap *NetMap) ([]Track, error) {
	if root.IsLeaf() {
		return nil, fmt.Errorf("expected root list")
	}

	// Find all (segment ...) nodes
	segmentNodes := findAllNodes(root, "segment")
	if len(segmentNodes) == 0 {
		// No tracks is valid (board might have no routing yet)
		return []Track{}, nil
	}

	var tracks []Track

	for _, segmentNode := range segmentNodes {
		track, err := parseSegment(segmentNode, netMap)
		if err != nil {
			return nil, fmt.Errorf("failed to parse segment: %w", err)
		}
		tracks = append(tracks, *track)
	}

	return tracks, nil
}

// parseVias extracts all via definitions from the root node
// Finds and parses all (via ...) nodes
func parseVias(root kicadsexp.Sexp, netMap *NetMap) ([]Via, error) {
	if root.IsLeaf() {
		return nil, fmt.Errorf("expected root list")
	}

	// Find all (via ...) nodes
	viaNodes := findAllNodes(root, "via")
	if len(viaNodes) == 0 {
		// No vias is valid
		return []Via{}, nil
	}

	var vias []Via

	for _, viaNode := range viaNodes {
		via, err := parseVia(viaNode, netMap)
		if err != nil {
			return nil, fmt.Errorf("failed to parse via: %w", err)
		}
		vias = append(vias, *via)
	}

	return vias, nil
}
