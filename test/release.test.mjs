import assert from "node:assert/strict"
import test from "node:test"
import {analyzeCommits} from "@semantic-release/commit-analyzer"
import {generateNotes} from "@semantic-release/release-notes-generator"
import config from "../.releaserc.cjs"

const analyzer = config.plugins.find(([name]) => name === "@semantic-release/commit-analyzer")[1]
const notesGenerator = config.plugins.find(([name]) => name === "@semantic-release/release-notes-generator")[1]

function context(messages) {
  return {
    commits: messages.map((message, index) => ({
      hash: String(index + 1).repeat(40),
      message,
    })),
    lastRelease: {version: "1.0.0", gitTag: "v1.0.0"},
    nextRelease: {version: "1.1.0", gitTag: "v1.1.0", type: "minor"},
    options: {repositoryUrl: "https://github.com/mtandersson/foodlist"},
    cwd: process.cwd(),
    env: {},
    logger: {log() {}, error() {}},
  }
}

test("release rules preserve feature, fix, UI style, and dependency decisions", async () => {
  for (const [message, expected] of [
    ["feat: add menu", "minor"],
    ["fix: repair list", "patch"],
    ["style(ui): improve spacing", "patch"],
    ["chore(deps): update package", null],
    ["fix(deps): update package", null],
    ["feat(deps): update UI library", null],
    ["perf(deps): update runtime library", null],
    ["fix(ci): repair pipeline", null],
    ["fix(build): repair bundler", null],
    ["fix(release): adjust version policy", null],
    ["refactor: simplify event projection", null],
    [`Revert "chore(deps): update package"\n\nThis reverts commit ${"a".repeat(40)}.`, null],
    [`Revert "refactor: simplify event projection"\n\nThis reverts commit ${"b".repeat(40)}.`, null],
    [`Revert "refactor(store): simplify projection"\n\nThis reverts commit ${"d".repeat(40)}.`, null],
    [`Revert "style(layout): format markup"\n\nThis reverts commit ${"f".repeat(40)}.`, null],
    [`Revert "feat: add menu"\n\nThis reverts commit ${"c".repeat(40)}.`, "patch"],
    [`Revert "style(ui): improve spacing"\n\nThis reverts commit ${"e".repeat(40)}.`, "patch"],
    ["feat!: change API\n\nBREAKING CHANGE: clients must update", "major"],
  ]) {
    assert.equal(await analyzeCommits(analyzer, context([message])), expected, message)
  }
})

test("maintenance-only commit ranges do not publish a release", async () => {
  for (const messages of [
    ["chore(deps): update package", "ci: update release action"],
    ["fix(deps): update runtime package", "refactor: simplify event projection"],
    ["docs: clarify setup", "build: update compiler"],
    ["fix(release): adjust version policy", "fix(ci): repair pipeline"],
  ]) {
    assert.equal(await analyzeCommits(analyzer, context(messages)), null, messages.join(", "))
  }
})

test("user-visible changes release even with maintenance commits", async () => {
  for (const [messages, expected] of [
    [["chore(deps): update package", "fix: repair list"], "patch"],
    [["refactor: simplify event projection", "feat: add menu"], "minor"],
    [["chore(deps): update package", "feat!: change API\n\nBREAKING CHANGE: clients must update"], "major"],
    [["fix(deps)!: remove old API\n\nBREAKING CHANGE: clients must update"], "major"],
  ]) {
    assert.equal(await analyzeCommits(analyzer, context(messages)), expected, messages.join(", "))
  }
})

test("release notes render configured feature and fix sections", async () => {
  const notes = await generateNotes(notesGenerator, context([
    "feat: add menu",
    "fix: repair list",
  ]))

  assert.match(notes, /### Features\n[\s\S]*add menu/)
  assert.match(notes, /### Bug Fixes\n[\s\S]*repair list/)
})

test("breaking releases explain the required client change", async () => {
  const notes = await generateNotes(notesGenerator, context([
    "feat!: change API\n\nBREAKING CHANGE: clients must update",
  ]))

  assert.match(notes, /BREAKING CHANGES\n[\s\S]*clients must update/)
})
