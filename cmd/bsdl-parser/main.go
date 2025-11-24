package main

import (
	"fmt"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <bsdl-file>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]

	// Create parser
	parser, err := bsdl.NewParser()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating parser: %v\n", err)
		os.Exit(1)
	}

	// Parse file
	file, err := parser.ParseFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Printf("Successfully parsed BSDL file: %s\n\n", filename)

	entity := file.Entity
	fmt.Printf("Entity Name: %s\n", entity.Name)

	// Generic clause
	if entity.Generic != nil {
		fmt.Printf("\nGeneric Parameters:\n")
		for _, gen := range entity.Generic.Generics {
			fmt.Printf("  %s : %s", gen.Name, gen.Type)
			if gen.DefaultValue != nil {
				fmt.Printf(" := %s", gen.DefaultValue.GetValue())
			}
			fmt.Println()
		}
	}

	// Port clause
	if entity.Port != nil {
		fmt.Printf("\nPorts (%d total):\n", len(entity.Port.Ports))
		// Show first 10 ports
		limit := len(entity.Port.Ports)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			port := entity.Port.Ports[i]
			fmt.Printf("  %s : %s %s\n", port.Name, port.Mode, port.Type.Name)
		}
		if len(entity.Port.Ports) > 10 {
			fmt.Printf("  ... and %d more ports\n", len(entity.Port.Ports)-10)
		}
	}

	// Use clause
	useClause := entity.GetUseClause()
	if useClause != nil {
		fmt.Printf("\nUse Clause: %s.%s\n", useClause.Package, useClause.Dot)
	}

	// Attributes
	attrs := entity.GetAttributes()
	if len(attrs) > 0 {
		fmt.Printf("\nAttributes (%d total):\n", len(attrs))
		// Show first 5 attributes
		limit := len(attrs)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			attr := attrs[i]
			if attr.Spec != nil {
				fmt.Printf("  attribute %s of %s: %s\n",
					attr.Spec.Name, attr.Spec.Of, attr.Spec.EntityType)
			} else if attr.Constant != nil {
				fmt.Printf("  constant %s: %s\n",
					attr.Constant.Name, attr.Constant.Type)
			}
		}
		if len(attrs) > 5 {
			fmt.Printf("  ... and %d more attributes\n", len(attrs)-5)
		}
	}

	fmt.Println("\nParsing completed successfully!")
}
