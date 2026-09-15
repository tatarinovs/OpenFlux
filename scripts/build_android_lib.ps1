param (
    [string[]]$Abis = @("arm64-v8a", "armeabi-v7a", "x86_64", "x86")
)

$ErrorActionPreference = "Stop"

$ndkPath = "C:\Users\Serg\AppData\Local\Android\Sdk\ndk\28.2.13676358"
if (-not (Test-Path $ndkPath)) {
    # Fallback to searching NDK
    $found = Get-ChildItem "C:\Users\Serg\AppData\Local\Android\Sdk\ndk" -Directory | Select-Object -Last 1
    if ($found) {
        $ndkPath = $found.FullName
    } else {
        Write-Error "NDK not found at $ndkPath"
    }
}

$llvmBin = "$ndkPath\toolchains\llvm\prebuilt\windows-x86_64\bin"

$targets = @{
    "arm64-v8a"   = @{ GoArch = "arm64"; GoArm = "";  Clang = "$llvmBin\aarch64-linux-android24-clang.cmd" }
    "armeabi-v7a" = @{ GoArch = "arm";   GoArm = "7"; Clang = "$llvmBin\armv7a-linux-androideabi24-clang.cmd" }
    "x86_64"      = @{ GoArch = "amd64"; GoArm = "";  Clang = "$llvmBin\x86_64-linux-android24-clang.cmd" }
    "x86"         = @{ GoArch = "386";   GoArm = "";  Clang = "$llvmBin\i686-linux-android24-clang.cmd" }
}

$keyLdflag = ""
$keyFile = "secret_key.txt"
if (-not (Test-Path $keyFile)) {
    $keyFile = "..\secret_key.txt"
}
if (Test-Path $keyFile) {
    $k = (Get-Content $keyFile -Raw).Trim()
    if ($k) {
        $keyLdflag = "-X openflux/mobile.DefaultSecretKey=$k"
        Write-Host "Embedding default secret key from $keyFile"
    }
}

$env:GOOS = "android"
$env:CGO_ENABLED = "1"
$env:CGO_CFLAGS = "-O3 -DNDEBUG"
$env:CGO_LDFLAGS = "-Wl,-O3,--as-needed"

foreach ($abi in $Abis) {
    if (-not $targets.ContainsKey($abi)) {
        Write-Warning "Unknown ABI: $abi. Skipping."
        continue
    }

    $target = $targets[$abi]
    $outputDir = "android\app\src\main\jniLibs\$abi"
    if (-not (Test-Path $outputDir)) {
        New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
    }

    $env:GOARCH = $target.GoArch
    $env:GOARM = $target.GoArm
    $env:CC = $target.Clang

    Write-Host "`n--> Building libopenflux.so for $abi (GOARCH=$($target.GoArch)) using $($target.Clang)..." -ForegroundColor Cyan
    go build -trimpath -buildmode=c-shared -ldflags="-s -w -checklinkname=0 $keyLdflag" -o "$outputDir\libopenflux.so" ./mobile

    if (Test-Path "$outputDir\libopenflux.so") {
        $file = Get-Item "$outputDir\libopenflux.so"
        Write-Host "    [OK] Build SUCCESS: $abi -> $([math]::Round($file.Length / 1MB, 2)) MB" -ForegroundColor Green
    } else {
        Write-Error "Build failed for $abi, output file not found"
    }
}

