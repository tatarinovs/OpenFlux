$ErrorActionPreference = "Stop"

$ndkPath = "C:\Users\Serg\AppData\Local\Android\Sdk\ndk\28.2.13676358"
if (-not (Test-Path $ndkPath)) {
    Write-Error "NDK not found at $ndkPath"
}

$outputDir = "android\app\src\main\jniLibs\arm64-v8a"
if (-not (Test-Path $outputDir)) {
    New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
}

$env:GOOS = "android"
$env:GOARCH = "arm64"
$env:CGO_ENABLED = "1"
$clang = "$ndkPath\toolchains\llvm\prebuilt\windows-x86_64\bin\aarch64-linux-android24-clang.cmd"
$keyLdflag = ""
$keyFile = "secret_key.txt"
if (-not (Test-Path $keyFile)) {
    $keyFile = "..\secret_key.txt"
}
if (Test-Path $keyFile) {
    $k = (Get-Content $keyFile -Raw).Trim()
    if ($k) {
        $keyLdflag = "-X universal-bypass-tool/mobile.DefaultSecretKey=$k"
        Write-Host "Embedding default secret key from $keyFile"
    }
}

$env:CGO_CFLAGS = "-O3 -DNDEBUG"
$env:CGO_LDFLAGS = "-Wl,-O3,--as-needed"

Write-Host "Building libopenflux.so for arm64-v8a using $clang..."

go build -trimpath -buildmode=c-shared -ldflags="-s -w -checklinkname=0 $keyLdflag" -o "$outputDir\libopenflux.so" ./mobile

if (Test-Path "$outputDir\libopenflux.so") {
    Write-Host "Build SUCCESS: $outputDir\libopenflux.so"
    Get-Item "$outputDir\libopenflux.so" | Select-Object Name, Length, LastWriteTime
} else {
    Write-Error "Build failed, output file not found"
}
