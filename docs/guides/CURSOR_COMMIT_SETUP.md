# Cursor IDE Semantic Commits Configuration

## What Was Configured

To make Cursor suggest semantic/conventional commits, the following files have been created:

### 1. `.cursorrules`
- Tells Cursor AI about your commit message convention
- Includes commit types, scopes, and examples
- Provides project context for better suggestions
- Cursor AI will read this file and suggest commits accordingly

### 2. `.vscode/settings.json`
- Configures VS Code / Cursor commit message convention
- Defines commit types with emojis (optional)
- Lists available scopes for the project
- Enables conventional commits mode

### 3. `.gitmessage`
- Git commit message template
- Shows up when you run `git commit` in terminal
- Provides inline examples and guidelines
- Already configured with `git config commit.template`

## How Cursor Will Suggest Commits

When you use Cursor's AI commit message feature (Cmd+K in the commit input), it will now:

1. **Analyze your changes** to determine the appropriate type and scope
2. **Suggest conventional commit messages** in the correct format
3. **Follow the project's conventions** defined in `.cursorrules`
4. **Include appropriate scopes** based on which files were changed

### Example Suggestions

**If you change backend files:**
```
feat(backend): add user preferences endpoint
fix(backend): resolve event store race condition
```

**If you change frontend files:**
```
feat(ui): implement dark mode toggle
fix(frontend): correct websocket reconnection logic
```

**If you change multiple areas:**
```
feat(backend,frontend): add real-time notifications
refactor(store,websocket): simplify state synchronization
```

**If you change documentation:**
```
docs(readme): update installation instructions
docs(api): document new endpoints
```

## Using Cursor's Commit Features

### Method 1: AI-Generated Commits (Recommended)

1. Stage your changes in Cursor
2. Click in the commit message input field
3. Press **Cmd+K** (Mac) or **Ctrl+K** (Windows/Linux)
4. Cursor AI will suggest a conventional commit message
5. Review and edit if needed
6. Commit!

### Method 2: Manual with Template

When committing from the terminal:

```bash
git commit
```

Your editor will open with the `.gitmessage` template showing:
- Format guidelines
- Available types
- Available scopes
- Examples

Just replace the template with your actual commit message.

### Method 3: Direct Command Line

```bash
git commit -m "feat(backend): add new feature"
```

## Commit Types Quick Reference

| Type | When to Use | Cursor Will Suggest For |
|------|-------------|-------------------------|
| `feat` | New features | New functions, components, endpoints |
| `fix` | Bug fixes | Bug fixes, error handling improvements |
| `docs` | Documentation | README, comments, markdown files |
| `style` | Code formatting | Prettier, linting fixes, whitespace |
| `refactor` | Code restructuring | Moving code, renaming, simplifying |
| `perf` | Performance | Optimizations, caching improvements |
| `test` | Tests | New tests, test updates |
| `build` | Build system | Package.json, go.mod, build scripts |
| `ci` | CI/CD | GitHub Actions, workflow changes |
| `chore` | Maintenance | Dependency updates, config changes |

## Scopes Quick Reference

Available scopes (use in parentheses):

- `backend` - Go backend code
- `frontend` - Svelte frontend code
- `ui` - User interface components
- `api` - API endpoints/contracts
- `websocket` - WebSocket functionality
- `store` - State management
- `docker` - Docker/containerization
- `ci` - CI/CD workflows
- `docs` - Documentation
- `tests` - Test files
- `e2e` - End-to-end tests

## Verification

To verify the configuration is working:

### 1. Check Git Template
```bash
git config commit.template
# Should output: .gitmessage
```

### 2. Test in Terminal
```bash
git commit
# Should open editor with template
```

### 3. Test Cursor AI
1. Make a small change to any file
2. Stage it in Cursor
3. In the commit message field, press **Cmd+K**
4. Cursor should suggest a conventional commit message

## Customization

### Add More Scopes

Edit `.cursorrules` and `.vscode/settings.json` to add project-specific scopes:

```json
"scopes": [
  "backend",
  "frontend",
  "your-new-scope"  // Add here
]
```

### Change Commit Types

Edit `.vscode/settings.json` to modify types:

```json
"types": [
  {
    "type": "feat",
    "description": "A new feature",
    "emoji": "✨"
  }
]
```

### Update Template

Edit `.gitmessage` to change the template text shown in terminal commits.

## Troubleshooting

### Cursor Not Suggesting Conventional Commits

1. **Restart Cursor IDE** - Configuration changes may require restart
2. **Check `.cursorrules` exists** - Cursor reads this file for context
3. **Try Cmd+K in commit field** - Make sure you're using the AI feature
4. **Update Cursor** - Ensure you have the latest version

### Git Template Not Working

```bash
# Reconfigure the template
git config commit.template .gitmessage

# Or set it globally
git config --global commit.template ~/.gitmessage
cp .gitmessage ~/.gitmessage
```

### Cursor Suggests Wrong Format

1. Check `.cursorrules` is properly formatted
2. Ensure no conflicting VS Code extensions
3. Clear Cursor cache: Cmd+Shift+P → "Developer: Reload Window"

## Additional Tools

### Enable Commit Validation

Install commitlint for automatic validation:

```bash
npm install --save-dev @commitlint/{config-conventional,cli} husky
npx husky install
npx husky add .husky/commit-msg 'npx --no -- commitlint --edit ${1}'
```

Now invalid commits will be rejected automatically.

### VS Code Extension (Optional)

Install "Conventional Commits" extension for additional UI support:
- Extension ID: `vivaxy.vscode-conventional-commits`
- Adds a UI picker for commit types and scopes

## Files Reference

```
foodlist/
├── .cursorrules              ← Cursor AI configuration
├── .gitmessage               ← Git commit template
├── .vscode/
│   └── settings.json         ← VS Code/Cursor settings
├── commitlint.config.js      ← Commit validation rules
└── validate-commit.sh        ← Manual validation script
```

## Documentation

For more details on conventional commits:
- `CONTRIBUTING.md` - Full guide with examples
- `COMMIT_QUICK_REFERENCE.md` - One-page cheat sheet
- `CI_CD_GUIDE.md` - How commits affect releases
- [Conventional Commits Website](https://www.conventionalcommits.org/)

---

**Status:** ✅ Cursor is now configured to suggest semantic commits!

**Next:** Start committing with Cursor AI (Cmd+K) and see conventional commits in action.

