# Sample prompt: a review job beside the factory

A scheduled job, run in a clone of the repository, that reviews the items in `.factory/review/` and records each review in the item file. It only reviews: it never edits code, merges, pushes branches or moves item files.

Before using it:

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

The job runs in its own clone, never in someone's working copy.

```
REPO=OWNER/REPO
CONTACT=<who to tell, and how>

Review the factory items in review/ of $REPO (this git repository is a
clone of it). Review only: never edit code, merge,
push branches, or move item files.

Use $review-code, run through $coder with Codex.

1. git pull. For each .factory/review/*.md, take `branch` from its header
   and its head: git rev-parse origin/<branch>.
2. Skip the item if its file already has a review section whose
   `head:` is that commit. A new head (after a fix round) is reviewed again.
3. Review origin/<base>...origin/<branch> against the item's request and
   plan (base is in .factory/factory.toml).
4. Append one section at the end of the item file, signed with
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
5. Commit only that file ("review: FT16 pass" or "review: FT16 fix"),
   git pull --rebase, push to base. Never commit anything outside
   .factory/. If the push is refused three times, stop and say so.
6. If any verdict is fix, tell $CONTACT: each item's ID, title and its
   P0/P1 findings, one line each.
```

To run it without a clone, through the GitHub API, see `review-cron.gh.md`.
