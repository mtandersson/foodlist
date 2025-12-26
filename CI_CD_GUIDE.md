# GitHub Actions CI/CD Pipeline

## Overview

This project uses GitHub Actions for continuous integration and automated releases with semantic versioning.

## Workflows

### 1. CI Workflow (`.github/workflows/ci.yml`)

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches

**Jobs:**

#### Backend Tests
- Sets up Go 1.25.5
- Runs `go test` with race detection and coverage
- Uploads coverage to Codecov

#### Frontend Tests
- Sets up Node.js 20
- Runs type checking with `svelte-check`
- Runs Vitest tests with coverage
- Uploads coverage to Codecov

#### E2E Tests
- Sets up both Go and Node.js
- Builds backend and frontend
- Runs Cypress end-to-end tests
- Uploads screenshots on failure

#### Lint
- Runs golangci-lint on backend code

#### Build Docker
- Tests Docker image build
- Uses build cache for faster builds

### 2. Release Workflow (`.github/workflows/release.yml`)

**Triggers:**
- Push to `main` branch only

**Jobs:**

#### Test
- Runs backend and frontend tests before release

#### Release
- Uses semantic-release to:
  - Analyze commit messages
  - Determine next version
  - Generate changelog
  - Create Git tag
  - Publish GitHub release
  - Update VERSION file
  - Update CHANGELOG.md

#### Build and Push
- Builds multi-platform Docker images (amd64, arm64)
- Pushes to GitHub Container Registry
- Tags with both version and `latest`
- Creates binary artifacts for:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64/Apple Silicon)
  - Windows (amd64)
- Uploads binaries to GitHub release

## Semantic Versioning

Version numbers follow [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH
```

### Version Bumps

Based on conventional commit messages:

| Commit Type | Example | Version Bump | Description |
|-------------|---------|--------------|-------------|
| `feat:` | `feat(api): add new endpoint` | MINOR (0.1.0 → 0.2.0) | New features |
| `fix:` | `fix(ui): resolve button bug` | PATCH (0.1.0 → 0.1.1) | Bug fixes |
| `feat!:` or `BREAKING CHANGE:` | `feat(api)!: change response format` | MAJOR (0.1.0 → 1.0.0) | Breaking changes |
| `docs:` | `docs: update README` | PATCH | Documentation |
| `perf:` | `perf: optimize query` | PATCH | Performance |
| Other | `chore:`, `refactor:`, `test:`, `ci:`, `build:` | PATCH | Other changes |

### Initial Version

The project starts at version `0.0.1` (defined in `VERSION` file).

## Configuration Files

### `.releaserc.json`

Semantic-release configuration:

```json
{
  "branches": ["main"],
  "plugins": [
    "@semantic-release/commit-analyzer",      // Analyze commits
    "@semantic-release/release-notes-generator", // Generate notes
    "@semantic-release/changelog",            // Update CHANGELOG.md
    "@semantic-release/exec",                 // Update VERSION file
    "@semantic-release/github",               // Create GitHub release
    "@semantic-release/git"                   // Commit changes
  ]
}
```

### `VERSION`

Current version file (managed by semantic-release):
- Updated automatically on each release
- Used by build scripts and Docker tags

### `CHANGELOG.md`

Automatically generated changelog:
- Updated on each release
- Organized by version and commit type
- Includes links to commits and PRs

### `commitlint.config.js`

Commitlint configuration for local validation:
- Enforces conventional commit format
- Validates commit message structure
- Can be integrated with Git hooks

## Using the Pipeline

### 1. Development Workflow

```bash
# Create a feature branch
git checkout -b feat/my-new-feature

# Make changes and commit using conventional format
git commit -m "feat(backend): add user preferences"

# Push and create PR
git push origin feat/my-new-feature
```

**Result:** CI workflow runs on PR (tests, linting, build)

### 2. Release Workflow

```bash
# Merge PR to main (after approval and passing tests)
git checkout main
git merge feat/my-new-feature
git push origin main
```

**Result:**
1. CI workflow runs first
2. If tests pass, Release workflow triggers:
   - Analyzes commits since last release
   - Creates new version (e.g., 0.1.0 → 0.2.0)
   - Updates CHANGELOG.md
   - Creates Git tag (e.g., v0.2.0)
   - Publishes GitHub release
   - Builds and pushes Docker images
   - Creates binary artifacts

### 3. Viewing Releases

- Go to your repository on GitHub
- Click "Releases" in the sidebar
- See automatically generated release notes
- Download binaries for your platform

### 4. Using Docker Images

```bash
# Pull latest
docker pull ghcr.io/YOUR_USERNAME/foodlist:latest

# Pull specific version
docker pull ghcr.io/YOUR_USERNAME/foodlist:0.2.0

# Run
docker run -p 8080:8080 ghcr.io/YOUR_USERNAME/foodlist:latest
```

## Local Setup (Optional)

### Install Commitlint

Validate commits locally before pushing:

```bash
# In project root
npm install --save-dev @commitlint/{config-conventional,cli} husky

# Setup git hooks
npx husky install
npx husky add .husky/commit-msg 'npx --no -- commitlint --edit ${1}'
```

Now Git will validate your commit messages before accepting them.

### Test Semantic Release Locally

```bash
# Install dependencies
npm install --save-dev \
  semantic-release \
  @semantic-release/git \
  @semantic-release/changelog \
  @semantic-release/exec \
  conventional-changelog-conventionalcommits

# Dry run (doesn't create actual release)
npx semantic-release --dry-run
```

## Troubleshooting

### Release Not Created

**Problem:** Pushed to main but no release was created.

**Solutions:**
1. Check commit messages follow conventional format
2. Ensure commits include releasable types (`feat`, `fix`, etc.)
3. View workflow logs in GitHub Actions tab
4. Check if tests passed (release only runs after tests)

### Docker Push Failed

**Problem:** Docker image build or push failed.

**Solutions:**
1. Check GitHub Container Registry permissions
2. Ensure GITHUB_TOKEN has write permissions
3. Repository settings → Actions → General → Workflow permissions → "Read and write permissions"

### Wrong Version Bump

**Problem:** Version bumped incorrectly (e.g., minor instead of patch).

**Solutions:**
1. Review commit messages between releases
2. Ensure commit types are correct:
   - Use `fix:` for patches
   - Use `feat:` for minor versions
   - Use `feat!:` or `BREAKING CHANGE:` for major versions
3. Check `.releaserc.json` configuration

### CI Tests Failing

**Problem:** CI tests fail but pass locally.

**Solutions:**
1. Check Go and Node.js versions match CI (Go 1.25.5, Node 20)
2. Ensure `go.mod` and `package-lock.json` are committed
3. Run tests in same environment: `make test`
4. Check for environment-specific issues (paths, dependencies)

## Customization

### Change Branch Strategy

Edit `.releaserc.json`:

```json
{
  "branches": [
    "main",
    { "name": "beta", "prerelease": true }
  ]
}
```

### Add More Platforms

Edit `.github/workflows/release.yml` to add more build targets:

```yaml
- name: Build for FreeBSD
  run: |
    cd backend
    GOOS=freebsd GOARCH=amd64 go build -o ../dist/foodlist-freebsd-amd64 .
```

### Change Release Frequency

Modify workflow triggers in `.github/workflows/release.yml`:

```yaml
on:
  push:
    branches:
      - main
  # Or use manual trigger
  workflow_dispatch:
```

## GitHub Secrets

No manual secrets needed! The workflows use:
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions
- Has permissions for releases, packages, and commits

## Monitoring

Check pipeline status:
1. Go to repository on GitHub
2. Click "Actions" tab
3. View workflow runs
4. Click on a run to see detailed logs

## Best Practices

1. **Write Clear Commits**
   - Use descriptive commit messages
   - Follow conventional format strictly
   - Include scope when relevant

2. **Test Before Merging**
   - Always test locally: `make test`
   - Review CI results on PRs
   - Don't merge failing PRs

3. **Meaningful Releases**
   - Group related changes
   - Don't push every commit to main
   - Use feature branches for development

4. **Document Changes**
   - Commit messages become release notes
   - Write for users, not just developers
   - Include context and reasoning

5. **Version Strategy**
   - Start with 0.x.x for initial development
   - Move to 1.0.0 when stable
   - Use 1.0.0 for production-ready releases

## Resources

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [Semantic Release](https://semantic-release.gitbook.io/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)

