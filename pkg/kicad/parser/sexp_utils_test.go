package parser

import (
	"testing"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/parser/kicadsexp"
)

// Helper to parse s-expression from string
func parseSexp(t *testing.T, input string) kicadsexp.Sexp {
	t.Helper()
	sexps, err := kicadsexp.ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse s-expression %q: %v", input, err)
	}
	if len(sexps) == 0 {
		t.Fatalf("No s-expressions parsed from %q", input)
	}
	return sexps[0]
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		index   int
		want    string
		wantErr bool
	}{
		{
			name:  "get first element",
			input: "(layer F.Cu)",
			index: 0,
			want:  "layer",
		},
		{
			name:  "get second element",
			input: "(layer F.Cu)",
			index: 1,
			want:  "F.Cu",
		},
		{
			name:  "get third element",
			input: "(at 100 50 90)",
			index: 3,
			want:  "90",
		},
		{
			name:    "index out of bounds",
			input:   "(layer F.Cu)",
			index:   5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parseSexp(t, tt.input)
			got, err := getString(s, tt.index)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getString() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("getString() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("getString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetFloat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		index   int
		want    float64
		wantErr bool
	}{
		{
			name:  "parse simple float",
			input: "(width 0.15)",
			index: 1,
			want:  0.15,
		},
		{
			name:  "parse integer as float",
			input: "(net 42)",
			index: 1,
			want:  42.0,
		},
		{
			name:  "parse large coordinate",
			input: "(at 100000000 50000000)",
			index: 1,
			want:  100000000.0,
		},
		{
			name:    "non-numeric string",
			input:   "(layer F.Cu)",
			index:   1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parseSexp(t, tt.input)
			got, err := getFloat(s, tt.index)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getFloat() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("getFloat() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("getFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		index   int
		want    int
		wantErr bool
	}{
		{
			name:  "parse simple int",
			input: "(net 42)",
			index: 1,
			want:  42,
		},
		{
			name:  "parse layer number",
			input: "(0 F.Cu signal)",
			index: 0,
			want:  0,
		},
		{
			name:    "non-integer float",
			input:   "(width 0.15)",
			index:   1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parseSexp(t, tt.input)
			got, err := getInt(s, tt.index)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getInt() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("getInt() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("getInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPosition(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantX   float64
		wantY   float64
		wantRot float64
		wantErr bool
	}{
		{
			name:    "simple position without angle",
			input:   "(at 100000000 50000000)",
			wantX:   100.0, // Converted from nm to mm
			wantY:   50.0,
			wantRot: 0.0,
		},
		{
			name:    "position with angle",
			input:   "(at 100000000 50000000 900)",
			wantX:   100.0,
			wantY:   50.0,
			wantRot: 90.0, // Converted from decidegrees to degrees
		},
		{
			name:    "negative coordinates",
			input:   "(at -25000000 -30000000)",
			wantX:   -25.0,
			wantY:   -30.0,
			wantRot: 0.0,
		},
		{
			name:    "position with negative angle",
			input:   "(at 100000000 50000000 -450)",
			wantX:   100.0,
			wantY:   50.0,
			wantRot: -45.0,
		},
		{
			name:    "wrong keyword",
			input:   "(pos 100 50)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parseSexp(t, tt.input)
			got, err := getPosition(s)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getPosition() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("getPosition() unexpected error: %v", err)
				return
			}

			if got.X != tt.wantX {
				t.Errorf("getPosition().X = %v, want %v", got.X, tt.wantX)
			}
			if got.Y != tt.wantY {
				t.Errorf("getPosition().Y = %v, want %v", got.Y, tt.wantY)
			}
			if got.Angle != Angle(tt.wantRot) {
				t.Errorf("getPosition().Angle = %v, want %v", got.Angle, tt.wantRot)
			}
		})
	}
}

func TestGetPositionXY(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantX   float64
		wantY   float64
		wantErr bool
	}{
		{
			name:  "start coordinate",
			input: "(start 100000000 50000000)",
			wantX: 100.0,
			wantY: 50.0,
		},
		{
			name:  "end coordinate",
			input: "(end 110000000 60000000)",
			wantX: 110.0,
			wantY: 60.0,
		},
		{
			name:  "center coordinate",
			input: "(center 50000000 50000000)",
			wantX: 50.0,
			wantY: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parseSexp(t, tt.input)
			got, err := getPositionXY(s)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getPositionXY() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("getPositionXY() unexpected error: %v", err)
				return
			}

			if got.X != tt.wantX {
				t.Errorf("getPositionXY().X = %v, want %v", got.X, tt.wantX)
			}
			if got.Y != tt.wantY {
				t.Errorf("getPositionXY().Y = %v, want %v", got.Y, tt.wantY)
			}
		})
	}
}

func TestGetAngle(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		index   int
		want    Angle
		wantErr bool
	}{
		{
			name:  "90 degrees (900 decidegrees)",
			input: "(at 100 50 900)",
			index: 3,
			want:  90.0,
		},
		{
			name:  "45 degrees (450 decidegrees)",
			input: "(at 100 50 450)",
			index: 3,
			want:  45.0,
		},
		{
			name:  "negative angle",
			input: "(at 100 50 -900)",
			index: 3,
			want:  -90.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parseSexp(t, tt.input)
			got, err := getAngle(s, tt.index)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getAngle() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("getAngle() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("getAngle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindNode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		key     string
		wantKey string
		found   bool
	}{
		{
			name:    "find layer",
			input:   "(segment (start 100 50) (end 110 60) (layer F.Cu) (net 1))",
			key:     "layer",
			wantKey: "layer",
			found:   true,
		},
		{
			name:    "find start",
			input:   "(segment (start 100 50) (end 110 60) (layer F.Cu))",
			key:     "start",
			wantKey: "start",
			found:   true,
		},
		{
			name:  "key not found",
			input: "(segment (start 100 50) (end 110 60))",
			key:   "layer",
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parseSexp(t, tt.input)
			got, found := findNode(s, tt.key)

			if found != tt.found {
				t.Errorf("findNode() found = %v, want %v", found, tt.found)
				return
			}

			if !tt.found {
				return
			}

			// Verify the found node starts with the expected key
			gotKey, err := getString(got, 0)
			if err != nil {
				t.Errorf("findNode() returned node that can't extract key: %v", err)
				return
			}

			if gotKey != tt.wantKey {
				t.Errorf("findNode() returned node with key %q, want %q", gotKey, tt.wantKey)
			}
		})
	}
}

func TestGetLayers(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    LayerSet
		wantErr bool
	}{
		{
			name:  "single layer",
			input: "(layer F.Cu)",
			want:  LayerSet{"F.Cu"},
		},
		{
			name:  "multiple layers",
			input: "(layers F.Cu B.Cu F.Mask)",
			want:  LayerSet{"F.Cu", "B.Cu", "F.Mask"},
		},
		{
			name:  "all copper layers wildcard",
			input: "(layers *.Cu *.Mask)",
			want:  LayerSet{"*.Cu", "*.Mask"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parseSexp(t, tt.input)
			got, err := getLayers(s)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getLayers() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("getLayers() unexpected error: %v", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("getLayers() length = %v, want %v", len(got), len(tt.want))
				return
			}

			for i, layer := range got {
				if layer != tt.want[i] {
					t.Errorf("getLayers()[%d] = %q, want %q", i, layer, tt.want[i])
				}
			}
		})
	}
}

func TestGetColor(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantR   float64
		wantG   float64
		wantB   float64
		wantA   float64
		wantErr bool
	}{
		{
			name:  "RGB color",
			input: "(color 255 128 64)",
			wantR: 1.0,
			wantG: 128.0 / 255.0,
			wantB: 64.0 / 255.0,
			wantA: 1.0, // Default alpha
		},
		{
			name:  "RGBA color",
			input: "(color 255 128 64 200)",
			wantR: 1.0,
			wantG: 128.0 / 255.0,
			wantB: 64.0 / 255.0,
			wantA: 200.0 / 255.0,
		},
		{
			name:  "black color",
			input: "(color 0 0 0)",
			wantR: 0.0,
			wantG: 0.0,
			wantB: 0.0,
			wantA: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parseSexp(t, tt.input)
			got, err := getColor(s)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getColor() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("getColor() unexpected error: %v", err)
				return
			}

			epsilon := 0.001
			if abs(got.R-tt.wantR) > epsilon {
				t.Errorf("getColor().R = %v, want %v", got.R, tt.wantR)
			}
			if abs(got.G-tt.wantG) > epsilon {
				t.Errorf("getColor().G = %v, want %v", got.G, tt.wantG)
			}
			if abs(got.B-tt.wantB) > epsilon {
				t.Errorf("getColor().B = %v, want %v", got.B, tt.wantB)
			}
			if abs(got.A-tt.wantA) > epsilon {
				t.Errorf("getColor().A = %v, want %v", got.A, tt.wantA)
			}
		})
	}
}

func TestGetStroke(t *testing.T) {
	input := "(stroke (width 150000) (type solid))"
	s := parseSexp(t, input)

	got, err := getStroke(s)
	if err != nil {
		t.Fatalf("getStroke() unexpected error: %v", err)
	}

	// Width should be converted from nm to mm
	wantWidth := 0.15
	if abs(got.Width-wantWidth) > 0.001 {
		t.Errorf("getStroke().Width = %v, want %v", got.Width, wantWidth)
	}

	if got.Type != "solid" {
		t.Errorf("getStroke().Type = %q, want %q", got.Type, "solid")
	}
}

func TestGetFill(t *testing.T) {
	input := "(fill (type solid))"
	s := parseSexp(t, input)

	got, err := getFill(s)
	if err != nil {
		t.Fatalf("getFill() unexpected error: %v", err)
	}

	if got.Type != "solid" {
		t.Errorf("getFill().Type = %q, want %q", got.Type, "solid")
	}
}

func TestHasSymbol(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		symbol string
		want   bool
	}{
		{
			name:   "symbol present",
			input:  "(via blind (at 100 50))",
			symbol: "blind",
			want:   true,
		},
		{
			name:   "symbol not present",
			input:  "(via (at 100 50))",
			symbol: "blind",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parseSexp(t, tt.input)
			got := hasSymbol(s, tt.symbol)

			if got != tt.want {
				t.Errorf("hasSymbol() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNodeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple node",
			input: "(segment (start 100 50))",
			want:  "segment",
		},
		{
			name:  "layer node",
			input: "(layer F.Cu)",
			want:  "layer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parseSexp(t, tt.input)
			got, err := getNodeName(s)

			if err != nil {
				t.Errorf("getNodeName() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("getNodeName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Helper function for float comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
