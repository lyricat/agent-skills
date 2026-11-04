# Messages

Every message the manager sends, as a Markdown template: to the contact, and as comments on GitHub. Edit the wording here; the manager fills in the placeholders and sends the result as it is.

## How to fill them

- `{name}` is replaced by its value. A line holding a placeholder with no value is dropped, such as `{link}` when the remote is not on GitHub.
- A line starting `- ` under a heading such as "for each finding" is repeated, once per item.
- Send ordinary Markdown. If the contact's tool refuses the markup, send the same text again as plain text.
- Keep the owner's language: translate the template into the language of the owner's sections in the item.
- No secrets, tokens, machine paths or long logs. Quote at most one failing test line.

Placeholders:

| Placeholder | Value |
|---|---|
| `{repo}` | the repository, `OWNER/REPO` |
| `{id}`, `{title}` | the item's ID and title |
| `{branch}` | the item's branch |
| `{link}` | the branch on GitHub: `https://github.com/OWNER/REPO/compare/<base>...<branch>` |
| `{round}` | the fix round number |
| `{base}` | the base branch |
| `{folder}` | the folder an item went to |
| `{run}` | `build`, `review` or `ship` |
| `{verdict}` | `pass` or `fix` |
| `{what happened}` | the item's move or result, in a few words |
| `{sha}` | a short commit hash |
| `{source}` | the linked GitHub issue or PR, such as `issue #47` |
| `{reason}` | why, in a sentence |
| `{test}` | the failing test, one line |
| `{install}`, `{why}` | what was installed, with which command, and why |
| `{finding}` | one review finding: `[P1] file:line: what` |
| `{ref}` | a branch or ref the remote refused |

## Build run

### Build started

```markdown
👨‍💻 {repo}
**{id} {title}**
Build started on `{branch}`.
```

### Fix round started

```markdown
👨‍💻 {repo}
**{id} {title}**
Fix round {round} started on `{branch}`.
```

### Installed

For each install:

```markdown
🛠️ {repo}
**{id} {title}**
installed `{install}`
reason: {why}
```

### Install needs sudo

```markdown
🛠️ {repo}
**{id} {title}**
needs `{install}`, which needs sudo; not installed.
reason: {why}
```

### Tests still failing

```markdown
⚠️ {repo}
**{id} {title}**
Tests still fail after a retry; going on, the review will see it:
`{test}`
```

### Built

```markdown
🏗️ {repo}
**{id} {title}**
Pushed, tests pass. Now in `review/`.
[Read the branch]({link})
```

### Built, tests failing

```markdown
🏗️⚠️ {repo}
**{id} {title}**
Pushed with a failing test. Now in `review/`.
`{test}`
[Read the branch]({link})
```

### Moved to backlog

When the item does not parse, cannot be built at all, or tries to change the factory's rules:

```markdown
⚠️ {repo}
**{id} {title}**
Moved to `backlog/`.
reason: {reason}
```

## Review run

### Review started

```markdown
🧐 {repo}
**{id} {title}**
Review started.
```

### Conflict resolved

```markdown
🧐 {repo}
**{id} {title}**
`{base}` conflicted with the branch; merged and resolved, pushed `{sha}`.
```

### Installed

As in the build run.

### Passed

```markdown
✅ {repo}
**{id} {title}**
Review passed. Now in `ready/`.
[Read the branch]({link})
```

### Needs fixes

For each P0 and P1 finding:

```markdown
⛔ {repo}
**{id} {title}**
Review found P0/P1; it stays in `review/` for you to decide.
- {finding}
To fix it: add a note and move it to `new/`.
[Read the branch]({link})
```

## Ship run

### Merged

```markdown
✅ {repo}
**{id} {title}**
Merged into `{base}` at `{sha}`. Now in `done/`.
Closed: {source}
```

### Sent back

```markdown
⚠️ {repo}
**{id} {title}**
Back in `{folder}` with your note.
```

### Merge conflict not resolved

```markdown
⚠️ {repo}
**{id} {title}**
Conflicts with `{base}` the coder could not resolve; still in `ready/`.
{reason}
```

### Tests failing on base

```markdown
⛔ {repo}
**Ship stopped**
tests fail on the merged `{base}`; nothing pushed, the items stay in `ready/`.
`{test}`
```

## Any run

### Run failed

```markdown
⛔ {repo}
Factory {run} run stopped: {reason}
```

### Push refused

```markdown
⛔ {repo}
Factory {run} run stopped: the remote refused `{ref}` ({reason}). Check the GitHub App's permissions and rules.
```

### Summary

At the end of a run, when anything happened. One line per item, those waiting for you first:

```markdown
🏭 {repo}
**Factory {run}**
- **{id}** {title}: {what happened}
```

## GitHub comments

### Review comment on a linked issue or PR

```markdown
Factory review of **{id}** on `{branch}` (`{sha}`): **{verdict}**
- {finding}
```

### Close comment

```markdown
Done in **{id}**: merged into `{base}` at `{sha}`.
```
