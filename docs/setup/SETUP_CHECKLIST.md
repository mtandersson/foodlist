# First-Time Setup Checklist

Follow these steps to complete the CI/CD setup for your repository.

## ✅ Pre-Push Checklist

### 1. Verify Files Created

All these files should now exist:

```bash
# GitHub Actions
.github/workflows/ci.yml
.github/workflows/release.yml

# Templates
.github/ISSUE_TEMPLATE/bug_report.yml
.github/ISSUE_TEMPLATE/feature_request.yml
.github/pull_request_template.md

# Configuration
.releaserc.cjs
VERSION
CHANGELOG.md
commitlint.config.js
.huskyrc.json

# Documentation
CONTRIBUTING.md
CI_CD_GUIDE.md
CI_CD_SETUP_SUMMARY.md
COMMIT_QUICK_REFERENCE.md

# Scripts
validate-commit.sh
```

### 2. Verify Initial Version

Check that VERSION file contains:
```
0.0.1
```

### 3. Create Initial Commit

Use conventional commit format:

```bash
git add .
git commit -m "ci: add GitHub Actions workflows and semantic versioning

- Added CI workflow for automated testing
- Added release workflow with semantic-release
- Created conventional commit templates and guides
- Initial version set to 0.0.1"
```

Or if you want a feature commit for first minor version:

```bash
git add .
git commit -m "feat: initial release with CI/CD pipeline

- Real-time todo list with WebSockets
- Go backend with event sourcing
- Svelte frontend with TypeScript
- Automated releases with semantic versioning
- Multi-platform Docker images"
```

## 🔧 GitHub Repository Setup

### 1. Create GitHub Repository (if not exists)

```bash
# Create on GitHub, then:
git remote add origin https://github.com/YOUR_USERNAME/foodlist.git
git branch -M main
```

### 2. Configure Repository Settings

**IMPORTANT:** Before pushing, configure these settings:

#### Go to: Settings → Actions → General

**Workflow permissions:**
- ✅ Select "Read and write permissions"
- ✅ Enable "Allow GitHub Actions to create and approve pull requests"

**Click "Save"**

This allows workflows to:
- Create GitHub releases
- Push to GitHub Container Registry
- Commit changelog updates
- Create tags

### 3. Configure Branch Protection (Recommended)

Go to: Settings → Branches → Add branch protection rule

For branch `main`:
- ✅ Require a pull request before merging
- ✅ Require status checks to pass before merging
  - Add: `Backend Tests`, `Frontend Tests`, `Lint`
- ✅ Require conversation resolution before merging

### 4. Configure GitHub Pages (Optional)

If you want to host documentation:

Go to: Settings → Pages
- Source: Deploy from a branch
- Branch: `main` / `docs` (if you create one)

## 📦 Package Registry Setup

### Public Repository (Default)

Docker images will automatically publish to:
```
ghcr.io/YOUR_USERNAME/foodlist
```

No additional setup needed!

### Private Repository

If your repo is private, you may need to:

1. Create a Personal Access Token (PAT):
   - Settings → Developer settings → Personal access tokens
   - Generate new token (classic)
   - Scopes: `write:packages`, `repo`

2. Add as repository secret:
   - Repository → Settings → Secrets and variables → Actions
   - New repository secret: `GHCR_TOKEN`
   - Value: Your PAT

3. Update `.github/workflows/release.yml`:
   ```yaml
   - name: Log in to GitHub Container Registry
     uses: docker/login-action@v3
     with:
       registry: ghcr.io
       username: ${{ github.actor }}
       password: ${{ secrets.GHCR_TOKEN }}  # Changed from GITHUB_TOKEN
   ```

## 🚀 First Push

### 1. Push to GitHub

```bash
git push -u origin main
```

### 2. Watch Workflows

1. Go to your repository on GitHub
2. Click "Actions" tab
3. You should see:
   - ✅ CI workflow running
   - ⏳ Release workflow waiting for CI to complete

### 3. Verify First Release

After workflows complete:

1. **Check Releases:**
   - Go to Releases in GitHub
   - Should see `v0.0.1` (or `v0.1.0` if you used `feat:`)
   - Release notes auto-generated from commit

2. **Check Docker Image:**
   ```bash
   docker pull ghcr.io/YOUR_USERNAME/foodlist:latest
   docker pull ghcr.io/YOUR_USERNAME/foodlist:0.0.1
   ```

3. **Check Files Updated:**
   - `CHANGELOG.md` - Should have release entry
   - `VERSION` - Should match release version
   - Git tags - `v0.0.1` tag created

## 🛠️ Local Development Setup (Optional)

### Install Commitlint

For local commit message validation:

```bash
cd /Users/martin/foodlist
npm install --save-dev @commitlint/{config-conventional,cli} husky
npx husky install
npx husky add .husky/commit-msg 'npx --no -- commitlint --edit ${1}'
```

Now Git will validate commits before accepting them.

### Test Commit Format

Try making an invalid commit:
```bash
git commit -m "added feature"  # ❌ Should fail
git commit -m "feat: add new feature"  # ✅ Should succeed
```

## 🧪 Testing the Pipeline

### Test CI Workflow

Create a test branch:
```bash
git checkout -b test/ci-pipeline
echo "# Test" >> TEST.md
git add TEST.md
git commit -m "test: verify CI pipeline"
git push origin test/ci-pipeline
```

Create a Pull Request on GitHub:
- CI should run automatically
- All jobs should pass
- PR template should appear

### Test Release Workflow

Merge PR to main:
```bash
git checkout main
git merge test/ci-pipeline
git push origin main
```

Watch the release workflow:
- Should create a new patch version
- Should update CHANGELOG
- Should publish Docker images

## 📋 Common Issues

### Issue: Workflow doesn't run

**Solution:**
- Check: Settings → Actions → General
- Ensure Actions are enabled
- Check workflow permissions

### Issue: Release not created

**Solution:**
- Ensure commit messages follow conventional format
- Check if commits since last release contain releasable types
- View workflow logs for errors

### Issue: Docker push fails

**Solution:**
- Check workflow permissions (read and write)
- For private repos, set up PAT
- Check package visibility settings

### Issue: Permission denied errors

**Solution:**
- Settings → Actions → General
- Select "Read and write permissions"
- Save and re-run workflow

## 📚 Next Steps

After successful setup:

1. **Read Documentation:**
   - `CONTRIBUTING.md` - Contribution guidelines
   - `CI_CD_GUIDE.md` - Detailed pipeline docs
   - `COMMIT_QUICK_REFERENCE.md` - Quick commit guide

2. **Start Developing:**
   ```bash
   git checkout -b feat/my-feature
   # Make changes
   git commit -m "feat: add my feature"
   git push origin feat/my-feature
   ```

3. **Create Pull Request:**
   - CI runs automatically
   - Use PR template
   - Wait for review

4. **Merge to Main:**
   - Release happens automatically
   - Version bumped based on commits
   - Docker images published

## ✨ Summary

After completing this checklist:

✅ GitHub Actions workflows configured  
✅ Semantic versioning enabled  
✅ Automated releases working  
✅ Docker images publishing  
✅ Conventional commits enforced  
✅ CI/CD pipeline operational  

**Your repository is now ready for automated releases!**

---

## Quick Commands

```bash
# View current version
cat VERSION

# View changelog
cat CHANGELOG.md

# Validate commit message manually
./validate-commit.sh .git/COMMIT_EDITMSG

# Pull latest Docker image
docker pull ghcr.io/YOUR_USERNAME/foodlist:latest

# View workflow status
# Go to: https://github.com/YOUR_USERNAME/foodlist/actions
```

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Semantic Release](https://semantic-release.gitbook.io/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [GHCR Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)

