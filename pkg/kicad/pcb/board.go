package pcb

// Board represents a complete KiCad PCB
type Board struct {
	Version    int         // File format version
	Generator  string      // Generator info (e.g., "pcbnew")
	General    General     // General board properties
	Layers     []Layer     // Layer definitions
	Setup      Setup       // Board setup and configuration
	Nets       []Net       // Electrical nets
	Footprints []Footprint // Component footprints
	Graphics   Graphics    // Graphical elements (lines, circles, arcs, etc.)
	Tracks     []Track     // Track segments
	Vias       []Via       // Vias
	Zones      []Zone      // Filled zones
	Groups     []Group     // Grouped elements
}

// General contains general board properties
type General struct {
	Thickness float64 // Board thickness in mm
	Title     string  // Board title
	Date      string  // Design date
	Revision  string  // Board revision
	Company   string  // Company name
}

// Setup contains board setup and default values
type Setup struct {
	Pad2MaskClearance float64 // Pad to mask clearance
	AuxAxisOrigin     Position // Auxiliary axis origin
	GridOrigin        Position // Grid origin
}

// Footprint represents a component footprint
type Footprint struct {
	Library   string        // Library name
	Name      string        // Footprint name
	Layer     string        // Layer (F.Cu or B.Cu typically)
	Position  PositionAngle // Position and rotation
	Pads      []Pad         // Pads
	Graphics  []Graphic     // Graphics (silk, fab, etc.)
	Reference string        // Reference designator (e.g., "R1")
	Value     string        // Component value
}

// Pad represents a footprint pad
type Pad struct {
	Number   string        // Pad number/name
	Type     string        // Pad type (thru_hole, smd, etc.)
	Shape    string        // Pad shape (circle, rect, oval, etc.)
	Position PositionAngle // Position and rotation
	Size     Size          // Pad size
	Drill    float64       // Drill diameter (0 for SMD)
	Layers   LayerSet      // Layers the pad appears on
	Net      *Net          // Connected net (if any)
}

// Graphic represents graphical elements
type Graphic struct {
	Type   string        // Type (line, arc, circle, rect, polygon, text)
	Layer  string        // Layer name
	Start  Position      // Start point (for lines, arcs)
	End    Position      // End point (for lines, arcs)
	Center Position      // Center point (for circles, arcs)
	Radius float64       // Radius (for circles)
	Angle  float64       // Angle (for arcs)
	Points []Position    // Points (for polygons)
	Text   string        // Text content (for text)
	Stroke Stroke        // Stroke definition
	Fill   Fill          // Fill definition
}

// Track represents a copper track segment
type Track struct {
	Start  Position // Start point
	End    Position // End point
	Width  float64  // Track width in mm
	Layer  string   // Layer name
	Net    *Net     // Connected net
	Locked bool     // Whether track is locked
}

// Via represents a via
type Via struct {
	Position Position // Via position
	Size     float64  // Via diameter
	Drill    float64  // Drill diameter
	Layers   LayerSet // Layer pair
	Net      *Net     // Connected net
	Locked   bool     // Whether via is locked
}

// Zone represents a filled copper zone
type Zone struct {
	Net            *Net       // Connected net
	Layer          string     // Layer name
	Outline        []Position // Zone outline polygon
	Fills          [][]Position // Filled polygon segments
	HatchThickness float64    // Hatch line thickness
	HatchGap       float64    // Hatch line gap
	MinThickness   float64    // Minimum thickness
}

// Group represents a logical grouping of elements
type Group struct {
	Name    string   // Group name
	Members []string // UUIDs of member elements
}

// GetNet returns a net by name, or nil if not found
func (b *Board) GetNet(name string) *Net {
	for i := range b.Nets {
		if b.Nets[i].Name == name {
			return &b.Nets[i]
		}
	}
	return nil
}

// GetNetPads returns all pads connected to a specific net
func (b *Board) GetNetPads(netName string) []Pad {
	var pads []Pad
	for _, fp := range b.Footprints {
		for _, pad := range fp.Pads {
			if pad.Net != nil && pad.Net.Name == netName {
				pads = append(pads, pad)
			}
		}
	}
	return pads
}

// GetNetTracks returns all tracks connected to a specific net
func (b *Board) GetNetTracks(netName string) []Track {
	var tracks []Track
	for _, track := range b.Tracks {
		if track.Net != nil && track.Net.Name == netName {
			tracks = append(tracks, track)
		}
	}
	return tracks
}

// GetNetVias returns all vias connected to a specific net
func (b *Board) GetNetVias(netName string) []Via {
	var vias []Via
	for _, via := range b.Vias {
		if via.Net != nil && via.Net.Name == netName {
			vias = append(vias, via)
		}
	}
	return vias
}

// NetInfo contains information about a net and its connections
type NetInfo struct {
	Net    *Net
	Pads   []Pad
	Tracks []Track
	Vias   []Via
}

// GetNetInfo returns complete information about a net
func (b *Board) GetNetInfo(netName string) *NetInfo {
	net := b.GetNet(netName)
	if net == nil {
		return nil
	}
	
	return &NetInfo{
		Net:    net,
		Pads:   b.GetNetPads(netName),
		Tracks: b.GetNetTracks(netName),
		Vias:   b.GetNetVias(netName),
	}
}

// GetAllNetNames returns a list of all net names in the board
func (b *Board) GetAllNetNames() []string {
	names := make([]string, len(b.Nets))
	for i, net := range b.Nets {
		names[i] = net.Name
	}
	return names
}
