---
name: ui-craft
description: Fit and finish for web UIs, the craft details that look simple and are hard to get right. Hairlines that meet on device pixels at any zoom, a selection highlight that glides between items (FLIP), scroll edges and long names that fade instead of clipping or "…", menus, dropdowns and dialogs that animate both in and out (with the caret turning alongside and spaced evenly from the edge), buttons that answer a press on touch and mouse, phone layouts that clear the status bar and home indicator (safe areas), controls in one row that share a height (and a button inside a field that is smaller by an even inset), dialogs that grow out of the control that opened them, rolling digits, calm hover, labels on buttons and tabs that don't get selected or change weight when selected, focus rings, tooltips, tabular digits, line breaking (incl. CJK), optical alignment, and keeping all of it cheap while the page re-renders. Use this skill whenever the user wants a UI to feel more polished, finished, refined, "less janky", quieter or more native; asks about micro-interactions, motion, jitter, flicker, blurry or uneven lines, clipped focus rings, truncation, or animation smoothness; reviews a frontend for polish; says a screen is busy, cluttered or repeats itself, or asks to remove labels, marks or decoration; or is building a component (list, sidebar, chat bubble, dialog, menu, dropdown, button, tooltip, counter, switch) or a mobile/PWA/WebView screen and cares how it looks and moves, even if they never say "craft" or "polish".
---

# UI craft

Craft details are the ones a user never names but always feels: a line that is a shade paler where
two parts meet, a highlight that jumps instead of moving, a fade that flickers as scrolling starts.
Each is small; together they decide whether an interface feels made or assembled. They are hard
because the obvious implementation is almost right, and the last bit lives in how the browser
rasterises, composites and lays out.

The skill is in four chapters. Each has a few rules, a catalogue (symptom → fix in one line) and a
reference file with the recipes, code and the reason each naive version fails. Read the entry in
the reference before implementing it.

| Chapter | About | Checked by | Reference |
| --- | --- | --- | --- |
| 1. UI rules | How it looks at rest: what is shown, how often, on which scale, where | A still screenshot | `references/ui-rules.md` |
| 2. UX rules | How it answers input: press, hover, open and close, navigate | Doing it (mouse, touch, keyboard) and sampling frames | `references/ux-rules.md` |
| 3. Performance | Keeping every detail smooth and cheap while the page is busy | A trace under re-render pressure | `references/performance.md` |
| 4. Details craft | The solved problems: pixel-exact lines, FLIP glides, fades, digits | Screenshots at several densities, per-frame samples | `references/details-craft.md` |

The worked examples come from one codebase (the mistermorph Console, a Vue app); the techniques
carry to any framework.

Every rule in the tables below has an ID and a slug, such as `UI-09` `icon-label-balance`. Either
one names the rule: use them in audits, findings, code comments and commit messages, and look a
rule up by searching for its ID. IDs are permanent. A new rule takes the next free number in its
chapter (`UI`, `UX`, `PF`, `DC`); a removed rule's ID is not reused, and rules are not renumbered
when the reference files are reordered.

## 1. UI rules

1. **Say each thing once.** One surface level: panels divided by space and one hairline, not
   boxes in boxes. No label that repeats its context (a job's name on that job's own page), no
   word beside a mark that already says it ("Running" next to a spinner), no decoration without
   a meaning (a shadow says "floats", so nothing set into the page has one). Before adding a
   detail, look for a louder one it should replace; before keeping one, cover it and ask what the
   reader would lose.
2. **One cue per state, by colour or fill.** A selected row is a soft fill and a stronger text
   colour; never also a frame, a dot, a coloured line down its left edge, or a change of font
   weight.
3. **Everything on a scale.** Type sizes, weights and control heights come from short token
   scales. Controls that can share a row share a height; a control inset in another is smaller by
   an even gap; spacing around an icon is measured from its ink, not its box.
4. **Leave the system its room.** On phones, content clears the status bar and the home
   indicator, while the bars' fills run to the edge.

| Rule | Detail | Symptom | Fix |
| --- | --- | --- | --- |
| `UI-01` `say-once` | Say each thing once | A label repeats its context, a word sits beside a mark that already says it, boxes sit in boxes | Cover each label, mark and decoration and ask what the reader would lose; remove what the context or another mark already says, and decoration without a meaning |
| `UI-02` `no-side-line` | No side line on selection | A selected nav item or row has a coloured bar down its left edge as well as a fill | Remove the bar (border, inset shadow or pseudo-element); the soft fill and text colour are the selected state |
| `UI-03` `same-weight` | Same weight in every state | Selecting a tab or nav item makes it bold, so it widens and its neighbours shift | One `font-weight` for normal, hover, pressed, selected, active and current; show the state with colour, a fill or an indicator (sliding highlight, underline, check) |
| `UI-04` `type-scale` | Type scale | Two sans faces, stray sizes and weights | One sans, one mono, a short size scale (e.g. 11–32), three weights; weights belong to kinds of text, not states |
| `UI-05` `tabular-digits` | Tabular digits | Live numbers twitch; columns don't line up | `font-variant-numeric: tabular-nums` on interface text; keep proportional digits in prose |
| `UI-06` `line-breaks` | Line breaks | One-word last lines; CJK punctuation starting a line | `text-wrap: pretty` for paragraphs, `balance` for titles, `line-break: strict`, `word-break: auto-phrase` for Japanese titles |
| `UI-07` `control-heights` | Control heights | A search field and its button are 36px and 34px; a button inside a field covers its border | Explicit heights from a size scale (`--control-sm/md/lg`, `box-sizing: border-box`); controls that can share a row use the same token; a control inset in a field is the field height − 2 × inset, radius − inset; raise the whole scale on touch |
| `UI-08` `caret-spacing` | Caret spacing | A dropdown's chevron is jammed against the border, or its right side looks wider than the left; the label runs under it | Gap from the caret's ink (not its icon box) to the right border = the text's left padding; label truncates a gap before it; reserve the room in a styled `<select>`'s padding; logical properties for RTL |
| `UI-09` `icon-label-balance` | Icon and label balance | A button with an icon and a label has 16px on both sides and still looks lopsided: the icon floats off-centre in the round end, the label's end looks tight | Icon's end: the same gap above, below and beside the icon (concentric with the round end), and at least that to the label; label's end: padding + what the round end takes away (0.107 × height on a pill), so the room looks equal, whatever the numbers say |
| `UI-10` `focus-rings` | Focus rings | Rings clipped by scrolling lists; inconsistent rings | One ring (e.g. 1.5px, accent 55%, offset 2px, follows corners); inside the edge for rows in scrollers |
| `UI-11` `selection-scrollbars` | Selection, caret, scrollbars | Browser-default blue selection, fat scrollbars | Soft accent `::selection`, accent `caret-color`; thin thumbs shown on hover/focus with a stable gutter so nothing shifts |
| `UI-12` `safe-areas` | Safe areas | On a phone the header sits under the clock, the tab bar under the home indicator | `viewport-fit=cover`; `--safe-*` vars from `env(safe-area-inset-*)` (overridable by a native shell); bars' fills run to the edge, their content is padded; `100dvh`; drop the bottom inset while the keyboard is open |

## 2. UX rules

1. **Every input answers at once**, on touch as on mouse: a press shows on its own frame, and
   motion starts on the input (the click, not when the next page's code has loaded).
2. **What appears also leaves.** Menus, dropdowns, popovers and dialogs animate in and out; a
   second input reverses from the current animated state rather than snapping.
3. **Nothing waits on motion.** The action runs on the click, content is never held behind an
   animation, and `prefers-reduced-motion` zeroes durations.
4. **Controls are not text.** Their labels don't get selected, hover is calm and absent on touch,
   and tooltips show on keyboard focus too.

| Rule | Detail | Symptom | Fix |
| --- | --- | --- | --- |
| `UX-01` `pressed-state` | Pressed state | A press gets no answer, or only the grey tap flash; taps too quick to see | `:active` + `.is-pressed` held ≥100ms; in ~40ms, out ~160ms; scale 0.97 on compact controls only, fill change on all; no tap highlight; works on iOS via a touch listener; cancel on scroll |
| `UX-02` `unselectable-labels` | Unselectable labels | Double-click or long-press on a button or tab highlights its label (iOS opens a callout) | `user-select: none` (+ `-webkit-` prefix) and `-webkit-touch-callout: none` on buttons, tabs, menu items, segmented items, chips, nav items, switch labels; never on content; a `.copyable` value inside stays selectable |
| `UX-03` `calm-hover` | Calm hover | Sweeping the pointer flashes every row | Hover fill transitions in after ~60ms and out at once; no hover fill under `(hover: none)` |
| `UX-04` `popup-show-hide` | Popup show/hide | Menus, dropdowns and dialogs pop in and vanish on close; the chevron flips | Fade + small scale on `opacity`/`transform` both ways (in ~150–200ms, out ~100ms), origin at the trigger; `@starting-style` + `allow-discrete` for `[popover]`/`<dialog>`, stay mounted until the exit ends otherwise; rotate the caret from `aria-expanded` over the same frames |
| `UX-05` `tooltips` | Tooltips | Native grey boxes, none on keyboard focus | One document-level tooltip: delay ~420ms, instant when moving between tips, flip below when no room, clamp to window, show on focus, lift `title` while shown (keep it as `aria-label` if it was the only name) |
| `UX-06` `page-entrance` | Page entrance | View transitions hold frames until the new page is built, so the nav glide waits | Select the new nav item from the pending route; animate the new page in (opacity + 6px rise) after mount; nothing waits for it |
| `UX-07` `glide-to-new` | Glide to new content | A sent message appears far below | Smooth-scroll to it only when it is near (≤1.5 screens); jump for streaming, loads and long distances so the view never lags the text |

## 3. Performance

1. **Animate only what the compositor can run.** `transform` and `opacity` stay smooth while the
   main thread is busy rendering the page the click opened; `width`, `height`, `top`, masks and
   filters repaint every frame and stutter exactly then. If something must change size, set the
   size at once and animate a transform from the old box (FLIP).
2. **Some things should not animate.** A mask or filter that animates repaints its content every
   frame, and text renders a touch heavier while it does, which shows as a flicker. Fades on
   scroll edges and similar switch at once.
3. **Craft must be cheap.** Frameworks re-render often (a streaming reply re-renders a chat many
   times a second). Never measure layout in a render hook; let ResizeObserver, MutationObserver
   and scroll events report real changes, coalesced to one measurement per animation frame.

| Rule | Detail | Symptom | Fix |
| --- | --- | --- | --- |
| `PF-01` `compositor-only` | Compositor-only motion | A glide, popup or resize stutters on the frame the click starts rendering a page | Animate `transform` and `opacity` only; for a size or place change, set the new box at once and animate a transform from the old one (FLIP) |
| `PF-02` `no-animated-masks` | No animated masks or filters | Text flickers heavier while a fade on a scroller's edge animates; a panel "unrolls" by height | Switch masks, filters and blurs at once; open panels with opacity + small scale, never `height` |
| `PF-03` `cheap-rerenders` | Cheap under re-renders | A streaming reply makes the page sluggish; profiles show layout per render | No layout reads in render hooks; observers (scroll, Resize, Mutation) coalesced to one measurement per frame; share observers |

## 4. Details craft

1. **Draw the way the browser draws.** A 1px CSS border is whole device pixels on device-pixel
   edges. Anything you draw yourself (SVG, canvas, gradients, masks) must land on the same grid or
   it reads wider and paler beside the real border. Think in device pixels: `devicePixelRatio`
   is 1, 1.25, 1.5, 2, 3 on real screens.
2. **Measure, don't eyeball.** Pixel claims get checked in screenshots at several densities;
   motion claims get checked by sampling every frame. A detail you did not measure is a guess.

| Rule | Detail | Symptom | Fix |
| --- | --- | --- | --- |
| `DC-01` `device-pixel-lines` | Device-pixel lines | A drawn shape (SVG tail, notch) looks wider/paler than the CSS border it continues | Build the shape for the current `devicePixelRatio`: line width `floor(dpr)/dpr` CSS px, straight parts on the border's pixels; rebuild on a `(resolution: Ndppx)` media query change |
| `DC-02` `sliding-selection` | Sliding selection | Selection jumps; or a glide that animates width/height stutters | One highlight behind the list; FLIP: set the new box at once, animate `transform` (translate + scale) from the old/animated box |
| `DC-03` `scroll-edges` | Scroll edges | Content cut off hard at a scroller's edge; or a fade that flickers | JS marks `is-scrolled` / `has-more-end`; CSS masks the content at those edges only; switch the mask at once, never animate it |
| `DC-04` `fade-truncation` | Fade truncation | "…" on names; or every name faded even when it fits | Mark `is-truncated` only while `scrollWidth > clientWidth`; mask the last ~2.4em; add the full text as a tooltip while cut |
| `DC-05` `cut-outs` | Cut-outs | A badge on an avatar sits on a page-coloured plate | Subtract the notch from the avatar with a mask (`mask-composite: exclude` or a radial gradient) so the real background shows through the gap |
| `DC-06` `dialog-origin` | Dialog origin | Dialogs pop from nowhere in the middle | Remember the last pressed control; if a dialog appears within ~900ms, set `transform-origin` on it before the first frame and scale from ~0.9 |
| `DC-07` `rolling-digits` | Rolling digits | Counters jump; or digits twitch in width | A wheel per digit with an invisible copy of the digit for width/baseline; clip with `clip-path` (not `overflow: hidden`, which moves the baseline); strip of digits 0–9 three times, turn the short way round |
| `DC-08` `auto-grow-field` | Auto-growing field | Textarea snaps between heights, or animates from 0 | Measure with transitions off and put the old height back before setting the new one, so it eases from old to new |
| `DC-09` `optical-alignment` | Optical alignment | An icon sits visibly low against capitals | Measure icon centre vs cap height; shift by half a pixel only at ≥2x (at 1x it blurs, which reads worse) |
| `DC-10` `one-stroke-check` | One-stroke check | Copy confirmation swaps icons abruptly | Draw the check with an animated `stroke-dashoffset` |

## Workflow

1. **Audit, chapter by chapter.** Walk the UI at 1x and 2x (and 1.25x/1.5x if the user has such a
   screen), light and dark, desktop and phone width (with the phone's safe-area insets forced
   on).
   - UI: run the cover test (`references/ui-rules.md` §1) on every label, mark and decoration;
     measure the controls that share a row; look at every selected state.
   - UX: open and close every menu, dropdown and dialog; press every kind of button with a mouse,
     with touch and with the keyboard.
   - Performance: repeat the interactions while the page is busy (navigating, streaming).
   - Craft: look for lines, edges, truncation and alignment that are almost right.

   List what is loud, repeated, misaligned, clipped, jumpy or flickering, grouped by chapter.
   Prefer removing a repeated cue over adding a new one.
2. **Pick the technique.** Match each finding to a catalogue row and cite its ID; read its entry
   in the chapter's reference before writing code, because the naive version of most of these is
   the bug being fixed.
3. **Build it once, centrally.** A craft detail used on one list will be wanted on six. Make it a
   directive, hook or utility class with a one-line usage, not per-component code.
4. **Verify.** Follow `references/verification.md`: screenshots at several densities, per-frame
   sampling of motion, a check with reduced motion on, and the render-cost check.
5. **Write down why.** Each detail gets a comment saying what the browser does that made the naive
   version wrong (e.g. "a 1px SVG stroke is 1.5 device pixels at 1.5x"). Without it the next
   person "simplifies" it back.

## When the user asks for "polish" generally

Do the audit first and show the list, grouped by chapter, with the cheapest high-impact items
first. Many "polish" requests are really UI rule 1 (too many cues) and are fixed by deleting
things; "quieter" almost always is. Apply `references/ui-rules.md` §1 on the first draft of a new
design too, not only when asked to reduce. Then implement in batches and verify each batch.

## Reference files

- `references/ui-rules.md`: reduction (the cover test, the three kinds of repetition, what to
  keep, how to report it), one cue per state, type and text setting, control heights, trailing
  icon spacing, icon and label balance in a button, focus/selection/scrollbars, safe areas. Read
  §1 before a first draft and whenever the user asks for "quieter" or "cleaner".
- `references/ux-rules.md`: pressed state, unselectable labels, calm hover, popup show/hide and
  the caret's turn, tooltips, page entrance, gliding to new content.
- `references/performance.md`: compositor-only motion and FLIP, what not to animate, staying
  cheap under re-renders.
- `references/details-craft.md`: device-pixel lines, sliding selection, scroll edges, fade
  truncation, cut-outs, dialog origin, rolling digits, auto-growing fields, optical alignment.
- `references/verification.md`: how to check pixel, motion and render-cost claims with headless
  Chrome.
- `references/mistermorph-console.md`: where each technique lives in the mistermorph Console
  (`/home/lyric/Codework/arch/mistermorph/web/console`), to copy from when working in a Vue app or
  in that repo.
