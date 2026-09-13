@echo off
setlocal enabledelayedexpansion

title OpenFlux Release Builder
chcp 65001 >nul

echo =======================================================================
echo                 OpenFlux Release APK Builder
echo =======================================================================
echo.

set "SCRIPT_DIR=%~dp0"
if "%SCRIPT_DIR:~-1%"=="\" set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"
set "RELEASES_DIR=%SCRIPT_DIR%\releases"

rem --- 1. Environment Detection ---
if not defined JAVA_HOME (
    if exist "C:\Program Files\Android\Android Studio\jbr" (
        set "JAVA_HOME=C:\Program Files\Android\Android Studio\jbr"
    )
)

if not defined ANDROID_HOME (
    if exist "%LOCALAPPDATA%\Android\Sdk" (
        set "ANDROID_HOME=%LOCALAPPDATA%\Android\Sdk"
    ) else if exist "C:\Users\Serg\AppData\Local\Android\Sdk" (
        set "ANDROID_HOME=C:\Users\Serg\AppData\Local\Android\Sdk"
    )
)

set "PATH=%JAVA_HOME%\bin;%PATH%"

where go >nul 2>nul
if errorlevel 1 (
    echo [ERROR] Go not found in PATH!
    pause
    exit /b 1
)

set "NDK_DIR="
if exist "%ANDROID_HOME%\ndk\28.2.13676358" (
    set "NDK_DIR=%ANDROID_HOME%\ndk\28.2.13676358"
) else (
    for /d %%D in ("%ANDROID_HOME%\ndk\*") do (
        set "NDK_DIR=%%D"
    )
)

if not defined NDK_DIR (
    echo [ERROR] Android NDK not found in %ANDROID_HOME%\ndk!
    pause
    exit /b 1
)

set "CLANG_CMD=%NDK_DIR%\toolchains\llvm\prebuilt\windows-x86_64\bin\aarch64-linux-android24-clang.cmd"
if not exist "%CLANG_CMD%" (
    echo [ERROR] NDK Clang not found: %CLANG_CMD%
    pause
    exit /b 1
)

echo [OK] Java Home:    %JAVA_HOME%
echo [OK] Android SDK:  %ANDROID_HOME%
echo [OK] NDK:          %NDK_DIR%
echo.

rem --- 2. Build Optimized Go Native Libraries (All ABIs) ---
echo =======================================================================
echo [1/3] Building Go libraries (arm64-v8a, armeabi-v7a, x86_64, x86)...
echo =======================================================================

powershell.exe -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\scripts\build_android_lib.ps1"
if errorlevel 1 (
    echo [ERROR] Native library build failed!
    pause
    exit /b 1
)

echo [OK] All native libraries built successfully.
echo.

rem --- 3. Check / Generate Keystore ---
echo =======================================================================
echo [2/3] Checking Release signing key...
echo =======================================================================

set "KEYSTORE=%SCRIPT_DIR%\android\openflux-release.jks"
if not exist "%KEYSTORE%" (
    if "!OPENFLUX_KEYSTORE_PASSWORD!"=="" (
        set /p "OPENFLUX_KEYSTORE_PASSWORD=Enter new release keystore password: "
    )
    echo    * Generating release keystore...
    keytool -genkeypair -v -keystore "%KEYSTORE%" -alias openflux -keyalg RSA -keysize 2048 -validity 10000 -storepass "!OPENFLUX_KEYSTORE_PASSWORD!" -keypass "!OPENFLUX_KEYSTORE_PASSWORD!" -dname "CN=OpenFlux, OU=Dev, O=OpenFlux, L=Moscow, ST=Moscow, C=RU" >nul 2>nul
    if errorlevel 1 (
        echo [WARNING] Could not auto-generate keystore.
    ) else (
        echo [OK] Keystore created: %KEYSTORE%
    )
) else (
    echo [OK] Keystore found: %KEYSTORE%
)
echo.

rem --- 4. Build Release APKs via Gradle ---
echo =======================================================================
echo [3/3] Building Release APKs (R8 Minify, ProGuard, Resource Shrinking)...
echo =======================================================================

cd /d "%SCRIPT_DIR%\android"
call gradlew.bat assembleRelease
if errorlevel 1 (
    echo [ERROR] Gradle assembleRelease failed!
    cd /d "%SCRIPT_DIR%"
    pause
    exit /b 1
)

cd /d "%SCRIPT_DIR%"

rem --- 5. Copy all architecture APKs to releases\ ---
if not exist "%RELEASES_DIR%" mkdir "%RELEASES_DIR%"

set "OUT_DIR=%SCRIPT_DIR%\android\app\build\outputs\apk\release"

if exist "%OUT_DIR%\app-arm64-v8a-release.apk" (
    copy /y "%OUT_DIR%\app-arm64-v8a-release.apk" "%RELEASES_DIR%\OpenFlux-v1.0.2-arm64-v8a.apk" >nul
    copy /y "%OUT_DIR%\app-arm64-v8a-release.apk" "%RELEASES_DIR%\OpenFlux-v1.0.2.apk" >nul
    copy /y "%OUT_DIR%\app-arm64-v8a-release.apk" "%RELEASES_DIR%\OpenFlux-release.apk" >nul
)
if exist "%OUT_DIR%\app-armeabi-v7a-release.apk" (
    copy /y "%OUT_DIR%\app-armeabi-v7a-release.apk" "%RELEASES_DIR%\OpenFlux-v1.0.2-armeabi-v7a.apk" >nul
)
if exist "%OUT_DIR%\app-x86_64-release.apk" (
    copy /y "%OUT_DIR%\app-x86_64-release.apk" "%RELEASES_DIR%\OpenFlux-v1.0.2-x86_64.apk" >nul
)
if exist "%OUT_DIR%\app-x86-release.apk" (
    copy /y "%OUT_DIR%\app-x86-release.apk" "%RELEASES_DIR%\OpenFlux-v1.0.2-x86.apk" >nul
)
if exist "%OUT_DIR%\app-universal-release.apk" (
    copy /y "%OUT_DIR%\app-universal-release.apk" "%RELEASES_DIR%\OpenFlux-v1.0.2-universal.apk" >nul
)

echo.
echo =======================================================================
echo                     RELEASE BUILDS SUCCESSFUL!
echo =======================================================================
echo.
echo Copied APK artifacts to: %RELEASES_DIR%
dir "%RELEASES_DIR%\*.apk"

echo.
echo Optimizations applied:
echo   - Go runtime: -trimpath, symbols stripped (-s -w), CGO -O3
echo   - Android code: R8 code minification and optimization
echo   - Android resources: shrinkResources and AAPT2 optimization
echo   - Signed: V2 Signature scheme (release key)
echo.
echo =======================================================================
if "%1"=="--no-pause" goto :end
pause
:end
