# Sample prompt: the ship job

A scheduled job in which the manager runs one factory **ship** on its own: it takes every item in `ready/` whose passing review still matches its branch, merges them into `base` one by one with the tests run after each, tags the result and pushes `base` and the tag. The items move to `done/`, and their linked issues and PRs are closed. An item that conflicts with `base` or fails its tests there goes back to `review/`, where the next review merges `base` into it and fixes it.

The actions come from `ship` in `.factory/factory.toml`, by default `["merge", "tag", "push"]`. Leave out `tag` to ship without tags; leave out `push` for a dry run that only reports what would ship; add `clean` to delete merged `factory/` branches. You can still ship by hand, in your own words, beside this job.

Before using it:

- `gh` must act on the repository with the rights in `github-app.md`: Contents write to push `base` and tags, and Issues and Pull requests write to close what the items link. Rules on `base` or on tags must let the App through. The job keeps its own clone in `~/.factory/OWNER/REPO/ship/clone`.
- Each run that merges something makes one tag, so the schedule sets how often you release: once or a few times a day suits most repositories.
- With this job, `ready/` no longer waits for you. To hold an item back, add a note to it (an owner section after its review) or move it to `stop/`.

## The prompt

The prompt holds only the run's inputs. Everything the run does is in the skill, so a change to the skill reaches every job on its next run; do not copy rules into the prompt. Fill in the values:

```
Run one factory ship with $factory.
run: ship
items: auto
REPO: OWNER/REPO
contact: <who to tell, and how: tg:@name, slack:#channel, or anything else>
```

One job per repository; only `REPO` (and perhaps `contact`) differs between them. Settings for one repository belong in that repository's `.factory/factory.toml`, not in the prompt.
