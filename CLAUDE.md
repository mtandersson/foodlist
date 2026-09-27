# Foodlist agent guidance

`AGENTS.md` links to this file. Read the root `README.md` before broad searches;
use `backend/CONFIG.md`, `frontend/README.md`, the schema, and source code for
details in the area being changed. Source and tests take precedence when a
document is stale.

## Commands and checks

- `make test` runs the Go and frontend suites.
- `make lint` runs the Go linter.
- `make build` builds the frontend and backend.
- CI also runs Go tests with the race detector, frontend type checking and
  coverage, a Docker build, and Go lint. Use the checks relevant to the change
  locally and confirm the PR's CI jobs before merging.
- For visual changes, inspect the rendered UI as well as its behavior.

## Issues and pull requests

Use GitHub issues as the unit of work. Keep each PR focused on one issue. When
an issue is too broad, create independently shippable sub-issues and use native
parent and blocked-by relationships to show the order. Check for an existing
issue before creating one for an unrelated finding; keep that finding out of
the current PR.

Use the existing `bug` or `enhancement` type label and the relevant component
in the issue body. Do not add priority labels without an agreed policy. Use
Conventional Commits as described in `CONTRIBUTING.md`; mark breaking changes
with `!` or a `BREAKING CHANGE:` footer. Independent tickets may run in
parallel, each in its own worktree and PR. Start a blocked ticket after its
dependency merges. A single-ticket run ends after its PR merges or the issue
has been decomposed.

The repository skills live in `.claude/skills` and are also available through
`.agents/skills` and `.codex/skills`:

- `issue-to-merge` for one ticket or parallel independent tickets through merge
- `adversarial-review` for an independent read-only review
- `writing-tests` when adding or changing behavior tests
- `grill-me` for a detailed design interview
