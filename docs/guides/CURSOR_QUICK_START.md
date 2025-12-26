# 🚀 Quick Start: Cursor Semantic Commits

## ✅ Already Configured!

Cursor IDE is now set up to suggest semantic/conventional commits automatically.

## How to Use

### 1. Make Your Changes
Edit files as usual in Cursor.

### 2. Stage Your Changes
Use Cursor's Git panel to stage files (Cmd+Shift+G).

### 3. Get AI Commit Suggestion
1. Click in the **commit message** input field
2. Press **Cmd+K** (Mac) or **Ctrl+K** (Windows/Linux)
3. Cursor AI will suggest a properly formatted commit message
4. Review, edit if needed, and commit!

## Example Suggestions You'll See

**Backend changes:**
```
feat(backend): add user preferences endpoint
fix(backend): resolve event store race condition
```

**Frontend changes:**
```
feat(ui): implement dark mode toggle
fix(frontend): correct websocket reconnection
```

**Documentation:**
```
docs(readme): update installation guide
```

**Tests:**
```
test(backend): add integration tests
```

## Commit Format

Cursor will suggest commits in this format:
```
<type>(<scope>): <description>
```

### Types
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `refactor` - Code refactoring
- `test` - Tests
- `ci` - CI/CD changes
- `chore` - Maintenance

### Scopes
- `backend`, `frontend`, `ui`, `api`, `websocket`, `store`, `docker`, `ci`, `docs`, `tests`, `e2e`

## Breaking Changes

For breaking changes, Cursor might suggest:
```
feat(api)!: change response format

BREAKING CHANGE: All endpoints now return timestamps.
```

## Manual Override

You can always type your own commit message. Just follow the format:
```
feat(scope): your description here
```

## Validation

All commits are checked by CI/CD. Invalid format will:
- Not affect local commits
- May skip release creation
- Won't fail CI (but release won't trigger)

## More Info

- **Full Guide:** `CURSOR_COMMIT_SETUP.md`
- **Commit Examples:** `COMMIT_QUICK_REFERENCE.md`
- **Contributing:** `CONTRIBUTING.md`

---

**That's it!** Just use Cmd+K in the commit field and let Cursor suggest semantic commits. 🎉

