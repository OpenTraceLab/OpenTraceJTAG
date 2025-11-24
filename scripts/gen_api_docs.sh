#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$ROOT/docs/api"

mkdir -p "$OUT_DIR"

packages=(
  "github.com/epkcfsm/jtag/pkg/bsdl"
  "github.com/epkcfsm/jtag/pkg/jtag"
  "github.com/epkcfsm/jtag/pkg/tap"
  "github.com/epkcfsm/jtag/pkg/chain"
)

cd "$ROOT"

for pkg in "${packages[@]}"; do
  name="$(basename "$pkg")"
  file="$OUT_DIR/${name}.md"
  {
    echo "# Package $name"
    echo
    echo '```'
    go doc -all "$pkg"
    echo '```'
  } > "$file"
  echo "Generated docs for $pkg -> docs/api/${name}.md"
done
