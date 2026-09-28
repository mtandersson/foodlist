import {appendFileSync} from "node:fs"
import semanticRelease from "semantic-release"

const outputFile = process.env.GITHUB_OUTPUT
if (!outputFile) {
  throw new Error("GITHUB_OUTPUT is required to pass release results to the Docker job")
}

const result = await semanticRelease()
appendFileSync(outputFile, `new_release_published=${Boolean(result)}\n`)

if (result) {
  const {version, gitHead} = result.nextRelease
  if (!version || !gitHead) {
    throw new Error("semantic-release did not provide a version and git head")
  }

  appendFileSync(outputFile, `new_release_version=${version}\nnew_release_git_head=${gitHead}\n`)
}
