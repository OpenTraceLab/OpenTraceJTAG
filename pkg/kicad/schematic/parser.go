package schematic

import (
	"fmt"
	"io"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp/kicadsexp"
)

// Minimum supported KiCad version for schematics (6.0 = 20211014)
const MinSupportedVersion = 20211014

// ParseFile reads and parses a KiCad schematic file
func ParseFile(filename string) (*Schematic, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return Parse(file)
}

// Parse reads and parses a KiCad schematic from an io.Reader
func Parse(r io.Reader) (*Schematic, error) {
	// Parse s-expressions directly from reader
	sexps, err := kicadsexp.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse s-expression: %w", err)
	}

	if len(sexps) == 0 {
		return nil, fmt.Errorf("empty file or no valid s-expressions found")
	}

	// The root should be a (kicad_sch ...) expression
	root := sexps[0]

	// Verify this is a kicad_sch file
	rootName, err := sexp.GetNodeName(root)
	if err != nil {
		return nil, fmt.Errorf("failed to get root node name: %w", err)
	}

	if rootName != "kicad_sch" {
		return nil, fmt.Errorf("not a KiCad schematic file: expected 'kicad_sch', got '%s'", rootName)
	}

	// Create schematic structure
	sch := &Schematic{}

	// Parse header (version and generator)
	if err := parseHeader(root, sch); err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Parse UUID
	if uuidNode, found := sexp.FindNode(root, "uuid"); found {
		uuid, err := sexp.GetUUID(uuidNode)
		if err == nil {
			sch.UUID = uuid
		}
	}

	// Parse paper size
	if paperNode, found := sexp.FindNode(root, "paper"); found {
		paper, err := sexp.GetQuotedString(paperNode, 1)
		if err == nil {
			sch.Paper = paper
		}
	}

	// Parse title block
	if titleBlockNode, found := sexp.FindNode(root, "title_block"); found {
		sch.TitleBlock = parseTitleBlock(titleBlockNode)
	}

	// Parse lib_symbols
	if libSymbolsNode, found := sexp.FindNode(root, "lib_symbols"); found {
		sch.LibSymbols = parseLibSymbols(libSymbolsNode)
	}

	// Parse symbols
	sch.Symbols = parseSymbols(root)

	// Parse wires
	sch.Wires = parseWires(root)

	// Parse buses
	sch.Buses = parseBuses(root)

	// Parse bus entries
	sch.BusEntries = parseBusEntries(root)

	// Parse junctions
	sch.Junctions = parseJunctions(root)

	// Parse no connects
	sch.NoConnects = parseNoConnects(root)

	// Parse labels
	sch.Labels = parseLabels(root)

	// Parse global labels
	sch.GlobalLabels = parseGlobalLabels(root)

	// Parse hierarchical labels
	sch.HierLabels = parseHierLabels(root)

	// Parse sheets
	sch.Sheets = parseSheets(root)

	// Parse sheet instances
	if instancesNode, found := sexp.FindNode(root, "sheet_instances"); found {
		sch.SheetInstances = parseSheetInstances(instancesNode)
	}

	// Parse polylines
	sch.Polylines = parsePolylines(root)

	// Parse text elements
	sch.Texts = parseTexts(root)

	return sch, nil
}

// parseHeader extracts version and generator information
func parseHeader(root kicadsexp.Sexp, sch *Schematic) error {
	// Find version node
	versionNode, found := sexp.FindNode(root, "version")
	if !found {
		return fmt.Errorf("missing required 'version' field")
	}

	// Extract version number
	ver, err := sexp.GetInt(versionNode, 1)
	if err != nil {
		return fmt.Errorf("failed to parse version: %w", err)
	}

	// Validate version
	if ver < MinSupportedVersion {
		return fmt.Errorf("unsupported KiCad version: %d (minimum required: %d / KiCad 6.0)", ver, MinSupportedVersion)
	}
	sch.Version = ver

	// Find generator
	if genNode, found := sexp.FindNode(root, "generator"); found {
		gen, err := sexp.GetQuotedString(genNode, 1)
		if err == nil {
			sch.Generator = gen
		}
	}

	// Find generator version
	if genVerNode, found := sexp.FindNode(root, "generator_version"); found {
		genVer, err := sexp.GetQuotedString(genVerNode, 1)
		if err == nil {
			sch.GeneratorVer = genVer
		}
	}

	return nil
}

// parseTitleBlock extracts title block information
func parseTitleBlock(node kicadsexp.Sexp) TitleBlock {
	tb := TitleBlock{}

	if titleNode, found := sexp.FindNode(node, "title"); found {
		tb.Title, _ = sexp.GetQuotedString(titleNode, 1)
	}
	if dateNode, found := sexp.FindNode(node, "date"); found {
		tb.Date, _ = sexp.GetQuotedString(dateNode, 1)
	}
	if revNode, found := sexp.FindNode(node, "rev"); found {
		tb.Revision, _ = sexp.GetQuotedString(revNode, 1)
	}
	if companyNode, found := sexp.FindNode(node, "company"); found {
		tb.Company, _ = sexp.GetQuotedString(companyNode, 1)
	}
	// Parse comments
	commentNodes := sexp.FindAllNodes(node, "comment")
	for _, cn := range commentNodes {
		num, _ := sexp.GetInt(cn, 1)
		text, _ := sexp.GetQuotedString(cn, 2)
		switch num {
		case 1:
			tb.Comment1 = text
		case 2:
			tb.Comment2 = text
		case 3:
			tb.Comment3 = text
		case 4:
			tb.Comment4 = text
		}
	}

	return tb
}

// parseLibSymbols parses embedded library symbols
func parseLibSymbols(node kicadsexp.Sexp) []LibSymbol {
	symbolNodes := sexp.FindAllNodes(node, "symbol")
	symbols := make([]LibSymbol, 0, len(symbolNodes))

	for _, symNode := range symbolNodes {
		sym := parseLibSymbol(symNode)
		symbols = append(symbols, sym)
	}

	return symbols
}

// parseLibSymbol parses a single library symbol definition
func parseLibSymbol(node kicadsexp.Sexp) LibSymbol {
	sym := LibSymbol{
		InBom:   true,
		OnBoard: true,
	}

	// Get symbol name
	sym.Name, _ = sexp.GetQuotedString(node, 1)

	// Parse properties
	propNodes := sexp.FindAllNodes(node, "property")
	for _, pn := range propNodes {
		prop, err := sexp.GetProperty(pn)
		if err == nil {
			sym.Properties = append(sym.Properties, prop)
		}
	}

	// Parse pin_numbers visibility
	if pnNode, found := sexp.FindNode(node, "pin_numbers"); found {
		sym.PinNumbers = !sexp.HasSymbol(pnNode, "hide")
	}

	// Parse pin_names visibility
	if pnNode, found := sexp.FindNode(node, "pin_names"); found {
		sym.PinNames = !sexp.HasSymbol(pnNode, "hide")
	}

	// Parse in_bom
	if ibNode, found := sexp.FindNode(node, "in_bom"); found {
		val, _ := sexp.GetString(ibNode, 1)
		sym.InBom = val == "yes"
	}

	// Parse on_board
	if obNode, found := sexp.FindNode(node, "on_board"); found {
		val, _ := sexp.GetString(obNode, 1)
		sym.OnBoard = val == "yes"
	}

	// Parse nested symbol units (these contain the actual graphics and pins)
	unitNodes := sexp.FindAllNodes(node, "symbol")
	for _, unitNode := range unitNodes {
		unit := parseSymbolUnit(unitNode)
		sym.Units = append(sym.Units, unit)

		// Also collect graphics and pins at top level for easier access
		sym.Graphics = append(sym.Graphics, unit.Graphics...)
		sym.Pins = append(sym.Pins, unit.Pins...)
	}

	return sym
}

// parseSymbolUnit parses a nested symbol unit (contains graphics and pins)
func parseSymbolUnit(node kicadsexp.Sexp) SymbolUnit {
	unit := SymbolUnit{}

	// Get unit name
	unit.Name, _ = sexp.GetQuotedString(node, 1)

	// Parse graphics elements
	// Rectangles
	rectNodes := sexp.FindAllNodes(node, "rectangle")
	for _, rn := range rectNodes {
		graphic := parseRectangle(rn)
		unit.Graphics = append(unit.Graphics, graphic)
	}

	// Circles
	circleNodes := sexp.FindAllNodes(node, "circle")
	for _, cn := range circleNodes {
		graphic := parseCircle(cn)
		unit.Graphics = append(unit.Graphics, graphic)
	}

	// Arcs
	arcNodes := sexp.FindAllNodes(node, "arc")
	for _, an := range arcNodes {
		graphic := parseArc(an)
		unit.Graphics = append(unit.Graphics, graphic)
	}

	// Polylines
	polyNodes := sexp.FindAllNodes(node, "polyline")
	for _, pn := range polyNodes {
		graphic := parseGraphicPolyline(pn)
		unit.Graphics = append(unit.Graphics, graphic)
	}

	// Parse pins
	pinNodes := sexp.FindAllNodes(node, "pin")
	for _, pn := range pinNodes {
		pin := parsePin(pn)
		unit.Pins = append(unit.Pins, pin)
	}

	return unit
}

// parsePin parses a pin definition
func parsePin(node kicadsexp.Sexp) Pin {
	pin := Pin{}

	// Pin type (input, output, etc.)
	pin.Type, _ = sexp.GetString(node, 1)

	// Pin style (line, inverted, etc.)
	pin.Style, _ = sexp.GetString(node, 2)

	// Position
	if atNode, found := sexp.FindNode(node, "at"); found {
		pos, _ := getPosition(atNode)
		pin.Position = pos.Position
		pin.Angle = pos.Angle
	}

	// Length
	if lenNode, found := sexp.FindNode(node, "length"); found {
		pin.Length, _ = sexp.GetFloat(lenNode, 1)
	}

	// Name
	if nameNode, found := sexp.FindNode(node, "name"); found {
		pin.Name.Name, _ = sexp.GetQuotedString(nameNode, 1)
		if effectsNode, found := sexp.FindNode(nameNode, "effects"); found {
			pin.Name.Effects, _ = sexp.GetEffects(effectsNode)
		}
	}

	// Number
	if numNode, found := sexp.FindNode(node, "number"); found {
		pin.Number.Number, _ = sexp.GetQuotedString(numNode, 1)
		if effectsNode, found := sexp.FindNode(numNode, "effects"); found {
			pin.Number.Effects, _ = sexp.GetEffects(effectsNode)
		}
	}

	// Hide
	pin.Hide = sexp.HasSymbol(node, "hide")

	return pin
}

// parseSymbols parses symbol instances
func parseSymbols(root kicadsexp.Sexp) []Symbol {
	symbolNodes := sexp.FindAllNodes(root, "symbol")
	symbols := make([]Symbol, 0, len(symbolNodes))

	for _, symNode := range symbolNodes {
		sym := parseSymbol(symNode)
		symbols = append(symbols, sym)
	}

	return symbols
}

// parseSymbol parses a single symbol instance
func parseSymbol(node kicadsexp.Sexp) Symbol {
	sym := Symbol{
		InBom:   true,
		OnBoard: true,
		Unit:    1,
	}

	// Get lib_id
	if libNode, found := sexp.FindNode(node, "lib_id"); found {
		sym.LibID, _ = sexp.GetQuotedString(libNode, 1)
	}

	// Position
	if atNode, found := sexp.FindNode(node, "at"); found {
		pos, _ := getPosition(atNode)
		sym.Position = pos.Position
		sym.Angle = pos.Angle
	}

	// Mirror
	if mirrorNode, found := sexp.FindNode(node, "mirror"); found {
		sym.Mirror, _ = sexp.GetString(mirrorNode, 1)
	}

	// Unit
	if unitNode, found := sexp.FindNode(node, "unit"); found {
		sym.Unit, _ = sexp.GetInt(unitNode, 1)
	}

	// UUID
	if uuidNode, found := sexp.FindNode(node, "uuid"); found {
		sym.UUID, _ = sexp.GetUUID(uuidNode)
	}

	// Properties
	propNodes := sexp.FindAllNodes(node, "property")
	for _, pn := range propNodes {
		prop, err := sexp.GetProperty(pn)
		if err == nil {
			sym.Properties = append(sym.Properties, prop)
		}
	}

	// Pin references
	pinNodes := sexp.FindAllNodes(node, "pin")
	for _, pn := range pinNodes {
		ref := PinRef{}
		ref.Number, _ = sexp.GetQuotedString(pn, 1)
		if uuidNode, found := sexp.FindNode(pn, "uuid"); found {
			ref.UUID, _ = sexp.GetUUID(uuidNode)
		}
		sym.Pins = append(sym.Pins, ref)
	}

	return sym
}

// parseRectangle parses a rectangle graphic element
func parseRectangle(node kicadsexp.Sexp) SymGraphic {
	graphic := SymGraphic{Type: "rectangle"}

	if startNode, found := sexp.FindNode(node, "start"); found {
		graphic.Start, _ = getPositionXY(startNode)
	}
	if endNode, found := sexp.FindNode(node, "end"); found {
		graphic.End, _ = getPositionXY(endNode)
	}
	if strokeNode, found := sexp.FindNode(node, "stroke"); found {
		graphic.Stroke, _ = sexp.GetStroke(strokeNode)
	}
	if fillNode, found := sexp.FindNode(node, "fill"); found {
		graphic.Fill, _ = sexp.GetFill(fillNode)
	}

	return graphic
}

// parseCircle parses a circle graphic element
func parseCircle(node kicadsexp.Sexp) SymGraphic {
	graphic := SymGraphic{Type: "circle"}

	if centerNode, found := sexp.FindNode(node, "center"); found {
		graphic.Center, _ = getPositionXY(centerNode)
	}
	if radiusNode, found := sexp.FindNode(node, "radius"); found {
		graphic.Radius, _ = sexp.GetFloat(radiusNode, 1)
	}
	if strokeNode, found := sexp.FindNode(node, "stroke"); found {
		graphic.Stroke, _ = sexp.GetStroke(strokeNode)
	}
	if fillNode, found := sexp.FindNode(node, "fill"); found {
		graphic.Fill, _ = sexp.GetFill(fillNode)
	}

	return graphic
}

// parseArc parses an arc graphic element
func parseArc(node kicadsexp.Sexp) SymGraphic {
	graphic := SymGraphic{Type: "arc"}

	if startNode, found := sexp.FindNode(node, "start"); found {
		graphic.Start, _ = getPositionXY(startNode)
	}
	if midNode, found := sexp.FindNode(node, "mid"); found {
		// Mid point can be used to calculate arc parameters
		// For now, store in Center as approximation
		graphic.Center, _ = getPositionXY(midNode)
	}
	if endNode, found := sexp.FindNode(node, "end"); found {
		graphic.End, _ = getPositionXY(endNode)
	}
	if strokeNode, found := sexp.FindNode(node, "stroke"); found {
		graphic.Stroke, _ = sexp.GetStroke(strokeNode)
	}
	if fillNode, found := sexp.FindNode(node, "fill"); found {
		graphic.Fill, _ = sexp.GetFill(fillNode)
	}

	return graphic
}

// parseGraphicPolyline parses a polyline graphic element
func parseGraphicPolyline(node kicadsexp.Sexp) SymGraphic {
	graphic := SymGraphic{Type: "polyline"}

	if ptsNode, found := sexp.FindNode(node, "pts"); found {
		// Parse all xy points
		xyNodes := sexp.FindAllNodes(ptsNode, "xy")
		for _, xy := range xyNodes {
			pos, err := getPositionXY(xy)
			if err == nil {
				graphic.Points = append(graphic.Points, pos)
			}
		}
	}
	if strokeNode, found := sexp.FindNode(node, "stroke"); found {
		graphic.Stroke, _ = sexp.GetStroke(strokeNode)
	}
	if fillNode, found := sexp.FindNode(node, "fill"); found {
		graphic.Fill, _ = sexp.GetFill(fillNode)
	}

	return graphic
}

// parseWires parses wire connections
func parseWires(root kicadsexp.Sexp) []Wire {
	wireNodes := sexp.FindAllNodes(root, "wire")
	wires := make([]Wire, 0, len(wireNodes))

	for _, wn := range wireNodes {
		wire := Wire{}

		// Parse points
		if ptsNode, found := sexp.FindNode(wn, "pts"); found {
			xyNodes := sexp.FindAllNodes(ptsNode, "xy")
			for _, xy := range xyNodes {
				pos, _ := getPositionXY(xy)
				wire.Points = append(wire.Points, pos)
			}
		}

		// Stroke
		if strokeNode, found := sexp.FindNode(wn, "stroke"); found {
			wire.Stroke, _ = sexp.GetStroke(strokeNode)
		}

		// UUID
		if uuidNode, found := sexp.FindNode(wn, "uuid"); found {
			wire.UUID, _ = sexp.GetUUID(uuidNode)
		}

		wires = append(wires, wire)
	}

	return wires
}

// parseBuses parses bus connections
func parseBuses(root kicadsexp.Sexp) []Bus {
	busNodes := sexp.FindAllNodes(root, "bus")
	buses := make([]Bus, 0, len(busNodes))

	for _, bn := range busNodes {
		bus := Bus{}

		// Parse points
		if ptsNode, found := sexp.FindNode(bn, "pts"); found {
			xyNodes := sexp.FindAllNodes(ptsNode, "xy")
			for _, xy := range xyNodes {
				pos, _ := getPositionXY(xy)
				bus.Points = append(bus.Points, pos)
			}
		}

		// Stroke
		if strokeNode, found := sexp.FindNode(bn, "stroke"); found {
			bus.Stroke, _ = sexp.GetStroke(strokeNode)
		}

		// UUID
		if uuidNode, found := sexp.FindNode(bn, "uuid"); found {
			bus.UUID, _ = sexp.GetUUID(uuidNode)
		}

		buses = append(buses, bus)
	}

	return buses
}

// parseBusEntries parses bus entry points
func parseBusEntries(root kicadsexp.Sexp) []BusEntry {
	entryNodes := sexp.FindAllNodes(root, "bus_entry")
	entries := make([]BusEntry, 0, len(entryNodes))

	for _, en := range entryNodes {
		entry := BusEntry{}

		// Position
		if atNode, found := sexp.FindNode(en, "at"); found {
			pos, _ := getPosition(atNode)
			entry.Position = pos.Position
		}

		// Size
		if sizeNode, found := sexp.FindNode(en, "size"); found {
			w, _ := sexp.GetFloat(sizeNode, 1)
			h, _ := sexp.GetFloat(sizeNode, 2)
			entry.Size = Size{Width: w, Height: h}
		}

		// Stroke
		if strokeNode, found := sexp.FindNode(en, "stroke"); found {
			entry.Stroke, _ = sexp.GetStroke(strokeNode)
		}

		// UUID
		if uuidNode, found := sexp.FindNode(en, "uuid"); found {
			entry.UUID, _ = sexp.GetUUID(uuidNode)
		}

		entries = append(entries, entry)
	}

	return entries
}

// parseJunctions parses wire junctions
func parseJunctions(root kicadsexp.Sexp) []Junction {
	juncNodes := sexp.FindAllNodes(root, "junction")
	junctions := make([]Junction, 0, len(juncNodes))

	for _, jn := range juncNodes {
		junc := Junction{}

		// Position
		if atNode, found := sexp.FindNode(jn, "at"); found {
			pos, _ := getPosition(atNode)
			junc.Position = pos.Position
		}

		// Diameter
		if diamNode, found := sexp.FindNode(jn, "diameter"); found {
			junc.Diameter, _ = sexp.GetFloat(diamNode, 1)
		}

		// Color
		if colorNode, found := sexp.FindNode(jn, "color"); found {
			junc.Color, _ = sexp.GetColor(colorNode)
		}

		// UUID
		if uuidNode, found := sexp.FindNode(jn, "uuid"); found {
			junc.UUID, _ = sexp.GetUUID(uuidNode)
		}

		junctions = append(junctions, junc)
	}

	return junctions
}

// parseNoConnects parses no-connect markers
func parseNoConnects(root kicadsexp.Sexp) []NoConnect {
	ncNodes := sexp.FindAllNodes(root, "no_connect")
	ncs := make([]NoConnect, 0, len(ncNodes))

	for _, ncn := range ncNodes {
		nc := NoConnect{}

		// Position
		if atNode, found := sexp.FindNode(ncn, "at"); found {
			pos, _ := getPosition(atNode)
			nc.Position = pos.Position
		}

		// UUID
		if uuidNode, found := sexp.FindNode(ncn, "uuid"); found {
			nc.UUID, _ = sexp.GetUUID(uuidNode)
		}

		ncs = append(ncs, nc)
	}

	return ncs
}

// parseLabels parses local wire labels
func parseLabels(root kicadsexp.Sexp) []Label {
	labelNodes := sexp.FindAllNodes(root, "label")
	labels := make([]Label, 0, len(labelNodes))

	for _, ln := range labelNodes {
		label := Label{}

		// Text
		label.Text, _ = sexp.GetQuotedString(ln, 1)

		// Position
		if atNode, found := sexp.FindNode(ln, "at"); found {
			pos, _ := getPosition(atNode)
			label.Position = pos.Position
			label.Angle = pos.Angle
		}

		// Effects
		if effectsNode, found := sexp.FindNode(ln, "effects"); found {
			label.Effects, _ = sexp.GetEffects(effectsNode)
		}

		// UUID
		if uuidNode, found := sexp.FindNode(ln, "uuid"); found {
			label.UUID, _ = sexp.GetUUID(uuidNode)
		}

		labels = append(labels, label)
	}

	return labels
}

// parseGlobalLabels parses global labels
func parseGlobalLabels(root kicadsexp.Sexp) []GlobalLabel {
	labelNodes := sexp.FindAllNodes(root, "global_label")
	labels := make([]GlobalLabel, 0, len(labelNodes))

	for _, ln := range labelNodes {
		label := GlobalLabel{}

		// Text
		label.Text, _ = sexp.GetQuotedString(ln, 1)

		// Shape
		if shapeNode, found := sexp.FindNode(ln, "shape"); found {
			label.Shape, _ = sexp.GetString(shapeNode, 1)
		}

		// Position
		if atNode, found := sexp.FindNode(ln, "at"); found {
			pos, _ := getPosition(atNode)
			label.Position = pos.Position
			label.Angle = pos.Angle
		}

		// Effects
		if effectsNode, found := sexp.FindNode(ln, "effects"); found {
			label.Effects, _ = sexp.GetEffects(effectsNode)
		}

		// UUID
		if uuidNode, found := sexp.FindNode(ln, "uuid"); found {
			label.UUID, _ = sexp.GetUUID(uuidNode)
		}

		// Properties
		propNodes := sexp.FindAllNodes(ln, "property")
		for _, pn := range propNodes {
			prop, err := sexp.GetProperty(pn)
			if err == nil {
				label.Properties = append(label.Properties, prop)
			}
		}

		labels = append(labels, label)
	}

	return labels
}

// parseHierLabels parses hierarchical labels
func parseHierLabels(root kicadsexp.Sexp) []HierLabel {
	labelNodes := sexp.FindAllNodes(root, "hierarchical_label")
	labels := make([]HierLabel, 0, len(labelNodes))

	for _, ln := range labelNodes {
		label := HierLabel{}

		// Text
		label.Text, _ = sexp.GetQuotedString(ln, 1)

		// Shape
		if shapeNode, found := sexp.FindNode(ln, "shape"); found {
			label.Shape, _ = sexp.GetString(shapeNode, 1)
		}

		// Position
		if atNode, found := sexp.FindNode(ln, "at"); found {
			pos, _ := getPosition(atNode)
			label.Position = pos.Position
			label.Angle = pos.Angle
		}

		// Effects
		if effectsNode, found := sexp.FindNode(ln, "effects"); found {
			label.Effects, _ = sexp.GetEffects(effectsNode)
		}

		// UUID
		if uuidNode, found := sexp.FindNode(ln, "uuid"); found {
			label.UUID, _ = sexp.GetUUID(uuidNode)
		}

		labels = append(labels, label)
	}

	return labels
}

// parseSheets parses hierarchical sheet references
func parseSheets(root kicadsexp.Sexp) []Sheet {
	sheetNodes := sexp.FindAllNodes(root, "sheet")
	sheets := make([]Sheet, 0, len(sheetNodes))

	for _, sn := range sheetNodes {
		sheet := Sheet{}

		// Position
		if atNode, found := sexp.FindNode(sn, "at"); found {
			pos, _ := getPosition(atNode)
			sheet.Position = pos.Position
		}

		// Size
		if sizeNode, found := sexp.FindNode(sn, "size"); found {
			w, _ := sexp.GetFloat(sizeNode, 1)
			h, _ := sexp.GetFloat(sizeNode, 2)
			sheet.Size = Size{Width: w, Height: h}
		}

		// Stroke
		if strokeNode, found := sexp.FindNode(sn, "stroke"); found {
			sheet.Stroke, _ = sexp.GetStroke(strokeNode)
		}

		// Fill
		if fillNode, found := sexp.FindNode(sn, "fill"); found {
			sheet.Fill, _ = sexp.GetFill(fillNode)
		}

		// UUID
		if uuidNode, found := sexp.FindNode(sn, "uuid"); found {
			sheet.UUID, _ = sexp.GetUUID(uuidNode)
		}

		// Properties
		propNodes := sexp.FindAllNodes(sn, "property")
		for _, pn := range propNodes {
			prop, err := sexp.GetProperty(pn)
			if err == nil {
				// Check for special properties
				if prop.Key == "Sheetname" {
					sheet.Name = SheetName{Name: prop.Value, Effects: prop.Effects}
				} else if prop.Key == "Sheetfile" {
					sheet.FileName = SheetFileName{Name: prop.Value, Effects: prop.Effects}
				} else {
					sheet.Properties = append(sheet.Properties, prop)
				}
			}
		}

		// Pins
		pinNodes := sexp.FindAllNodes(sn, "pin")
		for _, pn := range pinNodes {
			pin := SheetPin{}
			pin.Name, _ = sexp.GetQuotedString(pn, 1)
			pin.Shape, _ = sexp.GetString(pn, 2)

			if atNode, found := sexp.FindNode(pn, "at"); found {
				pos, _ := getPosition(atNode)
				pin.Position = pos.Position
			}
			if effectsNode, found := sexp.FindNode(pn, "effects"); found {
				pin.Effects, _ = sexp.GetEffects(effectsNode)
			}
			if uuidNode, found := sexp.FindNode(pn, "uuid"); found {
				pin.UUID, _ = sexp.GetUUID(uuidNode)
			}

			sheet.Pins = append(sheet.Pins, pin)
		}

		sheets = append(sheets, sheet)
	}

	return sheets
}

// parseSheetInstances parses sheet instance paths
func parseSheetInstances(node kicadsexp.Sexp) []SheetInstance {
	pathNodes := sexp.FindAllNodes(node, "path")
	instances := make([]SheetInstance, 0, len(pathNodes))

	for _, pn := range pathNodes {
		inst := SheetInstance{}
		inst.Path, _ = sexp.GetQuotedString(pn, 1)

		if pageNode, found := sexp.FindNode(pn, "page"); found {
			inst.Page, _ = sexp.GetQuotedString(pageNode, 1)
		}

		instances = append(instances, inst)
	}

	return instances
}

// parsePolylines parses graphical polylines
func parsePolylines(root kicadsexp.Sexp) []Polyline {
	polyNodes := sexp.FindAllNodes(root, "polyline")
	polys := make([]Polyline, 0, len(polyNodes))

	for _, pn := range polyNodes {
		poly := Polyline{}

		// Parse points
		if ptsNode, found := sexp.FindNode(pn, "pts"); found {
			xyNodes := sexp.FindAllNodes(ptsNode, "xy")
			for _, xy := range xyNodes {
				pos, _ := getPositionXY(xy)
				poly.Points = append(poly.Points, pos)
			}
		}

		// Stroke
		if strokeNode, found := sexp.FindNode(pn, "stroke"); found {
			poly.Stroke, _ = sexp.GetStroke(strokeNode)
		}

		// UUID
		if uuidNode, found := sexp.FindNode(pn, "uuid"); found {
			poly.UUID, _ = sexp.GetUUID(uuidNode)
		}

		polys = append(polys, poly)
	}

	return polys
}

// parseTexts parses graphical text elements
func parseTexts(root kicadsexp.Sexp) []Text {
	textNodes := sexp.FindAllNodes(root, "text")
	texts := make([]Text, 0, len(textNodes))

	for _, tn := range textNodes {
		text := Text{}

		// Text content
		text.Text, _ = sexp.GetQuotedString(tn, 1)

		// Position
		if atNode, found := sexp.FindNode(tn, "at"); found {
			pos, _ := getPosition(atNode)
			text.Position = pos.Position
			text.Angle = pos.Angle
		}

		// Effects
		if effectsNode, found := sexp.FindNode(tn, "effects"); found {
			text.Effects, _ = sexp.GetEffects(effectsNode)
		}

		// UUID
		if uuidNode, found := sexp.FindNode(tn, "uuid"); found {
			text.UUID, _ = sexp.GetUUID(uuidNode)
		}

		texts = append(texts, text)
	}

	return texts
}
