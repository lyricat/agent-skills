# Messages to people

How the manager words and formats a message to the contact: a build or review starting or finishing, an install, a failing test, a review asking for fixes, a ship, a run summary. The text is the same everywhere; only the markup changes with where it goes.

## Content

- One item per message block: the ID and title on the first line, then what happened and what the person should do, in a line or two.
- Name the item's branch when it has one. Link it on GitHub (`https://github.com/OWNER/REPO/compare/<base>...<branch>`) when the remote is on GitHub.
- A summary lists one block per item, the ones that need the person first.
- No secrets, tokens, machine paths or long logs. Quote at most one failing test line.
- Keep the person's language: write in the language of the owner's sections in the item.

## Pick the format from the destination

| Destination | Format |
|---|---|
| Telegram (`tg:`, a Telegram chat or tool) | Markdown, when the tool renders it (such as mistermorph's `contacts_send`); otherwise Telegram MarkdownV2 |
| Slack (`slack:`, a Slack channel or tool) | Slack mrkdwn |
| Anything else: mail, `notify-send`, a log, the run summary | Plain text |

A tool that renders Markdown for Telegram takes ordinary Markdown: do not add MarkdownV2 escapes, or the backslashes show. A tool that takes a parse mode instead gets MarkdownV2, with the parse mode set to `MarkdownV2`. If the message is refused for its markup, send it once more as plain text.

## Telegram Markdown

For a tool that renders Markdown. Bold `**text**`, italic `_text_`, code `` `text` ``, link `[text](url)`, list items start with `- `. No escaping, except a backslash before a character that would otherwise start markup, such as `\*` for a literal asterisk. No headings or tables: Telegram shows neither.

```
**FT16** Search box in the toolbar
Build started on `factory/FT16-search-box`
```

```
**FT16** Search box in the toolbar
Pushed, tests passing. Now in `review/`: [read the branch](https://github.com/OWNER/REPO/compare/master...factory/FT16-search-box)
```

```
**FT17** Export as CSV
Review asks for fixes:
- [P1] `src/export.go:42`: the header row is written twice.
```

## Telegram MarkdownV2

For a tool that sends with the parse mode set to `MarkdownV2`.

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
