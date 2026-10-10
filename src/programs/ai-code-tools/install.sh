#!/usr/bin/env bash
set -euo pipefail

PREFIX="${1:-$HOME/.ai-tools}"
mkdir -p "$PREFIX"

cp generate-ai-index "$PREFIX/generate-ai-index"
cp ai-code-index "$PREFIX/ai-code-index"
chmod +x "$PREFIX/generate-ai-index" "$PREFIX/ai-code-index"

echo "Installed:"
echo "  $PREFIX/generate-ai-index"
echo "  $PREFIX/ai-code-index"
echo ""
echo "Add this to your shell profile if not already present:"
echo "  export PATH=\"$PREFIX:\$PATH\""
