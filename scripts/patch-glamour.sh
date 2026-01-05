#!/bin/bash
# patch-glamour.sh - Updates vendored glamour and re-applies marker patches
#
# Usage: ./scripts/patch-glamour.sh [glamour-version]
#
# This script:
# 1. Downloads a fresh copy of glamour (specified version or latest)
# 2. Removes glamour's go.mod/go.sum (integrates into main module)
# 3. Applies the marker injection patches for link and header position tracking
# 4. Updates all import paths to use the local module path
# 5. Runs go mod tidy to clean up dependencies
# 6. Runs tests to verify the patches work correctly
#
# The patches inject invisible Unicode markers around link text and headers
# during glamour's rendering process. These markers enable 100% reliable
# position detection for navigation purposes.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GLAMOUR_DIR="$PROJECT_ROOT/glamour"
PATCH_DIR="$SCRIPT_DIR/glamour-patches"

VERSION="${1:-v0.10.0}"

echo "=== Glamour Patch Script ==="
echo "Version: $VERSION"
echo "Glamour dir: $GLAMOUR_DIR"
echo "Patch dir: $PATCH_DIR"
echo ""

# Check if patches exist
if [[ ! -f "$PATCH_DIR/link-markers.patch" ]] || [[ ! -f "$PATCH_DIR/heading-markers.patch" ]]; then
    echo "ERROR: Patch files not found in $PATCH_DIR"
    echo "Expected:"
    echo "  - $PATCH_DIR/link-markers.patch"
    echo "  - $PATCH_DIR/heading-markers.patch"
    exit 1
fi

# Backup existing glamour if it exists
if [[ -d "$GLAMOUR_DIR" ]]; then
    echo "Backing up existing glamour directory..."
    rm -rf "$GLAMOUR_DIR.bak"
    mv "$GLAMOUR_DIR" "$GLAMOUR_DIR.bak"
fi

# Clone glamour
echo "Downloading glamour $VERSION..."
git clone --depth 1 --branch "$VERSION" https://github.com/charmbracelet/glamour.git "$GLAMOUR_DIR" 2>&1

# Remove .git directory (we're vendoring, not maintaining a submodule)
rm -rf "$GLAMOUR_DIR/.git"

# Remove glamour's go.mod (integrate into main module)
echo ""
echo "Removing glamour's go.mod files..."
rm -f "$GLAMOUR_DIR/go.mod" "$GLAMOUR_DIR/go.sum"

# Apply patches
echo ""
echo "Applying marker patches..."

echo "  - link-markers.patch"
if ! patch -p1 -d "$GLAMOUR_DIR" < "$PATCH_DIR/link-markers.patch"; then
    echo "ERROR: Failed to apply link-markers.patch"
    echo "The patch may be incompatible with glamour $VERSION"
    echo "You may need to manually update the patch."
    exit 1
fi

echo "  - heading-markers.patch"
if ! patch -p1 -d "$GLAMOUR_DIR" < "$PATCH_DIR/heading-markers.patch"; then
    echo "ERROR: Failed to apply heading-markers.patch"
    echo "The patch may be incompatible with glamour $VERSION"
    echo "You may need to manually update the patch."
    exit 1
fi

# Update import paths to use local module
echo ""
echo "Updating import paths to use local module..."
find "$GLAMOUR_DIR" -name "*.go" -type f -exec sed -i '' 's|github\.com/charmbracelet/glamour|github.com/boolean-maybe/navidown/glamour|g' {} \;

# Clean up dependencies
echo ""
echo "Cleaning up go.mod dependencies..."
cd "$PROJECT_ROOT"
go mod tidy

# Verify build
echo ""
echo "Verifying build..."
cd "$PROJECT_ROOT"
if ! go build ./...; then
    echo "ERROR: Build failed after applying patches"
    exit 1
fi

# Run tests
echo ""
echo "Running tests..."
if ! go test ./navidown/...; then
    echo "ERROR: Tests failed after applying patches"
    exit 1
fi

# Update golden test files
echo ""
echo "Updating golden test files..."
go test ./glamour/ansi -update
go test ./glamour -update

# Cleanup backup
if [[ -d "$GLAMOUR_DIR.bak" ]]; then
    rm -rf "$GLAMOUR_DIR.bak"
fi

echo ""
echo "=== Success ==="
echo "Glamour $VERSION has been updated with marker patches applied."
echo ""
echo "Changes applied:"
echo "  - glamour/ansi/link.go (link text markers)"
echo "  - glamour/ansi/heading.go (header markers with level encoding)"
echo "  - Removed glamour/go.mod and glamour/go.sum (integrated into main module)"
echo "  - Updated all import paths to github.com/boolean-maybe/navidown/glamour"
echo "  - Cleaned up go.mod dependencies"
