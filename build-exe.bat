@echo off
chcp 65001 >nul
title Desktop Proxy - Build EXE

echo ============================================
echo   Desktop Proxy - Building EXE
echo ============================================
echo.

:: Check wails CLI
where wails >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] wails CLI not found! Please install: go install github.com/wailsapp/wails/v2/cmd/wails@latest
    pause
    exit /b 1
)

:: Check Go
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Go not found! Please install Go first.
    pause
    exit /b 1
)

:: Check Node.js
where node >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Node.js not found! Please install Node.js first.
    pause
    exit /b 1
)

echo [1/3] Installing frontend dependencies...
cd /d "%~dp0frontend"
call npm install
if %errorlevel% neq 0 (
    echo [ERROR] npm install failed!
    pause
    exit /b 1
)

echo.
echo [2/3] Building project (wails build)...
cd /d "%~dp0"
wails build
if %errorlevel% neq 0 (
    echo [ERROR] wails build failed!
    pause
    exit /b 1
)

echo.
echo [3/3] Done!
echo.
echo ============================================
echo   Build successful!
echo   Output: %~dp0build\bin\desktop-proxy.exe
echo ============================================
echo.

:: Ask to open output folder
set /p OPEN="Open output folder? (Y/N): "
if /i "%OPEN%"=="Y" explorer "%~dp0build\bin"

pause
