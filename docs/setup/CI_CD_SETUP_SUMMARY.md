# GitHub Actions CI/CD Setup Summary

## What Was Created

### 1. GitHub Actions Workflows

#### `.github/workflows/ci.yml`

- **Purpose:** Continuous Integration on every push and PR
- **Jobs:**
  - Backend tests (Go 1.25.5)
  - Frontend tests (Node 20, Vitest, Svelte type checking)
  - Linting (golangci-lint)
  - Docker build validation
- **Triggers:** Push/PR to `main` or `develop` branches

#### `.github/workflows/release.yml`

- **Purpose:** Automated releases with semantic versioning
- **Jobs:**
  - Run tests before release
  - Semantic release (version bump, changelog, Git tags)
  - Build multi-platform Docker images
  - Create binary artifacts for multiple platforms
  - Publish to GitHub Container Registry
- **Triggers:** Push to `main` branch (after CI passes)
- **Outputs:**
  - GitHub releases with auto-generated notes
  - Docker images (amd64, arm64)
  - Binary downloads (Linux, macOS, Windows)

### 2. Configuration Files

#### `.releaserc.cjs`

- Semantic-release configuration
- Defines version bump rules based on commit types
- Configures changelog generation
- Manages Git tags and GitHub releases

#### `VERSION`

- Current version: `0.0.1`
- Automatically updated by semantic-release

#### `CHANGELOG.md`

- Initial changelog template
- Automatically updated on each release

#### `commitlint.config.js`

- Commit message validation rules
- Can be used with Git hooks for local validation

#### `.huskyrc.json`

- Git hooks configuration (optional)
- Validates commits before they're accepted

### 3. GitHub Templates

#### `.github/pull_request_template.md`

- Standardized PR template
- Includes commit type checklist
- Conventional commit format guidance

#### `.github/ISSUE_TEMPLATE/bug_report.yml`

- Structured bug report template
- Pre-filled with conventional commit format (`fix:`)

#### `.github/ISSUE_TEMPLATE/feature_request.yml`

- Structured feature request template
- Pre-filled with conventional commit format (`feat:`)

### 4. Documentation

#### `CONTRIBUTING.md`

- Comprehensive guide to conventional commits
- Examples for each commit type
- Explanation of version bumping rules
- Best practices and tips

#### `CI_CD_GUIDE.md`

- Complete CI/CD pipeline documentation
- Workflow explanations
- Usage instructions
- Troubleshooting guide
- Customization options

#### `validate-commit.sh`

- Shell script for local commit validation
- Can be used as Git hook
- Provides helpful error messages

#### `.cursorrules`

- Cursor IDE AI configuration
- Tells Cursor about commit conventions
- Enables semantic commit suggestions

#### `.gitmessage`

- Git commit message template
- Shows format and examples when committing
- Auto-configured with `git config commit.template`

#### `.vscode/settings.json`

- VS Code/Cursor IDE settings
- Defines commit types and scopes
- Enables conventional commits mode

#### Updated `README.md`

- Added CI/CD and Releases section
- Conventional commits overview
- Cursor IDE commit suggestions section
- Docker image pull instructions
- Links to documentation

#### `CURSOR_COMMIT_SETUP.md`

- Cursor IDE-specific documentation
- How to use AI commit suggestions
- Configuration verification steps
- Troubleshooting guide for Cursor

## Semantic Versioning Rules

| Commit Type | Example                   | Version Change | Description             |
| ----------- | ------------------------- | -------------- | ----------------------- |
| `feat:`     | `feat(api): add endpoint` | 0.0.1 → 0.1.0  | New feature (MINOR)     |
| `fix:`      | `fix(ui): button bug`     | 0.0.1 → 0.0.2  | Bug fix (PATCH)         |
| `feat!:`    | `feat(api)!: breaking`    | 0.0.1 → 1.0.0  | Breaking change (MAJOR) |
| `docs:`     | `docs: update readme`     | 0.0.1 → 0.0.2  | Documentation (PATCH)   |
| `perf:`     | `perf: optimize query`    | 0.0.1 → 0.0.2  | Performance (PATCH)     |
| Others      | `chore:`, `test:`, etc.   | 0.0.1 → 0.0.2  | Other changes (PATCH)   |

## How It Works

### Development Flow

1. **Create Feature Branch**

   ```bash
   git checkout -b feat/my-feature
   ```

2. **Make Changes & Commit**

   ```bash
   git add .
   git commit -m "feat(backend): add new feature"
   ```

3. **Push & Create PR**

   ```bash
   git push origin feat/my-feature
   ```

   - CI workflow runs automatically
   - Tests, linting, and build validation

4. **Merge to Main**

   - After PR approval and passing tests
   - Merge commits to `main` branch

5. **Automatic Release**
   - Release workflow triggers
   - Analyzes commits since last release
   - Determines version bump
   - Creates changelog
   - Tags release
   - Builds Docker images
   - Creates binary artifacts
   - Publishes to GitHub

### Release Process

When you push to `main`:

```
Push to main
    ↓
CI Workflow
    ├─ Backend Tests
    ├─ Frontend Tests
    ├─ Linting
    └─ Docker Build
    ↓ (if pass)
Release Workflow
    ├─ Analyze Commits
    ├─ Determine Version
    ├─ Update CHANGELOG.md
    ├─ Update VERSION file
    ├─ Create Git Tag
    ├─ Generate Release Notes
    ├─ Build Multi-Platform Binaries
    ├─ Build Docker Images
    ├─ Push to GitHub Container Registry
    └─ Publish GitHub Release
```

## What You Get

### GitHub Releases

- Automatically created with each release
- Version tags (e.g., `v0.1.0`)
- Auto-generated release notes from commits
- Downloadable binary artifacts

### Docker Images

Available at: `ghcr.io/YOUR_USERNAME/foodlist`

```bash
# Latest version
docker pull ghcr.io/YOUR_USERNAME/foodlist:latest

# Specific version
docker pull ghcr.io/YOUR_USERNAME/foodlist:0.1.0
```

Supports:

- `linux/amd64`
- `linux/arm64`

### Binary Downloads

Each release includes binaries for:

- Linux (amd64, arm64)
- macOS (amd64, arm64/Apple Silicon)
- Windows (amd64)

## Next Steps

### 1. Update Repository Settings (Required)

Go to your GitHub repository:

**Settings → Actions → General → Workflow permissions**

Select:

- ✅ "Read and write permissions"
- ✅ "Allow GitHub Actions to create and approve pull requests"

This allows workflows to:

- Create releases
- Push Docker images
- Commit changelog updates

### 2. First Release

To create your first release (`v0.0.1`), create an initial commit with conventional format:

```bash
# If you haven't pushed to main yet
git add .
git commit -m "chore: initial commit with CI/CD setup"
git push origin main
```

Or create a release commit:

```bash
git add .
git commit -m "feat: initial release of FoodList application"
git push origin main
```

This will:

- Trigger CI tests
- Create release v0.0.1 (or v0.1.0 for feat)
- Build and publish Docker images
- Create GitHub release with artifacts

### 3. Local Validation Setup (Optional)

Install commitlint for local commit validation:

```bash
cd /Users/martin/foodlist
npm install --save-dev @commitlint/{config-conventional,cli} husky
npx husky install
npx husky add .husky/commit-msg 'npx --no -- commitlint --edit ${1}'
```

Or use the provided script manually:

```bash
./validate-commit.sh .git/COMMIT_EDITMSG
```

### 4. Configure Git Hook (Alternative)

Add to `.git/hooks/commit-msg`:

```bash
#!/bin/sh
exec < /dev/tty
./validate-commit.sh "$1"
```

```bash
chmod +x .git/hooks/commit-msg
```

## Commit Message Examples

### Features (Minor Version Bump)

```bash
feat(backend): add user preferences API
feat(ui): implement dark mode toggle
feat(websocket): add connection pooling
```

### Bug Fixes (Patch Version Bump)

```bash
fix(auth): resolve token expiration issue
fix(ui): correct button alignment in modal
fix(store): prevent race condition in state update
```

### Breaking Changes (Major Version Bump)

```bash
feat(api)!: change response format to include metadata

BREAKING CHANGE: All API responses now include a metadata object.
Clients must be updated to handle the new structure.
```

### Documentation

```bash
docs(readme): add installation instructions
docs(api): document new endpoints
```

### Other Changes

```bash
chore(deps): update dependencies
refactor(store): simplify state management logic
test(backend): add integration tests for auth
ci(actions): optimize build cache
```

## Monitoring Releases

### View Workflow Status

1. Go to GitHub repository
2. Click "Actions" tab
3. See all workflow runs
4. Click on run for detailed logs

### View Releases

1. Go to GitHub repository
2. Click "Releases" in sidebar
3. See all published releases
4. Download artifacts

### Pull Docker Images

```bash
docker pull ghcr.io/YOUR_USERNAME/foodlist:latest
docker run -p 8080:8080 ghcr.io/YOUR_USERNAME/foodlist:latest
```

## Files Created

```
foodlist/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                    # CI pipeline
│   │   └── release.yml               # Release pipeline
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.yml           # Bug report template
│   │   └── feature_request.yml      # Feature request template
│   └── pull_request_template.md     # PR template
├── .releaserc.cjs                    # Semantic-release config
├── VERSION                           # Current version (0.0.1)
├── CHANGELOG.md                      # Auto-generated changelog
├── CONTRIBUTING.md                   # Contribution guide
├── CI_CD_GUIDE.md                    # CI/CD documentation
├── commitlint.config.js              # Commitlint config
├── .huskyrc.json                     # Husky config
├── validate-commit.sh                # Commit validator script
└── README.md                         # Updated with CI/CD info
```

## Benefits

✅ **Automated Testing** - Every push runs full test suite  
✅ **Automated Releases** - No manual version management  
✅ **Semantic Versioning** - Automatic version bumps based on commits  
✅ **Generated Changelogs** - Always up-to-date release notes  
✅ **Multi-Platform Builds** - Docker images and binaries for all platforms  
✅ **Conventional Commits** - Standardized commit messages  
✅ **Quality Gates** - Tests must pass before release  
✅ **GitHub Integration** - Native releases, container registry  
✅ **Developer-Friendly** - Clear guidelines and templates

## Troubleshooting

If releases don't work:

1. Check GitHub Settings → Actions → Workflow permissions
2. Enable "Read and write permissions"
3. Enable "Allow GitHub Actions to create and approve pull requests"
4. Re-run the workflow

If Docker push fails:

1. Check repository visibility (public vs private)
2. For private repos, may need Personal Access Token
3. Check workflow logs for specific errors

## Resources

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [Semantic Release Docs](https://semantic-release.gitbook.io/)
- [GitHub Actions Docs](https://docs.github.com/en/actions)

---

**Current Version:** 0.0.1  
**Ready for:** First release on next push to `main`
