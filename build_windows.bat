@echo off
setlocal enabledelayedexpansion

title OpenFlux Windows Desktop Builder
chcp 65001 >nul

echo =======================================================================
echo                 OpenFlux Windows Desktop GUI Builder
echo =======================================================================
echo.

set "SCRIPT_DIR=%~dp0"
if "%SCRIPT_DIR:~-1%"=="\" set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"
set "DESKTOP_DIR=%SCRIPT_DIR%\desktop"
set "RELEASES_DIR=%SCRIPT_DIR%\releases"

rem Check Go
where go >nul 2>nul
if errorlevel 1 (
    echo [ERROR] Go not found in PATH!
    pause
    exit /b 1
)

rem Check Wails
where wails >nul 2>nul
if errorlevel 1 (
    if exist "%USERPROFILE%\go\bin\wails.exe" (
        set "PATH=%USERPROFILE%\go\bin;%PATH%"
    ) else (
        echo [ERROR] Wails CLI not found! Install it via: go install github.com/wailsapp/wails/v2/cmd/wails@latest
        pause
        exit /b 1
    )
)

rem Stop running instance if open
taskkill /F /IM OpenFlux.exe >nul 2>nul
timeout /t 1 /nobreak >nul 2>nul

echo [1/3] Compiling OpenFlux Desktop with Wails (Clean, Trimpath, Stripped)...
pushd "%DESKTOP_DIR%"
call wails build -clean -trimpath -ldflags "-s -w"
if errorlevel 1 (
    echo [ERROR] Wails build failed!
    popd
    pause
    exit /b 1
)
popd

echo.
echo [2/3] Preparing release artifacts...
if not exist "%RELEASES_DIR%" mkdir "%RELEASES_DIR%"
copy /y "%DESKTOP_DIR%\build\bin\OpenFlux.exe" "%RELEASES_DIR%\OpenFlux.exe" >nul
copy /y "%DESKTOP_DIR%\pkg\wintun\embed\wintun.dll" "%RELEASES_DIR%\wintun.dll" >nul

echo.
echo [3/3] Packaging Release ZIP...
powershell -NoProfile -Command "Compress-Archive -Path '%RELEASES_DIR%\OpenFlux.exe', '%RELEASES_DIR%\wintun.dll' -DestinationPath '%RELEASES_DIR%\OpenFlux-Windows-GUI.zip' -Force"

echo.
echo =======================================================================
echo  SUCCESS! Build complete:
echo  - %RELEASES_DIR%\OpenFlux.exe
echo  - %RELEASES_DIR%\OpenFlux-Windows-GUI.zip
echo =======================================================================
echo.
if "%~1"=="" pause
