#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
url="https://github.com/JetBrains/JetBrainsMono/raw/v2.304/fonts/ttf/JetBrainsMono-Regular.ttf"
name="JetBrainsMono-Regular.ttf"
if [ ! -f "$name" ]; then
  curl -L "$url" -o "$name"
fi
