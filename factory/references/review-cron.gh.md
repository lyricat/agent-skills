# Sample prompt: a review job beside the factory, through gh

A scheduled job that, using `gh` and no clone of the project, reviews the items in `.factory/review/` on the `factory_board` branch and records each review in the item file. It only reviews: it never edits code, merges, pushes branches or moves item files.

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

The job reads and writes the board with the GitHub API. Its token needs Contents: write and permission to push to `factory_board`, not to `base` (see `github-app.md`).

```
REPO=OWNER/REPO
CONTACT=<who to tell, and how: tg:@name, slack:#channel, or anything else>

Review the factory items in review/ of $REPO, using gh only (no local
clone of the project). Review only: never edit code, merge, push branches,
or move item files.

Anything that talks to GitHub goes through gh (gh api, gh repo clone,
gh auth status), never plain git or curl against github.com.

Let R=repos/$REPO. The board is on the branch factory_board. Read
.factory/factory.toml there for base, owners and timezone:
  gh api "$R/contents/.factory/factory.toml?ref=factory_board" --jq .content | base64 -d

1. Lock: mkdir -p ~/.factory/$REPO && mkdir ~/.factory/$REPO/review.lock
   (one lock per repository, apart from the factory's pass.lock).
   If it exists and was touched in the last 10 minutes, another run is
   reviewing: stop. If older, take it over. Then delete any leftover
   ~/.factory/$REPO/review-* directories from a stopped run. Touch the
   lock before and after each review; remove it when the run ends, also
   on failure.
2. List the items:
     gh api "$R/contents/.factory/review?ref=factory_board" --jq '.[] | select(.name|endswith(".md")) | .path'
   For each, read the file and keep its blob sha:
     gh api "$R/contents/<path>?ref=factory_board" --jq '.sha, .content'
   Take `branch` from its header and the branch head:
     gh api "$R/branches/<branch>" --jq .commit.sha
   Skip the item when:
   - its file already has a review section whose `head:` is that commit
     (a new head, after a fix round, is reviewed again); or
   - it waits: its last factory section is a question, with no section
     from a name in factory.toml's owners after it.
3. Make a throwaway checkout of the branch for context:
     gh repo clone "$REPO" ~/.factory/$REPO/review-<ID> -- --depth 50 --branch <branch> --single-branch
4. Review there with $review-code, run through $coder with Codex, against
   the item's request and plan. Give it the diff too:
     gh api -H "Accept: application/vnd.github.diff" "$R/compare/<base>...<head>"
   In the task you give Codex, say plainly that it only reads and
   reports: it changes no files, makes no commits, and never pushes,
   fetches or runs gh.
   Then delete the checkout.
5. Append one section at the end of the item file, signed with
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
6. Write the whole file back to factory_board in one commit, passing the blob sha
   you read:
     gh api -X PUT "$R/contents/<path>" -f branch=factory_board \
       -f message="review: FT16 pass" -f sha=<blob sha> \
       -f content="$(base64 -w0 < item.md)"
   A 409 or 422 means the file changed meanwhile: read it again, append
   your section to the new text, and retry, at most three times; then
   stop and say so. A 404 means the item left review/: skip it.
7. Messages:
   Before each review, tell $CONTACT that it starts (ID and title).
   After writing it, tell $CONTACT the verdict: pass, or fix with its
   P0/P1 findings. If a message cannot be sent, note it for the summary
   and go on. Format every message as the factory's
   references/messages.md says for that destination (Telegram, Slack, or
   plain text).
```

The blob `sha` does the job of `git pull --rebase`: if the factory changed the file in between, the write is refused instead of overwriting it.

To run it in a clone it keeps (`~/.factory/OWNER/REPO/clone-review`) instead, see `review-cron.md`.
