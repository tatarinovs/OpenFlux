# OpenFlux Android APK Builder
$ErrorActionPreference = "Stop"

Write-Host "=== 1. Building Native Library (libopenflux.so) ===" -ForegroundColor Cyan
& .\scripts\build_android_lib.ps1

Write-Host "`n=== 2. Building Android APK ===" -ForegroundColor Cyan
$env:JAVA_HOME = "C:\Program Files\Android\Android Studio\jbr"
$env:ANDROID_HOME = "C:\Users\Serg\AppData\Local\Android\Sdk"
$gradleBat = "C:\Users\Serg\.gradle\wrapper\dists\gradle-8.13-bin\5xuhj0ry160q40clulazy9h7d\gradle-8.13\bin\gradle.bat"

Push-Location "android"
try {
    & $gradleBat assembleDebug
} finally {
    Pop-Location
}

$apk = Get-Item "android\app\build\outputs\apk\debug\app-debug.apk" -ErrorAction SilentlyContinue
if ($apk) {
    Write-Host "`n=== SUCCESS! APK Created ===" -ForegroundColor Green
    Write-Host "Path: $($apk.FullName)" -ForegroundColor Yellow
    Write-Host "Size: $([math]::Round($apk.Length / 1MB, 2)) MB" -ForegroundColor Yellow
} else {
    Write-Error "APK build failed, output not found."
}
