# Chapter 1: UI rules

How the interface looks at rest: what is shown, how many times, on which scale, and where it sits
on the screen. Every rule here can be checked in a still screenshot.

Rules: say each thing once (§1); show a state once, by colour or fill (§2); take sizes, weights
and heights from short scales (§3–§5); measure spacing from the ink, not the box (§6); balance an icon and a label in one button by
eye, not by number (§7); keep focus, selection and scrollbars quiet and consistent (§8); leave the
system's own areas free on phones (§9).

## Contents

1. Say each thing once
2. One cue per state: no side line, no weight change
3. Type on one scale
4. Setting text: digits and line breaks
5. Control heights: siblings match, inset controls are smaller
6. The caret and other trailing icons: spacing from the edge
7. Icon and label in one button: balanced by eye, not by number
8. Focus rings, text selection, caret colour, scrollbars
9. Safe areas on phones (status bar and home indicator)

---

## 1. Say each thing once

`UI-01` `say-once`

A quiet interface is not one with fewer features. It is one where every piece of information
appears once. Most requests for "quieter", "cleaner" or "less busy" are answered by removing what
repeats, not by restyling what is there.

Apply this on the first draft. A design that needs five rounds of "remove this too" was drawn
without the test.

### The cover test

For each label, mark, line and decoration: cover it and ask what the reader no longer knows.

- Nothing: remove it.
- Something another element also shows: keep the quieter of the two.
- Something only it shows: keep it, even if it looks like clutter.

Run the test on the finished screen, not element by element while drawing. Repetition comes from
two elements that each looked reasonable alone.

### Three kinds of repetition

#### 1. The context already says it

The page, the heading above, or the place an element sits in has told the reader.

| Remove | Because |
| --- | --- |
| A job's name beside "Runs" on that job's own page | The page is that job |
| "7 jobs · 1 running" above a visible list of seven | The list shows it |
| "What ran" beside a past day in a calendar | A past day can hold nothing else |
| A section title repeated as the first field's label | The section named it |

#### 2. Another mark already says it

Two encodings of one state: keep one.

| Remove | Keep |
| --- | --- |
| "Running" next to a spinner | The spinner, with the elapsed time |
| "Expected" next to a hollow mark | The hollow mark |
| An arrowhead on a line that ends at the item | The line |
| An underline on a phrase that already has its own colour | The colour, and a tint on hover |
| A group heading for an attribute shown in every row | The rows |
| A coloured line down the left edge of a selected item | The soft fill (and the text colour) |

Which one to keep: the mark, when it is already learned elsewhere in the product; the word, when
the mark is new or ambiguous. Never neither.

#### 3. Decoration must carry meaning

A visual treatment is a statement. Use it only where the statement is true.

- A shadow says "this floats over the page". A toolbar set into the page has none; the same
  toolbar floating over a scroller keeps it.
- A border says "this is a separate surface". Rows of one list are not separate surfaces.
- Text for an event says "this is news". Repeating events that went as usual are not news: fold
  them to a mark, name the repeating things once, and write out only what differs (a failure, a
  run started some other way).
- A group says "these belong to one kind". Do not group by an attribute an item can have several
  of; a job with two triggers has no group.

### What to keep

Removing is not the goal; one clear statement is. Keep:

- **The only place something appears.** The date beside "Today" stays if nothing else shows the
  date.
- **Outcomes that have no mark of their own**, or whose mark differs only by colour. "Failed"
  beside a red square stays unless shape or position also tells it apart.
- **Accessible names.** When a visible label goes, its text stays as the control's
  `aria-label` and tooltip. Icon-only is a visual choice, not a removal of the name.
- **Affordance on touch.** Hover is not available on a phone: a changeable phrase that shows
  itself only on hover on a desktop needs a resting cue there (a faint tint).
- **First-use clarity.** If the remaining mark is one the product has not used before, keep the
  word until the mark is established.

### Reporting a reduction

After reducing, say three things:

1. What was removed, grouped by the kinds above.
2. What was kept on purpose and why, so the user can overrule it in one line.
3. Any affordance that now relies on colour or hover alone.

When the user asks for one removal ("remove X, and similar things"), find the similar things
across every view, not only the one they were looking at, and list them.

## 2. One cue per state: no side line, no weight change

`UI-02` `no-side-line`, `UI-03` `same-weight`

A state is shown once, by colour or fill. A selected item is a soft fill and a stronger text
colour; it is not also framed, dotted, lined or bolded. Each extra cue says the state again
(rule 1) and most of them also move or reshape something.

**No coloured line on the left (or any side) of the selected item**, in nav, sidebars, lists,
menus or settings indexes. A straight bar against the fill's rounded corners looks stuck on;
drawn as a `border-left` it pushes the label in by its width when selected; and it doesn't
travel with the sliding highlight (details-craft.md §2). The fill and the text colour are the
whole selected state. (A tab strip's underline is a different thing: there the indicator is the
selection, with no fill beside it.)

**A state never changes the font weight.** Normal, hover, pressed, focused, selected, active,
current page, checked, open: one weight for all of them. Bolder text is wider, so a tab or nav
item that turns 600 when selected grows a few pixels, its neighbours shift, a centred label
wobbles, a label that just fit starts to truncate, and the sliding highlight has to chase a box
that changed size. Show the state with colour (text to the strong foreground or the accent), a
fill, or an indicator (the sliding highlight, a tab's underline, a check), never with weight,
`<strong>`/`<b>` or `font-synthesis`.

```css
/* One weight per control, whatever its state. */
.tab, .nav-item, .segmented > *, [role=option], [role=menuitem] { font-weight: 500; }
.tab { color: var(--text-muted); }
.tab[aria-selected="true"] { color: var(--text); }  /* colour + indicator carry the state */
```

Reserving the bold width with a hidden bold copy (`::after { content: attr(data-text);
font-weight: 600; visibility: hidden; }`) stops the shift but keeps the second cue and the
flicker of a weight change; don't use it as the fix. Weight still differs between kinds of text
(a heading vs. its body); the rule is about one element across its states.

**Check:** verification.md, Pixels: weight across states, and no side line on selection.

## 3. Type on one scale

`UI-04` `type-scale`

- Count families and sizes before adding any: a UI that mixes the system sans with a brand sans,
  and two monos, looks unsettled. Aim for one sans, one mono, a short size scale (e.g. 11, 12, 14,
  16, 18, 20, 32) and three weights (400, 500, 600). Snap stray sizes and weights to it.
- Weights belong to kinds of text (a heading, a label, body), never to states (§2).
- For CJK with a Latin brand font, fall back to system CJK faces, Japanese first under `:lang(ja)`
  and Chinese first otherwise, since the same characters are drawn differently.
- Bundle fonts (e.g. `@fontsource/*`) rather than loading from a CDN when the app must look the same
  offline or in a desktop shell; subsets per script keep it light.

## 4. Setting text: digits and line breaks

`UI-05` `tabular-digits`, `UI-06` `line-breaks`

- `font-variant-numeric: tabular-nums` on interface text (counters, tables, timestamps), so live
  numbers don't twitch and columns line up. Keep proportional digits in prose and documents.
- `text-wrap: pretty` on paragraphs, list items, `dd`, quotes (no one-word last line);
  `text-wrap: balance` on titles and empty-state copy.
- CJK: `line-break: strict` (no line starting with closing punctuation or small kana); Japanese
  titles `word-break: auto-phrase` (break between phrases).

## 5. Control heights: siblings match, inset controls are smaller

`UI-07` `control-heights`

**Why the naive version fails.** When height comes from padding, it is padding + line height +
border, and those differ between controls: an input has a 1px border and the button beside it
has none, the button's label is 14px and the input's text 16px, an icon-only button's height is its
16px SVG while a text button's is its line box, and a native `<select>` brings its own. So a
search field and its button come out 36px and 34px, and the row's edges don't line up: the eye
reads that 2px at once. Going the other way, a button set inside a field at the field's own
height covers the field's border and looks pasted on.

**Recipe.** Heights come from a short scale of tokens and are set explicitly; padding only places
the content inside.

```css
:root {
  --control-sm: 28px;
  --control-md: 36px;
  --control-lg: 44px;
  --control-inset: 4px;                       /* gap around a control set inside a field */
  --control-radius: 8px;
}
/* Touch: raise the whole scale together, so siblings still match each other. */
@media (pointer: coarse) { :root { --control-md: 44px; } }

/* Controls by class, not every <button>: menu items, rows and link-like buttons aren't controls. */
:where(.input, .select, .btn, .segmented) {
  box-sizing: border-box;                     /* the border is inside the height */
  block-size: var(--control-md);
  border-radius: var(--control-radius);
}
.btn--icon { inline-size: var(--control-md); padding: 0; } /* square, the same height as text buttons */
.row { display: flex; align-items: center; gap: 8px; }    /* center, so a stray height shows up in review */
```

- **Siblings: the same token.** Anything that can sit in one row as its own control (input,
  select, button, icon button, segmented control, date picker, filter chip in a toolbar, search
  field) uses the same size token as its neighbours. A search bar of input + button is two
  `--control-md`s; a toolbar that mixes sizes picks one. A button text, icon or no border never
  changes the height; only the size token does.
- **Inset: smaller by the gap, corners concentric.** A control set inside another (a search button
  in the field's end slot, a clear "×", a unit picker, a send button in a composer) is the outer
  height minus the inset on both sides, and its radius is the outer radius minus the inset, so the
  gap around it is even on every side, round the corners too:

```css
.field { position: relative; display: flex; align-items: center; block-size: var(--control-md); }
.field > input { block-size: 100%; inline-size: 100%;
  padding-inline-end: calc(var(--control-md) - var(--control-inset)); } /* text stops before the button */
.field > .field-action {
  position: absolute; inset-block: var(--control-inset); inset-inline-end: var(--control-inset);
  block-size: auto;                           /* top + bottom inset decide it: md − 2 × inset */
  inline-size: auto; aspect-ratio: 1;         /* icon-only (overrides .btn--icon); drop for text */
  border-radius: max(0px, calc(var(--control-radius) - var(--control-inset)));
}
```

  Measure the inset from the field's border box (the edge you see), and with a 1px border make
  the gap the same on every side; check it in a screenshot rather than trusting the arithmetic.
  The touch target of an inset button may stay larger than the button: extend it with an
  invisible `::before` rather than growing what is drawn.
- **Joined groups** (input and button sharing an edge, a segmented control): one height, the
  shared border drawn once, inner corners square, outer corners at `--control-radius`.
- **Not controls:** badges, tags and counts inside text or a row's label follow the text size,
  not the control scale; they are part of a line, not siblings of the controls.
- Align a row's controls by their centres; text inside each is centred in its box, so labels share
  a baseline once heights match.

**Check:** verification.md, Pixels: for each row of controls, every control's
`getBoundingClientRect().height` is equal (within 0.5px), and each inset control has the same gap
above, below and at its end, at 1x and at a fractional density.

## 6. The caret and other trailing icons: spacing from the edge

`UI-08` `caret-spacing`

The usual bugs: the browser's own arrow, or a chevron
background placed at `right 4px`, jammed against the border while the text starts 12px in; the
label running under the caret; or the caret placed by its 16px box, whose ink (a ~10px chevron)
sits 3px further in than the box, so the right side looks wider than the left. The rule: the
gap from the caret's **ink** to the right border equals the gap from the text to the left
border, so the content sits evenly between the two edges; and the label stops a gap before the
caret.

```css
.select-trigger {
  --pad-x: 12px;                              /* the text's left padding */
  --caret: 16px;                              /* the icon's box */
  --caret-ink: 10px;                          /* the chevron's drawn width inside that box */
  --caret-pad: calc(var(--pad-x) - (var(--caret) - var(--caret-ink)) / 2); /* box edge → border */
  display: flex; align-items: center; gap: 8px;
  padding-inline: var(--pad-x) var(--caret-pad);
}
.select-trigger .label { flex: 1; min-inline-size: 0; } /* fades out before the caret */
.select-trigger .caret { flex: none; inline-size: var(--caret); block-size: var(--caret); }

/* A styled native <select>: no element for the caret, so draw it as a background and
   reserve its room in the padding. background-position has no logical keyword: flip it for RTL. */
select.select {
  appearance: none;
  padding-inline: var(--pad-x) calc(var(--caret-pad) + var(--caret) + 8px);
  background: var(--chevron) no-repeat right var(--caret-pad) center / var(--caret) var(--caret);
}
[dir=rtl] select.select { background-position: left var(--caret-pad) center; }
```

Measure `--caret-ink` from the icon you use (its path's bounding box), rather than assuming it; the
same rule holds for any trailing icon in a control (a calendar on a date field, a "×" that clears
it). Rotating the caret (ux-rules.md §4) keeps its spacing, since it turns about its own centre.
Use logical properties (`padding-inline`, `inset-inline-end`) so the caret moves to the left edge
in right-to-left text.

**Check:** verification.md, Pixels: trailing icon spacing, at 1x and 2x, and in
right-to-left if the app supports it.

## 7. Icon and label in one button: balanced by eye, not by number

`UI-09` `icon-label-balance`

Most buttons need only one of the two: an icon that repeats the label says the thing twice (§1).
When a button does carry both, equal numbers on its two sides do not look equal.

**Why the naive version fails.** `padding: 0 16px` puts the icon 16px from the start edge and
about 10px from the top and bottom, so it sits off-centre in the button's round end. At the other
end the 16px is measured to the furthest point of the curve; above and below the centre line the
edge comes in, so the label has less room than 16px and looks tighter than the icon side. Both
sides say 16 and the button looks lopsided.

**Recipe.** Each side gets the padding that suits what sits there.

```css
.btn {
  --h: var(--control-md);                     /* the height comes from the scale (§5) */
  --pad-x: 1em;                               /* the room the label should appear to have */
  --pad-y: calc((var(--h) - 1lh) / 2);        /* what is left above and below a 1lh icon */
  --round: calc(var(--h) * 0.107);            /* what a pill's end takes away: (1 − π/4) × radius */
  display: inline-flex; align-items: center;
  gap: max(var(--pad-y), var(--pad-x) - (1lh - 1cap) / 2);
  padding-inline: calc(var(--pad-x) + var(--round));
  border-radius: calc(var(--h) / 2);
}
.btn > .icon { flex: none; inline-size: 1lh; block-size: 1lh; }
.btn:has(> .icon:first-child) { padding-inline-start: var(--pad-y); } /* the icon's end: concentric */
.btn:has(> .icon:last-child)  { padding-inline-end: var(--pad-y); }
```

- **The icon's end: concentric.** The icon is one line of the label tall (`1lh`) and has the same
  gap above, below and at its end, so it sits in the centre of the round end. The gap to the label
  is never smaller than that, so the icon has even room on all four sides.
- **The label's end: the average, not the furthest point.** A round end removes room beside the
  label. Averaged over the button's height, a corner of radius r on a button of height h removes
  0.43 × r² / h; for a pill that is 0.107 × h. Add it to the padding, so the room the eye sees is
  `--pad-x`. In a 45px pill with 16px of padding the label's end becomes 21px, and looks like 16.
- **The gap.** A line of text has `(1lh − 1cap) / 2` of empty room above and below its capitals,
  and reads as looser than a solid icon at the same distance. Take that off `--pad-x`, down to
  `--pad-y` and no further.
- **Small radii.** With a radius under a quarter of the height the correction is below 1px: set
  `--round` to 0 and keep the concentric icon.
- **Icons with air in their box.** The numbers above assume the icon's ink fills its box (a filled
  circle, an avatar). For an outline glyph, measure from its ink, as with the caret (§6).
- A text-only button keeps the label's padding on both ends; an icon-only button is square (§5).
- Where `lh` or `cap` are not supported, set the icon's size and the gap in px from the same
  arithmetic.

**Check:** verification.md, Pixels: icon and label balance.

## 8. Focus rings, text selection, caret colour, scrollbars

`UI-10` `focus-rings`, `UI-11` `selection-scrollbars`

```css
/* One focus ring, following corners. */
:focus-visible { outline: 1.5px solid color-mix(in srgb, var(--accent) 55%, transparent); outline-offset: 2px; }
/* Rows in a scroller: inside the edge, where the scroller can't clip it. */
.scroller .row:focus-visible { outline-offset: -2px; }

::selection { background: color-mix(in srgb, var(--accent) 22%, transparent); color: inherit; }
:where(input, textarea, [contenteditable="true"]) { caret-color: var(--accent); }

/* Thin scrollbars that appear on hover/focus; the gutter keeps its width so nothing shifts. */
* { scrollbar-width: thin; scrollbar-color: transparent transparent; }
*:hover, *:focus-within { scrollbar-color: var(--line) transparent; }
.scroller { scrollbar-gutter: stable; } /* on scrolling areas only */
```

Text fields keep their own focused border instead of the ring. When the ring shows (keyboard
focus, not clicks) is `:focus-visible`'s job; don't re-create it with mouse tracking.

## 9. Safe areas on phones (status bar and home indicator)

`UI-12` `safe-areas`

**Why the naive version fails.** In a full-screen web app (an iOS home-screen app with a
translucent status bar, a Capacitor/Cordova/WebView shell, Android's edge-to-edge default) the page
draws under the system status bar and the home indicator or gesture bar. A header with `top: 0`
puts its title under the clock and battery; a tab bar or composer with `bottom: 0` sits under the
home indicator, where taps start a system gesture. Padding `body` instead leaves strips of page
colour where the bars' own fills should run to the edge. And `100vh` is taller than what a
mobile browser actually shows, so bottom bars end up under its toolbar.

**Recipe.**

```html
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
```

```css
/* One source for the insets. A native shell may override these on <html> (see below). */
:root {
  --safe-top: env(safe-area-inset-top, 0px);
  --safe-right: env(safe-area-inset-right, 0px);
  --safe-bottom: env(safe-area-inset-bottom, 0px);
  --safe-left: env(safe-area-inset-left, 0px);
}
.app { min-height: 100dvh; }                 /* not 100vh */

/* Backgrounds run to the edge; only content is inset. */
.app-header { padding-top: calc(var(--safe-top) + 8px); }       /* clear the status bar fully */
.tab-bar, .composer, .bottom-sheet {
  padding-bottom: max(var(--safe-bottom), 12px);                  /* the indicator area has room */
}
.app-header, .tab-bar, .content {
  padding-left: max(var(--safe-left), 16px);                      /* landscape notch */
  padding-right: max(var(--safe-right), 16px);
}
/* Scrolling content clears the bottom bar and the inset, so the last item can be reached. */
.content { padding-bottom: calc(var(--tab-bar-height) + var(--safe-bottom)); scroll-padding-bottom: calc(var(--tab-bar-height) + var(--safe-bottom)); }
```

- Inset everything fixed to an edge, not just the header and tab bar: toasts, drawers, sheets,
  floating buttons, full-screen dialogs, and the tooltip clamp of ux-rules.md §5 (clamp 8px inside
  the safe area, not the window).
- On desktop and in a normal browser tab the insets are 0, so this needs no media query.
- The keyboard: when it is open, a composer resting on it should drop the home-indicator inset
  (iOS keeps reporting it, leaving a ~34px gap above the keyboard). Detect it with
  `visualViewport` (its height well below `innerHeight`) and set a class that zeroes
  `--safe-bottom` on the composer.
- Native shells on Android: some WebView setups report 0 for `env(safe-area-inset-*)` while
  drawing edge to edge. Check on a device; if so, have the shell read the window insets
  (status bar, navigation bar, display cutout) and set `--safe-top` etc. as inline styles on
  `<html>`, which override the `:root` defaults above (Capacitor plugins exist for this).
- Match the status bar to the header: `<meta name="theme-color">` (one per colour scheme with
  `media`) in a browser, the shell's status-bar style (light or dark icons) in an app.

**Check:** verification.md, Safe areas: screenshot at phone width with the insets forced to real
values, and once on a real device or simulator.
