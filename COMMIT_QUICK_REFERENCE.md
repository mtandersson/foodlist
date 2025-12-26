# Quick Reference: Conventional Commits

## Commit Format

```
<type>(<scope>): <subject>
```

## Common Types

| Type | When to Use | Version Bump | Example |
|------|-------------|--------------|---------|
| `feat` | New feature | MINOR (0.1.0 → 0.2.0) | `feat(api): add user endpoint` |
| `fix` | Bug fix | PATCH (0.1.0 → 0.1.1) | `fix(ui): button alignment` |
| `docs` | Documentation | PATCH | `docs: update README` |
| `style` | Code formatting | PATCH | `style: fix indentation` |
| `refactor` | Code restructuring | PATCH | `refactor(store): simplify logic` |
| `perf` | Performance | PATCH | `perf: optimize queries` |
| `test` | Tests | PATCH | `test: add unit tests` |
| `build` | Build system | PATCH | `build: update webpack` |
| `ci` | CI/CD | PATCH | `ci: add new workflow` |
| `chore` | Maintenance | PATCH | `chore: update deps` |

## Breaking Changes

Add `!` after type or include `BREAKING CHANGE:` in footer:

```bash
feat(api)!: change endpoint structure

BREAKING CHANGE: All endpoints now require authentication.
```

**Version Bump:** MAJOR (0.1.0 → 1.0.0)

## Scopes

Common scopes for this project:
- `backend` - Go backend code
- `frontend` - Svelte frontend code
- `ui` - User interface components
- `api` - API changes
- `websocket` - WebSocket functionality
- `store` - State management
- `docker` - Docker/deployment
- `ci` - CI/CD pipelines
- `docs` - Documentation

## Examples

```bash
# Features
feat(backend): add structured logging
feat(ui): implement dark mode
feat(websocket): add auto-reconnect

# Bug Fixes
fix(store): prevent race condition
fix(ui): resolve CSS overflow issue
fix(api): correct response format

# Breaking Changes
feat(api)!: change event structure
feat(backend)!: require authentication

# Documentation
docs(readme): add installation guide
docs(api): document new endpoints

# Multiple files
feat(backend,frontend): add categories
fix(ui,store): sync state correctly
```

## Validation

Before committing:
```bash
# Manually validate
./validate-commit.sh .git/COMMIT_EDITMSG

# Or install commitlint
npm install --save-dev @commitlint/{config-conventional,cli} husky
npx husky install
```

## Tips

✅ Use imperative mood: "add" not "adds" or "added"  
✅ Don't capitalize first letter: "add feature" not "Add feature"  
✅ No period at end: "add feature" not "add feature."  
✅ Keep subject under 72 characters  
✅ Use body for detailed explanations  

## Full Example

```bash
git commit -m "feat(backend): add user preferences API

Implemented CRUD operations for user preferences.
Includes validation and error handling.

Closes #123"
```

## Resources

- `CONTRIBUTING.md` - Full guide
- `CI_CD_GUIDE.md` - Pipeline details
- [conventionalcommits.org](https://www.conventionalcommits.org/)

