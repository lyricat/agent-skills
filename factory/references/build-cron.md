# Sample prompt: the build job

A scheduled job in which the manager runs one factory **build**: it takes one item from `new/`, has the coder build it (or fix it, when it comes back from a review) on its own branch, pushes the branch, moves the item to `review/` and tells you each step.

Before using it:

- `gh` must act on the repository with the rights in `github-app.md`. Nothing to clone by hand: the job keeps its own clone in `~/.factory/OWNER/REPO/build/clone`, and each item gets its own worktree beside it.
- Give the job a way to reach you (a chat or mail tool), or read its summary.
- Schedule it every 30 minutes or so. A run builds one item; the lock keeps two builds from overlapping. The review job (`review-cron.md`) runs beside it in its own directory.

Fill in the two lines at the top of the prompt.

## The prompt

```
REPO=OWNER/REPO
CONTACT=<who to tell, and how: tg:@name, slack:#channel, or anything else>

Run one factory build on $REPO with $factory (run: build, contact:
$CONTACT). You are the manager; use $coder with Codex as the coder.

Anything that talks to GitHub goes through gh: cloning (gh repo clone),
the API, issues and PRs, auth checks (gh auth status). Local work stays
git: commits, worktrees, merges. Run every fetch, pull and push with gh
as the credential helper, for that command only:
  GH_REPO=$REPO git -c credential.helper= \
    -c 'credential.helper=!gh auth git-credential' push origin <branch>
gh mints the token and hands it to git; never ask gh for a token
yourself, or put one in a variable, file, remote URL or git config.

1. Take one item from new/, as the factory says. None: end quietly.
2. Tell $CONTACT before the coder starts: the item's ID and title, and
   whether it is a build or fix round N.
3. In the task you give the coder, say plainly: finish the work on your
   own without asking; merge the base branch in first and resolve any
   conflict; install missing project dependencies and user-level tools
   (no sudo) and list them; fix failing tests; commit on the branch,
   but never push, fetch or run gh.
4. Tell $CONTACT, without waiting for an answer: what the coder
   installed, and any test that still fails.
5. Push the branch, move the item to review/, push factory_board, then
   tell $CONTACT: ID, title, branch link, test result, now in review/.
6. Format every message as the factory's references/messages.md says
   for that destination. If one cannot be sent, put it in the summary
   and go on.
```
