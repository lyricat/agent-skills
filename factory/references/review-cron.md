# Sample prompt: the review job

A scheduled job in which the manager runs one factory **review**: it takes one item from `review/`, has the coder review its branch, records the review in the item and on the linked GitHub issue or PR, and moves the item to `ready/` when there is no P0 or P1 finding. With one, the item stays in `review/` and you decide: send it back to `new/` with a note to have it fixed.

Before using it:

- `gh` must act on the repository with the rights in `github-app.md`, including Issues and Pull requests: write for the comments. The job keeps its own clone in `~/.factory/OWNER/REPO/review/clone`.
- The coder should have the review-code skill; it reviews without it, less thoroughly.
- Schedule it every 30 minutes or so, beside the build job (`build-cron.md`). A run reviews one item.

Fill in the two lines at the top of the prompt.

## The prompt

```
REPO=OWNER/REPO
CONTACT=<who to tell, and how: tg:@name, slack:#channel, or anything else>

Run one factory review on $REPO with $factory (run: review, contact:
$CONTACT). You are the manager; use $coder with Codex as the coder.

Anything that talks to GitHub goes through gh: cloning (gh repo clone),
the API, issues and PRs, auth checks (gh auth status). Local work stays
git: commits, worktrees, merges. Run every fetch, pull and push with gh
as the credential helper, for that command only:
  GH_REPO=$REPO git -c credential.helper= \
    -c 'credential.helper=!gh auth git-credential' push origin <branch>
gh mints the token and hands it to git; never ask gh for a token
yourself, or put one in a variable, file, remote URL or git config.

1. Take one item from review/ whose branch head has no review yet, as
   the factory says. None: end quietly.
2. Tell $CONTACT before the review starts: the item's ID and title.
3. If the branch conflicts with the base branch, have the coder merge
   it in and resolve the conflict, then push the branch. Do not stop.
4. Have the coder review with $review-code. In its task, say plainly:
   report findings as [P0-P3] file:line; run the tests (a failure is
   P1); install missing project dependencies and user-level tools
   (no sudo) and list them; change no files, make no commits, never
   push, fetch or run gh.
5. Write the review into the item; comment on its linked GitHub issues
   and PRs; with no P0 or P1, move it to ready/. Push factory_board.
6. Tell $CONTACT: the verdict, and either "in ready/" or the P0 and P1
   findings for you to decide. Mention installs too.
7. Format every message as the factory's references/messages.md says
   for that destination. If one cannot be sent, put it in the summary
   and go on.
```
