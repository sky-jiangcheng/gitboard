#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== GitBoard Build Script ==="
echo ""

# Step 1: Build frontend
echo "[1/2] Building React frontend..."
cd "$PROJECT_ROOT/web"
npm install --silent
npm run build
echo "  Frontend built to web/dist/"

# Step 2: Build Go binary
echo "[2/2] Building Go binary..."
cd "$PROJECT_ROOT"
go build -ldflags="-s -w" -o gitboard .
echo "  Binary: $PROJECT_ROOT/gitboard"

echo ""
echo "=== Build complete ==="
ls -lh gitboard
