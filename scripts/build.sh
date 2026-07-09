#!/bin/bash
set -e

echo "=== GitBoard Build Script ==="
echo ""

# Step 1: Build frontend
echo "[1/2] Building React frontend..."
cd "$(dirname "$0")/../web"
npm install --silent
npm run build
echo "  Frontend built to web/dist/"

# Step 2: Build Go binary
echo "[2/2] Building Go binary..."
cd "$(dirname "$0")/.."
export GOPROXY=https://goproxy.cn,direct
go build -ldflags="-s -w" -o gitboard .
echo "  Binary: $(pwd)/gitboard"

echo ""
echo "=== Build complete ==="
ls -lh gitboard
