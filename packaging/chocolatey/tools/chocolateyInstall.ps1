$ErrorActionPreference = 'Stop'

$version = $env:ChocolateyPackageVersion
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

# Checksums injected by release workflow - DO NOT EDIT MANUALLY
$checksumAmd64 = 'CHECKSUM_AMD64_PLACEHOLDER'
$checksumArm64 = 'CHECKSUM_ARM64_PLACEHOLDER'

# Architecture detection with ARM64 support
if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {
    $arch = 'arm64'
    $checksum = $checksumArm64
} elseif ([Environment]::Is64BitOperatingSystem) {
    $arch = 'amd64'
    $checksum = $checksumAmd64
} else {
    throw "32-bit Windows is not supported. slck requires 64-bit Windows."
}

$baseUrl = "https://github.com/open-cli-collective/slack-chat-api/releases/download/v${version}"
$zipFile = "slck_v${version}_windows_${arch}.zip"
$url = "${baseUrl}/${zipFile}"

Write-Host "Installing slck ${version} for Windows ${arch}..."
Write-Host "URL: ${url}"
Write-Host "Checksum (SHA256): ${checksum}"

Install-ChocolateyZipPackage -PackageName $env:ChocolateyPackageName `
    -Url $url `
    -UnzipLocation $toolsDir `
    -Checksum $checksum `
    -ChecksumType 'sha256'

Write-Host "slck installed successfully. Run 'slck --help' to get started."
