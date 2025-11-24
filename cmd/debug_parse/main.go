package main

import (
	"fmt"
	"io"
	"os"

	"github.com/chewxy/sexp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug_parse <board_file>")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Open file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Read file content
	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("File size: %d bytes\n", len(data))
	fmt.Printf("First 100 chars: %s\n", string(data[:100]))

	// Try to parse with sexp
	sexps, err := sexp.ParseString(string(data))
	if err != nil {
		fmt.Printf("Error parsing s-expression: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Number of s-expressions: %d\n", len(sexps))

	if len(sexps) > 0 {
		fmt.Printf("First sexp type: %T\n", sexps[0])
		fmt.Printf("Is leaf: %v\n", sexps[0].IsLeaf())
		if !sexps[0].IsLeaf() {
			fmt.Printf("Leaf count: %d\n", sexps[0].LeafCount())
		}
	}
}
