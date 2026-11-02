# Sample prompt: a coder job that runs the factory

A scheduled job, run in a clone of the repository, that runs one factory pass with Codex as the coding agent. A pass numbers new items, asks when a request is unclear, builds at most one clear item and moves it to `review/`. It also makes fix rounds from review notes, moves reviewed items to `ready/` and merged ones to `done/`.

Run the whole pass, not only `new/`: the fix rounds that a review job's `verdict: fix` asks for happen here too (see `review-cron.md`).

Before using it:

- Run the job in its own clone of the repository, never a working copy someone edits.
- Nothing to set up for builds: the clone stays on `base` for board commits, and each build gets its own checkout of its branch in `~/.factory/OWNER/REPO/`, so Codex's files never touch the clone and a build cut short continues there on the next pass. The factory creates and removes these folders itself.
- Give the job a way to reach you (a chat or mail tool), or read its summary.
- Schedule it every 30 minutes or so. A pass builds at most one new item, and the factory's lock keeps two passes from overlapping.

Fill in the two lines at the top of the prompt.

## The prompt

```
REPO=OWNER/REPO
CONTACT=<who to tell, and how>

Run one factory pass on $REPO with $factory, in this repository (a clone
used only by the factory).

For every build and fix round, use $coder with Codex as the coding agent.
Follow the factory's rules: build at most one new item per pass; when a
request is unclear, ask in the item instead of guessing; Codex gets no
push or GitHub credentials, only the factory pushes.

At the end, if an item reached ready/ or is waiting on a question,
tell $CONTACT: each item's ID, title and what it needs, one line each.
```

To run it without a clone kept by the scheduler, through `gh`, see `coder-cron.gh.md`.
