$ErrorActionPreference = 'Stop'

$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

Write-Host "Uninstalling slck..."

# Remove extracted files
Remove-Item "$toolsDir\slck.exe" -Force -ErrorAction SilentlyContinue
Remove-Item "$toolsDir\LICENSE" -Force -ErrorAction SilentlyContinue
Remove-Item "$toolsDir\README.md" -Force -ErrorAction SilentlyContinue

# Remove .ignore files created during install
Remove-Item "$toolsDir\*.ignore" -Force -ErrorAction SilentlyContinue

Write-Host "slck has been uninstalled."
