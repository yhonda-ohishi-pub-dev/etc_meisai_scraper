#!/bin/bash
# Windows Release Build Script for ETC Meisai Scraper

set -e

# Get version from git tag or use default
VERSION=${1:-$(git describe --tags --always 2>/dev/null || echo "dev")}

echo "Building ETC Meisai Scraper ${VERSION} for Windows..."

# Build for Windows amd64
echo "Building Windows amd64..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
  -o "etc_meisai_scraper-${VERSION}-windows-amd64.exe" \
  -ldflags="-s -w -X main.Version=${VERSION}" \
  .

echo "✅ Build completed: etc_meisai_scraper-${VERSION}-windows-amd64.exe"

# Create zip archive if zip is available
if command -v zip &> /dev/null; then
  echo "Creating zip archive..."
  zip "etc_meisai_scraper-${VERSION}-windows-amd64.zip" \
    "etc_meisai_scraper-${VERSION}-windows-amd64.exe" \
    README.md \
    CLAUDE.md
  echo "✅ Archive created: etc_meisai_scraper-${VERSION}-windows-amd64.zip"
else
  echo "⚠️ zip command not found, skipping archive creation"
fi

echo ""
echo "Build completed successfully!"
echo "Binary: etc_meisai_scraper-${VERSION}-windows-amd64.exe"
