# OpenFlux Android APK Builder
param (
    [switch]$DebugBuild
)

$ErrorActionPreference = "Stop"

Write-Host "=== 1. Building Native Libraries (arm64-v8a, armeabi-v7a, x86_64, x86) ===" -ForegroundColor Cyan
& .\scripts\build_android_lib.ps1

Write-Host "`n=== 2. Building Android APKs ===" -ForegroundColor Cyan
$env:JAVA_HOME = "C:\Program Files\Android\Android Studio\jbr"
$env:ANDROID_HOME = "C:\Users\Serg\AppData\Local\Android\Sdk"
$gradleBat = Join-Path $PSScriptRoot "android\gradlew.bat"
if (-not (Test-Path $gradleBat)) {
    $gradleBat = "C:\Users\Serg\.gradle\wrapper\dists\gradle-8.13-bin\5xuhj0ry160q40clulazy9h7d\gradle-8.13\bin\gradle.bat"
}

$task = if ($DebugBuild) { "assembleDebug" } else { "assembleRelease" }
$subDir = if ($DebugBuild) { "debug" } else { "release" }

Push-Location "android"
try {
    & $gradleBat $task
} finally {
    Pop-Location
}

$outputDir = "android\app\build\outputs\apk\$subDir"
$releasesDir = Join-Path $PSScriptRoot "releases"
if (-not (Test-Path $releasesDir)) {
    New-Item -ItemType Directory -Path $releasesDir -Force | Out-Null
}

$apks = Get-ChildItem $outputDir -Filter "*.apk" -ErrorAction SilentlyContinue
if ($apks) {
    Write-Host "`n=== SUCCESS! APKs Created ===" -ForegroundColor Green
    foreach ($a in $apks) {
        $cleanName = $a.Name -replace "^app-", "OpenFlux-v1.0.2-" -replace "-release\.apk$", ".apk" -replace "-debug\.apk$", "-debug.apk"
        Copy-Item $a.FullName (Join-Path $releasesDir $cleanName) -Force
        if ($cleanName -like "*arm64*") {
            Copy-Item $a.FullName (Join-Path $releasesDir "OpenFlux-release.apk") -Force
            Copy-Item $a.FullName (Join-Path $releasesDir "OpenFlux-v1.0.2.apk") -Force
        }
        Write-Host "  * $($a.Name) -> $cleanName ($([math]::Round($a.Length / 1MB, 2)) MB)" -ForegroundColor Yellow
    }
} else {
    Write-Error "APK build failed, output not found."
}

