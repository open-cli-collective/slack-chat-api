# Winget Package for slack-chat-api

This directory contains the Winget manifest templates for slack-chat-api.

## Package Structure

```
packaging/winget/
├── OpenCLICollective.slack-chat-api.yaml              # Version manifest
├── OpenCLICollective.slack-chat-api.installer.yaml    # Installer manifest
├── OpenCLICollective.slack-chat-api.locale.en-US.yaml # Locale manifest
└── README.md                                           # This file
```

## How It Works

Unlike Chocolatey (direct push), Winget uses **pull requests** to microsoft/winget-pkgs:

1. **Release Workflow**: When a new version is released, the GitHub Actions workflow:
   - Checks if the package already exists in winget-pkgs
   - **For updates**: Uses `wingetcreate update` with new URLs
   - **For new packages**: Processes templates with version/checksums and uses `wingetcreate submit`
   - Creates a PR to microsoft/winget-pkgs

2. **Microsoft Validation**: Automated validation runs on the PR
3. **Auto-merge**: On success, PRs are typically auto-merged within minutes
4. **Availability**: Users can install immediately after merge

## Manifest Templates

The manifest files use placeholders:
- `0.0.0` → Replaced with actual version
- 64 zeros → Replaced with SHA256 checksums (x64 first, then arm64)

## Package Identifier

```
OpenCLICollective.slack-chat-api
```

Manifests are stored at: `manifests/o/OpenCLICollective/slack-chat-api/`

## Manual Publishing

If automated publishing fails, use the manual workflow:

```bash
gh workflow run winget-publish.yml -f version=X.Y.Z
```

Or via the GitHub Actions UI: Actions → "Publish to Winget" → Run workflow

## Required Secrets

- `WINGET_GITHUB_TOKEN`: PAT with `public_repo` scope for creating PRs to microsoft/winget-pkgs

## Local Validation

To validate manifests locally:

```powershell
# Create a test directory with processed manifests
$testDir = "winget-test"
New-Item -ItemType Directory -Path $testDir -Force

# Copy and update manifests (replace placeholders with test values)
# ... see test-winget.yml for full example ...

# Validate
winget validate --manifest $testDir/
```

## Installation (after package is approved)

```powershell
winget install OpenCLICollective.slack-chat-api
```
