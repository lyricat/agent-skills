# Worked examples: the mistermorph Console

The recipes of `details-craft.md`, and the tooltips, hover, page entrance, glide, focus and type
rules of `ux-rules.md` and `ui-rules.md`, are implemented in the mistermorph Console, a Vue 3 app
(template-string components, no SFCs). Paths below are relative to
`/home/lyric/Codework/arch/mistermorph/web/console/src`. The design write-up, with the reasoning
and the measurements, is `/home/lyric/Codework/arch/mistermorph/docs/feat/feat_20261023_console_quiet_details.md`.

The later rules (pressed state, popup show/hide, caret spacing, control heights, unselectable
labels, one cue per state, safe areas) are not implemented there yet; use the recipes in the
chapter files. When working in that repo, reuse what is listed below rather than writing new
ones. When working elsewhere, read the file for a tested implementation to adapt.

| Technique | Code | Styles | Used on |
| --- | --- | --- | --- |
| Device-pixel lines (bubble tail) | `core/chat-bubble-tail.js` (`borderWidth`, `chatBubbleTail`, `installChatBubbleTail`), test `core/chat-bubble-tail.test.mjs` | `views/ChatView.css` (`--chat-tail-*` masks on `::before`/`::after`) | user chat bubbles |
| Sliding selection | `core/sliding-selection.js` (`v-sliding-selection`, selector argument, default `.is-active`) | `styles/base.css` (`.has-sliding-selection`, `.sliding-selection`) | sidebar nav, Topics, Settings index, Skills, Contacts, TODO |
| Scroll edges | `core/scroll-edges.js` (`v-scroll-edges`) | `styles/base.css` (`.has-scroll-edges`, `.is-scrolled`, `.has-more-end`; pinned-head rule) | sidebar, Topics, chat history, Settings, Skills, Contacts, TODO, Audit |
| Fade truncation | `core/fade-truncate.js` (`v-fade-truncate`, one shared ResizeObserver) | `styles/base.css` (`.fade-truncate.is-truncated`) | topic titles, contact details, agent switcher |
| Cut-outs | (CSS only) | `views/OverviewView.css` (status notch, `mask-composite: exclude`), `views/ContactsView.css` (round badge cut-out) | Overview avatars, Contacts |
| Dialog origin | `core/dialog-origin.js` (`installDialogOrigin`) | `styles/base.css` (`.q-dialog.is-from-source`, scale 0.9 vs 0.98) | every `QDialog` |
| Page entrance | `core/page-transitions.js` (installed from the router; a Web Animation, no CSS); pending-route selection in `composables/useAppShell.js` | |  top-level page changes |
| Rolling digits | `components/RollingNumber.js` + `.css`, `core/rolling-wheel.js` (`wheelStep`) | `components/RollingNumber.css` (`clip-path` window) | Overview readouts, Usage totals |
| Tooltips | `core/tooltips.js` (`installTooltips`, `splitShortcut`) | `styles/base.css` (`.app-tooltip`, `.app-tooltip-keys kbd`) | every `title` |
| Digits, line breaks, hover, focus, selection, scrollbars | (CSS only) | `styles/base.css` | global |
| Composer growth, glide to sent message | `components/ChatComposer.js`, `views/ChatView.js` (`scrollHistoryToBottom({ smooth: true })`) | `components/ChatComposer.css` | chat |
| Shared tests | `core/ui-details.test.mjs` | | |

Installation: directives are registered and the `install*` functions called in `main.js`.

Conventions in that codebase worth keeping when adding a detail there:

- A comment at the top of each module says what the browser does that made the obvious version
  wrong, and the usage in one line.
- Motion durations come from tokens (`--motion-fast`, `--motion-normal`, `--motion-slow`,
  `--motion-ease`) that reduced motion sets to 0, so JS reads the token instead of hard-coding.
- The UI kit (`components/ui/`, `styles/ui-kit.css`) is local, derived from quail-ui; change it
  there instead of overriding with `!important`.
- Type scale tokens: `--font-size-label` 11, `-meta` 12, `-ui` 14, `-body` 16, `-heading` 18,
  `-title` 20, `-display` 32; families `--font-ui`, `--font-mono`, `--font-cjk`.
- Frontend tests: `node --test` over `src/**/*.test.mjs`; build with `pnpm build` (pnpm only).
