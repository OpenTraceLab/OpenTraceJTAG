package pcb

import (
	"fmt"
	"io"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp/kicadsexp"
)

// Minimum supported KiCad version (6.0 = 20211014)
const MinSupportedVersion = 20211014

// Parser handles parsing of KiCad board files
type Parser struct {
	// Future: will contain parser state if needed
}

// NewParser creates a new KiCad board parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile reads and parses a KiCad board file
func ParseFile(filename string) (*Board, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return Parse(file)
}

// Parse reads and parses a KiCad board from an io.Reader
func Parse(r io.Reader) (*Board, error) {
	// Parse s-expressions directly from reader (streaming, no memory limit)
	sexps, err := kicadsexp.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse s-expression: %w", err)
	}

	if len(sexps) == 0 {
		return nil, fmt.Errorf("empty file or no valid s-expressions found")
	}

	// The root should be a (kicad_pcb ...) expression
	root := sexps[0]

	// Verify this is a kicad_pcb file
	rootName, err := getNodeName(root)
	if err != nil {
		return nil, fmt.Errorf("failed to get root node name: %w", err)
	}

	if rootName != "kicad_pcb" {
		return nil, fmt.Errorf("not a KiCad PCB file: expected 'kicad_pcb', got '%s'", rootName)
	}

	// Parse header (version and generator)
	version, generator, err := parseHeader(root)
	if err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Create board structure
	board := &Board{
		Version:   version,
		Generator: generator,
	}

	// Parse general section
	if generalNode, found := findNode(root, "general"); found {
		general, err := parseGeneral(generalNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse general section: %w", err)
		}
		board.General = *general
	}

	// Parse layers section
	if layersNode, found := findNode(root, "layers"); found {
		layers, err := parseLayers(layersNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layers section: %w", err)
		}
		board.Layers = layers
	}

	// Parse nets section
	nets, err := parseNets(root)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nets: %w", err)
	}
	board.Nets = nets

	// Parse graphics section
	graphics, err := parseGraphics(root)
	if err != nil {
		return nil, fmt.Errorf("failed to parse graphics: %w", err)
	}
	board.Graphics = *graphics

	// Create net map for lookups
	netMap := NewNetMap(board.Nets)

	// Parse tracks section
	tracks, err := parseTracks(root, netMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tracks: %w", err)
	}
	board.Tracks = tracks

	// Parse vias section
	vias, err := parseVias(root, netMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vias: %w", err)
	}
	board.Vias = vias

	// Parse footprints section
	footprints, err := parseFootprints(root, netMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse footprints: %w", err)
	}
	board.Footprints = footprints

	// Parse zones
	zones, err := parseZones(root, netMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse zones: %w", err)
	}
	board.Zones = zones

	return board, nil
}

// parseHeader extracts version and generator information from the root node
// Expected format: (kicad_pcb (version 20221018) (generator pcbnew) ...)
func parseHeader(root kicadsexp.Sexp) (version int, generator string, err error) {
	// Find version node
	versionNode, found := findNode(root, "version")
	if !found {
		return 0, "", fmt.Errorf("missing required 'version' field")
	}

	// Extract version number
	ver, err := getInt(versionNode, 1)
	if err != nil {
		return 0, "", fmt.Errorf("failed to parse version: %w", err)
	}

	// Validate version (must be KiCad 6.0 or later)
	if ver < MinSupportedVersion {
		return 0, "", fmt.Errorf("unsupported KiCad version: %d (minimum required: %d / KiCad 6.0)", ver, MinSupportedVersion)
	}

	// Find generator/host node (optional in some files)
	gen := "unknown"
	if hostNode, found := findNode(root, "host"); found {
		// Format: (host tool build)
		// Example: (host pcbnew "(6.0.0)")
		toolName, err := getString(hostNode, 1)
		if err == nil {
			gen = toolName
		}
	} else if genNode, found := findNode(root, "generator"); found {
		// Newer format: (generator "pcbnew")
		generatorName, err := getString(genNode, 1)
		if err == nil {
			gen = generatorName
		}
	}

	return ver, gen, nil
}

// parseGeneral extracts general board properties
// Expected format: (general (thickness 1.6) (title "Board") ...)
func parseGeneral(node kicadsexp.Sexp) (*General, error) {
	general := &General{}

	// Parse thickness (required)
	if thicknessNode, found := findNode(node, "thickness"); found {
		thickness, err := getFloat(thicknessNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse thickness: %w", err)
		}
		general.Thickness = thickness
	}

	// Parse title (optional)
	if titleNode, found := findNode(node, "title"); found {
		title, err := getQuotedString(titleNode, 1)
		if err == nil {
			general.Title = title
		}
	}

	// Parse date (optional)
	if dateNode, found := findNode(node, "date"); found {
		date, err := getQuotedString(dateNode, 1)
		if err == nil {
			general.Date = date
		}
	}

	// Parse revision (optional)
	if revNode, found := findNode(node, "rev"); found {
		rev, err := getQuotedString(revNode, 1)
		if err == nil {
			general.Revision = rev
		}
	}

	// Parse company (optional)
	if companyNode, found := findNode(node, "company"); found {
		company, err := getQuotedString(companyNode, 1)
		if err == nil {
			general.Company = company
		}
	}

	return general, nil
}

// parseLayers extracts layer definitions
// Expected format: (layers (0 "F.Cu" signal) (31 "B.Cu" signal) ...)
func parseLayers(node kicadsexp.Sexp) ([]Layer, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected (layers ...) list")
	}

	// Get all layer sub-nodes
	layerNodes := getListItems(node)
	if len(layerNodes) == 0 {
		return nil, fmt.Errorf("no layers defined")
	}

	var layers []Layer

	for _, layerNode := range layerNodes {
		if layerNode.IsLeaf() {
			continue // Skip any leaf nodes
		}

		// Parse individual layer: (number "name" type)
		// Example: (0 "F.Cu" signal)
		number, err := getInt(layerNode, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layer number: %w", err)
		}

		name, err := getQuotedString(layerNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layer name: %w", err)
		}

		layerType, err := getString(layerNode, 2)
		if err != nil {
			// Layer type is optional in some cases
			layerType = "user"
		}

		layer := Layer{
			Number: number,
			Name:   name,
			Type:   layerType,
		}

		layers = append(layers, layer)
	}

	return layers, nil
}

// parseSetup extracts board setup configuration
func parseSetup(sexp interface{}) (*Setup, error) {
	// TODO: Implement
	return nil, fmt.Errorf("not implemented")
}

// parseNets extracts net definitions from the root node
// Expected format: (net 0 "") (net 1 "GND") (net 2 "+5V") ...
// Each net is a top-level node in the board file
func parseNets(root kicadsexp.Sexp) ([]Net, error) {
	if root.IsLeaf() {
		return nil, fmt.Errorf("expected root list")
	}

	// Find all (net ...) nodes
	netNodes := findAllNodes(root, "net")
	if len(netNodes) == 0 {
		// No nets is valid (minimal boards might have no nets)
		return []Net{}, nil
	}

	var nets []Net

	for _, netNode := range netNodes {
		// Parse individual net: (net <number> "<name>")
		// Example: (net 1 "GND")

		number, err := getInt(netNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse net number: %w", err)
		}

		// Name is optional (net 0 often has empty name)
		name := ""
		nameStr, err := getQuotedString(netNode, 2)
		if err == nil {
			name = nameStr
		}

		net := Net{
			Number: number,
			Name:   name,
		}

		nets = append(nets, net)
	}

	return nets, nil
}

// parseZones extracts all zone definitions
func parseZones(root kicadsexp.Sexp, netMap *NetMap) ([]Zone, error) {
	zoneNodes := findAllNodes(root, "zone")
	zones := make([]Zone, 0, len(zoneNodes))

	for i, zoneNode := range zoneNodes {
		parsedZones, err := parseZone(zoneNode, netMap)
		if err != nil {
			fmt.Printf("[WARN] Failed to parse zone %d: %v\n", i, err)
			continue
		}
		
		// parseZone now returns a slice (for multi-layer zones)
		for _, zone := range parsedZones {
			if len(zone.Fills) == 0 {
				fmt.Printf("[WARN] Zone %d on layer %s has no fills\n", i, zone.Layer)
			} else {
				fmt.Printf("[INFO] Zone %d on layer %s has %d fills with %d points\n", i, zone.Layer, len(zone.Fills), len(zone.Fills[0]))
			}
			zones = append(zones, zone)
		}
	}

	fmt.Printf("[INFO] Parsed %d zones out of %d zone nodes\n", len(zones), len(zoneNodes))
	return zones, nil
}

// TODO: Future parsing functions:
// - parseGroups() - Extract grouped elements
