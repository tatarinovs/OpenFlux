# OpenFlux Multi-Platform Release Builder (PowerShell)
param (
    [switch]$SkipWindows,
    [switch]$SkipLinux,
    [switch]$SkipAndroid,
    [switch]$NoPause
)

$ErrorActionPreference = "Stop"

Write-Host "=======================================================================" -ForegroundColor Cyan
Write-Host "              OpenFlux Multi-Platform Release Builder" -ForegroundColor Cyan
Write-Host "         Targets: Windows (GUI) | Linux (Server/CLI) | Android (APK)" -ForegroundColor Cyan
Write-Host "=======================================================================" -ForegroundColor Cyan
Write-Host ""

$scriptDir = $PSScriptRoot
$releasesDir = Join-Path $scriptDir "releases"

if (!(Test-Path $releasesDir)) {
    New-Item -ItemType Directory -Path $releasesDir -Force | Out-Null
}

# --- 1. Windows Desktop GUI ---
if (!$SkipWindows) {
    Write-Host "`n=======================================================================" -ForegroundColor Yellow
    Write-Host " [1/3] Building Windows Desktop GUI (OpenFlux.exe + Wintun)..." -ForegroundColor Yellow
    Write-Host "=======================================================================" -ForegroundColor Yellow
    & "$scriptDir\build_windows.ps1"
    if ($LASTEXITCODE -and $LASTEXITCODE -ne 0) {
        Write-Error "Windows build failed!"
    }
} else {
    Write-Host "`n[SKIPPED] Windows build (-SkipWindows)" -ForegroundColor DarkGray
}

# --- 2. Linux Binaries ---
if (!$SkipLinux) {
    Write-Host "`n=======================================================================" -ForegroundColor Yellow
    Write-Host " [2/3] Building Linux Binaries (amd64 + arm64 exit node / CLI)..." -ForegroundColor Yellow
    Write-Host "=======================================================================" -ForegroundColor Yellow

    Push-Location $scriptDir
    try {
        Write-Host "   * Compiling Linux amd64 (x86_64, VPS / Server)..."
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        $env:CGO_ENABLED = "0"
        go build -trimpath -ldflags="-s -w" -o (Join-Path $releasesDir "openflux-linux-amd64") .
        if ($LASTEXITCODE -ne 0) { throw "Linux amd64 compilation failed" }

        Write-Host "   * Compiling Linux arm64 (aarch64, ARM VPS / Oracle / Pi)..."
        $env:GOOS = "linux"
        $env:GOARCH = "arm64"
        $env:CGO_ENABLED = "0"
        go build -trimpath -ldflags="-s -w" -o (Join-Path $releasesDir "openflux-linux-arm64") .
        if ($LASTEXITCODE -ne 0) { throw "Linux arm64 compilation failed" }

        Copy-Item (Join-Path $releasesDir "openflux-linux-amd64") (Join-Path $releasesDir "universal-bypass-tool") -Force

        Write-Host "   * Packaging Linux release archive (binaries + deploy configs)..."
        $linuxZip = Join-Path $releasesDir "OpenFlux-Linux.zip"
        Compress-Archive -Path (Join-Path $releasesDir "openflux-linux-amd64"), (Join-Path $releasesDir "openflux-linux-arm64"), (Join-Path $scriptDir "deploy") -DestinationPath $linuxZip -Force
        Write-Host "   [OK] Created: $linuxZip" -ForegroundColor Green
    } finally {
        $env:GOOS = ""
        $env:GOARCH = ""
        $env:CGO_ENABLED = ""
        Pop-Location
    }
} else {
    Write-Host "`n[SKIPPED] Linux build (-SkipLinux)" -ForegroundColor DarkGray
}

# --- 3. Android Release APK ---
if (!$SkipAndroid) {
    Write-Host "`n=======================================================================" -ForegroundColor Yellow
    Write-Host " [3/3] Building Android Release APK (libopenflux.so + R8 ProGuard)..." -ForegroundColor Yellow
    Write-Host "=======================================================================" -ForegroundColor Yellow
    cmd.exe /c "build_release.bat --no-pause"
    if ($LASTEXITCODE -and $LASTEXITCODE -ne 0) {
        Write-Error "Android release build failed!"
    }
} else {
    Write-Host "`n[SKIPPED] Android build (-SkipAndroid)" -ForegroundColor DarkGray
}

# --- Summary ---
Write-Host "`n=======================================================================" -ForegroundColor Green
Write-Host "                  ALL RELEASE BUILDS FINISHED!" -ForegroundColor Green
Write-Host "=======================================================================" -ForegroundColor Green
Write-Host "Generated Release Artifacts in: $releasesDir`n"

Get-ChildItem $releasesDir -File | Where-Object { $_.Name -match '\.(exe|apk|zip)$|linux' } |
    Select-Object Name, @{Name="Size (MB)";Expression={[math]::Round($_.Length / 1MB, 2)}}, LastWriteTime |
    Format-Table -AutoSize

Write-Host "Ready for distribution!`n" -ForegroundColor Green

if (!$NoPause) {
    Read-Host "Press Enter to exit"
}
