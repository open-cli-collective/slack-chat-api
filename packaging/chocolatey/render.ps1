[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern('^\d+\.\d+\.\d+$')]
    [string]$Version,

    [Parameter(Mandatory)]
    [ValidatePattern('^[0-9a-fA-F]{64}$')]
    [string]$Amd64Checksum,

    [Parameter(Mandatory)]
    [ValidatePattern('^[0-9a-fA-F]{64}$')]
    [string]$Arm64Checksum,

    [string]$PackageDirectory = $PSScriptRoot,
    [string]$Repository = 'open-cli-collective/slack-chat-api'
)

$nuspec = Join-Path $PackageDirectory 'slack-chat-cli.nuspec'
$installScript = Join-Path $PackageDirectory 'tools/chocolateyInstall.ps1'
if (-not (Test-Path $nuspec) -or -not (Test-Path $installScript)) {
    throw "Chocolatey package files not found under $PackageDirectory"
}

$nuspecContent = Get-Content $nuspec -Raw
if (-not $nuspecContent.Contains('<version>0.0.0</version>')) {
    throw "$nuspec has no version placeholder"
}
Set-Content $nuspec ($nuspecContent.Replace('<version>0.0.0</version>', "<version>$Version</version>"))

$baseUrl = "https://github.com/$Repository/releases/download/v$Version"
$values = [ordered]@{
    URL_AMD64_PLACEHOLDER = "$baseUrl/slck_v${Version}_windows_amd64.zip"
    URL_ARM64_PLACEHOLDER = "$baseUrl/slck_v${Version}_windows_arm64.zip"
    CHECKSUM_AMD64_PLACEHOLDER = $Amd64Checksum
    CHECKSUM_ARM64_PLACEHOLDER = $Arm64Checksum
}
$content = Get-Content $installScript -Raw
foreach ($placeholder in $values.Keys) {
    if (-not $content.Contains($placeholder)) {
        throw "$installScript has no $placeholder"
    }
    $content = $content.Replace($placeholder, $values[$placeholder])
}
$tokens = $null
$errors = $null
$null = [System.Management.Automation.Language.Parser]::ParseInput($content, [ref]$tokens, [ref]$errors)
if ($errors.Count) {
    throw $errors[0]
}
if ($content -match 'URL_(?:AMD64|ARM64)_PLACEHOLDER|CHECKSUM_(?:AMD64|ARM64)_PLACEHOLDER|ChocolateyPackageVersion|\$\{\s*version\s*\}') {
    throw "$installScript still contains an unrendered placeholder or runtime version expression"
}
foreach ($value in $values.Values) {
    if (-not $content.Contains($value)) {
        throw "$installScript is missing rendered value $value"
    }
}
Set-Content $installScript $content
