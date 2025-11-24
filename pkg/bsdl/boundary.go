package bsdl

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var boundaryEntryRegexp = regexp.MustCompile(`(?m)(\d+)\s*\(([^)]+)\)`)

// GetBoundaryCells parses the BOUNDARY_REGISTER attribute and returns the
// ordered list of boundary scan cells. The slice is sorted by cell number (LSB
// first) to make indexing by BoundaryLength straightforward.
func (e *Entity) GetBoundaryCells() ([]BoundaryCell, error) {
	attr := e.getAttributeSpec("BOUNDARY_REGISTER")
	if attr == nil || attr.Is == nil {
		return nil, fmt.Errorf("bsdl: BOUNDARY_REGISTER attribute missing")
	}

	raw := attr.Is.GetConcatenatedString()
	matches := boundaryEntryRegexp.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("bsdl: BOUNDARY_REGISTER parse failure")
	}

	cells := make([]BoundaryCell, 0, len(matches))
	for _, m := range matches {
		idx, err := strconv.Atoi(strings.TrimSpace(m[1]))
		if err != nil {
			return nil, fmt.Errorf("bsdl: invalid boundary index %q", m[1])
		}
		fields := splitAndTrim(m[2])
		if len(fields) < 3 {
			return nil, fmt.Errorf("bsdl: boundary entry %d has <3 fields", idx)
		}

		cell := BoundaryCell{
			Number:   idx,
			CellType: fields[0],
			Port:     fields[1],
			Function: fields[2],
			Control:  -1,
			Disable:  -1,
		}
		if len(fields) >= 4 {
			cell.Safe = fields[3]
		}
		if len(fields) >= 5 {
			if v, ok := parseOptionalInt(fields[4]); ok {
				cell.Control = v
			}
		}
		if len(fields) >= 6 {
			if v, ok := parseOptionalInt(fields[5]); ok {
				cell.Disable = v
			}
		}
		if len(fields) >= 7 {
			cell.Result = fields[6]
		}

		cells = append(cells, cell)
	}

	sort.Slice(cells, func(i, j int) bool {
		return cells[i].Number < cells[j].Number
	})
	return cells, nil
}

func splitAndTrim(body string) []string {
	parts := strings.Split(body, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseOptionalInt(val string) (int, bool) {
	if val == "*" || strings.EqualFold(val, "X") {
		return -1, false
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return -1, false
	}
	return parsed, true
}

func (e *Entity) getAttributeSpec(name string) *AttributeSpec {
	for _, attr := range e.GetAttributes() {
		if attr.Spec != nil && strings.EqualFold(attr.Spec.Name, name) {
			return attr.Spec
		}
	}
	return nil
}
