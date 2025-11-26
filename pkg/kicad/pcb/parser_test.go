package pcb

import (
	"strings"
	"testing"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp/kicadsexp"
)

// Test parseHeader function
func TestParseHeader(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantVersion int
		wantGen     string
		wantErr     bool
	}{
		{
			name:        "valid KiCad 6.0 with generator",
			input:       "(kicad_pcb (version 20211014) (generator pcbnew))",
			wantVersion: 20211014,
			wantGen:     "pcbnew",
			wantErr:     false,
		},
		{
			name:        "valid KiCad 6.0 with host",
			input:       "(kicad_pcb (version 20221018) (host pcbnew \"(6.0.10)\"))",
			wantVersion: 20221018,
			wantGen:     "pcbnew",
			wantErr:     false,
		},
		{
			name:        "valid KiCad 7.0",
			input:       "(kicad_pcb (version 20230314) (generator pcbnew))",
			wantVersion: 20230314,
			wantGen:     "pcbnew",
			wantErr:     false,
		},
		{
			name:    "missing version",
			input:   "(kicad_pcb (generator pcbnew))",
			wantErr: true,
		},
		{
			name:    "old version (KiCad 5)",
			input:   "(kicad_pcb (version 20171130))",
			wantErr: true,
		},
		{
			name:        "no generator (should default to unknown)",
			input:       "(kicad_pcb (version 20211014))",
			wantVersion: 20211014,
			wantGen:     "unknown",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse s-expression
			sexps, err := kicadsexp.ParseString(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse s-expression: %v", err)
			}

			version, gen, err := parseHeader(sexps[0])

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseHeader() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseHeader() unexpected error: %v", err)
				return
			}

			if version != tt.wantVersion {
				t.Errorf("parseHeader() version = %d, want %d", version, tt.wantVersion)
			}

			if gen != tt.wantGen {
				t.Errorf("parseHeader() generator = %q, want %q", gen, tt.wantGen)
			}
		})
	}
}

// Test parseGeneral function
func TestParseGeneral(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantThickness float64
		wantTitle     string
		wantDate      string
		wantRev       string
		wantCompany   string
		wantErr       bool
	}{
		{
			name: "complete general section",
			input: `(general
				(thickness 1.6)
				(title "Example Board")
				(date "2024-01-15")
				(rev "1.0")
				(company "Acme Corp")
			)`,
			wantThickness: 1.6,
			wantTitle:     "Example Board",
			wantDate:      "2024-01-15",
			wantRev:       "1.0",
			wantCompany:   "Acme Corp",
			wantErr:       false,
		},
		{
			name: "minimal general section",
			input: `(general
				(thickness 1.6)
			)`,
			wantThickness: 1.6,
			wantTitle:     "",
			wantDate:      "",
			wantRev:       "",
			wantCompany:   "",
			wantErr:       false,
		},
		{
			name: "partial general section",
			input: `(general
				(thickness 1.2)
				(title "Prototype")
				(rev "A")
			)`,
			wantThickness: 1.2,
			wantTitle:     "Prototype",
			wantDate:      "",
			wantRev:       "A",
			wantCompany:   "",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse s-expression
			sexps, err := kicadsexp.ParseString(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse s-expression: %v", err)
			}

			general, err := parseGeneral(sexps[0])

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseGeneral() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseGeneral() unexpected error: %v", err)
				return
			}

			epsilon := 0.001
			if abs(general.Thickness-tt.wantThickness) > epsilon {
				t.Errorf("parseGeneral() thickness = %v, want %v", general.Thickness, tt.wantThickness)
			}

			if general.Title != tt.wantTitle {
				t.Errorf("parseGeneral() title = %q, want %q", general.Title, tt.wantTitle)
			}

			if general.Date != tt.wantDate {
				t.Errorf("parseGeneral() date = %q, want %q", general.Date, tt.wantDate)
			}

			if general.Revision != tt.wantRev {
				t.Errorf("parseGeneral() revision = %q, want %q", general.Revision, tt.wantRev)
			}

			if general.Company != tt.wantCompany {
				t.Errorf("parseGeneral() company = %q, want %q", general.Company, tt.wantCompany)
			}
		})
	}
}

// Test Parse function with minimal board
func TestParseMinimalBoard(t *testing.T) {
	minimalBoard := `(kicad_pcb
		(version 20211014)
		(generator pcbnew)
		(general
			(thickness 1.6)
			(title "Minimal Test Board")
			(date "2024-01-15")
			(rev "1.0")
		)
	)`

	board, err := Parse(strings.NewReader(minimalBoard))
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if board == nil {
		t.Fatal("Parse() returned nil board")
	}

	// Verify version
	if board.Version != 20211014 {
		t.Errorf("Parse() version = %d, want %d", board.Version, 20211014)
	}

	// Verify generator
	if board.Generator != "pcbnew" {
		t.Errorf("Parse() generator = %q, want %q", board.Generator, "pcbnew")
	}

	// Verify general properties
	if board.General.Thickness != 1.6 {
		t.Errorf("Parse() general.thickness = %v, want %v", board.General.Thickness, 1.6)
	}

	if board.General.Title != "Minimal Test Board" {
		t.Errorf("Parse() general.title = %q, want %q", board.General.Title, "Minimal Test Board")
	}

	if board.General.Date != "2024-01-15" {
		t.Errorf("Parse() general.date = %q, want %q", board.General.Date, "2024-01-15")
	}

	if board.General.Revision != "1.0" {
		t.Errorf("Parse() general.revision = %q, want %q", board.General.Revision, "1.0")
	}
}

// Test Parse with invalid input
func TestParseInvalid(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "empty file",
			input:   "",
			wantErr: "empty file",
		},
		{
			name:    "not a kicad_pcb file",
			input:   "(kicad_sch (version 20211014))",
			wantErr: "not a KiCad PCB file",
		},
		{
			name:    "missing version",
			input:   "(kicad_pcb (generator pcbnew))",
			wantErr: "missing required 'version'",
		},
		{
			name:    "old version",
			input:   "(kicad_pcb (version 20171130))",
			wantErr: "unsupported KiCad version",
		},
		{
			name:    "invalid s-expression",
			input:   "(kicad_pcb (version",
			wantErr: "unexpected EOF", // Malformed s-expression
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.input))

			if err == nil {
				t.Errorf("Parse() expected error containing %q, got nil", tt.wantErr)
				return
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Parse() error = %q, want error containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// Test parseLayers function
func TestParseLayers(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantCount  int
		wantLayers []Layer
		wantErr    bool
	}{
		{
			name: "2-layer board",
			input: `(layers
				(0 "F.Cu" signal)
				(31 "B.Cu" signal)
			)`,
			wantCount: 2,
			wantLayers: []Layer{
				{Number: 0, Name: "F.Cu", Type: "signal"},
				{Number: 31, Name: "B.Cu", Type: "signal"},
			},
			wantErr: false,
		},
		{
			name: "4-layer board with inner layers",
			input: `(layers
				(0 "F.Cu" signal)
				(1 "In1.Cu" signal)
				(2 "In2.Cu" signal)
				(31 "B.Cu" signal)
			)`,
			wantCount: 4,
			wantLayers: []Layer{
				{Number: 0, Name: "F.Cu", Type: "signal"},
				{Number: 1, Name: "In1.Cu", Type: "signal"},
				{Number: 2, Name: "In2.Cu", Type: "signal"},
				{Number: 31, Name: "B.Cu", Type: "signal"},
			},
			wantErr: false,
		},
		{
			name: "complete layer stack with technical layers",
			input: `(layers
				(0 "F.Cu" signal)
				(31 "B.Cu" signal)
				(36 "B.SilkS" user)
				(37 "F.SilkS" user)
				(44 "Edge.Cuts" user)
			)`,
			wantCount: 5,
			wantErr:   false,
		},
		{
			name: "layer with canonical name",
			input: `(layers
				(32 "B.Adhes" user "B.Adhesive")
			)`,
			wantCount: 1,
			wantLayers: []Layer{
				{Number: 32, Name: "B.Adhes", Type: "user"},
			},
			wantErr: false,
		},
		{
			name:    "empty layers section",
			input:   `(layers)`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse s-expression
			sexps, err := kicadsexp.ParseString(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse s-expression: %v", err)
			}

			layers, err := parseLayers(sexps[0])

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseLayers() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseLayers() unexpected error: %v", err)
				return
			}

			if len(layers) != tt.wantCount {
				t.Errorf("parseLayers() layer count = %d, want %d", len(layers), tt.wantCount)
			}

			// Verify specific layers if provided
			for _, wantLayer := range tt.wantLayers {
				found := false
				for _, layer := range layers {
					if layer.Number == wantLayer.Number {
						found = true
						if layer.Name != wantLayer.Name {
							t.Errorf("Layer %d name = %q, want %q", layer.Number, layer.Name, wantLayer.Name)
						}
						if layer.Type != wantLayer.Type {
							t.Errorf("Layer %d type = %q, want %q", layer.Number, layer.Type, wantLayer.Type)
						}
						break
					}
				}
				if !found {
					t.Errorf("Layer %d (%q) not found in parsed layers", wantLayer.Number, wantLayer.Name)
				}
			}
		})
	}
}

// Test LayerMap utilities
func TestLayerMap(t *testing.T) {
	layers := []Layer{
		{Number: 0, Name: "F.Cu", Type: "signal"},
		{Number: 1, Name: "In1.Cu", Type: "signal"},
		{Number: 31, Name: "B.Cu", Type: "signal"},
		{Number: 37, Name: "F.SilkS", Type: "user"},
		{Number: 44, Name: "Edge.Cuts", Type: "user"},
	}

	lm := NewLayerMap(layers)

	t.Run("GetByName", func(t *testing.T) {
		layer, ok := lm.GetByName("F.Cu")
		if !ok {
			t.Errorf("GetByName(\"F.Cu\") not found")
		}
		if layer.Number != 0 {
			t.Errorf("GetByName(\"F.Cu\") number = %d, want 0", layer.Number)
		}

		_, ok = lm.GetByName("NonExistent")
		if ok {
			t.Errorf("GetByName(\"NonExistent\") should not be found")
		}
	})

	t.Run("GetByNumber", func(t *testing.T) {
		layer, ok := lm.GetByNumber(31)
		if !ok {
			t.Errorf("GetByNumber(31) not found")
		}
		if layer.Name != "B.Cu" {
			t.Errorf("GetByNumber(31) name = %q, want \"B.Cu\"", layer.Name)
		}

		_, ok = lm.GetByNumber(999)
		if ok {
			t.Errorf("GetByNumber(999) should not be found")
		}
	})

	t.Run("IsCopperLayer", func(t *testing.T) {
		tests := []struct {
			name   string
			want   bool
		}{
			{"F.Cu", true},
			{"In1.Cu", true},
			{"B.Cu", true},
			{"F.SilkS", false},
			{"Edge.Cuts", false},
			{"NonExistent", false},
		}

		for _, tt := range tests {
			got := lm.IsCopperLayer(tt.name)
			if got != tt.want {
				t.Errorf("IsCopperLayer(%q) = %v, want %v", tt.name, got, tt.want)
			}
		}
	})
}

// Test Parse with layers
func TestParseWithLayers(t *testing.T) {
	boardWithLayers := `(kicad_pcb
		(version 20211014)
		(generator pcbnew)
		(general
			(thickness 1.6)
		)
		(layers
			(0 "F.Cu" signal)
			(31 "B.Cu" signal)
			(37 "F.SilkS" user)
			(44 "Edge.Cuts" user)
		)
	)`

	board, err := Parse(strings.NewReader(boardWithLayers))
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if board == nil {
		t.Fatal("Parse() returned nil board")
	}

	// Verify layers were parsed
	if len(board.Layers) != 4 {
		t.Errorf("Parse() layer count = %d, want 4", len(board.Layers))
	}

	// Verify specific layers
	layerMap := NewLayerMap(board.Layers)

	fCu, ok := layerMap.GetByName("F.Cu")
	if !ok {
		t.Errorf("F.Cu layer not found")
	} else if fCu.Number != 0 {
		t.Errorf("F.Cu layer number = %d, want 0", fCu.Number)
	}

	bCu, ok := layerMap.GetByName("B.Cu")
	if !ok {
		t.Errorf("B.Cu layer not found")
	} else if bCu.Number != 31 {
		t.Errorf("B.Cu layer number = %d, want 31", bCu.Number)
	}

	// Verify copper layer detection
	if !layerMap.IsCopperLayer("F.Cu") {
		t.Errorf("F.Cu should be copper layer")
	}

	if layerMap.IsCopperLayer("Edge.Cuts") {
		t.Errorf("Edge.Cuts should not be copper layer")
	}
}

// Test parseNets function
func TestParseNets(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantNets  []Net
		wantErr   bool
	}{
		{
			name: "basic nets",
			input: `(kicad_pcb
				(net 0 "")
				(net 1 "GND")
				(net 2 "+5V")
			)`,
			wantCount: 3,
			wantNets: []Net{
				{Number: 0, Name: ""},
				{Number: 1, Name: "GND"},
				{Number: 2, Name: "+5V"},
			},
			wantErr: false,
		},
		{
			name: "nets with special characters",
			input: `(kicad_pcb
				(net 1 "/DATA")
				(net 2 "/CLK")
				(net 3 "Net-(R1-Pad1)")
			)`,
			wantCount: 3,
			wantNets: []Net{
				{Number: 1, Name: "/DATA"},
				{Number: 2, Name: "/CLK"},
				{Number: 3, Name: "Net-(R1-Pad1)"},
			},
			wantErr: false,
		},
		{
			name: "no nets",
			input: `(kicad_pcb
				(version 20211014)
			)`,
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse s-expression
			sexps, err := kicadsexp.ParseString(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse s-expression: %v", err)
			}

			nets, err := parseNets(sexps[0])

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseNets() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseNets() unexpected error: %v", err)
				return
			}

			if len(nets) != tt.wantCount {
				t.Errorf("parseNets() net count = %d, want %d", len(nets), tt.wantCount)
			}

			// Verify specific nets if provided
			for _, wantNet := range tt.wantNets {
				found := false
				for _, net := range nets {
					if net.Number == wantNet.Number {
						found = true
						if net.Name != wantNet.Name {
							t.Errorf("Net %d name = %q, want %q", net.Number, net.Name, wantNet.Name)
						}
						break
					}
				}
				if !found {
					t.Errorf("Net %d (%q) not found in parsed nets", wantNet.Number, wantNet.Name)
				}
			}
		})
	}
}

// Test NetMap utilities
func TestNetMap(t *testing.T) {
	nets := []Net{
		{Number: 0, Name: ""},
		{Number: 1, Name: "GND"},
		{Number: 2, Name: "+5V"},
		{Number: 3, Name: "+3V3"},
		{Number: 4, Name: "/DATA"},
	}

	nm := NewNetMap(nets)

	t.Run("GetByName", func(t *testing.T) {
		net, ok := nm.GetByName("GND")
		if !ok {
			t.Errorf("GetByName(\"GND\") not found")
		}
		if net.Number != 1 {
			t.Errorf("GetByName(\"GND\") number = %d, want 1", net.Number)
		}

		_, ok = nm.GetByName("NonExistent")
		if ok {
			t.Errorf("GetByName(\"NonExistent\") should not be found")
		}

		// Empty name should not be indexed
		_, ok = nm.GetByName("")
		if ok {
			t.Errorf("GetByName(\"\") should not be found (empty names not indexed)")
		}
	})

	t.Run("GetByNumber", func(t *testing.T) {
		net, ok := nm.GetByNumber(2)
		if !ok {
			t.Errorf("GetByNumber(2) not found")
		}
		if net.Name != "+5V" {
			t.Errorf("GetByNumber(2) name = %q, want \"+5V\"", net.Name)
		}

		// Net 0 should be found
		net0, ok := nm.GetByNumber(0)
		if !ok {
			t.Errorf("GetByNumber(0) not found")
		}
		if net0.Name != "" {
			t.Errorf("GetByNumber(0) name = %q, want empty", net0.Name)
		}

		_, ok = nm.GetByNumber(999)
		if ok {
			t.Errorf("GetByNumber(999) should not be found")
		}
	})

	t.Run("IsUnconnected", func(t *testing.T) {
		if !nm.IsUnconnected(0) {
			t.Errorf("IsUnconnected(0) should be true")
		}

		if nm.IsUnconnected(1) {
			t.Errorf("IsUnconnected(1) should be false")
		}
	})
}

// Test Parse with nets
func TestParseWithNets(t *testing.T) {
	boardWithNets := `(kicad_pcb
		(version 20211014)
		(generator pcbnew)
		(general
			(thickness 1.6)
		)
		(layers
			(0 "F.Cu" signal)
			(31 "B.Cu" signal)
		)
		(net 0 "")
		(net 1 "GND")
		(net 2 "+5V")
		(net 3 "+3V3")
	)`

	board, err := Parse(strings.NewReader(boardWithNets))
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if board == nil {
		t.Fatal("Parse() returned nil board")
	}

	// Verify nets were parsed
	if len(board.Nets) != 4 {
		t.Errorf("Parse() net count = %d, want 4", len(board.Nets))
	}

	// Verify specific nets
	netMap := NewNetMap(board.Nets)

	gnd, ok := netMap.GetByName("GND")
	if !ok {
		t.Errorf("GND net not found")
	} else if gnd.Number != 1 {
		t.Errorf("GND net number = %d, want 1", gnd.Number)
	}

	net2, ok := netMap.GetByNumber(2)
	if !ok {
		t.Errorf("Net 2 not found")
	} else if net2.Name != "+5V" {
		t.Errorf("Net 2 name = %q, want \"+5V\"", net2.Name)
	}

	// Verify net 0 is unconnected
	if !netMap.IsUnconnected(0) {
		t.Errorf("Net 0 should be unconnected")
	}
}

// TestParseGraphics tests parsing of graphic elements
func TestParseGraphics(t *testing.T) {
	board, err := ParseFile("../../../testdata/boards/test_with_graphics.kicad_pcb")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	t.Run("Lines", func(t *testing.T) {
		if len(board.Graphics.Lines) != 2 {
			t.Errorf("Lines count = %d, want 2", len(board.Graphics.Lines))
		}

		if len(board.Graphics.Lines) > 0 {
			line := board.Graphics.Lines[0]
			if line.Start.X != 100 || line.Start.Y != 50 {
				t.Errorf("Line start = (%v, %v), want (100, 50)", line.Start.X, line.Start.Y)
			}
			if line.End.X != 150 || line.End.Y != 50 {
				t.Errorf("Line end = (%v, %v), want (150, 50)", line.End.X, line.End.Y)
			}
			if line.Stroke.Width != 0.15 {
				t.Errorf("Line stroke width = %v, want 0.15", line.Stroke.Width)
			}
			if line.Layer != "F.Cu" {
				t.Errorf("Line layer = %q, want \"F.Cu\"", line.Layer)
			}
		}
	})

	t.Run("Circles", func(t *testing.T) {
		if len(board.Graphics.Circles) != 2 {
			t.Errorf("Circles count = %d, want 2", len(board.Graphics.Circles))
		}

		if len(board.Graphics.Circles) > 0 {
			circle := board.Graphics.Circles[0]
			if circle.Center.X != 200 || circle.Center.Y != 100 {
				t.Errorf("Circle center = (%v, %v), want (200, 100)", circle.Center.X, circle.Center.Y)
			}
			if circle.End.X != 220 || circle.End.Y != 100 {
				t.Errorf("Circle end = (%v, %v), want (220, 100)", circle.End.X, circle.End.Y)
			}
			if circle.Fill.Type != "none" {
				t.Errorf("Circle fill type = %q, want \"none\"", circle.Fill.Type)
			}
		}
	})

	t.Run("Arcs", func(t *testing.T) {
		if len(board.Graphics.Arcs) != 1 {
			t.Errorf("Arcs count = %d, want 1", len(board.Graphics.Arcs))
		}

		if len(board.Graphics.Arcs) > 0 {
			arc := board.Graphics.Arcs[0]
			if arc.Start.X != 300 || arc.Start.Y != 100 {
				t.Errorf("Arc start = (%v, %v), want (300, 100)", arc.Start.X, arc.Start.Y)
			}
			if arc.Mid.X != 320 || arc.Mid.Y != 120 {
				t.Errorf("Arc mid = (%v, %v), want (320, 120)", arc.Mid.X, arc.Mid.Y)
			}
			if arc.End.X != 340 || arc.End.Y != 100 {
				t.Errorf("Arc end = (%v, %v), want (340, 100)", arc.End.X, arc.End.Y)
			}
		}
	})

	t.Run("Rectangles", func(t *testing.T) {
		if len(board.Graphics.Rects) != 2 {
			t.Errorf("Rects count = %d, want 2", len(board.Graphics.Rects))
		}

		if len(board.Graphics.Rects) > 0 {
			rect := board.Graphics.Rects[0]
			if rect.Start.X != 400 || rect.Start.Y != 100 {
				t.Errorf("Rect start = (%v, %v), want (400, 100)", rect.Start.X, rect.Start.Y)
			}
			if rect.End.X != 450 || rect.End.Y != 150 {
				t.Errorf("Rect end = (%v, %v), want (450, 150)", rect.End.X, rect.End.Y)
			}
			if rect.Fill.Type != "none" {
				t.Errorf("Rect fill type = %q, want \"none\"", rect.Fill.Type)
			}
		}
	})

	t.Run("Polygons", func(t *testing.T) {
		if len(board.Graphics.Polys) != 1 {
			t.Errorf("Polys count = %d, want 1", len(board.Graphics.Polys))
		}

		if len(board.Graphics.Polys) > 0 {
			poly := board.Graphics.Polys[0]
			if len(poly.Points) != 3 {
				t.Errorf("Polygon points count = %d, want 3", len(poly.Points))
			}
			if len(poly.Points) >= 3 {
				if poly.Points[0].X != 500 || poly.Points[0].Y != 100 {
					t.Errorf("Polygon point 0 = (%v, %v), want (500, 100)", poly.Points[0].X, poly.Points[0].Y)
				}
				if poly.Points[1].X != 550 || poly.Points[1].Y != 100 {
					t.Errorf("Polygon point 1 = (%v, %v), want (550, 100)", poly.Points[1].X, poly.Points[1].Y)
				}
				if poly.Points[2].X != 525 || poly.Points[2].Y != 150 {
					t.Errorf("Polygon point 2 = (%v, %v), want (525, 150)", poly.Points[2].X, poly.Points[2].Y)
				}
			}
		}
	})

	t.Run("Text", func(t *testing.T) {
		if len(board.Graphics.Texts) != 2 {
			t.Errorf("Texts count = %d, want 2", len(board.Graphics.Texts))
		}

		if len(board.Graphics.Texts) > 0 {
			text := board.Graphics.Texts[0]
			if text.Text != "Test Board" {
				t.Errorf("Text content = %q, want \"Test Board\"", text.Text)
			}
			if text.Position.X != 100 || text.Position.Y != 200 {
				t.Errorf("Text position = (%v, %v), want (100, 200)", text.Position.X, text.Position.Y)
			}
			if text.Size.Width != 1.0 || text.Size.Height != 1.0 {
				t.Errorf("Text size = (%v, %v), want (1.0, 1.0)", text.Size.Width, text.Size.Height)
			}
			if text.Layer != "F.SilkS" {
				t.Errorf("Text layer = %q, want \"F.SilkS\"", text.Layer)
			}
		}

		if len(board.Graphics.Texts) > 1 {
			text := board.Graphics.Texts[1]
			if text.Text != "Rev 1.0" {
				t.Errorf("Text content = %q, want \"Rev 1.0\"", text.Text)
			}
			if !text.Bold {
				t.Errorf("Text should be bold")
			}
			if !text.Italic {
				t.Errorf("Text should be italic")
			}
			if text.Justify != "left mirror" {
				t.Errorf("Text justify = %q, want \"left mirror\"", text.Justify)
			}
		}
	})
}

// TestParseGrLine tests individual line parsing
func TestParseGrLine(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *GrLine
		wantErr bool
	}{
		{
			name:  "basic line",
			input: `(gr_line (start 10 20) (end 30 40) (stroke (width 0.15) (type solid)) (layer "F.Cu"))`,
			want: &GrLine{
				Start:  Position{X: 10, Y: 20},
				End:    Position{X: 30, Y: 40},
				Stroke: Stroke{Width: 0.15, Type: "solid"},
				Layer:  "F.Cu",
			},
			wantErr: false,
		},
		{
			name:    "missing start",
			input:   `(gr_line (end 30 40) (stroke (width 0.15) (type solid)) (layer "F.Cu"))`,
			wantErr: true,
		},
		{
			name:    "missing end",
			input:   `(gr_line (start 10 20) (stroke (width 0.15) (type solid)) (layer "F.Cu"))`,
			wantErr: true,
		},
		{
			name:    "missing layer",
			input:   `(gr_line (start 10 20) (end 30 40) (stroke (width 0.15) (type solid)))`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sexp, err := kicadsexp.ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString failed: %v", err)
			}
			if len(sexp) == 0 {
				t.Fatal("ParseString returned empty")
			}

			got, err := parseGrLine(sexp[0])
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGrLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.Start.X != tt.want.Start.X || got.Start.Y != tt.want.Start.Y {
				t.Errorf("Start = (%v, %v), want (%v, %v)", got.Start.X, got.Start.Y, tt.want.Start.X, tt.want.Start.Y)
			}
			if got.End.X != tt.want.End.X || got.End.Y != tt.want.End.Y {
				t.Errorf("End = (%v, %v), want (%v, %v)", got.End.X, got.End.Y, tt.want.End.X, tt.want.End.Y)
			}
			if got.Layer != tt.want.Layer {
				t.Errorf("Layer = %q, want %q", got.Layer, tt.want.Layer)
			}
		})
	}
}

// TestParseGrCircle tests individual circle parsing
func TestParseGrCircle(t *testing.T) {
	input := `(gr_circle (center 100 100) (end 120 100) (stroke (width 0.15) (type solid)) (fill (type none)) (layer "F.Cu"))`

	sexp, err := kicadsexp.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	if len(sexp) == 0 {
		t.Fatal("ParseString returned empty")
	}

	circle, err := parseGrCircle(sexp[0])
	if err != nil {
		t.Fatalf("parseGrCircle failed: %v", err)
	}

	if circle.Center.X != 100 || circle.Center.Y != 100 {
		t.Errorf("Center = (%v, %v), want (100, 100)", circle.Center.X, circle.Center.Y)
	}
	if circle.End.X != 120 || circle.End.Y != 100 {
		t.Errorf("End = (%v, %v), want (120, 100)", circle.End.X, circle.End.Y)
	}
	if circle.Layer != "F.Cu" {
		t.Errorf("Layer = %q, want \"F.Cu\"", circle.Layer)
	}
	if circle.Fill.Type != "none" {
		t.Errorf("Fill type = %q, want \"none\"", circle.Fill.Type)
	}
}

// TestParseGrRect tests individual rectangle parsing
func TestParseGrRect(t *testing.T) {
	input := `(gr_rect (start 100 100) (end 200 150) (stroke (width 0.15) (type solid)) (fill (type solid)) (layer "F.Cu"))`

	sexp, err := kicadsexp.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	if len(sexp) == 0 {
		t.Fatal("ParseString returned empty")
	}

	rect, err := parseGrRect(sexp[0])
	if err != nil {
		t.Fatalf("parseGrRect failed: %v", err)
	}

	if rect.Start.X != 100 || rect.Start.Y != 100 {
		t.Errorf("Start = (%v, %v), want (100, 100)", rect.Start.X, rect.Start.Y)
	}
	if rect.End.X != 200 || rect.End.Y != 150 {
		t.Errorf("End = (%v, %v), want (200, 150)", rect.End.X, rect.End.Y)
	}
	if rect.Fill.Type != "solid" {
		t.Errorf("Fill type = %q, want \"solid\"", rect.Fill.Type)
	}
}

// TestParseGrPoly tests individual polygon parsing
func TestParseGrPoly(t *testing.T) {
	input := `(gr_poly (pts (xy 10 10) (xy 20 10) (xy 15 20)) (stroke (width 0.15) (type solid)) (fill (type none)) (layer "F.Cu"))`

	sexp, err := kicadsexp.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	if len(sexp) == 0 {
		t.Fatal("ParseString returned empty")
	}

	poly, err := parseGrPoly(sexp[0])
	if err != nil {
		t.Fatalf("parseGrPoly failed: %v", err)
	}

	if len(poly.Points) != 3 {
		t.Errorf("Points count = %d, want 3", len(poly.Points))
	}
	if len(poly.Points) >= 3 {
		if poly.Points[0].X != 10 || poly.Points[0].Y != 10 {
			t.Errorf("Point 0 = (%v, %v), want (10, 10)", poly.Points[0].X, poly.Points[0].Y)
		}
		if poly.Points[1].X != 20 || poly.Points[1].Y != 10 {
			t.Errorf("Point 1 = (%v, %v), want (20, 10)", poly.Points[1].X, poly.Points[1].Y)
		}
		if poly.Points[2].X != 15 || poly.Points[2].Y != 20 {
			t.Errorf("Point 2 = (%v, %v), want (15, 20)", poly.Points[2].X, poly.Points[2].Y)
		}
	}
}

// TestParseGrText tests individual text parsing
func TestParseGrText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *GrText)
	}{
		{
			name:    "basic text",
			input:   `(gr_text "Hello" (at 100 200 0) (layer "F.SilkS") (effects (font (size 1.0 1.0) (thickness 0.15))))`,
			wantErr: false,
			check: func(t *testing.T, text *GrText) {
				if text.Text != "Hello" {
					t.Errorf("Text = %q, want \"Hello\"", text.Text)
				}
				if text.Position.X != 100 || text.Position.Y != 200 {
					t.Errorf("Position = (%v, %v), want (100, 200)", text.Position.X, text.Position.Y)
				}
			},
		},
		{
			name:    "text with formatting",
			input:   `(gr_text "Bold" (at 50 50) (layer "F.SilkS") (effects (font (size 2.0 2.0) (thickness 0.3) bold italic) (justify left)))`,
			wantErr: false,
			check: func(t *testing.T, text *GrText) {
				if !text.Bold {
					t.Errorf("Text should be bold")
				}
				if !text.Italic {
					t.Errorf("Text should be italic")
				}
				if text.Size.Width != 2.0 || text.Size.Height != 2.0 {
					t.Errorf("Size = (%v, %v), want (2.0, 2.0)", text.Size.Width, text.Size.Height)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sexp, err := kicadsexp.ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString failed: %v", err)
			}
			if len(sexp) == 0 {
				t.Fatal("ParseString returned empty")
			}

			text, err := parseGrText(sexp[0])
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGrText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if tt.check != nil {
				tt.check(t, text)
			}
		})
	}
}

// TestParseTracks tests parsing of track segments
func TestParseTracks(t *testing.T) {
	board, err := ParseFile("../../../testdata/boards/test_with_tracks.kicad_pcb")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(board.Tracks) != 4 {
		t.Errorf("Tracks count = %d, want 4", len(board.Tracks))
	}

	if len(board.Tracks) > 0 {
		track := board.Tracks[0]
		if track.Start.X != 100 || track.Start.Y != 50 {
			t.Errorf("Track 0 start = (%v, %v), want (100, 50)", track.Start.X, track.Start.Y)
		}
		if track.End.X != 120 || track.End.Y != 50 {
			t.Errorf("Track 0 end = (%v, %v), want (120, 50)", track.End.X, track.End.Y)
		}
		if track.Width != 0.25 {
			t.Errorf("Track 0 width = %v, want 0.25", track.Width)
		}
		if track.Layer != "F.Cu" {
			t.Errorf("Track 0 layer = %q, want \"F.Cu\"", track.Layer)
		}
		if track.Net == nil {
			t.Errorf("Track 0 net is nil, want net 1")
		} else if track.Net.Number != 1 {
			t.Errorf("Track 0 net = %d, want 1", track.Net.Number)
		}
		if track.Locked {
			t.Errorf("Track 0 should not be locked")
		}
	}

	// Check locked track
	if len(board.Tracks) > 2 {
		track := board.Tracks[2]
		if !track.Locked {
			t.Errorf("Track 2 should be locked")
		}
		if track.Width != 0.5 {
			t.Errorf("Track 2 width = %v, want 0.5", track.Width)
		}
	}
}

// TestParseVias tests parsing of vias
func TestParseVias(t *testing.T) {
	board, err := ParseFile("../../../testdata/boards/test_with_tracks.kicad_pcb")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(board.Vias) != 3 {
		t.Errorf("Vias count = %d, want 3", len(board.Vias))
	}

	if len(board.Vias) > 0 {
		via := board.Vias[0]
		if via.Position.X != 120 || via.Position.Y != 70 {
			t.Errorf("Via 0 position = (%v, %v), want (120, 70)", via.Position.X, via.Position.Y)
		}
		if via.Size != 0.8 {
			t.Errorf("Via 0 size = %v, want 0.8", via.Size)
		}
		if via.Drill != 0.4 {
			t.Errorf("Via 0 drill = %v, want 0.4", via.Drill)
		}
		if len(via.Layers) != 2 {
			t.Errorf("Via 0 layers count = %d, want 2", len(via.Layers))
		}
		if via.Net == nil {
			t.Errorf("Via 0 net is nil, want net 1")
		} else if via.Net.Number != 1 {
			t.Errorf("Via 0 net = %d, want 1", via.Net.Number)
		}
		if via.Locked {
			t.Errorf("Via 0 should not be locked")
		}
	}

	// Check locked via
	if len(board.Vias) > 1 {
		via := board.Vias[1]
		if !via.Locked {
			t.Errorf("Via 1 should be locked")
		}
		if via.Size != 0.6 {
			t.Errorf("Via 1 size = %v, want 0.6", via.Size)
		}
	}
}

// TestParseSegment tests individual segment parsing
func TestParseSegment(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Track)
	}{
		{
			name: "basic segment",
			input: `(segment
				(start 10 20)
				(end 30 40)
				(width 0.25)
				(layer "F.Cu")
				(net 1)
			)`,
			wantErr: false,
			check: func(t *testing.T, track *Track) {
				if track.Start.X != 10 || track.Start.Y != 20 {
					t.Errorf("Start = (%v, %v), want (10, 20)", track.Start.X, track.Start.Y)
				}
				if track.End.X != 30 || track.End.Y != 40 {
					t.Errorf("End = (%v, %v), want (30, 40)", track.End.X, track.End.Y)
				}
				if track.Width != 0.25 {
					t.Errorf("Width = %v, want 0.25", track.Width)
				}
				if track.Layer != "F.Cu" {
					t.Errorf("Layer = %q, want \"F.Cu\"", track.Layer)
				}
			},
		},
		{
			name: "missing start",
			input: `(segment
				(end 30 40)
				(width 0.25)
				(layer "F.Cu")
			)`,
			wantErr: true,
		},
		{
			name: "missing end",
			input: `(segment
				(start 10 20)
				(width 0.25)
				(layer "F.Cu")
			)`,
			wantErr: true,
		},
		{
			name: "missing layer",
			input: `(segment
				(start 10 20)
				(end 30 40)
				(width 0.25)
			)`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sexp, err := kicadsexp.ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString failed: %v", err)
			}
			if len(sexp) == 0 {
				t.Fatal("ParseString returned empty")
			}

			track, err := parseSegment(sexp[0], nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSegment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if tt.check != nil {
				tt.check(t, track)
			}
		})
	}
}

// TestParseViaUnit tests individual via parsing
func TestParseViaUnit(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Via)
	}{
		{
			name: "basic via",
			input: `(via
				(at 100 200)
				(size 0.8)
				(drill 0.4)
				(layers "F.Cu" "B.Cu")
				(net 1)
			)`,
			wantErr: false,
			check: func(t *testing.T, via *Via) {
				if via.Position.X != 100 || via.Position.Y != 200 {
					t.Errorf("Position = (%v, %v), want (100, 200)", via.Position.X, via.Position.Y)
				}
				if via.Size != 0.8 {
					t.Errorf("Size = %v, want 0.8", via.Size)
				}
				if via.Drill != 0.4 {
					t.Errorf("Drill = %v, want 0.4", via.Drill)
				}
				if len(via.Layers) != 2 {
					t.Errorf("Layers count = %d, want 2", len(via.Layers))
				}
			},
		},
		{
			name: "missing position",
			input: `(via
				(size 0.8)
				(drill 0.4)
				(layers "F.Cu" "B.Cu")
			)`,
			wantErr: true,
		},
		{
			name: "missing size",
			input: `(via
				(at 100 200)
				(drill 0.4)
				(layers "F.Cu" "B.Cu")
			)`,
			wantErr: true,
		},
		{
			name: "missing drill",
			input: `(via
				(at 100 200)
				(size 0.8)
				(layers "F.Cu" "B.Cu")
			)`,
			wantErr: true,
		},
		{
			name: "missing layers",
			input: `(via
				(at 100 200)
				(size 0.8)
				(drill 0.4)
			)`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sexp, err := kicadsexp.ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString failed: %v", err)
			}
			if len(sexp) == 0 {
				t.Fatal("ParseString returned empty")
			}

			via, err := parseVia(sexp[0], nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVia() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if tt.check != nil {
				tt.check(t, via)
			}
		})
	}
}

// TestParseFootprints tests parsing of footprints
func TestParseFootprints(t *testing.T) {
	board, err := ParseFile("../../../testdata/boards/test_with_footprints.kicad_pcb")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(board.Footprints) != 3 {
		t.Errorf("Footprints count = %d, want 3", len(board.Footprints))
	}

	// Check first footprint (resistor)
	if len(board.Footprints) > 0 {
		fp := board.Footprints[0]
		if fp.Library != "Resistor_SMD" {
			t.Errorf("Footprint 0 library = %q, want \"Resistor_SMD\"", fp.Library)
		}
		if fp.Name != "R_0603" {
			t.Errorf("Footprint 0 name = %q, want \"R_0603\"", fp.Name)
		}
		if fp.Reference != "R1" {
			t.Errorf("Footprint 0 reference = %q, want \"R1\"", fp.Reference)
		}
		if fp.Value != "10k" {
			t.Errorf("Footprint 0 value = %q, want \"10k\"", fp.Value)
		}
		if fp.Layer != "F.Cu" {
			t.Errorf("Footprint 0 layer = %q, want \"F.Cu\"", fp.Layer)
		}
		if fp.Position.X != 100 || fp.Position.Y != 50 {
			t.Errorf("Footprint 0 position = (%v, %v), want (100, 50)", fp.Position.X, fp.Position.Y)
		}
		if fp.Position.Angle != 0 {
			t.Errorf("Footprint 0 angle = %v, want 0", fp.Position.Angle)
		}
		if len(fp.Pads) != 2 {
			t.Errorf("Footprint 0 pads count = %d, want 2", len(fp.Pads))
		}
	}

	// Check second footprint (capacitor with rotation)
	if len(board.Footprints) > 1 {
		fp := board.Footprints[1]
		if fp.Reference != "C1" {
			t.Errorf("Footprint 1 reference = %q, want \"C1\"", fp.Reference)
		}
		if fp.Position.Angle != 90 {
			t.Errorf("Footprint 1 angle = %v, want 90", fp.Position.Angle)
		}
	}

	// Check third footprint (through-hole connector)
	if len(board.Footprints) > 2 {
		fp := board.Footprints[2]
		if fp.Reference != "J1" {
			t.Errorf("Footprint 2 reference = %q, want \"J1\"", fp.Reference)
		}
		if len(fp.Pads) != 2 {
			t.Errorf("Footprint 2 pads count = %d, want 2", len(fp.Pads))
		}
	}
}

// TestParsePads tests pad parsing within footprints
func TestParsePads(t *testing.T) {
	board, err := ParseFile("../../../testdata/boards/test_with_footprints.kicad_pcb")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(board.Footprints) == 0 {
		t.Fatal("No footprints found")
	}

	// Check SMD pads from first footprint
	fp := board.Footprints[0]
	if len(fp.Pads) != 2 {
		t.Fatalf("Footprint 0 pads count = %d, want 2", len(fp.Pads))
	}

	pad := fp.Pads[0]
	if pad.Number != "1" {
		t.Errorf("Pad 0 number = %q, want \"1\"", pad.Number)
	}
	if pad.Type != "smd" {
		t.Errorf("Pad 0 type = %q, want \"smd\"", pad.Type)
	}
	if pad.Shape != "rect" {
		t.Errorf("Pad 0 shape = %q, want \"rect\"", pad.Shape)
	}
	if pad.Size.Width != 0.9 || pad.Size.Height != 1.0 {
		t.Errorf("Pad 0 size = (%v, %v), want (0.9, 1.0)", pad.Size.Width, pad.Size.Height)
	}
	if pad.Position.X != -0.8 || pad.Position.Y != 0 {
		t.Errorf("Pad 0 position = (%v, %v), want (-0.8, 0)", pad.Position.X, pad.Position.Y)
	}
	if pad.Net == nil {
		t.Errorf("Pad 0 net is nil, want net 1")
	} else if pad.Net.Number != 1 {
		t.Errorf("Pad 0 net = %d, want 1", pad.Net.Number)
	}
	if pad.Drill != 0 {
		t.Errorf("Pad 0 drill = %v, want 0 (SMD pad)", pad.Drill)
	}

	// Check through-hole pads from third footprint
	if len(board.Footprints) > 2 {
		fp := board.Footprints[2]
		if len(fp.Pads) != 2 {
			t.Fatalf("Footprint 2 pads count = %d, want 2", len(fp.Pads))
		}

		pad := fp.Pads[0]
		if pad.Type != "thru_hole" {
			t.Errorf("Pad type = %q, want \"thru_hole\"", pad.Type)
		}
		if pad.Shape != "circle" {
			t.Errorf("Pad shape = %q, want \"circle\"", pad.Shape)
		}
		if pad.Drill != 1.0 {
			t.Errorf("Pad drill = %v, want 1.0", pad.Drill)
		}
	}
}

// TestParseFootprint tests individual footprint parsing
func TestParseFootprint(t *testing.T) {
	input := `(footprint "Library:Component"
		(layer "F.Cu")
		(at 50 60 45)
		(property "Reference" "U1"
			(at 0 0 0)
			(layer "F.SilkS")
			(uuid "ref-uuid")
			(effects (font (size 1 1)))
		)
		(property "Value" "IC"
			(at 0 0 0)
			(layer "F.Fab")
			(uuid "val-uuid")
			(effects (font (size 1 1)))
		)
		(pad "1" smd rect
			(at 0 0 0)
			(size 1.0 1.0)
			(layers "F.Cu" "F.Mask")
			(net 1)
		)
	)`

	sexp, err := kicadsexp.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	if len(sexp) == 0 {
		t.Fatal("ParseString returned empty")
	}

	fp, err := parseFootprint(sexp[0], nil)
	if err != nil {
		t.Fatalf("parseFootprint failed: %v", err)
	}

	if fp.Library != "Library" {
		t.Errorf("Library = %q, want \"Library\"", fp.Library)
	}
	if fp.Name != "Component" {
		t.Errorf("Name = %q, want \"Component\"", fp.Name)
	}
	if fp.Reference != "U1" {
		t.Errorf("Reference = %q, want \"U1\"", fp.Reference)
	}
	if fp.Value != "IC" {
		t.Errorf("Value = %q, want \"IC\"", fp.Value)
	}
	if fp.Layer != "F.Cu" {
		t.Errorf("Layer = %q, want \"F.Cu\"", fp.Layer)
	}
	if fp.Position.X != 50 || fp.Position.Y != 60 {
		t.Errorf("Position = (%v, %v), want (50, 60)", fp.Position.X, fp.Position.Y)
	}
	if fp.Position.Angle != 45 {
		t.Errorf("Angle = %v, want 45", fp.Position.Angle)
	}
	if len(fp.Pads) != 1 {
		t.Errorf("Pads count = %d, want 1", len(fp.Pads))
	}
}

// TestParsePadUnit tests individual pad parsing
func TestParsePadUnit(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Pad)
	}{
		{
			name: "SMD pad",
			input: `(pad "1" smd rect
				(at 1.0 2.0 90)
				(size 0.8 1.2)
				(layers "F.Cu" "F.Mask")
				(net 1)
			)`,
			wantErr: false,
			check: func(t *testing.T, pad *Pad) {
				if pad.Number != "1" {
					t.Errorf("Number = %q, want \"1\"", pad.Number)
				}
				if pad.Type != "smd" {
					t.Errorf("Type = %q, want \"smd\"", pad.Type)
				}
				if pad.Shape != "rect" {
					t.Errorf("Shape = %q, want \"rect\"", pad.Shape)
				}
				if pad.Position.X != 1.0 || pad.Position.Y != 2.0 {
					t.Errorf("Position = (%v, %v), want (1.0, 2.0)", pad.Position.X, pad.Position.Y)
				}
				if pad.Position.Angle != 90 {
					t.Errorf("Angle = %v, want 90", pad.Position.Angle)
				}
				if pad.Size.Width != 0.8 || pad.Size.Height != 1.2 {
					t.Errorf("Size = (%v, %v), want (0.8, 1.2)", pad.Size.Width, pad.Size.Height)
				}
			},
		},
		{
			name: "through-hole pad",
			input: `(pad "2" thru_hole circle
				(at 0 0)
				(size 1.7 1.7)
				(drill 1.0)
				(layers "*.Cu" "*.Mask")
			)`,
			wantErr: false,
			check: func(t *testing.T, pad *Pad) {
				if pad.Type != "thru_hole" {
					t.Errorf("Type = %q, want \"thru_hole\"", pad.Type)
				}
				if pad.Drill != 1.0 {
					t.Errorf("Drill = %v, want 1.0", pad.Drill)
				}
			},
		},
		{
			name: "missing position",
			input: `(pad "1" smd rect
				(size 1.0 1.0)
				(layers "F.Cu")
			)`,
			wantErr: true,
		},
		{
			name: "missing size",
			input: `(pad "1" smd rect
				(at 0 0)
				(layers "F.Cu")
			)`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sexp, err := kicadsexp.ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString failed: %v", err)
			}
			if len(sexp) == 0 {
				t.Fatal("ParseString returned empty")
			}

			pad, err := parsePad(sexp[0], nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePad() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if tt.check != nil {
				tt.check(t, pad)
			}
		})
	}
}
