# CI/CD Pipeline Visual Overview

## 📁 Files Created

```
foodlist/
│
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                      ← Continuous Integration
│   │   └── release.yml                 ← Automated Releases
│   │
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.yml             ← Bug report template
│   │   └── feature_request.yml        ← Feature request template
│   │
│   └── pull_request_template.md       ← PR template
│
├── Configuration Files
│   ├── .releaserc.cjs                 ← Semantic-release config
│   ├── VERSION                        ← Current version (0.0.1)
│   ├── CHANGELOG.md                   ← Auto-generated changelog
│   ├── commitlint.config.js           ← Commit validation rules
│   └── .huskyrc.json                  ← Git hooks config
│
├── Documentation
│   ├── CONTRIBUTING.md                ← Contribution guidelines
│   ├── CI_CD_GUIDE.md                 ← Detailed pipeline docs
│   ├── CI_CD_SETUP_SUMMARY.md         ← Setup summary
│   ├── COMMIT_QUICK_REFERENCE.md      ← Quick commit guide
│   └── SETUP_CHECKLIST.md             ← First-time setup steps
│
├── Scripts
│   └── validate-commit.sh             ← Local commit validator
│
└── README.md (updated)                ← Added CI/CD section
```

## 🔄 CI Workflow (on Push/PR)

```
┌─────────────────────────────────────────────────────────────┐
│  Developer pushes code or creates PR                        │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  GitHub Actions: CI Workflow (.github/workflows/ci.yml)     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │   Backend    │  │   Frontend   │                        │
│  │    Tests     │  │    Tests     │                        │
│  │              │  │              │                        │
│  │  • Go 1.25   │  │  • Node 20   │                        │
│  │  • go test   │  │  • Vitest    │                        │
│  │  • Coverage  │  │  • Coverage  │                        │
│  └──────────────┘  └──────────────┘                        │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │     Lint     │  │    Docker    │                        │
│  │              │  │    Build     │                        │
│  │  • golangci  │  │              │                        │
│  │  • lint      │  │  • Validate  │                        │
│  └──────────────┘  └──────────────┘                        │
│                                                              │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
              ┌──────────────┐
              │  All Pass?   │
              └──────┬───────┘
                     │
        ┌────────────┴────────────┐
        │                         │
        ▼                         ▼
     ✅ SUCCESS                 ❌ FAILURE
     Continue to Release       Fix Issues
```

## 🚀 Release Workflow (on Push to main)

```
┌─────────────────────────────────────────────────────────────┐
│  Code merged to main branch                                 │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  CI Workflow Passes ✅                                       │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  GitHub Actions: Release Workflow                           │
│  (.github/workflows/release.yml)                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Step 1: Run Tests                                          │
│  ┌────────────────────────────────────────┐                │
│  │  • Backend tests                        │                │
│  │  • Frontend tests                       │                │
│  │  • Must pass to continue                │                │
│  └────────────────────────────────────────┘                │
│                     │                                        │
│                     ▼                                        │
│  Step 2: Semantic Release                                   │
│  ┌────────────────────────────────────────┐                │
│  │  Analyze Commits                        │                │
│  │  ├─ feat: → Minor version bump          │                │
│  │  ├─ fix:  → Patch version bump          │                │
│  │  └─ feat! → Major version bump          │                │
│  │                                          │                │
│  │  Determine Next Version                 │                │
│  │  ├─ Current: 0.0.1                      │                │
│  │  └─ Next:    0.1.0 (if feat)            │                │
│  │                                          │                │
│  │  Update Files                            │                │
│  │  ├─ CHANGELOG.md                        │                │
│  │  └─ VERSION                              │                │
│  │                                          │                │
│  │  Create Git Tag                          │                │
│  │  └─ v0.1.0                               │                │
│  │                                          │                │
│  │  Generate Release Notes                  │                │
│  │  └─ From commit messages                │                │
│  └────────────────────────────────────────┘                │
│                     │                                        │
│                     ▼                                        │
│  Step 3: Build Artifacts                                    │
│  ┌────────────────────────────────────────┐                │
│  │  Build Go Binaries                      │                │
│  │  ├─ linux-amd64                        │                │
│  │  ├─ linux-arm64                        │                │
│  │  ├─ darwin-amd64                       │                │
│  │  ├─ darwin-arm64 (Apple Silicon)       │                │
│  │  └─ windows-amd64.exe                  │                │
│  │                                          │                │
│  │  Build Frontend                          │                │
│  │  └─ npm run build → dist/              │                │
│  │                                          │                │
│  │  Create Archives                         │                │
│  │  ├─ .tar.gz for Unix                   │                │
│  │  ├─ .zip for Windows                   │                │
│  │  └─ Full package with frontend         │                │
│  └────────────────────────────────────────┘                │
│                     │                                        │
│                     ▼                                        │
│  Step 4: Build & Push Docker Images                        │
│  ┌────────────────────────────────────────┐                │
│  │  Multi-Platform Build                   │                │
│  │  ├─ linux/amd64                        │                │
│  │  └─ linux/arm64                        │                │
│  │                                          │                │
│  │  Tag Images                              │                │
│  │  ├─ ghcr.io/USER/foodlist:latest       │                │
│  │  └─ ghcr.io/USER/foodlist:0.1.0        │                │
│  │                                          │                │
│  │  Push to GitHub Container Registry      │                │
│  └────────────────────────────────────────┘                │
│                     │                                        │
│                     ▼                                        │
│  Step 5: Publish GitHub Release                            │
│  ┌────────────────────────────────────────┐                │
│  │  Create Release                          │                │
│  │  ├─ Title: v0.1.0                       │                │
│  │  ├─ Tag: v0.1.0                         │                │
│  │  ├─ Release Notes (auto-generated)     │                │
│  │  └─ Attach binary artifacts            │                │
│  └────────────────────────────────────────┘                │
│                                                              │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
              ✅ Release Complete!
```

## 📊 Version Bumping Logic

```
┌─────────────────────────────────────────────────────────────┐
│  Commits Since Last Release                                 │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
              Analyze Commit Types
                     │
        ┌────────────┴────────────┐
        │                         │
        ▼                         ▼
  Breaking Change?           No Breaking Change
  (feat! or BREAKING)              │
        │                    ┌─────┴─────┐
        ▼                    ▼           ▼
   MAJOR Bump          New Feature?   Bug Fix?
   X.0.0               (feat:)        (fix:)
   │                         │           │
   │                         ▼           ▼
   │                    MINOR Bump   PATCH Bump
   │                    X.Y.0        X.Y.Z
   │                         │           │
   └─────────┬───────────────┴───────────┘
             │
             ▼
      Create New Version
             │
             ▼
   ┌─────────────────────┐
   │  Current: 0.0.1     │
   │  Next:    0.1.0     │ ← Example with feat:
   │  Tag:     v0.1.0    │
   └─────────────────────┘
```

## 🔐 GitHub Settings Required

```
Repository Settings
│
├── Actions → General
│   ├── ✅ Workflow permissions
│   │   └── "Read and write permissions"
│   │
│   └── ✅ Allow GitHub Actions to:
│       └── "Create and approve pull requests"
│
├── Branches (Optional but Recommended)
│   └── Branch protection rules for 'main'
│       ├── ✅ Require pull request reviews
│       ├── ✅ Require status checks to pass
│       └── ✅ Require conversation resolution
│
└── Packages
    └── GitHub Container Registry
        └── Auto-configured (no setup needed for public repos)
```

## 🎯 Developer Workflow

```
┌─────────────────────────────────────────────────────────────┐
│  1. Create Feature Branch                                   │
│     git checkout -b feat/my-feature                         │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  2. Make Changes & Commit                                   │
│     git commit -m "feat(backend): add new feature"          │
│     ↓                                                        │
│     [optional] commitlint validates format                  │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  3. Push & Create PR                                        │
│     git push origin feat/my-feature                         │
│     ↓                                                        │
│     CI Workflow runs on PR                                  │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  4. Review & Merge                                          │
│     • Code review                                           │
│     • Tests must pass                                       │
│     • Merge to main                                         │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  5. Automatic Release                                       │
│     • CI runs on main                                       │
│     • Release workflow triggers                             │
│     • New version published                                 │
│     • Docker images pushed                                  │
└─────────────────────────────────────────────────────────────┘
```

## 📦 Release Artifacts

```
GitHub Release (v0.1.0)
│
├── Release Notes (auto-generated from commits)
│
├── Binaries
│   ├── foodlist-linux-amd64.tar.gz
│   ├── foodlist-linux-arm64.tar.gz
│   ├── foodlist-darwin-amd64.tar.gz
│   ├── foodlist-darwin-arm64.tar.gz
│   ├── foodlist-windows-amd64.zip
│   └── foodlist-v0.1.0-full.tar.gz (includes frontend)
│
└── Docker Images (GitHub Container Registry)
    ├── ghcr.io/USER/foodlist:latest
    └── ghcr.io/USER/foodlist:0.1.0
```

## 🌟 Benefits Summary

```
┌─────────────────────────────────────────────────────────────┐
│  BEFORE                    →    AFTER                       │
├─────────────────────────────────────────────────────────────┤
│  Manual version bumps      →    Automatic versioning       │
│  Manual changelog          →    Auto-generated changelog   │
│  Manual testing            →    Automated CI/CD            │
│  Manual releases           →    Push to main = release     │
│  Manual Docker builds      →    Auto build & push          │
│  Inconsistent commits      →    Standardized format        │
│  No release artifacts      →    Multi-platform binaries    │
└─────────────────────────────────────────────────────────────┘
```

## 🚦 Commit Type Examples

```
Type        Version     Example
────────────────────────────────────────────────────────────
feat:       MINOR       feat(ui): add dark mode
                        0.1.0 → 0.2.0

fix:        PATCH       fix(api): resolve timeout issue
                        0.1.0 → 0.1.1

feat!:      MAJOR       feat(api)!: change auth flow
                        0.1.0 → 1.0.0

docs:       NONE        docs: update README
                        no release

refactor:   NONE        refactor: simplify store logic
                        no release
```

## 📚 Documentation Files

```
For Users:
├── README.md                    → Project overview + CI/CD info
├── COMMIT_QUICK_REFERENCE.md    → Quick commit format guide
└── SETUP_CHECKLIST.md           → First-time setup steps

For Contributors:
├── CONTRIBUTING.md              → Contribution guidelines
├── CI_CD_GUIDE.md               → Detailed pipeline docs
└── CI_CD_SETUP_SUMMARY.md       → Technical setup details

For Developers:
├── commitlint.config.js         → Commit rules
├── .releaserc.cjs               → Release config
└── validate-commit.sh           → Local validator
```

---

**Current Status:** Ready for first release at version 0.0.1

**Next Step:** Push to GitHub to trigger first automated release
