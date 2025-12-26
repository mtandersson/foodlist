#!/usr/bin/env bash

# Validate commit message format
# This script checks if a commit message follows conventional commits format

set -e

COMMIT_MSG_FILE=$1
COMMIT_MSG=$(cat "$COMMIT_MSG_FILE")

# Conventional commit pattern
PATTERN="^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+\))?(!)?: .{1,}"

# Check if message matches pattern
if ! echo "$COMMIT_MSG" | grep -Eq "$PATTERN"; then
    echo "❌ Invalid commit message format!"
    echo ""
    echo "Your commit message:"
    echo "---"
    echo "$COMMIT_MSG"
    echo "---"
    echo ""
    echo "Expected format:"
    echo "  <type>(<scope>): <subject>"
    echo ""
    echo "Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"
    echo ""
    echo "Examples:"
    echo "  feat(backend): add user authentication"
    echo "  fix(ui): resolve button alignment issue"
    echo "  docs(readme): update installation instructions"
    echo "  feat(api)!: change endpoint structure (breaking change)"
    echo ""
    echo "See CONTRIBUTING.md for more details."
    exit 1
fi

echo "✅ Commit message format is valid"

