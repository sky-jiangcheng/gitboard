#!/bin/bash
# GitBoard install script for macOS and Linux
set -e

INSTALL_DIR="/usr/local/bin"
BINARY_NAME="gitboard"
REPO="sky-jiangcheng/gitboard"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux)   TARGET="linux-amd64" ;;
  darwin)
    if [ "$ARCH" = "arm64" ]; then
      TARGET="darwin-arm64"
    else
      TARGET="darwin-amd64"
    fi
    ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

echo "Downloading GitBoard for $TARGET..."

DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/gitboard-$TARGET"

if [ ! -w "$INSTALL_DIR" ]; then
  echo "Need sudo to install to $INSTALL_DIR"
  sudo curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/$BINARY_NAME"
  sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
  curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/$BINARY_NAME"
  chmod +x "$INSTALL_DIR/$BINARY_NAME"
fi

echo ""
echo "GitBoard installed to $INSTALL_DIR/$BINARY_NAME"
echo "Run 'gitboard' to start!"
