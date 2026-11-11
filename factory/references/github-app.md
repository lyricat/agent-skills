# GitHub App access

How to give an agent that runs the factory the least GitHub access it needs. The agent works through a GitHub App: a wrapper fetches short-lived installation tokens for `gh` and acts as git's credential helper, so the agent never holds a long-lived token.

## Installation

Install the App on the factory's repository only ("Only select repositories"), not on all repositories of the account.

## Repository permissions

| Permission | Access | Why |
|---|---|---|
| Contents | Read and write | Fetch; push board commits to `factory_board` and `factory/` branches; push `base` and a new tag in a ship run. |
| Issues | Read and write | Read an item's source issue; comment on it after a review; close it after a ship. |
| Pull requests | Read and write | The same for PRs, and fetch a PR's head to build on. |
| Metadata | Read-only | Required by GitHub for every App. |

Leave every other permission at "No access". In particular:

- **Workflows**: no access. GitHub then refuses any push that changes `.github/workflows/`, which enforces that `never` entry on the server.
- **Administration**, **Secrets**, **Actions**, **Environments**: no access.

The factory opens no issues or PRs: the board is files. Without Issues and Pull requests write, its comments and closes are refused, and it says so in the summary; the items still move.

No account or organization permissions are needed, and the App needs no webhook.

## Branch rules

Set rules on the repository (Settings → Rules → Rulesets) so that the server also enforces what the skill forbids. Bypass works per ruleset, so keep these separate.

1. **`base` only by ship.** A ship run merges into `base` and pushes it, so if `base` requires pull requests or status checks, add the App to that ruleset's bypass list. A ship also pushes a `vX.Y.Z` tag (unless `ship` in `factory.toml` leaves out `tag`): if a tag ruleset restricts creating such tags, add the App to its bypass list too, but keep "Restrict updates" and "Restrict deletions" on tags without bypass. Build and review runs never push `base`; the skill enforces that, not the server. Do not require signed commits on `base`, `factory_board` or `factory/**`: the factory's commits are unsigned.
2. **No force-push, no deletion.** A ruleset targeting `base`, `factory_board` and `factory/**` with "Block force pushes", and "Restrict deletions" on `base` and `factory_board`, with no bypass for the App. With `clean` in `ship`, a ship deletes merged `factory/` branches: keep "Restrict deletions" for `factory/**` in a ruleset of its own, with the App on its bypass list. Without `clean`, add `factory/**` to the first one.
3. **Protected paths** (push rulesets, where the plan offers them). Restrict file paths such as `.factory/factory.toml` and `.github/**`, with no bypass for the App. A push ruleset applies to every branch, so it cannot say "only `.factory/` on `factory_board`"; the skill's own check (Writing to the board, step 4) covers that.

Do not require linear history on `base` or `factory/**`: a ship merges with merge commits, and builds merge `base` into a branch.

## Keeping the token from the coding agent (optional)

The skill runs the coding agent and the tests in the factory's own environment: their task tells them not to push, and the factory checks their commits before it pushes. For a stricter setup, where they cannot push even if they try, the runner starts them like this:

- Unset `GH_TOKEN`, `GITHUB_TOKEN` and any variable the wrapper reads.
- Disable git's credential helper for them, so a `git push` in the worktree fails. Worktrees share the repository's git config, so set it in the environment: `GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=credential.helper GIT_CONFIG_VALUE_0=` (an empty value clears the helper list).
- Keep the `gh` wrapper off their `PATH` where the runner allows it.

## Checking the setup

From a throwaway clone, as the App:

- `git push` of a commit that changes only `.factory/` to `factory_board`: accepted.
- `git push` of a merge commit to `base`: accepted.
- `git push` of a new `v*` tag: accepted (delete the test tag afterwards, as yourself).
- `git push --force` to `base`, `factory_board` or a `factory/` branch: refused.
- `git push origin --delete` of `base` or `factory_board`: refused. Of a test `factory/` branch: accepted with `clean` in `ship`, refused without.
- A push that changes `.github/workflows/`: refused.
- With the stricter setup above: `git push` from a build worktree with the coding agent's environment fails for lack of credentials.
