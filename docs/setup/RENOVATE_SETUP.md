# Renovate Dependency Updates Setup

## Overview

Renovate is configured to automatically update dependencies across:

- Go modules (backend/go.mod)
- npm packages (frontend/package.json)
- GitHub Actions workflows
- Docker base images

## Setup Options

You have **two options** for running Renovate:

### Option 1: Renovate GitHub App (Recommended ✅)

**Easiest setup, no configuration needed!**

1. **Install the Renovate GitHub App:**

   - Go to: https://github.com/apps/renovate
   - Click "Install"
   - Select your repository
   - Approve permissions

2. **That's it!**
   - Renovate will automatically detect `renovate.json`
   - PRs will be created according to the configuration
   - No secrets or workflows needed

**Pros:**

- ✅ Zero maintenance
- ✅ Runs on Renovate's infrastructure
- ✅ Automatic updates
- ✅ No GitHub Actions minutes used
- ✅ Better performance

**Cons:**

- ⚠️ Third-party app access to repository

### Option 2: Self-Hosted via GitHub Actions

**More control, uses your GitHub Actions minutes.**

#### Prerequisites

Choose one authentication method:

**Method A: Using GITHUB_TOKEN (Simple)**

The workflow already uses the built-in token:

Edit `.github/workflows/renovate.yml`:

```yaml
- name: Self-hosted Renovate
  uses: renovatebot/github-action@v46.3.5
  with:
    configurationFile: renovate.json
    token: ${{ secrets.GITHUB_TOKEN }} # Built-in token
```

**Method B: Using GitHub App (Advanced)**

Better for rate limits and more control:

1. **Create a GitHub App:**

   - Go to: Settings → Developer settings → GitHub Apps → New GitHub App
   - Name: "Renovate Bot" (or your choice)
   - Homepage URL: Your repo URL
   - Webhook: Uncheck "Active"
   - Permissions needed:
     - Repository permissions:
       - Contents: Read & Write
       - Pull Requests: Read & Write
       - Issues: Read & Write
       - Metadata: Read-only
   - Click "Create GitHub App"

2. **Generate private key:**

   - In your app settings, scroll to "Private keys"
   - Click "Generate a private key"
   - Download the `.pem` file

3. **Add secrets to your repository:**

   - Go to: Repository → Settings → Secrets and variables → Actions
   - Add secrets:
     - `RENOVATE_APP_ID`: Your app ID (found in app settings)
     - `RENOVATE_APP_PRIVATE_KEY`: Contents of the `.pem` file

4. **Install the app on your repository:**

   - In app settings, click "Install App"
   - Select your repository

5. **The workflow is already configured!**
   - It will use the app credentials automatically

The self-hosted workflow runs hourly on Monday from 10:17 through 14:17
Europe/Stockholm time. It has no manual trigger so runs stay within the
configured window.

## Configuration

The `renovate.json` file controls Renovate's behavior:

### Key Features

✅ **Semantic Commits**

- All PRs use conventional commit format
- Format: `chore(deps): update <package> to <version>`
- Works perfectly with your semantic-release setup

✅ **Auto-merge**

- Patch updates auto-merge after CI passes
- Minor updates require manual review (for stability)
- Major updates always require manual review

✅ **Smart Scheduling**

- Runs on Monday between 10:00 and 15:00 Europe/Stockholm time
- Creates, updates, and auto-merges dependency PRs only in that window
- The weekly dependency release runs Monday at 16:30 after the update window
- Limits concurrent PRs to avoid noise
- Groups related updates together

✅ **Multi-language Support**

- **Go**: Updates go.mod, runs `go mod tidy`
- **npm**: Updates package.json, package-lock.json
- **GitHub Actions**: Updates action versions
- **Docker**: Updates base image versions

✅ **Security Updates**

- Vulnerability alerts can create PRs outside the normal update schedule when
  the hosted Renovate app runs; Renovate deliberately exempts them from `schedule`
- Auto-labeled as "security"
- Can auto-merge security patches

### Commit Message Format

Renovate creates commits like:

```
chore(deps): update golang to 1.26.0
chore(deps): update node to 20.11.0
chore(deps): update actions/checkout to v4.1.2
```

These will:

- ✅ Pass conventional commit validation
- ✅ Trigger PATCH version bumps (via semantic-release)
- ✅ Appear in changelog under "Chores" section

## Customization

### Change Update Schedule

Update the matching window in `renovate.json` and the cron schedule in
`.github/workflows/renovate.yml`. The current Renovate settings are:

```json
{
  "schedule": ["* 10-14 * * 1"],
  "automergeSchedule": ["* 10-14 * * 1"],
  "timezone": "Europe/Stockholm"
}
```

Renovate's cron schedule uses `*` for minutes; the workflow starts at minute
17 of each allowed hour. A separately installed hosted Renovate app may still
check the repository outside this window, but this configuration limits its
ordinary dependency updates and auto-merges.

### Disable Auto-merge

Edit `renovate.json`:

```json
{
  "packageRules": [
    {
      "matchUpdateTypes": ["minor", "patch"],
      "automerge": false // Changed from true
    }
  ]
}
```

### Group Updates

Group all Go dependencies together:

```json
{
  "packageRules": [
    {
      "matchManagers": ["gomod"],
      "groupName": "Go dependencies"
    }
  ]
}
```

### Ignore Specific Dependencies

Edit `renovate.json`:

```json
{
  "ignoreDeps": ["some-package-name", "another-package"]
}
```

### Pin to Major Versions

Prevent major version updates:

```json
{
  "packageRules": [
    {
      "matchUpdateTypes": ["major"],
      "enabled": false
    }
  ]
}
```

## Workflow

### What Renovate Does

1. **Scans for outdated dependencies** (hourly during the Monday window)
2. **Creates PRs** with updates
3. **Runs CI tests** on each PR
4. **Auto-merges** (if configured and tests pass)
5. **Closes PRs** if dependency is no longer needed

### Example PR

```
Title: chore(deps): update golang to 1.26.0

Description:
This PR updates golang from 1.25.5 to 1.26.0

Changelog: https://go.dev/doc/go1.26

---
✅ All CI checks passed
🤖 Auto-merge enabled for patch updates
```

### How It Integrates with Your Pipeline

```
Renovate detects update
    ↓
Creates PR: chore(deps): update <package>
    ↓
CI Workflow runs (tests, lint, build)
    ↓
    ├─ Tests fail → PR needs manual fix
    └─ Tests pass → Auto-merge (if enabled)
        ↓
    Merged to main
        ↓
    Release workflow runs
        ↓
    New PATCH version created
    (e.g., 0.1.0 → 0.1.1)
```

## Labels

Renovate PRs will be labeled:

- `dependencies` - All dependency updates
- `renovate` - Created by Renovate
- `security` - Security vulnerability fixes

Add to `.github/workflows/ci.yml` to require these PRs to pass CI:

```yaml
on:
  pull_request:
    branches: [main]
    types: [opened, synchronize, reopened]
```

## Monitoring

### View Renovate Activity

**If using GitHub App:**

- Check the "Dependency graph" tab in your repository
- View PRs with "renovate" label

**If using self-hosted:**

- Go to: Actions → Renovate
- View workflow runs and logs

### Troubleshooting

#### Renovate not creating PRs

1. **Check configuration:**

   ```bash
   # Validate renovate.json
   npx --yes renovate-config-validator
   ```

2. **Check logs:**

   - GitHub App: Check PR comments from Renovate bot
   - Self-hosted: Check GitHub Actions logs

3. **Common issues:**
   - Schedule hasn't run yet (Monday 10:17–14:17 Europe/Stockholm)
   - No updates available (all dependencies current)
   - Rate limited (wait or use GitHub App)

#### PRs not auto-merging

1. **Check CI is passing:**

   - All tests must pass
   - No merge conflicts
   - Branch protection rules allow auto-merge

2. **Check configuration:**

   ```json
   "automerge": true,
   "automergeType": "pr"
   ```

3. **Enable auto-merge in GitHub:**
   - Settings → General → Allow auto-merge

## Best Practices

✅ **Start conservative:**

- Disable auto-merge initially
- Review a few PRs manually
- Enable auto-merge once comfortable

✅ **Monitor the first few updates:**

- Ensure tests catch breaking changes
- Verify semantic commits are correct
- Adjust configuration as needed

✅ **Use branch protection:**

- Require CI to pass before merge
- Require reviews for major updates
- Enable auto-merge in settings

✅ **Review security updates quickly:**

- Renovate prioritizes security fixes
- Auto-merge is safe for security patches
- Check changelog for breaking changes

## Files

```
foodlist/
├── renovate.json                    # Renovate configuration
├── .github/workflows/
│   └── renovate.yml                 # Self-hosted workflow (optional)
└── RENOVATE_SETUP.md               # This file
```

## Resources

- [Renovate Documentation](https://docs.renovatebot.com/)
- [Configuration Options](https://docs.renovatebot.com/configuration-options/)
- [Conventional Commits Preset](https://docs.renovatebot.com/presets-default/#semanticcommits)
- [GitHub App](https://github.com/apps/renovate)

---

## Quick Start

### Option 1: GitHub App (Recommended)

1. Install: https://github.com/apps/renovate
2. That's it! ✅

### Option 2: Self-Hosted

1. Push the configured `.github/workflows/renovate.yml` to GitHub
2. Check the scheduled run on Monday 10:17–14:17 Europe/Stockholm

**Current Status:** Configuration ready, choose your setup option!
