# Messages to people

How the factory and its jobs word and format a notice: a build starting or finishing, an item reaching `ready/`, a review asking for fixes, a pass summary. The text is the same everywhere; only the markup changes with where it goes.

## Content

- One item per message block: the ID and title on the first line, then what happened and what the person should do, in a line or two.
- Name the item's branch when it has one. Link it on GitHub (`https://github.com/OWNER/REPO/compare/<base>...<branch>`) when the remote is on GitHub.
- A summary lists one block per item, the ones that need the person first.
- No secrets, tokens, machine paths or long logs. Quote at most one failing test line.
- Keep the person's language: write in the language of the owner's sections in the item.

## Pick the format from the destination

| Destination | Format |
|---|---|
| Telegram (`tg:`, a Telegram chat or tool) | Telegram MarkdownV2 |
| Slack (`slack:`, a Slack channel or tool) | Slack mrkdwn |
| Anything else: mail, `notify-send`, a log, the run summary | Plain text |

When a tool takes a parse mode, set it (`MarkdownV2` for Telegram). If the message is refused for its markup, send it once more as plain text.

## Telegram MarkdownV2

Bold `*text*`, italic `_text_`, code `` `text` ``, link `[text](url)`. Outside code, escape each of these with a backslash: `_ * [ ] ( ) ~ ` > # + - = | { } . !`. Inside code, escape only `` ` `` and `\`. Inside a link's `(url)`, escape only `)` and `\`.

```
*FT16* Search box in the toolbar
Build started on `factory/FT16-search-box`
```

```
*FT16* Search box in the toolbar
Pushed, tests passing\. Now in `review/`: [read the branch](https://github.com/OWNER/REPO/compare/master...factory/FT16-search-box)
```

```
*FT17* Export as CSV
Review asks for fixes:
• \[P1\] `src/export.go:42`: the header row is written twice\.
```

## Slack mrkdwn

Bold `*text*`, italic `_text_`, code `` `text` ``, link `<url|text>`, list items start with `•`. Escape only `&`, `<` and `>` as `&amp;`, `&lt;` and `&gt;`. Slack has no headings or tables.

```
*FT16* Search box in the toolbar
Pushed, tests passing. Now in `review/`: <https://github.com/OWNER/REPO/compare/master...factory/FT16-search-box|read the branch>
```

## Plain text

No markup. Put links as bare URLs on their own line.

```
FT16 Search box in the toolbar
Pushed, tests passing. Now in review/.
https://github.com/OWNER/REPO/compare/master...factory/FT16-search-box
```
