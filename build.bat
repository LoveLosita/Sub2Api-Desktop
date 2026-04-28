@echo off
chcp 65001 >nul

cd /d "%~dp0"

:: Backup data files before clean
if exist "build\bin\data.db" (
    echo [0/3] Backing up data files...
    if not exist "build\backup" mkdir "build\backup"
    copy /Y "build\bin\data.db" "build\backup\data.db" >nul 2>&1
    copy /Y "build\bin\data.db-wal" "build\backup\data.db-wal" >nul 2>&1
    copy /Y "build\bin\data.db-shm" "build\backup\data.db-shm" >nul 2>&1
    copy /Y "build\bin\config.yaml" "build\backup\config.yaml" >nul 2>&1
    echo       Done.
)

echo [1/3] Building frontend...
cd /d "%~dp0frontend"
call npm run build
if %errorlevel% neq 0 (
    echo Frontend build failed!
    pause
    exit /b 1
)

echo [2/3] Building Wails app...
cd /d "%~dp0"
wails build -clean
if %errorlevel% neq 0 (
    echo Wails build failed!
    pause
    exit /b 1
)

:: Restore data files after clean
if exist "build\backup\data.db" (
    echo [3/3] Restoring data files...
    copy /Y "build\backup\data.db" "build\bin\data.db" >nul 2>&1
    copy /Y "build\backup\data.db-wal" "build\bin\data.db-wal" >nul 2>&1
    copy /Y "build\backup\data.db-shm" "build\bin\data.db-shm" >nul 2>&1
    copy /Y "build\backup\config.yaml" "build\bin\config.yaml" >nul 2>&1
    echo       Done.
) else (
    echo [3/3] Done!
)

echo.
echo Output: build\bin\desktop-proxy.exe
pause
