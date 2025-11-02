# Windows Release Build Script for ETC Meisai Scraper
# PowerShell version

param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

# Get version from git tag or use default
if ($Version -eq "") {
    try {
        $Version = git describe --tags --always 2>$null
    } catch {
        $Version = "dev"
    }
    if ($Version -eq "") {
        $Version = "dev"
    }
}

Write-Host "Building ETC Meisai Scraper $Version for Windows..." -ForegroundColor Cyan

# Build for Windows amd64
Write-Host "Building Windows amd64..." -ForegroundColor Yellow

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

$outputName = "etc_meisai_scraper-$Version-windows-amd64.exe"

go build -o $outputName -ldflags="-s -w -X main.Version=$Version" .

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Build completed: $outputName" -ForegroundColor Green
} else {
    Write-Host "❌ Build failed" -ForegroundColor Red
    exit 1
}

# Create zip archive
Write-Host "Creating zip archive..." -ForegroundColor Yellow

$zipName = "etc_meisai_scraper-$Version-windows-amd64.zip"

try {
    Compress-Archive -Path $outputName, "README.md", "CLAUDE.md" -DestinationPath $zipName -Force
    Write-Host "✅ Archive created: $zipName" -ForegroundColor Green
} catch {
    Write-Host "⚠️ Failed to create archive: $_" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Build completed successfully!" -ForegroundColor Green
Write-Host "Binary: $outputName"
