# Chocolatey Package for slack-chat-api

This directory contains the Chocolatey package definition for slack-chat-api.

## Package Structure

```
packaging/chocolatey/
├── slack-chat-api.nuspec          # Package metadata
├── tools/
│   ├── chocolateyInstall.ps1      # Install script
│   └── chocolateyUninstall.ps1    # Uninstall script
└── README.md                       # This file
```

## How It Works

1. **Release Workflow**: When a new version is released, the GitHub Actions workflow:
   - Downloads `checksums.txt` from the release
   - Injects the Windows checksums into `chocolateyInstall.ps1`
   - Updates the version in `slack-chat-api.nuspec`
   - Packs and pushes to Chocolatey

2. **Checksum Placeholders**: The install script uses placeholders that are replaced at build time:
   - `CHECKSUM_AMD64_PLACEHOLDER` → SHA256 of Windows x64 zip
   - `CHECKSUM_ARM64_PLACEHOLDER` → SHA256 of Windows ARM64 zip

3. **Architecture Detection**: The install script automatically detects:
   - ARM64 Windows → Downloads `windows_arm64.zip`
   - x64 Windows → Downloads `windows_amd64.zip`
   - 32-bit Windows → Throws an error (not supported)

## Manual Publishing

If automated publishing fails, use the manual workflow:

```bash
gh workflow run chocolatey-publish.yml -f version=X.Y.Z
```

Or via the GitHub Actions UI: Actions → "Publish to Chocolatey" → Run workflow

## Required Secrets

- `CHOCOLATEY_API_KEY`: API key from https://community.chocolatey.org/account

## Local Testing

To test the package locally:

```powershell
# Pack the package (creates .nupkg file)
choco pack

# Install locally for testing
choco install slack-chat-api -s . -y

# Verify installation
slack-chat-api --version

# Uninstall
choco uninstall slack-chat-api -y
```

## Moderation Notes

Chocolatey has automated moderation. Key rules followed:

- **CPMR0041**: `projectUrl` uses `#readme` suffix to differ from `projectSourceUrl`
- **CPMR0055**: Only uses `Install-ChocolateyZipPackage` (no custom downloaders)
- **CPMR0073**: Checksums are required and injected at build time

First submissions typically take 1-3 days for human review. Subsequent versions are usually auto-approved.
