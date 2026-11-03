# Sample prompt: a review job beside the factory

A scheduled job, run in its own clone of the repository, that reviews the items in `.factory/review/` on the `factory_board` branch and records each review in the item file. It only reviews: it never edits code, merges, pushes branches or moves item files.

Before using it:

- Make sure the job can push to `factory_board` (see `github-app.md`); it never pushes to `base`.
- Add the reviewer's name (its `git config user.name`) to `owners` in `factory.toml`.
- Give the job a way to reach you (a chat or mail tool), or read its summary.
- Set `review = true` in `factory.toml`. The factory then moves a `review/` item to `ready/` only when the latest review of its current head says `verdict: pass`, and turns `verdict: fix` into a fix round on the `fix:` list. The fix makes a new head, which this job reviews again.

Fill in the two lines at the top of the prompt.

## The review section

The job appends this section. `head:` records the commit that was reviewed, so the same head is never reviewed twice and a new head after a fix round is reviewed again.

```markdown
## reviewer · review · 2026-11-05 10:00
head: 3f2a1c9e0b4d…
verdict: fix
fix:
- [P1] src/sync.go:88: the token is logged in plain text; drop it from the log line.
notes:
- [P3] src/sync.go:120: the retry delay never grows.
```

Only P0 and P1 findings block (`verdict: fix`); P2 and P3 go under `notes:`.

## The prompt

The job runs in its own clone, `~/.factory/OWNER/REPO/clone-review`, never in someone's working copy, and makes it with `gh repo clone` the first time. It is not the coder job's `clone` next to it: the recovery step below would clear what a pass left there. The name does not start with `review-`, so that step's cleanup of `review-*` worktrees leaves it alone.

```
REPO=OWNER/REPO
CONTACT=<who to tell, and how: tg:@name, slack:#channel, or anything else>

Review the factory items in .factory/review/ on the factory_board branch
of $REPO, in ~/.factory/$REPO/clone-review (a clone used only by this
job). If it does not exist, gh repo clone "$REPO"
~/.factory/$REPO/clone-review. Never use anyone's working copy. Review only: never edit code, merge, push
branches, or move item files.

If git has no credential helper here (for example with a GitHub App gh
wrapper), run every fetch, pull and push with gh as the helper, for that
command only:
  GH_REPO=$REPO git -c credential.helper= \
    -c 'credential.helper=!gh auth git-credential' push origin factory_board
Never ask gh for a token, or put one in a variable, file, remote URL or
git config.

1. Lock: mkdir -p ~/.factory/$REPO && mkdir ~/.factory/$REPO/review.lock
   (one lock per repository, apart from the factory's pass.lock).
   If it exists and was touched in the last 10 minutes, another run is
   reviewing: stop. If older, take it over. Touch it before and after
   each review; remove it when the run ends, also on failure.
2. Recover what a stopped run left, and say so in the summary: abort an
   unfinished rebase or merge; go back to factory_board; stash
   uncommitted changes as "review: left by a stopped run <time>"; remove leftover
   ~/.factory/$REPO/review-* worktrees (git worktree remove --force, then
   git worktree prune). Local commits on factory_board that are not on
   origin/factory_board: save them as refs/factory/stopped/<YYYYMMDD-HHMM>,
   then reset factory_board to origin/factory_board after the pull below.
3. git fetch --prune. Check out factory_board (the first time,
   git checkout -b factory_board origin/factory_board) and pull it.
   Read base from .factory/factory.toml. For each .factory/review/*.md,
   take `branch` from its header and its head:
   git rev-parse origin/<branch>.
   Skip the item when:
   - its file already has a review section whose `head:` is that commit
     (a new head, after a fix round, is reviewed again); or
   - it waits: its last factory section is a question, with no section
     from a name in factory.toml's owners after it.
4. Check the branch out in a temporary worktree, so the reviewer reads
   the branch's files:
     git worktree add --detach ~/.factory/$REPO/review-<ID> origin/<branch>
5. Review there with $review-code, run through $coder with Codex:
   origin/<base>...origin/<branch> against the item's request and plan.
   In the task you give Codex, say plainly that it only reads and
   reports: it changes no files, makes no commits, and never pushes,
   fetches or runs gh.
   Then remove the worktree:
     git worktree remove --force ~/.factory/$REPO/review-<ID>
6. Append one section at the end of the item file, signed with
   `git config user.name`, time in factory.toml's timezone:

   ## <name> · review · <YYYY-MM-DD HH:MM>
   head: <full commit hash>
   verdict: pass | fix
   fix:
   - [P0|P1] file:line: what is wrong and what to change
   notes:
   - [P2|P3] file:line: what is wrong

   verdict is fix when there is any P0 or P1 finding; only those go under
   fix:. Leave out empty lists. Never edit earlier sections.
7. Commit only that file ("review: FT16 pass" or "review: FT16 fix"),
   git pull --rebase, push to factory_board. Never push to base, or
   commit anything outside .factory/. If the push is refused three times, stop and say so.
8. Messages:
   Before each review, tell $CONTACT that it starts (ID and title).
   After writing it, tell $CONTACT the verdict: pass, or fix with its
   P0/P1 findings. If a message cannot be sent, note it for the summary
   and go on. Format every message as the factory's
   references/messages.md says for that destination (Telegram, Slack, or
   plain text).
```

To run it without a clone, through the GitHub API, see `review-cron.gh.md`.
