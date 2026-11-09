# Sample prompt: the review job

A scheduled job in which the manager runs one factory **review**: it takes one item from `review/`, has the coder review its branch, records the review in the item and on the linked GitHub issue or PR, and moves the item to `ready/` when there is no P0 or P1 finding. With one, the item goes back to `new/` for a fix round while its round is below `max_rounds` in `.factory/factory.toml`; at the limit it stays in `review/` and you decide: send it back to `new/` with a note to have it fixed.

Before using it:

- `gh` must act on the repository with the rights in `github-app.md`, including Issues and Pull requests: write for the comments. The job keeps its own clone in `~/.factory/OWNER/REPO/review/clone`.
- The coder should have the review-code skill; it reviews without it, less thoroughly.
- Schedule it every 30 minutes or so, beside the build job (`build-cron.md`) and the ship job (`ship-cron.md`). A run reviews one item.

## The prompt

The prompt holds only the run's inputs. Everything the run does is in the skill, so a change to the skill reaches every job on its next run; do not copy rules into the prompt. Fill in the values:

```
Run one factory review with $factory.
run: review
REPO: OWNER/REPO
contact: <who to tell, and how: tg:@name, slack:#channel, or anything else>
coder: $coder with Codex
```

One job per repository; only `REPO` (and perhaps `contact`) differs between them. Settings for one repository belong in that repository's `.factory/factory.toml`, not in the prompt.
