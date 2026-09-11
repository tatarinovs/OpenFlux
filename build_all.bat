@echo off
setlocal enabledelayedexpansion

title OpenFlux Multi-Platform Release Builder
chcp 65001 >nul

echo =======================================================================
echo              OpenFlux Multi-Platform Release Builder
echo         Targets: Windows - GUI ^| Linux - Server/CLI ^| Android - APK
echo =======================================================================
echo.

set "SCRIPT_DIR=%~dp0"
if "%SCRIPT_DIR:~-1%"=="\" set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"
set "RELEASES_DIR=%SCRIPT_DIR%\releases"

if not exist "%RELEASES_DIR%" mkdir "%RELEASES_DIR%"

rem --- Parse Arguments ---
set "NO_PAUSE=0"
set "SKIP_WINDOWS=0"
set "SKIP_LINUX=0"
set "SKIP_ANDROID=0"

for %%A in (%*) do (
    if "%%A"=="--no-pause" set "NO_PAUSE=1"
    if "%%A"=="--skip-windows" set "SKIP_WINDOWS=1"
    if "%%A"=="--skip-linux" set "SKIP_LINUX=1"
    if "%%A"=="--skip-android" set "SKIP_ANDROID=1"
)

rem --- Check Go Toolchain ---
where go >nul 2>nul
if errorlevel 1 (
    echo [FATAL ERROR] Go compiler not found in PATH!
    echo Please install Go 1.22+ and add it to your PATH.
    echo.
    if "%NO_PAUSE%"=="0" pause
    exit /b 1
)

rem =======================================================================
rem  TARGET 1: Windows Desktop GUI
rem =======================================================================
if "%SKIP_WINDOWS%"=="1" goto :skip_windows
echo.
echo =======================================================================
echo  [1/3] Building Windows Desktop GUI: OpenFlux.exe + Wintun...
echo =======================================================================
call "%SCRIPT_DIR%\build_windows.bat" --no-pause
if errorlevel 1 (
    echo [ERROR] Windows build failed!
    if "%NO_PAUSE%"=="0" pause
    exit /b 1
)
echo [OK] Windows build finished.
:skip_windows

rem =======================================================================
rem  TARGET 2: Linux Binaries (amd64 + arm64)
rem =======================================================================
if "%SKIP_LINUX%"=="1" goto :skip_linux
echo.
echo =======================================================================
echo  [2/3] Building Linux Binaries: amd64 + arm64 exit node / CLI...
echo =======================================================================

cd /d "%SCRIPT_DIR%"

echo    * Compiling Linux amd64 [x86_64, VPS / Server]...
set "GOOS=linux"
set "GOARCH=amd64"
set "CGO_ENABLED=0"
go build -trimpath -ldflags="-s -w" -o "%RELEASES_DIR%\openflux-linux-amd64" .
if errorlevel 1 (
    echo [ERROR] Linux amd64 compilation failed!
    if "%NO_PAUSE%"=="0" pause
    exit /b 1
)

echo    * Compiling Linux arm64 [aarch64, ARM VPS / Oracle / Pi]...
set "GOOS=linux"
set "GOARCH=arm64"
set "CGO_ENABLED=0"
go build -trimpath -ldflags="-s -w" -o "%RELEASES_DIR%\openflux-linux-arm64" .
if errorlevel 1 (
    echo [ERROR] Linux arm64 compilation failed!
    if "%NO_PAUSE%"=="0" pause
    exit /b 1
)

rem Compatibility copy for universal-bypass-tool name
copy /y "%RELEASES_DIR%\openflux-linux-amd64" "%RELEASES_DIR%\universal-bypass-tool" >nul

echo    * Packaging Linux release archive with deployment configs...
powershell -NoProfile -Command "Compress-Archive -Path '%RELEASES_DIR%\openflux-linux-amd64', '%RELEASES_DIR%\openflux-linux-arm64', '%SCRIPT_DIR%\deploy' -DestinationPath '%RELEASES_DIR%\OpenFlux-Linux.zip' -Force"
if errorlevel 1 (
    echo [WARNING] Failed to create OpenFlux-Linux.zip
) else (
    echo [OK] Linux archive created: %RELEASES_DIR%\OpenFlux-Linux.zip
)
echo [OK] Linux builds finished.
:skip_linux

rem =======================================================================
rem  TARGET 3: Android Release APK
rem =======================================================================
if "%SKIP_ANDROID%"=="1" goto :skip_android
echo.
echo =======================================================================
echo  [3/3] Building Android Release APK: libopenflux.so + R8 ProGuard...
echo =======================================================================
call "%SCRIPT_DIR%\build_release.bat" --no-pause
if errorlevel 1 (
    echo [ERROR] Android release build failed!
    if "%NO_PAUSE%"=="0" pause
    exit /b 1
)
echo [OK] Android build finished.
:skip_android

rem =======================================================================
rem  ALL BUILDS SUMMARY
rem =======================================================================
echo.
echo =======================================================================
echo                  ALL RELEASE BUILDS FINISHED!
echo =======================================================================
echo.
echo Generated Release Artifacts in: %RELEASES_DIR%
echo.

powershell -NoProfile -Command "Get-ChildItem '%RELEASES_DIR%' -File | Where-Object { $_.Name -match '\.(exe|apk|zip)$|linux' } | Select-Object Name, @{Name='Size (MB)';Expression={[math]::Round($_.Length / 1MB, 2)}}, LastWriteTime | Format-Table -AutoSize"

echo =======================================================================
echo Ready for distribution!
echo.

if "%NO_PAUSE%"=="0" pause
