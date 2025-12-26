# Conventional Commits Guide

This project uses [Conventional Commits](https://www.conventionalcommits.org/) specification for commit messages. This allows us to automatically generate changelogs and determine semantic version bumps.

## Commit Message Format

Each commit message consists of a **header**, a **body**, and a **footer**. The header has a special format that includes a **type**, an optional **scope**, and a **subject**:

```
<type>(<scope>): <subject>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

### Type

Must be one of the following:

- **feat**: A new feature (triggers a MINOR version bump, e.g., 1.0.0 → 1.1.0)
- **fix**: A bug fix (triggers a PATCH version bump, e.g., 1.0.0 → 1.0.1)
- **docs**: Documentation only changes (triggers a PATCH version bump)
- **style**: Changes that do not affect the meaning of the code (white-space, formatting, etc.)
- **refactor**: A code change that neither fixes a bug nor adds a feature
- **perf**: A code change that improves performance (triggers a PATCH version bump)
- **test**: Adding missing tests or correcting existing tests
- **build**: Changes that affect the build system or external dependencies
- **ci**: Changes to our CI configuration files and scripts
- **chore**: Other changes that don't modify src or test files
- **revert**: Reverts a previous commit

### Scope (Optional)

The scope should be the name of the component affected (e.g., `backend`, `frontend`, `websocket`, `store`, `ui`, `docker`, etc.)

### Subject

The subject contains a succinct description of the change:

- use the imperative, present tense: "change" not "changed" nor "changes"
- don't capitalize the first letter
- no period (.) at the end

### Breaking Changes

Breaking changes must be indicated by:
1. A `!` after the type/scope
2. And/or a footer that starts with `BREAKING CHANGE:`

Breaking changes trigger a MAJOR version bump (e.g., 1.0.0 → 2.0.0)

## Examples

### Feature (minor version bump)

```
feat(backend): add structured logging support

Implemented log/slog with support for both logfmt and JSON formats.
Users can now configure LOG_FORMAT environment variable.
```

### Bug Fix (patch version bump)

```
fix(websocket): prevent connection leak on reconnect

Fixed an issue where WebSocket connections were not properly
closed before reconnecting, leading to memory leaks.

Closes #123
```

### Breaking Change (major version bump)

```
feat(api)!: change event format to include timestamps

BREAKING CHANGE: All events now include a mandatory timestamp field.
Clients must be updated to handle the new event structure.
```

Or alternatively:

```
feat(api): change event format to include timestamps

All events now include a mandatory timestamp field for better
event ordering and debugging.

BREAKING CHANGE: Clients must be updated to handle the new event structure.
```

### Documentation (patch version bump)

```
docs(readme): add network development mode instructions

Added documentation for the new dev-network make target
for testing on mobile devices.
```

### Chore (patch version bump)

```
chore(deps): update Go dependencies

Updated gorilla/websocket to v1.5.3 and other dependencies.
```

### Build System (patch version bump)

```
build(docker): optimize multi-stage build

Reduced final image size by 40% through better layer caching
and removal of unnecessary build dependencies.
```

### CI Changes (patch version bump)

```
ci: add semantic release workflow

Implemented automated versioning and changelog generation
using semantic-release and conventional commits.
```

### Multiple Scopes

When changes affect multiple components:

```
feat(backend,frontend): add todo categories support

- Backend: Added category field to todo events
- Frontend: Implemented category selector UI component

Closes #45
```

### Revert

```
revert: feat(backend): add experimental caching

This reverts commit 667ecc1654a317a13331b17617d973392f415f02.

The caching implementation caused issues with real-time updates.
```

## Version Bumping Rules

- **MAJOR** version (X.0.0): Breaking changes (`BREAKING CHANGE:` or `!`)
- **MINOR** version (0.X.0): New features (`feat:`)
- **PATCH** version (0.0.X): Bug fixes, docs, performance, and other changes

## Tips

1. **Write clear, descriptive commit messages**: Future you (and your team) will thank you
2. **One logical change per commit**: Makes it easier to review and revert if needed
3. **Use the body to explain *why***: The diff shows *what* changed, use the body to explain *why*
4. **Reference issues**: Use `Closes #123` or `Fixes #456` in the footer
5. **Keep the subject line under 72 characters**: For better readability in git logs

## Checking Your Commits

Before pushing, you can check if your commits follow the convention:

```bash
git log --oneline
```

Look for the pattern: `type(scope): subject`

## Resources

- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/)

