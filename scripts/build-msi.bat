@echo off
REM Build MSI for Pangolin CLI (WiX v4). Package Version comes from pangolin-cli.wxs (set via scripts\set-version.sh).
REM Requires the `wix` CLI on PATH.
REM 1) Build:  make go-build-release-windows-amd64 VERSION=...
REM 2) Run from repo root:  scripts\build-msi.bat
REM    Or:  scripts\build-msi.bat C:\path\to\folder-with-exe

setlocal
cd /d "%~dp0.."

if "%~1"=="" (
  set "BUILDDIR=bin"
) else (
  set "BUILDDIR=%~1"
)

if not exist "%BUILDDIR%\pangolin-cli_windows_amd64.exe" (
  echo ERROR: %BUILDDIR%\pangolin-cli_windows_amd64.exe not found. Build the Windows binary first. 1>&2
  exit /b 1
)

if not exist "third_party\wintun\amd64\wintun.dll" (
  echo ERROR: third_party\wintun\amd64\wintun.dll not found. Download Wintun from https://www.wintun.net/ 1>&2
  exit /b 1
)

wix build -arch x64 -define "BuildDir=%BUILDDIR%" -o "%BUILDDIR%\pangolin-cli_windows_installer.msi" "pangolin-cli.wxs"
if errorlevel 1 exit /b %errorlevel%
echo Created "%BUILDDIR%\pangolin-cli_windows_installer.msi"
