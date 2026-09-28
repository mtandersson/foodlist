import assert from "node:assert/strict"
import {execFileSync} from "node:child_process"
import {mkdtempSync, rmSync, writeFileSync} from "node:fs"
import {tmpdir} from "node:os"
import {join} from "node:path"
import {Writable} from "node:stream"
import semanticRelease from "semantic-release"
import config from "../.releaserc.cjs"

function git(cwd, ...args) {
  return execFileSync("git", args, {cwd, stdio: "pipe"}).toString().trim()
}

const base = mkdtempSync(join(tmpdir(), "foodlist-release-"))
try {
  const remote = join(base, "remote.git")
  const repo = join(base, "repo")

  git(base, "init", "--bare", remote)
  git(base, "init", "-b", "main", repo)
  git(repo, "config", "user.name", "Release Test")
  git(repo, "config", "user.email", "release@example.test")
  writeFileSync(join(repo, "README.md"), "initial\n")
  git(repo, "add", "README.md")
  git(repo, "commit", "-m", "chore: initial")
  git(repo, "tag", "v1.0.0")
  writeFileSync(join(repo, "README.md"), "initial\nfixed\n")
  git(repo, "add", "README.md")
  git(repo, "commit", "-m", "fix: repair list")
  writeFileSync(join(repo, "README.md"), "initial\nfeature\n")
  git(repo, "add", "README.md")
  git(repo, "commit", "-m", "feat: add menu")
  git(repo, "remote", "add", "origin", `file://${remote}`)
  git(repo, "push", "-u", "origin", "main", "--tags")

  const logs = []
  const silent = new Writable({write(chunk, _encoding, callback) { logs.push(chunk.toString()); callback() }})
  // The temporary repository is on main even when the outer CI job checks out a PR ref.
  const env = Object.fromEntries(Object.entries(process.env).filter(([name]) => !name.startsWith("GITHUB_")))
  const result = await semanticRelease(
    {
      branches: ["main"],
      repositoryUrl: `file://${remote}`,
      plugins: config.plugins.slice(0, 2),
      dryRun: true,
      ci: false,
    },
    {cwd: repo, env, stdout: silent, stderr: silent},
  )

  assert.ok(result?.nextRelease, logs.join(""))
  assert.equal(result.nextRelease.version, "1.1.0")
  assert.match(result.nextRelease.notes, /### Features\n[\s\S]*add menu/)
  assert.match(result.nextRelease.notes, /### Bug Fixes\n[\s\S]*repair list/)
  console.log("semantic-release dry run generated feature and fix notes")
} finally {
  rmSync(base, {recursive: true, force: true})
}
