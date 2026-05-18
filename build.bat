@echo off
setlocal EnableDelayedExpansion

echo.
echo  GitLab Pipeline Viewer - Build Script
echo  ======================================
echo.

echo  [+] Building main binary (glab-pipe.exe)...
go build -ldflags="-s -w" -o dist/glab-pipe.exe .
if errorlevel 1 (
    echo  [!] Failed to build main binary
    pause
    exit /b 1
)
echo      OK  Main binary built: dist\glab-pipe.exe

echo.
echo  [+] Building installer (installer.exe)...
go build -ldflags="-s -w" -o dist/installer.exe ./cmd/installer
if errorlevel 1 (
    echo  [!] Failed to build installer
    pause
    exit /b 1
)
echo      OK  Installer built: dist\installer.exe

echo.
echo  ======================================
echo  Build complete!
echo.
echo  Output binaries:
echo    - dist\glab-pipe.exe        (main application)
echo    - dist\installer.exe        (TUI installer)
echo.
echo  To install:
echo    dist\installer.exe
echo.
echo  To debug manually:
echo    dist\glab-pipe.exe
echo    dist\glab-pipe.exe .
echo    dist\glab-pipe.exe --source dfs/support/dfs-case-management/casemanagement/case-gateway
echo.
pause
