# OpenFlux Windows Desktop GUI Builder
$ErrorActionPreference = "Stop"

Write-Host "=======================================================================" -ForegroundColor Cyan
Write-Host "                OpenFlux Windows Desktop GUI Builder" -ForegroundColor Cyan
Write-Host "=======================================================================" -ForegroundColor Cyan

$scriptDir = $PSScriptRoot
$desktopDir = Join-Path $scriptDir "desktop"
$releasesDir = Join-Path $scriptDir "releases"

if (!(Get-Command wails -ErrorAction SilentlyContinue)) {
    if (Test-Path "$env:USERPROFILE\go\bin\wails.exe") {
        $env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"
    } else {
        Write-Error "Wails CLI not found! Install it via: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
    }
}

# Stop any running instances
Get-Process -Name "OpenFlux" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 500

Write-Host "`n[1/3] Building OpenFlux Desktop with Wails..." -ForegroundColor Yellow
Push-Location $desktopDir
try {
    & wails build -clean -trimpath -ldflags "-s -w"
} finally {
    Pop-Location
}

Write-Host "`n[2/3] Preparing release artifacts..." -ForegroundColor Yellow
if (!(Test-Path $releasesDir)) {
    New-Item -ItemType Directory -Path $releasesDir -Force | Out-Null
}

# Stop running instance if open
Get-Process -Name "OpenFlux" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 500

try {
    # If file is locked, move to .old first (allowed by Windows NTFS even for running files)
    if (Test-Path "$releasesDir\OpenFlux.old.exe") {
        Remove-Item "$releasesDir\OpenFlux.old.exe" -Force -ErrorAction SilentlyContinue
    }
    if (Test-Path "$releasesDir\OpenFlux.exe") {
        try {
            Remove-Item "$releasesDir\OpenFlux.exe" -Force -ErrorAction Stop
        } catch {
            Move-Item "$releasesDir\OpenFlux.exe" "$releasesDir\OpenFlux.old.exe" -Force -ErrorAction SilentlyContinue
        }
    }
    Copy-Item "$desktopDir\build\bin\OpenFlux.exe" "$releasesDir\OpenFlux.exe" -Force
    Copy-Item "$desktopDir\pkg\wintun\embed\wintun.dll" "$releasesDir\wintun.dll" -Force
    $zipPath = Join-Path $releasesDir "OpenFlux-Windows-GUI.zip"
    Compress-Archive -Path "$releasesDir\OpenFlux.exe", "$releasesDir\wintun.dll" -DestinationPath $zipPath -Force
    Write-Host "`n=======================================================================" -ForegroundColor Green
    Write-Host " SUCCESS! Build complete:" -ForegroundColor Green
    Write-Host " - EXE: $releasesDir\OpenFlux.exe" -ForegroundColor Yellow
    Write-Host " - ZIP: $zipPath" -ForegroundColor Yellow
    Write-Host "=======================================================================" -ForegroundColor Green
} catch {
    Write-Warning "Не удалось обновить releases\OpenFlux.exe: $_. Бинарник доступен в: $desktopDir\build\bin\OpenFlux.exe"
}
