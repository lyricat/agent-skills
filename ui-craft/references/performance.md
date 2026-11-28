# Chapter 3: Performance

Craft that costs frames is not craft. Frameworks re-render often (a streaming reply re-renders a
chat many times a second), and the moment a detail most needs to be smooth is the moment the page
is busiest.

Rules: animate only what the compositor can run (§1); some things should not animate at all
(§2); never measure layout in a render hook, and coalesce real changes to one measurement per
frame (§3).

## Contents

1. Animate on the compositor
2. Some things should not animate
3. Keeping it cheap under re-renders

---

## 1. Animate on the compositor

`PF-01` `compositor-only`

**Why the naive version fails.** The frame an animation most needs to be smooth in is the one
where the click also starts rendering whatever it opened. `width`, `height`, `top`, `left`,
`padding`, masks and filters are laid out or repainted on the main thread every frame, so they
stall exactly then. `transform` and `opacity` run on the compositor and keep moving while the main
thread is busy.

- Animate `transform` (translate, scale, rotate) and `opacity` only: popups (ux-rules.md §4),
  pressed buttons (ux-rules.md §1), page entrance (ux-rules.md §6), the caret's turn.
- **Something must change size or place: FLIP.** Set the final box at once (one layout), then
  animate a transform from the old box to none: First, Last, Invert, Play. If a second change
  arrives mid-animation, start the new one from the element's current on-screen box, not from the
  old target. The full recipe is the sliding selection (details-craft.md §2).
- Prefer Web Animations (`el.animate`) or CSS transitions to rAF loops that write styles; they
  hand the work to the compositor and give you `finished` and `reverse()` for free.
- `will-change: transform` only on elements that actually move often (a sliding highlight), not
  as a blanket rule: each one is a layer that costs memory.
- Durations come from motion tokens that `prefers-reduced-motion` sets to 0, and JS reads the
  token rather than hard-coding milliseconds.

**Check:** verification.md, Motion: main-thread pressure.

## 2. Some things should not animate

`PF-02` `no-animated-masks`

**Why the naive version fails.** A mask, filter or `backdrop-filter` that animates repaints its
content every frame, and text renders a touch heavier while it is being repainted, so an
animated fade on a scroller's edge shows as a flicker of the text weight just as scrolling starts.

- Masks (scroll edges, fade truncation), filters and blurs switch at once; they never get a
  transition (details-craft.md §3, §4).
- A height animation (`0 → auto`) to "unroll" a panel is a layout per frame; use opacity and a
  small scale instead (ux-rules.md §4).
- A font weight never animates or changes with state (ui-rules.md §2).

**Check:** verification.md, Motion: flicker.

## 3. Keeping it cheap under re-renders

`PF-03` `cheap-rerenders`

Frameworks call update hooks on every parent render, and some UIs re-render many times a second
(a streaming chat reply). A detail that measures layout in that hook costs a forced layout per
element per render: a hundred truncated titles is a hundred layouts.

- In update hooks, only restore your classes (a `:class`/`className` binding may have replaced
  them) and re-check cheap invariants (did the selector or text change?).
- Real changes reach you through observers: scroll events, ResizeObserver for size,
  MutationObserver for content and class changes. Coalesce all of them to one measurement per
  `requestAnimationFrame`.
- Share one observer across many elements where you can (fade truncation).
- Check it: record a performance profile while the page re-renders rapidly and confirm there is no
  layout from your code per render.
