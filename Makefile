.PHONY: help build build-otj build-kicad build-jtag test clean install lint fmt coverage run-viewer run-jtag run-otj docs

# Use a project-local Go build cache
GOCACHE ?= $(CURDIR)/.gocache
export GOCACHE

# Binary output directory
BIN_DIR := bin

# Default target
help:
	@echo "OpenTraceJTAG - Unified Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  make build          - Build all tools"
	@echo "  make build-otj      - Build unified otj CLI (recommended)"
	@echo "  make build-kicad    - Build standalone KiCad tools"
	@echo "  make build-jtag     - Build standalone JTAG tools"
	@echo "  make test           - Run all tests"
	@echo "  make coverage       - Run tests with coverage report"
	@echo "  make lint           - Run linter"
	@echo "  make fmt            - Format code"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make install        - Install all CLI tools"
	@echo "  make run-otj        - Run unified otj CLI"
	@echo "  make run-viewer     - Run KiCad board viewer"
	@echo "  make run-jtag       - Run JTAG CLI"
	@echo "  make docs           - Generate documentation"

# Build all tools
build: build-otj build-kicad build-jtag

# Build unified otj CLI (recommended)
build-otj:
	@echo "Building unified otj CLI..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/otj ./cmd/otj
	@echo "✓ otj built successfully"

# Build KiCad tools
build-kicad:
	@echo "Building KiCad tools..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/gio-viewer ./cmd/gio-viewer
	go build -o $(BIN_DIR)/sch-viewer ./cmd/sch-viewer
	go build -o $(BIN_DIR)/net-info ./cmd/net-info
	go build -o $(BIN_DIR)/sch-info ./cmd/sch-info
	@echo "✓ KiCad tools built"

# Build JTAG tools
build-jtag:
	@echo "Building JTAG tools..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/bsdl-parser ./cmd/bsdl-parser
	go build -o $(BIN_DIR)/jtag ./cmd/jtag
	@echo "✓ JTAG tools built"

# Run tests
test:
	go test -v ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
	@which goimports > /dev/null && goimports -w . || echo "goimports not found, using go fmt only"

# Clean build artifacts
clean:
	rm -rf $(BIN_DIR)/
	rm -f coverage.out coverage.html
	go clean

# Install all CLI tools
install:
	go install ./cmd/otj
	go install ./cmd/gio-viewer
	go install ./cmd/net-info
	go install ./cmd/sch-info
	go install ./cmd/bsdl-parser
	go install ./cmd/jtag

# Run KiCad board viewer with sample file
run-viewer: build-kicad
	@if [ -f testdata/boards/test_with_footprints.kicad_pcb ]; then \
		./$(BIN_DIR)/gio-viewer testdata/boards/test_with_footprints.kicad_pcb; \
	else \
		echo "No sample board file found"; \
	fi

# Run unified otj CLI
run-otj: build-otj
	./$(BIN_DIR)/otj --help

# Run JTAG CLI
run-jtag: build-jtag
	./$(BIN_DIR)/jtag --help

# Get dependencies
deps:
	go mod download
	go mod tidy

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Generate API docs
docs:
	@if [ -x scripts/gen_api_docs.sh ]; then \
		./scripts/gen_api_docs.sh; \
	else \
		echo "Documentation script not found"; \
	fi
	@command -v mkdocs >/dev/null && (echo "Building MkDocs site..." && mkdocs build >/dev/null && echo "MkDocs output: site/") || echo "mkdocs not found, skipped site build"

# Quick build for development
quick:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/otj ./cmd/otj
