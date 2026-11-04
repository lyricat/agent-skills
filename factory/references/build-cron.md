# Sample prompt: the build job

A scheduled job in which the manager runs one factory **build**: it takes one item from `new/`, has the coder build it (or fix it, when it comes back from a review) on its own branch, pushes the branch, moves the item to `review/` and tells you each step.

Before using it:

- `gh` must act on the repository with the rights in `github-app.md`. Nothing to clone by hand: the job keeps its own clone in `~/.factory/OWNER/REPO/build/clone`, and each item gets its own worktree beside it.
- Give the job a way to reach you (a chat or mail tool), or read its summary.
- Schedule it every 30 minutes or so. A run builds one item; the lock keeps two builds from overlapping. The review job (`review-cron.md`) runs beside it in its own directory.

## The prompt

The prompt holds only the run's inputs. Everything the run does is in the skill, so a change to the skill reaches every job on its next run; do not copy rules into the prompt. Fill in the values:

```
Run one factory build with $factory.
run: build
REPO: OWNER/REPO
contact: <who to tell, and how: tg:@name, slack:#channel, or anything else>
coder: $coder with Codex
```

One job per repository; only `REPO` (and perhaps `contact`) differs between them. Settings for one repository (its test command, rules, paths it must not change) belong in that repository's `.factory/factory.toml`, not in the prompt.
