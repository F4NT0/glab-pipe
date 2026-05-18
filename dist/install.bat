@echo off
setlocal EnableDelayedExpansion

echo.
echo  ██████╗ ██╗      █████╗ ██████╗        ██████╗ ██╗██████╗ ███████╗██╗     ██╗███╗   ██╗███████╗
echo ██╔════╝ ██║     ██╔══██╗██╔══██╗      ██╔══██╗██║██╔══██╗██╔════╝██║     ██║████╗  ██║██╔════╝
echo ██║  ███╗██║     ███████║██████╔╝█████╗██████╔╝██║██████╔╝█████╗  ██║     ██║██╔██╗ ██║█████╗  
echo ██║   ██║██║     ██╔══██║██╔══██╗╚════╝██╔═══╝ ██║██╔═══╝ ██╔══╝  ██║     ██║██║╚██╗██║██╔══╝  
echo ╚██████╔╝███████╗██║  ██║██████╔╝      ██║     ██║██║     ███████╗███████╗██║██║ ╚████║███████╗
echo  ╚═════╝ ╚══════╝╚═╝  ╚═╝╚═════╝       ╚═╝     ╚═╝╚═╝     ╚══════╝╚══════╝╚═╝╚═╝  ╚═══╝╚══════╝
echo.
echo   GitLab Pipeline Management Tool — Installer
echo.
echo   This will install the GitLab Pipeline Viewer and configure the 'glab-pipe' command.
echo.

:: Check if installer exists
if not exist "%~dp0installer.exe" (
    echo   [!] ERROR: installer.exe not found in dist directory.
    echo   Please build the project first:
    echo       go build -o dist\gitlab-pipeline.exe .
    echo       go build -o dist\installer.exe ./cmd/installer
    pause
    exit /b 1
)

:: Run the Go installer
echo   [+] Running installer...
"%~dp0installer.exe"

if errorlevel 1 (
    echo   [!] Installation failed.
    pause
    exit /b 1
)

echo.
echo   Installation complete!
echo.
echo   Restart PowerShell and run: glab-pipe
echo.
pause
