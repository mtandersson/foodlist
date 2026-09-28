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
    ["feat!: change API\n\nBREAKING CHANGE: clients must update", "major"],
  ]) {
    assert.equal(await analyzeCommits(analyzer, context([message])), expected, message)
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
