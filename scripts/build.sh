#!/bin/bash
set -e

echo "=== Git Dashboard Build Script ==="
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
go build -ldflags="-s -w" -o git-dashboard .
echo "  Binary: $(pwd)/git-dashboard"

echo ""
echo "=== Build complete ==="
ls -lh git-dashboard
