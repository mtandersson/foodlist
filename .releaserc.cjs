// Semantic-release configuration
// - Default behavior: do NOT publish releases for `chore` commits
// - Daily chore release workflow: set DAILY_CHORE_RELEASE=true to publish a patch
//   release even when only `chore(...)` commits happened since the last release.
const dailyChoreRelease = process.env.DAILY_CHORE_RELEASE === "true"

const baseReleaseRules = [
  {type: "feat", release: "minor"},
  {type: "fix", release: "patch"},
  {type: "perf", release: "patch"},
  {type: "revert", release: "patch"},
  {type: "docs", release: false},
  {type: "style", release: false},
  {type: "refactor", release: "patch"},
  {type: "test", release: false},
  {type: "build", release: "patch"},
  {type: "ci", release: false},
  {breaking: true, release: "major"},
]

const releaseRules = dailyChoreRelease
  ? [...baseReleaseRules, {type: "chore", release: "patch"}]
  : baseReleaseRules

module.exports = {
  branches: ["main"],
  plugins: [
    [
      "@semantic-release/commit-analyzer",
      {
        preset: "conventionalcommits",
        releaseRules,
      },
    ],
    [
      "@semantic-release/release-notes-generator",
      {
        preset: "conventionalcommits",
        presetConfig: {
          types: [
            {type: "feat", section: "Features"},
            {type: "fix", section: "Bug Fixes"},
            {type: "perf", section: "Performance Improvements"},
            {type: "revert", section: "Reverts"},
            {type: "docs", section: "Documentation"},
            {type: "style", section: "Styles"},
            {type: "refactor", section: "Code Refactoring"},
            {type: "test", section: "Tests"},
            {type: "build", section: "Build System"},
            {type: "ci", section: "Continuous Integration"},
            {type: "chore", section: "Chores"},
          ],
        },
      },
    ],
    ["@semantic-release/changelog", {changelogFile: "CHANGELOG.md"}],
    [
      "@semantic-release/exec",
      {prepareCmd: "echo ${nextRelease.version} > VERSION"},
    ],
    [
      "@semantic-release/github",
      {
        assets: [
          {path: "dist/*.tar.gz", label: "Binary archives"},
          {path: "dist/*.zip", label: "Binary archives (Windows)"},
        ],
      },
    ],
    [
      "@semantic-release/git",
      {
        assets: ["CHANGELOG.md", "VERSION"],
        message:
          "chore(release): ${nextRelease.version} [skip ci]\n\n${nextRelease.notes}",
      },
    ],
  ],
  tagFormat: "v${version}",
}
