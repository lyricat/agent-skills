# Sample prompt: a coder job that runs the factory, through gh

A scheduled job that, using `gh` for access, runs one factory pass with Codex as the coding agent. A pass numbers new items, asks when a request is unclear, builds at most one clear item and moves it to `review/`. It also makes fix rounds from review notes, moves reviewed items to `ready/` and merged ones to `done/`.

Building needs a git checkout, so the job keeps its own clone in `~/.factory/OWNER/REPO/clone`, next to the build worktrees, made with `gh repo clone` the first time. Git has no stored login: each fetch, pull and push uses `gh auth git-credential` as its credential helper, for that command only, so `gh` mints a fresh token each time and the job never sees it.

Run the whole pass, not only `new/`: the fix rounds that a review job's `verdict: fix` asks for happen here too (see `review-cron.gh.md`).

Before using it:

- Make sure `gh` can act on the repository with rights to push to `factory/**` branches and to `factory_board`, and none to push to `base` (see `github-app.md`). A GitHub App wrapper works as long as it passes `gh auth git-credential` through; nothing else about `gh` login is used.
- Nothing to set up for builds: the clone's own checkout holds the `factory_board` branch for board commits, and each build gets its own checkout of its branch in `~/.factory/OWNER/REPO/`, so Codex's files never touch the clone and a build cut short continues there on the next pass. The factory creates and removes these folders itself.
- Give the job a way to reach you (a chat or mail tool), or read its summary.
- Schedule it every 30 minutes or so. A pass builds at most one new item, and the factory's lock keeps two passes from overlapping.

Fill in the two lines at the top of the prompt.

## The prompt

```
REPO=OWNER/REPO
CONTACT=<who to tell, and how: tg:@name, slack:#channel, or anything else>

Anything that talks to GitHub goes through gh: cloning (gh repo clone),
the API, branches and auth checks (gh api, gh auth status). Local work
stays git: commits, worktrees, merges. git has no credential helper here:
run every fetch, pull and push with gh as the helper, for that command
only, as the factory's Access rules show:
  GH_REPO=$REPO git -c credential.helper= \
    -c 'credential.helper=!gh auth git-credential' push origin <branch>
gh mints the token and hands it to git; never ask gh for a token
yourself, or put one in a variable, file, remote URL or git config.

1. gh auth status must succeed; if not, stop and say so. If
   ~/.factory/$REPO/clone does not exist,
   gh repo clone "$REPO" ~/.factory/$REPO/clone. Never use anyone's
   working copy.
2. Run one factory pass on $REPO with $factory, with
   repo: ~/.factory/$REPO/clone.
3. For every build and fix round, use $coder with Codex as the coding
   agent. In the task you give Codex, say plainly that it must commit
   its work on the branch but never push, fetch or run gh; only the
   factory pushes.
4. Each time, before calling $coder, tell $CONTACT: the item's ID
   and title, and whether it is a build or fix round N. When
   that build or fix round is finished (verified and pushed, or stopped
   with a question), tell $CONTACT again: the item's ID, the
   outcome (pushed, tests passing; or waiting on a question) and where
   the item is now. If a message cannot be sent, note it for the summary
   and go on.
5. Follow the factory's rules: build at most one new item per pass; when
   a request is unclear, ask in the item instead of guessing.
6. At the end, if an item reached ready/ or is waiting on a question,
   tell $CONTACT: each item's ID, title and what it needs.
7. Format every message to $CONTACT as the factory's references/messages.md
   says for that destination (Telegram, Slack, or plain text).
```

When git already has a credential helper for the repository, see `coder-cron.md`.
