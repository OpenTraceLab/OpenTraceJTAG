// sch-info queries information from KiCad schematic files
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: sch-info <schematic.kicad_sch> [component]")
		fmt.Println("")
		fmt.Println("Without component argument: shows schematic summary")
		fmt.Println("With component argument: shows details for that component")
		os.Exit(1)
	}

	filename := os.Args[1]
	sch, err := schematic.ParseFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing schematic: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) >= 3 {
		// Show details for specific component
		showComponentDetails(sch, os.Args[2])
	} else {
		// Show summary
		showSummary(sch, filename)
	}
}

func showSummary(sch *schematic.Schematic, filename string) {
	fmt.Printf("Schematic: %s\n", filename)
	fmt.Printf("Version: %d\n", sch.Version)
	fmt.Printf("Generator: %s", sch.Generator)
	if sch.GeneratorVer != "" {
		fmt.Printf(" v%s", sch.GeneratorVer)
	}
	fmt.Println()
	fmt.Printf("Paper: %s\n", sch.Paper)
	fmt.Println()

	// Title block
	if sch.TitleBlock.Title != "" || sch.TitleBlock.Revision != "" {
		fmt.Println("Title Block:")
		if sch.TitleBlock.Title != "" {
			fmt.Printf("  Title: %s\n", sch.TitleBlock.Title)
		}
		if sch.TitleBlock.Date != "" {
			fmt.Printf("  Date: %s\n", sch.TitleBlock.Date)
		}
		if sch.TitleBlock.Revision != "" {
			fmt.Printf("  Revision: %s\n", sch.TitleBlock.Revision)
		}
		if sch.TitleBlock.Company != "" {
			fmt.Printf("  Company: %s\n", sch.TitleBlock.Company)
		}
		fmt.Println()
	}

	// Statistics
	fmt.Println("Statistics:")
	fmt.Printf("  Components: %d\n", len(sch.Symbols))
	fmt.Printf("  Library symbols: %d\n", len(sch.LibSymbols))
	fmt.Printf("  Wires: %d\n", len(sch.Wires))
	fmt.Printf("  Buses: %d\n", len(sch.Buses))
	fmt.Printf("  Junctions: %d\n", len(sch.Junctions))
	fmt.Printf("  Labels: %d\n", len(sch.Labels))
	fmt.Printf("  Global labels: %d\n", len(sch.GlobalLabels))
	fmt.Printf("  Hierarchical labels: %d\n", len(sch.HierLabels))
	fmt.Printf("  Sheets: %d\n", len(sch.Sheets))
	fmt.Printf("  No-connects: %d\n", len(sch.NoConnects))
	fmt.Println()

	// Component list
	if len(sch.Symbols) > 0 {
		fmt.Println("Components:")

		// Group by reference prefix
		byPrefix := make(map[string][]string)
		for _, sym := range sch.Symbols {
			ref := getProperty(sym.Properties, "Reference")
			if ref != "" {
				prefix := getRefPrefix(ref)
				byPrefix[prefix] = append(byPrefix[prefix], ref)
			}
		}

		// Sort prefixes
		var prefixes []string
		for p := range byPrefix {
			prefixes = append(prefixes, p)
		}
		sort.Strings(prefixes)

		for _, prefix := range prefixes {
			refs := byPrefix[prefix]
			sort.Strings(refs)
			fmt.Printf("  %s: %s\n", prefix, strings.Join(refs, ", "))
		}
		fmt.Println()
	}

	// Labels
	labels := sch.GetLabels()
	if len(labels) > 0 {
		fmt.Println("Net Labels:")
		sort.Strings(labels)
		for _, l := range labels {
			fmt.Printf("  %s\n", l)
		}
		fmt.Println()
	}

	// Hierarchical sheets
	if len(sch.Sheets) > 0 {
		fmt.Println("Hierarchical Sheets:")
		for _, sheet := range sch.Sheets {
			fmt.Printf("  %s (%s)\n", sheet.Name.Name, sheet.FileName.Name)
			if len(sheet.Pins) > 0 {
				fmt.Printf("    Pins: ")
				var pinNames []string
				for _, p := range sheet.Pins {
					pinNames = append(pinNames, p.Name)
				}
				fmt.Printf("%s\n", strings.Join(pinNames, ", "))
			}
		}
	}
}

func showComponentDetails(sch *schematic.Schematic, ref string) {
	sym := sch.GetSymbol(ref)
	if sym == nil {
		fmt.Fprintf(os.Stderr, "Component '%s' not found\n", ref)
		os.Exit(1)
	}

	fmt.Printf("Component: %s\n", ref)
	fmt.Printf("Library: %s\n", sym.LibID)
	fmt.Printf("Position: (%.2f, %.2f)\n", sym.Position.X, sym.Position.Y)
	if sym.Angle != 0 {
		fmt.Printf("Rotation: %.1f°\n", sym.Angle)
	}
	if sym.Mirror != "" {
		fmt.Printf("Mirror: %s\n", sym.Mirror)
	}
	fmt.Printf("Unit: %d\n", sym.Unit)
	fmt.Println()

	// Properties
	if len(sym.Properties) > 0 {
		fmt.Println("Properties:")
		for _, prop := range sym.Properties {
			fmt.Printf("  %s: %s\n", prop.Key, prop.Value)
		}
		fmt.Println()
	}

	// Find library symbol for pin info
	var libSym *schematic.LibSymbol
	for i := range sch.LibSymbols {
		if sch.LibSymbols[i].Name == sym.LibID {
			libSym = &sch.LibSymbols[i]
			break
		}
	}

	if libSym != nil && len(libSym.Pins) > 0 {
		fmt.Println("Pins:")
		for _, pin := range libSym.Pins {
			fmt.Printf("  %s (%s): %s %s\n", pin.Number.Number, pin.Name.Name, pin.Type, pin.Style)
		}
	}
}

func getProperty(props []schematic.Property, key string) string {
	for _, p := range props {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

func getRefPrefix(ref string) string {
	// Extract prefix (letters before numbers)
	for i, c := range ref {
		if c >= '0' && c <= '9' {
			return ref[:i]
		}
	}
	return ref
}
