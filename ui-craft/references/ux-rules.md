# Chapter 2: UX rules

How the interface answers input: presses, hovers, opening and closing, moving between pages.
These rules are checked by doing (with a mouse, with touch, with a keyboard) and by sampling
every frame.

Rules: every press answers at once, on touch as on mouse (§1); a control's label is not text to
select (§2); hover is calm, and absent on touch (§3); whatever appears also leaves with motion,
and a second input reverses it from where it is (§4); tooltips wait a beat and show on focus
(§5); motion starts on the input, never holds content back, and is zeroed by
`prefers-reduced-motion` (§4, §6, §7).

## Contents

1. Pressed buttons on touch and mouse
2. Labels on controls are not selectable
3. Calm hover
4. Menus, dropdowns and dialogs: show, hide and the caret
5. Tooltips
6. Page entrance
7. Gliding to new content

---

## 1. Pressed buttons on touch and mouse

`UX-01` `pressed-state`

**Why the naive version fails.** Desktop styles usually give buttons a hover and nothing for the
press, so a click gets no answer until whatever it triggers renders. On phones it is worse: there
is no hover (and §3 removes hover fills under `(hover: none)`), iOS Safari doesn't apply `:active`
to an element unless a touch listener is on it or an ancestor, the grey
`-webkit-tap-highlight-color` flash covers the button's own shape, and a quick tap sets and clears
`:active` within a frame or two, so even a styled press is never seen. A pressed state with a
single transition also feels slow: the same 160ms that is right for the release lags the finger
on the way in.

**Recipe.**

```css
.btn {
  -webkit-tap-highlight-color: transparent;  /* our pressed state replaces the grey flash */
  touch-action: manipulation;                /* no double-tap-zoom delay */
  transition: transform 160ms var(--ease-out), background-color 160ms ease; /* release: settle back */
}
/* The transition on the pressed rule governs going in: fast, so it follows the finger. */
.btn:where(:active, .is-pressed):not(:disabled, [aria-disabled="true"]) {
  transform: scale(0.97);
  background: var(--btn-pressed);            /* a step darker than hover, in both themes */
  transition-duration: 40ms;
}
@media (prefers-reduced-motion: reduce) {
  .btn:where(:active, .is-pressed) { transform: none; } /* keep the fill change: it is state, not motion */
}
```

```js
// A tap is shorter than a frame or two of :active. Hold .is-pressed for a minimum time so every
// press is seen, delay it a beat on touch so a scroll that starts on a button doesn't flash it,
// and drop it when the browser takes the gesture over (pointercancel).
function installPressed(doc = document, { min = 100, touchDelay = 50 } = {}) {
  const sel = "button, [role=button], a.btn";
  let el = null, shownAt = 0, showTimer = 0;
  const show = (b) => { clearTimeout(showTimer); showTimer = 0; b.classList.add("is-pressed"); shownAt = performance.now(); };
  const down = (e) => {
    const b = e.target.closest?.(sel);
    if (!b || b.disabled || b.getAttribute("aria-disabled") === "true") return;
    el = b;
    if (e.pointerType === "touch") showTimer = setTimeout(() => show(b), touchDelay);
    else show(b);
  };
  const up = (e) => {
    const b = el; el = null;
    if (!b) return;
    if (e.type === "pointercancel") {          // the browser took the gesture (a scroll): no press
      clearTimeout(showTimer); showTimer = 0;
      return b.classList.remove("is-pressed");
    }
    if (showTimer) show(b);                    // a tap shorter than the delay: show it now
    setTimeout(() => b.classList.remove("is-pressed"), Math.max(0, min - (performance.now() - shownAt)));
  };
  doc.addEventListener("pointerdown", down, { capture: true, passive: true });
  doc.addEventListener("pointerup", up, { capture: true, passive: true });
  doc.addEventListener("pointercancel", up, { capture: true, passive: true });
  doc.addEventListener("touchstart", () => {}, { passive: true }); // lets iOS apply :active at all
}
```

- Press in ~40ms, release ~160ms, on desktop and mobile alike; the action itself runs on the
  click, never after the animation.
- Scale only compact controls (icon buttons, chips, small buttons) and keep it ≥ 0.96: on a wide
  button or a full-width row, 0.97 moves the edges several pixels and reads as a lurch, and a
  scaled 1px border goes soft at fractional densities (details-craft.md §1). Those get the fill
  change only.
- Pressed wins over hover: put the pressed rule after the hover rule, and don't let hover's
  delayed transition (§3) hold back the press.
- Dragging off the button cancels the press on desktop as it does natively: `:active` stays while
  the pointer is held outside, so if that matters, also clear `.is-pressed` on `pointerleave`.
- Keyboard: Space shows `:active` in browsers; Enter doesn't. Add `.is-pressed` on Enter keydown
  (and clear it on keyup) for the same feedback.
- Disabled controls have no pressed state.

**Check:** with a touch context (verification.md), tap and confirm the pressed frame is visible
for ≥ 100ms; start a scroll on a button and confirm it never shows pressed; on desktop, sample the
transform through a click: fast in, slower out.

## 2. Labels on controls are not selectable

`UX-02` `unselectable-labels`

Double-clicking a button, a quick
double-tap on a tab, a drag across a toolbar or a long press on a phone otherwise highlights
the label (and on iOS opens the copy/look-up callout), which reads as the control breaking. Their
labels are not content, so nobody needs to copy them.

```css
:where(button, .btn, [role=button], [role=tab], [role=menuitem], [role=option], [role=switch],
       .segmented, .chip, .nav-item, summary, label:has(> input[type=checkbox], > input[type=radio])) {
  -webkit-user-select: none;                  /* Safari still needs the prefix */
  user-select: none;
  -webkit-touch-callout: none;                /* no iOS long-press callout on the control */
}
/* Inside a control, something meant to be copied (an ID, a path, a value) stays selectable. */
:where(.copyable) { -webkit-user-select: text; user-select: text; }
```

- Only the controls, never `body` or a whole panel: content, messages, prose links, table cells
  and form values must stay selectable and findable. A clickable list row is a control only
  where its label is; its secondary text (an email, an ID) is often what people want to copy.
- Inputs and textareas are left alone: their text is the user's.
- `user-select: none` doesn't hide the label from screen readers or the browser's find; it only
  stops the highlight.

**Check:** verification.md, Motion: unselectable labels.

## 3. Calm hover

`UX-03` `calm-hover`

**Why the naive version fails.** A hover fill that appears at once flashes every row the pointer
crosses on its way somewhere else. On touch screens a tap leaves a "sticky" hover on the last
row touched.

```css
/* Calm hover: in after a beat, out at once, none on touch. */
.row { transition: background-color 0ms; }
.row:hover { background: var(--hover); transition: background-color 120ms ease 60ms; }
@media (hover: none) { .row:hover { background: none; } }
```

Touch gets its feedback from the pressed state instead (§1). Anything that shows itself only on
hover on a desktop needs a resting cue on touch (ui-rules.md §1, What to keep).

## 4. Menus, dropdowns and dialogs: show, hide and the caret

`UX-04` `popup-show-hide`

**Why the naive version fails.** A popup shown with `display`, `hidden`, `v-if` or a conditional
render appears in one frame and disappears in one frame: `transition` does not run from or to
`display: none`, and an element removed from the DOM has no exit. So menus pop in, vanish on
close, and the chevron beside the trigger flips from down to up with nothing between. Animating the
panel's `height` (0 → auto) to "unroll" it relays out the page every frame and stutters.

**Recipe.** Every popup surface (dropdown panel, menu, context menu, popover, dialog and its
backdrop) fades and scales in on open **and** out on close, on `opacity` + `transform` only.

- Timing: in ~150ms (menus) to ~200ms (dialogs), decelerating (`cubic-bezier(0.2, 0, 0, 1)`);
  out faster, ~100–120ms, accelerating (`cubic-bezier(0.4, 0, 1, 1)`), because the user has
  already moved on. Read both from motion tokens that reduced motion sets to 0.
- Scale small: menus from `scale(0.96)` plus a 4px shift toward the trigger, dialogs from 0.98 (0.9
  when they grow out of their control, see details-craft.md §6). Opacity alone looks like a
  rendering fade; scale alone looks like a zoom.
- Origin: a menu grows from the edge that touches its trigger (`transform-origin: top left` for a
  menu below and right-aligned to the trigger's left edge; flip it when the menu is placed above
  or to the other side, so the origin follows the final placement, not the default).
- The panel moves as one; don't stagger its items.
- Native top-layer elements (`[popover]`, `<dialog>`): let CSS keep them displayed through the exit.

```css
.menu, dialog.dialog, dialog.dialog::backdrop {
  transition:
    opacity var(--motion-out) var(--ease-in),
    transform var(--motion-out) var(--ease-in),
    display var(--motion-out) allow-discrete,
    overlay var(--motion-out) allow-discrete; /* stay in the top layer until the exit ends */
}
.menu { opacity: 0; transform: translateY(-4px) scale(0.96); transform-origin: top left; }
.menu:popover-open {
  opacity: 1; transform: none;
  transition-duration: var(--motion-in); transition-timing-function: var(--ease-out);
}
@starting-style { .menu:popover-open { opacity: 0; transform: translateY(-4px) scale(0.96); } }

dialog.dialog { opacity: 0; transform: scale(0.98); }
dialog.dialog[open] { opacity: 1; transform: none; transition-duration: var(--motion-in); }
@starting-style { dialog.dialog[open] { opacity: 0; transform: scale(0.98); } }
dialog.dialog::backdrop { opacity: 0; }
dialog.dialog[open]::backdrop { opacity: 1; }
@starting-style { dialog.dialog[open]::backdrop { opacity: 0; } }
```

- Framework-mounted popups: keep the element mounted until its exit finishes (Vue `<Transition>`,
  a presence wrapper in React), or run the exit yourself and unmount on `finished`:

```js
// Close: run the exit on the compositor, then remove. Nothing else waits for it.
function closePopup(el, remove, duration) {
  el.style.pointerEvents = "none";            // clicks fall through to the page at once
  if (!duration) return remove();             // reduced motion
  const a = el.animate(
    [{ opacity: 1, transform: "none" }, { opacity: 0, transform: "translateY(-4px) scale(0.96)" }],
    { duration, easing: "cubic-bezier(0.4, 0, 1, 1)", fill: "forwards" },
  );
  a.finished.then(remove, remove);
  return a;                                   // keep it: reopening mid-exit calls a.reverse()
}
```

- Interruptions: reopening during the exit must reverse from where it is, not restart from
  nothing. CSS transitions do this by themselves; with Web Animations, `reverse()` the running one.
- Nothing waits on the exit: the chosen menu action runs on the click, focus returns to the
  trigger at once, and the leaving popup takes no pointer events.
- **The caret (chevron).** Drive it from the same state as the popup, ideally `aria-expanded`, so
  the visual and the announced state cannot disagree, and rotate it with a transform over the same
  duration and easing as the panel so the two move together:

```css
.trigger .caret { transition: transform var(--motion-out) var(--ease-in); }
.trigger[aria-expanded="true"] .caret {
  transform: rotate(180deg);                  /* 90deg for a submenu that opens to the side */
  transition-duration: var(--motion-in); transition-timing-function: var(--ease-out);
}
/* Rotating a shape inside an inline SVG: rotate about its own box, not the SVG's origin. */
.caret path { transform-box: fill-box; transform-origin: center; }
```

- Native `<select>`: its picker can't be animated except as a customizable select
  (`appearance: base-select`, styling `::picker(select)` and `::picker-icon`, with `:open` for the
  icon) where supported; gate that behind `@supports (appearance: base-select)` and leave the
  native control alone elsewhere.
- The caret's distance from the control's edge is a layout rule: ui-rules.md §6.

**Check:** sample the popup's opacity and transform every frame through open, close, and a reopen
mid-close (verification.md, Motion): continuous both ways, the exit shorter than the entrance, the
caret's rotation spanning the same frames; with reduced motion, both ends set at once.

## 5. Tooltips

`UX-05` `tooltips`

One document-level implementation replaces native `title` boxes:

- Delay ~420ms; if another tooltip closed within ~300ms, show at once (moving along a toolbar).
- Position centred above the element, below if there is no room, clamped 8px inside the window.
- Show on keyboard focus too (native tooltips never do).
- While the pointer is over the element, move `title` to a data attribute so the browser's box
  does not also appear; if the title was the element's only accessible name, set it as
  `aria-label` meanwhile; restore both on leave. Leave `title` alone on focus.
- A trailing shortcut like "New topic (Ctrl+N)" renders as `<kbd>` caps.
- `role="tooltip"` and `aria-describedby` while shown.

## 6. Page entrance

`UX-06` `page-entrance`

**Why the naive version fails.** The View Transitions API holds every frame until the new page has
rendered, including other animations, so the nav selection's glide waits ~100–200ms and then
freezes. A new page that appears with no motion feels abrupt.

**Recipe.**

- Select the nav item for the page being opened as soon as navigation starts (follow the router's
  pending route), so the selection glide starts on the click.
- After the new page mounts, run a compositor animation on it: opacity 0→1 and translateY 6px→0,
  ~200ms. Don't fade the old page out; replace it. Nothing waits for the animation.
- Only between top-level pages, not between items within a page (another topic, another settings
  section).

## 7. Gliding to new content

`UX-07` `glide-to-new`

**Why the naive version fails.** A sent message that appears far below the view, with no motion,
makes the reader hunt for it; smooth-scrolling to every new piece of content makes the view lag
behind streaming text and long jumps take seconds.

Smooth-scroll to the bottom when the user sends something and the bottom is within ~1.5 screens;
otherwise jump. Streaming text, initial loads and long distances always jump, so the view never
lags behind the text.
