// Semantic-release configuration
// - `chore` commits do NOT trigger a release on push to `main`.
//   They accumulate on `main` and are rolled up into a single patch release
//   once per week by `.github/workflows/release-weekly.yml`, which creates
//   an empty `fix(deps): weekly dependency release` commit.
const releaseRules = [
  {type: "feat", release: "minor"},
  {type: "fix", release: "patch"},
  {type: "perf", release: "patch"},
  {type: "revert", release: "patch"},
  {type: "docs", release: false},
  {type: "style", scope: "ui", release: "patch"},
  {type: "style", release: false},
  {type: "refactor", release: "patch"},
  {type: "test", release: false},
  {type: "build", release: false},
  {type: "ci", release: false},
  {type: "chore", release: false},
  {breaking: true, release: "major"},
]

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
    // No release assets: the deployable artifact is the multi-arch Docker
    // image published by the build-and-push job in release.yml. The GitHub
    // release just carries the tag + changelog.
    "@semantic-release/github",
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
