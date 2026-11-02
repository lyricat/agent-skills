# Sample prompt: a coder job that runs the factory, through gh

A scheduled job that, using `gh` for access, runs one factory pass with Codex as the coding agent. A pass numbers new items, asks when a request is unclear, builds at most one clear item and moves it to `review/`. It also makes fix rounds from review notes, moves reviewed items to `ready/` and merged ones to `done/`.

Building needs a git checkout, so the job keeps its own clone in the directory it runs in, made with `gh repo clone` the first time. Git authenticates through `gh` (`gh auth setup-git`, set up once by whoever runs the scheduler).

Run the whole pass, not only `new/`: the fix rounds that a review job's `verdict: fix` asks for happen here too (see `review-cron.gh.md`).

Before using it:

- Log `gh` in on the machine with a token that can push to `factory/**` branches and to `base` (see `github-app.md`), and run `gh auth setup-git` once.
- Nothing to set up for builds: the clone stays on `base` for board commits, and each build gets its own checkout of its branch in `~/.factory/OWNER/REPO/`, so Codex's files never touch the clone and a build cut short continues there on the next pass. The factory creates and removes these folders itself.
- Give the job a way to reach you (a chat or mail tool), or read its summary.
- Schedule it every 30 minutes or so. A pass builds at most one new item, and the factory's lock keeps two passes from overlapping.

Fill in the two lines at the top of the prompt.

## The prompt

```
REPO=OWNER/REPO
CONTACT=<who to tell, and how>

1. gh auth status must succeed; if not, stop and say so. If ./<repo name>
   does not exist, gh repo clone "$REPO" it. Never use anyone's working
   copy.
2. Run one factory pass on $REPO with $factory, with repo: ./<repo name>.
3. For every build and fix round, use $coder with Codex as the coding
   agent. Codex gets no push or GitHub credentials, including gh's
   stored login (the factory runs it with an empty GH_CONFIG_DIR). Only
   the factory pushes.
4. Follow the factory's rules: build at most one new item per pass; when
   a request is unclear, ask in the item instead of guessing.
5. At the end, if an item reached ready/ or is waiting on a question,
   tell $CONTACT: each item's ID, title and what it needs, one line each.
```

To run it in a clone the scheduler already keeps, see `coder-cron.md`.
