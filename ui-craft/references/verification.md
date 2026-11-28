# Verifying craft details

Craft claims are about pixels and frames, so they are checked with pixels and frames. Headless
Chrome (Playwright or the DevTools protocol directly) does all of it against the running app.

## Setup

- Playwright's `chromium.launch()` with a `newContext({ viewport, deviceScaleFactor })`. Use
  `deviceScaleFactor` 1, 1.25, 1.5, 2 and 3: the fractional ones are where device-pixel bugs show.
- Run against the real app (dev server or built app), logged in, with realistic data. A component
  story alone misses the backgrounds, scroll containers and re-render pressure the detail lives in.
- If a site needs a login, do it once and save `storageState` for later scripts.

## Pixels

- **Screenshot the element**, not the page: `locator.screenshot({ path })`, at each density.
- **Measure lines** by reading pixels: load the PNG (e.g. `pngjs` or `sharp`), walk a row or column
  across the line, and compare the drawn line's run length and colour with the CSS border's. Equal
  run length and colour on straight segments is the pass condition; only slopes and curves may be
  antialiased.
- **Alignment**: find the icon's ink bounding box and the capitals' ink bounding box in the same
  row; compare their vertical centres. Report in CSS px (device px / dpr).
- **Control heights**: for each row (a flex or grid parent with two or more controls among its
  children), compare the children's `getBoundingClientRect().height`; any difference over 0.5px
  is a finding. For a control inside a field, compare its top, bottom and end gaps to the field's
  border box; they should be equal, and equal to the inset token.
- **Trailing icon spacing**: in a screenshot of each dropdown (and any field with a trailing
  icon), find the caret's ink bounding box; the gap from its right edge to the border's inner edge
  must equal the gap from the text's first ink to the left border (within 0.5px), at 1x and 2x,
  and the longest label must stop before the caret. Repeat with `dir="rtl"` if the app supports it.
- **Icon and label balance**: in a screenshot of each button that has both, find the icon's ink
  bounding box; its gaps to the top, bottom and near edge of the button must be equal (within
  0.5px), and the gap to the label's first ink no smaller. At the label's end, measure from the
  label's last ink column to the button's edge on every pixel row of the button and average: the
  result must equal the intended padding (within 1px), not the CSS padding.
- **Weight across states**: for each tab, nav item, menu item, option and segmented item, read
  `getComputedStyle(el).fontWeight` and `getBoundingClientRect().width` before and after
  selecting, hovering and pressing it; both must be unchanged, and so must the neighbours'
  positions.
- **No side line on selection**: on each selected nav item, list row and menu item, the computed
  `border-inline-start`, `box-shadow` (inset) and `::before`/`::after` must draw nothing coloured
  along an edge; the screenshot shows only the fill.
- **Phone width**: 390×844 at 3x. Look for clipping, overflow, double scrollbars, hover-only
  affordances.
- **Safe areas**: headless Chrome reports 0 for `env(safe-area-inset-*)`, so force the insets
  through the variables (`--safe-top: 59px; --safe-bottom: 34px` for an iPhone with a Dynamic
  Island in portrait, `--safe-top: 24px`–`48px` for an Android status bar) with
  `page.addStyleTag`, and screenshot at 390×844: no text or control may sit in those bands, while
  header and bar fills run through them. Repeat in landscape with left/right insets. Confirm once
  on a real device or simulator, and say so if you couldn't.
- Also take one screenshot with `colorScheme: "dark"` if the app has a dark theme.

## Motion

- **Sample every frame**: in the page, `requestAnimationFrame` a loop that pushes
  `el.getBoundingClientRect()` (or computed transform/opacity) with `performance.now()`, start it,
  trigger the interaction, collect ~40 frames, return the array.

```js
const samples = await page.evaluate(async (sel) => {
  const el = document.querySelector(sel);
  const out = [];
  const t0 = performance.now();
  await new Promise((done) => {
    const tick = () => {
      const r = el.getBoundingClientRect();
      out.push({ t: Math.round(performance.now() - t0), y: r.y, h: r.height });
      out.length < 40 ? requestAnimationFrame(tick) : done();
    };
    requestAnimationFrame(tick);
    document.querySelectorAll(".item")[3].click();
  });
  return out;
}, ".sliding-selection");
```

  Pass: values change monotonically with easing (no jumps between frames larger than the curve
  allows), start on the click's frame (not 100–200ms later), and end exactly on the target box.
- **Interruptions**: trigger a second change mid-animation; the samples must continue from the
  current position without a snap.
- **Main-thread pressure**: trigger the interaction together with something heavy (navigating to a
  big page) and confirm the animation still advances every frame. Only compositor-run properties
  (transform, opacity) will.
- **Reduced motion**: `newContext({ reducedMotion: "reduce" })` and confirm the end state is set at
  once.
- **Show and hide**: sample a menu's and a dialog's opacity/transform through open, close, and a
  reopen during the close. Both directions must animate (an exit that is one frame is the common
  bug), the exit must be shorter than the entrance, and the caret's rotation must run over the
  same frames as the panel.
- **Pressed state**: in a `newContext({ hasTouch: true, isMobile: true })`, `page.touchscreen.tap()`
  a button while sampling its computed transform/background every frame: the pressed value must
  hold for at least ~100ms. Then start a scroll on a button (`page.mouse` drag or CDP touch events)
  and confirm it never shows pressed. On desktop, sample a `page.mouse.down()`/`up()`: fast in,
  slower out.
- **Unselectable labels**: `page.dblclick()` each kind of control (button, tab, segmented item,
  menu item, chip) and then `page.evaluate(() => getSelection().toString())` must be empty; in a
  touch context, a long press (`touchstart`, wait ~600ms, `touchend` via CDP) must select nothing.
  Then select a paragraph and a copyable value next to them to confirm content still selects.
- **Flicker**: for fades on masks or filters, capture frames while scrolling starts and compare text
  pixels between frames; text weight should not change.

## Render cost

- Record a trace (`page.tracing` / Performance panel) while the app re-renders rapidly (e.g. a
  streaming reply, typing in a large form). Look for Layout/Recalculate Style events attributed to
  the detail's code on every render. There should be none; measurements happen only after real
  changes, at most once per frame.
- Quick proxy: wrap `getBoundingClientRect`/`scrollWidth` getters in a counter during a burst of
  renders.

## What headless can't tell you

Say so in the report rather than implying it was checked:

- Smoothness on a real GPU and real display refresh.
- Real safe-area values and keyboard behaviour on iOS and Android (forced insets only test the
  layout's response to them).
- Hover timing with a real mouse (headless reports no hover for some interactions; synthetic hover
  works for `:hover` styles but not for timing feel).
- Anything that needs real side effects to trigger (sending a real message, a real payment).

## Reporting

Report each detail with what was measured and at which densities, e.g. "tail's top edge: 1 device
px, same colour as the border, at 1, 1.25, 1.5, 2, 3x; slope antialiased". Keep the scripts in a
scratch directory so they can be rerun after the next change.
