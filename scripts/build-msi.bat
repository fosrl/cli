@echo off
REM Build MSI for Pangolin CLI (WiX v4+). CI does this automatically (sign-windows job in
REM .github/workflows/cicd.yml); this script is for local/manual builds.
REM Requires the `wix` CLI on PATH (dotnet tool install --global wix --version 5.0.2).
REM 1) Build:  make go-build-release-windows-amd64 VERSION=...
REM 2) Run from repo root:  scripts\build-msi.bat <version>
REM    Or:  scripts\build-msi.bat <version> C:\path\to\folder-with-exe
REM    <version> must be numeric X.Y.Z (MSI versions cannot carry an -rc suffix).

setlocal
cd /d "%~dp0.."

if "%~1"=="" (
  echo Usage: %~nx0 ^<version^> [builddir] 1>&2
  exit /b 1
)
set "MSIVERSION=%~1"

if "%~2"=="" (
  set "BUILDDIR=bin"
) else (
  set "BUILDDIR=%~2"
)

if not exist "%BUILDDIR%\pangolin-cli_windows_amd64.exe" (
  echo ERROR: %BUILDDIR%\pangolin-cli_windows_amd64.exe not found. Build the Windows binary first. 1>&2
  exit /b 1
)

if not exist "third_party\wintun\amd64\wintun.dll" (
  echo ERROR: third_party\wintun\amd64\wintun.dll not found. Download Wintun from https://www.wintun.net/ 1>&2
  exit /b 1
)

wix build -arch x64 -define "Version=%MSIVERSION%" -define "BuildDir=%BUILDDIR%" -o "%BUILDDIR%\pangolin-cli_windows_installer.msi" "pangolin-cli.wxs"
if errorlevel 1 exit /b %errorlevel%
echo Created "%BUILDDIR%\pangolin-cli_windows_installer.msi"
