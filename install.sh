#!/usr/bin/env bash

set -e

echo "Building jtk..."
# Determine version from git or fallback to dev
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
echo "Version: $VERSION"

# Build the binary
go build -ldflags="-s -w -X 'github.com/zach-snell/jtk/internal/version.Version=$VERSION'" -o jtk ./cmd/jtk

# Determine destination directory
DEST_DIR="$HOME/.local/bin"

if [ ! -d "$DEST_DIR" ]; then
    echo "Creating $DEST_DIR..."
    mkdir -p "$DEST_DIR"
fi

echo "Installing jtk to $DEST_DIR..."
mv jtk "$DEST_DIR/"

echo "Installation complete!"
echo "Ensure that $DEST_DIR is in your system PATH using:"
echo '  export PATH="$HOME/.local/bin:$PATH"'
