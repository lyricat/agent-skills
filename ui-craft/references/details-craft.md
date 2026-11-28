# Chapter 4: Details craft

The solved problems: details that look simple, whose obvious implementation is almost right, and
whose last bit lives in how the browser rasterises, composites and lays out. Each recipe starts
with why the naive version fails; read that before writing code, because the naive version is the
bug being fixed.

Recipes are framework-neutral, written as a plain `attach(el)` function returning a cleanup, which
maps onto a Vue directive (`mounted`/`unmounted`), a React effect, a Svelte action or a custom
element.

Rules: draw the way the browser draws, on device pixels (§1); measure, don't eyeball: every
pixel claim is checked in screenshots at several densities and every motion claim by sampling
frames (verification.md).

## Contents

1. Device-pixel lines
2. Sliding selection (FLIP)
3. Scroll edges
4. Fade truncation
5. Cut-outs and notches
6. Dialog origin
7. Rolling digits
8. Auto-growing fields
9. Optical alignment

---

## 1. Device-pixel lines

`DC-01` `device-pixel-lines`

**Why the naive version fails.** A 1px CSS border is rasterised as whole device pixels, snapped to
device-pixel edges: at 1.25x and 1.5x it is one device pixel, at 2x two. A 1px SVG stroke is
exactly 1 CSS px, so at 1.5x it is 1.5 device pixels smoothed across two, and reads wider and paler
than the border next to it. The same goes for a stroke centred on a coordinate that is not on the
device grid.

**Recipe.**

```js
// The width the browser draws a 1px border at, in CSS px: whole device pixels, at least one.
function borderWidth(dpr) {
  const r = Number.isFinite(dpr) && dpr > 0 ? dpr : 1;
  return Math.max(1, Math.floor(r + 1e-6)) / r;
}
```

- Build the shape for the current density: stroke width `borderWidth(dpr)`, and place straight
  segments so their centre is `line / 2` from the box edge where the CSS border starts. Then the
  straight parts land on the border's own pixels; only slopes and curves are antialiased (as any
  non-straight line is).
- Paint it with two masks of the same outline (`::before` = fill, `::after` = stroke), set as CSS
  variables holding `url("data:image/svg+xml,...")`, so colour comes from `background` and themes
  just work.
- Rebuild when the density changes (another screen, browser zoom):

```js
function watchDensity(win, apply) {
  const run = () => {
    const dpr = win.devicePixelRatio || 1;
    apply(dpr);
    // This query stops matching when the density changes.
    win.matchMedia(`(resolution: ${dpr}dppx)`).addEventListener("change", run, { once: true });
  };
  run();
}
```

- To make a drawn shape grow out of a bordered box (a chat bubble's tail), let the two meet with a
  concave fillet (~4px) and round the tip (~1.6px), so it reads as one outline rather than a
  triangle stuck on a corner.

**Check:** screenshot at 1, 1.25, 1.5, 2, 3x and compare the drawn line's pixels and colour with
the border's, row by row where they meet.

## 2. Sliding selection (FLIP)

`DC-02` `sliding-selection`

**Why the naive version fails.** Each item lighting up on its own is a jump. Transitioning a single
highlight's `top/left/width/height` repaints and lays out every frame, so it stutters exactly when
the click also starts rendering a new page.

**Recipe.**

- One highlight element, prepended to the list, `position: absolute; top: 0; left: 0; z-index: -1;
  transform-origin: 0 0; will-change: transform`. The list gets `position: relative; isolation:
  isolate` so the highlight sits behind items but inside the list. Items' own selected background
  becomes transparent.
- On change: measure the selected item's box in the list's coordinates (add `scrollTop`, subtract
  `clientTop`), set the highlight's width/height/translate to it **at once**, copy its
  `border-radius`, then animate only the transform from the previous box:

```js
const from = animation?.playState === "running"
  ? boxIn(list, highlight.getBoundingClientRect(), listRect) // mid-glide: start where it is now
  : lastBox;
animation?.cancel();
// ...set the new box at once...
animation = highlight.animate(
  [
    { transform: `translate(${from.x}px, ${from.y}px) scale(${from.w / box.w}, ${from.h / box.h})` },
    { transform: `translate(${box.x}px, ${box.y}px)` },
  ],
  { duration, easing: "cubic-bezier(0.2, 0, 0, 1)" },
);
```

- First placement, and a selection after none: no animation, just place it.
- If the list changed but the selected box is where it was (within 0.5px), do nothing, so a glide
  in flight is not restarted.
- Follow changes with a MutationObserver (`childList`, `subtree`, `attributes` filtered to `class`,
  ignoring records whose target is the highlight itself) and a ResizeObserver on the list and the
  selected item; coalesce to one `requestAnimationFrame`.
- Read the duration from a CSS token (e.g. `--motion-slow: 260ms`) that reduced motion sets to 0.
- Hovering the selected item: deepen the highlight via `:has(.is-active:hover) > .highlight`.
- The highlight is the whole selected state: no side line, no weight change (ui-rules.md §2).
- A framework may re-insert its own nodes before the highlight; on update, re-prepend it.

**Check:** click between items and sample the highlight's `getBoundingClientRect()` every frame:
it should be continuous, eased, end exactly on the item, and turn smoothly when clicked again
mid-glide.

## 3. Scroll edges

`DC-03` `scroll-edges`

**Why the naive version fails.** A hard cut at a scroller's edge reads as broken. A gradient
overlay in the page colour fails on any tinted, gridded or translucent background. Fading the mask
in with a transition makes text flicker: the content is repainted each frame and text renders
slightly heavier while it is.

**Recipe.**

```js
function attachScrollEdges(el) {
  let frame = 0;
  const update = () => {
    frame = 0;
    el.classList.toggle("is-scrolled", el.scrollTop > 1);
    el.classList.toggle("has-more-end", el.scrollTop + el.clientHeight < el.scrollHeight - 1);
  };
  const schedule = () => { if (!frame) frame = requestAnimationFrame(update); };
  el.classList.add("has-scroll-edges");
  el.addEventListener("scroll", schedule, { passive: true });
  const ro = new ResizeObserver(schedule); ro.observe(el);
  const mo = new MutationObserver(schedule); mo.observe(el, { subtree: true, childList: true, characterData: true });
  update();
  return () => { cancelAnimationFrame(frame); el.removeEventListener("scroll", schedule); ro.disconnect(); mo.disconnect(); };
}
```

```css
.has-scroll-edges {
  --fade-start: 0px; --fade-end: 0px; --fade: 22px;
  mask-image: linear-gradient(to bottom, transparent 0, #000 var(--fade-start),
    #000 calc(100% - var(--fade-end)), transparent 100%);
}
.has-scroll-edges.is-scrolled { --fade-start: var(--fade); }
.has-scroll-edges.has-more-end { --fade-end: var(--fade); }
/* No transition on the mask: it switches at once. */
```

- Under a pinned header: no top fade; instead show the header's hairline/blur only once something
  has scrolled under it (`.head:has(+ .has-scroll-edges:not(.is-scrolled)) { box-shadow: none }`).
- Under a floating composer or toolbar: end the fade at the overlay's top edge so content is gone
  beneath it, rather than showing through a translucent panel; keep the last item clear of that
  edge at the bottom.
- Known cost: the mask also fades the scrollbar at that edge.

## 4. Fade truncation

`DC-04` `fade-truncation`

**Why the naive version fails.** `text-overflow: ellipsis` is fine, but a fade reads calmer and
lets a row's hover or selection tint show through. Fading every label (even ones that fit) looks
like a rendering bug.

**Recipe.** One shared ResizeObserver; mark `is-truncated` only while
`el.scrollWidth > el.clientWidth + 0.5`; while truncated, set `title` to the full text unless the
element had its own title (remember which you set, and remove only that).

```css
.fade-truncate { overflow: hidden; white-space: nowrap; }
.fade-truncate.is-truncated {
  text-overflow: clip;
  mask-image: linear-gradient(to right, #000 calc(100% - 2.4em), transparent);
}
```

Re-check on text change, not on every render. Multi-line clamps keep "…": a clamp can't be faded
without measuring each line.

## 5. Cut-outs and notches

`DC-05` `cut-outs`

**Why the naive version fails.** A status dot or badge on an avatar is usually drawn on a plate
the colour of the page, which breaks on any grid, tint or hover background.

**Recipe.** Subtract the notch from the avatar itself:

```css
/* Round badge bottom-right: a radial-gradient hole */
.avatar {
  mask: radial-gradient(circle 9px at calc(100% - 6px) calc(100% - 6px), transparent 98%, #000);
}
/* Rounded-square notch: two masks, excluded */
.avatar {
  mask: linear-gradient(#000 0 0), var(--notch-shape) no-repeat right -2px bottom -2px / 13px 13px;
  mask-composite: exclude;
}
```

The notch is the badge size plus the gap (e.g. 2px) on each side, so the real background shows
through the gap. Add a faint inner hairline (`box-shadow: inset 0 0 0 1px rgb(0 0 0 / 9%)`) so a
pale avatar doesn't melt into a pale page; the notch cuts it too.

## 6. Dialog origin

`DC-06` `dialog-origin`

**Why the naive version fails.** A dialog that scales from the centre of the window has no
relationship to what opened it.

**Recipe.**

- Capture-phase `pointerdown` (and `keydown` Enter/Space) on the document remembers the closest
  `button, a, [role=button], [role=menuitem]` and a timestamp.
- A MutationObserver on `body` sees dialogs added. If one appears within ~900ms of the press and
  doesn't contain the control, set `transform-origin` to the control's centre relative to the
  dialog's box, and add a class. The observer's callback runs as a microtask, before the first
  frame, so the first painted frame already has the origin.
- CSS: dialogs normally enter from `scale: 0.98` + opacity; with the class, from `scale: 0.9`, and
  leave back into the control the same way.

## 7. Rolling digits

`DC-07` `rolling-digits`

**Why the naive version fails.** Digits that jump are fine; rolling ones are nicer but usually get
the width, baseline or direction wrong: `overflow: hidden` on an inline-block moves the baseline to
its bottom edge; a 0–9 strip makes 9→0 spin all the way back.

**Recipe.**

- One wheel per digit. The wheel is an inline-block holding an invisible copy of its current digit
  (so it has the font's exact width, line height and baseline), and an absolutely positioned strip
  over it.
- Clip the strip with `clip-path: inset(0)`, not `overflow: hidden`, so the baseline stays the
  text's.
- The strip is the digits 0–9 three times (30 stops). Turn the short way round, forward on a tie:

```js
// How far a wheel at stop `at` (0..29) turns to show `digit`.
function wheelStep(at, digit) {
  const step = (digit - (at % 10) + 10) % 10;
  return step > 5 ? step - 10 : step;
}
```

  So 9→0 is one step forward and 999→1,000 turns each wheel once. Once settled, jump back to the
  middle run without motion.
- Keep wheels aligned from the right when the number of digits changes. Give screen readers and
  text selection the plain value (`aria-label` / a visually hidden copy), and mark the wheels
  `aria-hidden`.
- Reduced motion: set the new value without turning.

## 8. Auto-growing fields

`DC-08` `auto-grow-field`

**Why the naive version fails.** The usual measurement sets `height: auto` (or 0) to read
`scrollHeight`; with a height transition that makes every keystroke animate from nothing.

**Recipe.** Measure with the transition disabled, restore the old height, force a reflow, then
re-enable the transition and set the new height, so it eases old → new.

## 9. Optical alignment

`DC-09` `optical-alignment`

Measure, don't trust the box model: compare an icon's visual centre with the capitals' centre
(cap height / 2) in a screenshot. Shift icons that sit low by half a pixel, but only at ≥2x; at 1x
a half-pixel shift blurs the icon, which reads worse than being 0.5px off. Icons whose offsets vary
by shape are left alone.
