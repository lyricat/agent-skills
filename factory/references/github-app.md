# GitHub App access

How to give an agent that runs the factory the least GitHub access it needs. The agent works through a GitHub App: a wrapper fetches short-lived installation tokens for `gh` and acts as git's credential helper, so the agent never holds a long-lived token.

## Installation

Install the App on the factory's repository only ("Only select repositories"), not on all repositories of the account.

## Repository permissions

| Permission | Access | Why |
|---|---|---|
| Contents | Read and write | Fetch; push board commits to `base`; push `factory/` branches. |
| Metadata | Read-only | Required by GitHub for every App. |

Leave every other permission at "No access". In particular:

- **Workflows**: no access. GitHub then refuses any push that changes `.github/workflows/`, which enforces that `never` entry on the server.
- **Pull requests**, **Issues**: no access. The factory opens no PRs and writes no issues; the board is files.
- **Administration**, **Secrets**, **Actions**, **Environments**: no access.

Add a permission only when a repository setting needs it, such as a `notify` command that comments on an issue (Issues: write).

No account or organization permissions are needed, and the App needs no webhook.

## Branch rules

Set rules on the repository (Settings → Rules → Rulesets) so that the server also enforces what the skill forbids. Bypass works per ruleset, so keep these separate.

1. **Board pushes to `base`.** If `base` requires pull requests or status checks, add the App to that ruleset's bypass list; the factory commits `.factory/` changes straight to `base`. Do not require signed commits on `base`: the factory's commits are unsigned.
2. **No force-push, no deletion.** A ruleset targeting `base` and `factory/**` with "Block force pushes" and "Restrict deletions", with no bypass for the App.
3. **Protected paths** (push rulesets, where the plan offers them). Restrict file paths such as `.factory/factory.toml` and `.github/**`, with no bypass for the App. A push ruleset applies to every branch, so it cannot say "only `.factory/` on `base`"; the skill's own check (Writing to the board, step 4) covers that.

Do not require linear history on `factory/**`: fix rounds merge `base` into the branch.

## Keeping the token from the coding agent (optional)

The skill runs the coding agent and the tests in the factory's own environment: their task tells them not to push, and the factory checks their commits before it pushes. For a stricter setup, where they cannot push even if they try, the runner starts them like this:

- Unset `GH_TOKEN`, `GITHUB_TOKEN` and any variable the wrapper reads.
- Disable git's credential helper for them, so a `git push` in the worktree fails. Worktrees share the repository's git config, so set it in the environment: `GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=credential.helper GIT_CONFIG_VALUE_0=` (an empty value clears the helper list).
- Keep the `gh` wrapper off their `PATH` where the runner allows it.

## Checking the setup

From a throwaway clone, as the App:

- `git push` of a commit that changes only `.factory/` to `base`: accepted.
- `git push --force` to `base` or a `factory/` branch: refused.
- A push that changes `.github/workflows/`: refused.
- With the stricter setup above: `git push` from a build worktree with the coding agent's environment fails for lack of credentials.
