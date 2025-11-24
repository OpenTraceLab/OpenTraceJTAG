package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/chewxy/sexp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: investigate_sexp <board_file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Get file info
	info, _ := file.Stat()
	fmt.Printf("File size: %d bytes (%.2f MB)\n", info.Size(), float64(info.Size())/1024/1024)

	// Try parsing with Parse(io.Reader)
	fmt.Println("\nAttempt 1: Using sexp.Parse(io.Reader)...")
	sexps, err := sexp.Parse(file)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  Success! Parsed %d s-expressions\n", len(sexps))
		if len(sexps) > 0 {
			fmt.Printf("  First sexp is leaf: %v\n", sexps[0].IsLeaf())
			if !sexps[0].IsLeaf() {
				fmt.Printf("  Leaf count: %d\n", sexps[0].LeafCount())
			}
		}
	}

	// Reset file pointer
	file.Seek(0, 0)

	// Try with NewParser
	fmt.Println("\nAttempt 2: Using sexp.NewParser(io.Reader, strict=false)...")
	parser := sexp.NewParser(file, false)

	count := 0
	timeout := 0
	for s := range parser.Output {
		if s != nil {
			count++
			if count <= 3 {
				fmt.Printf("  Got sexp #%d (leaf: %v)\n", count, s.IsLeaf())
			}
		}
		timeout++
		if timeout > 1000 {
			fmt.Println("  Timeout - stopping after 1000 iterations")
			break
		}
	}
	fmt.Printf("  Total s-expressions received: %d\n", count)

	// Try parsing just the first 1000 lines
	fmt.Println("\nAttempt 3: Parsing first 1000 lines only...")
	file.Seek(0, 0)
	buf := make([]byte, 100000)
	n, _ := file.Read(buf)

	// Find a good stopping point (closing paren)
	content := string(buf[:n])
	lastParen := strings.LastIndex(content, ")")
	if lastParen > 0 {
		content = content[:lastParen+1]
	}

	sexps3, err := sexp.ParseString(content)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  Success! Parsed %d s-expressions from first 100KB\n", len(sexps3))
	}
}
