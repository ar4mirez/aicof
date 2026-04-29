# HTML/CSS Patterns Reference

## Holy Grail Layout (Grid)

```css
.layout {
  display: grid;
  grid-template:
    "header  header  header" auto
    "sidebar content aside"  1fr
    "footer  footer  footer" auto / 14rem 1fr 14rem;
  min-height: 100dvh;
}
.layout > header { grid-area: header; }
.layout > nav    { grid-area: sidebar; }
.layout > main   { grid-area: content; }
.layout > aside  { grid-area: aside; }
.layout > footer { grid-area: footer; }

@media (max-width: 60rem) {
  .layout { grid-template: "header" auto "content" 1fr "footer" auto / 1fr; }
  .layout > nav, .layout > aside { display: none; }
}
```

## Responsive Card Grid

```css
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr));
  gap: var(--space-md);
}
.card {
  display: flex; flex-direction: column;
  background: var(--color-surface); border-radius: var(--radius-md);
  overflow: hidden; box-shadow: 0 1px 3px oklch(0% 0 0 / 0.12);
}
.card__body { flex: 1; padding: var(--space-md); }
```

## Centering and Sticky Footer

```css
/* Content centering */
.centered { max-width: 65ch; margin-inline: auto; padding-inline: var(--space-md); }
.grid-center { display: grid; place-items: center; min-height: 100dvh; }

/* Sticky footer: header auto, main stretches, footer auto */
body { display: grid; grid-template-rows: auto 1fr auto; min-height: 100dvh; }
```

## Accessible Modal Dialog

Use the native `<dialog>` element with `showModal()` for focus trapping and backdrop.

```html
<dialog id="confirm" aria-labelledby="dlg-title">
  <form method="dialog">
    <h2 id="dlg-title">Confirm Deletion</h2>
    <p>This action cannot be undone.</p>
    <button value="cancel">Cancel</button>
    <button value="confirm" class="btn--danger">Delete</button>
  </form>
</dialog>
```

```css
dialog { max-width: min(90vw, 32rem); border: none; border-radius: var(--radius-md); padding: var(--space-lg); }
dialog::backdrop { background: oklch(0% 0 0 / 0.5); backdrop-filter: blur(4px); }
```

## Skip Navigation

```css
.skip-link {
  position: absolute; top: -100%; left: var(--space-sm); z-index: 9999;
  padding: var(--space-sm) var(--space-md); background: var(--color-primary);
  color: white; border-radius: 0 0 var(--radius-sm) var(--radius-sm);
}
.skip-link:focus { top: 0; }
/* Usage: <a href="#main" class="skip-link">Skip to content</a> */
```

## Fluid Typography Scale

```css
:root {
  --text-sm:  clamp(0.875rem, 0.8rem + 0.35vw, 1rem);
  --text-base: clamp(1rem, 0.9rem + 0.5vw, 1.125rem);
  --text-xl:  clamp(1.375rem, 1.1rem + 1.2vw, 2rem);
  --text-2xl: clamp(1.75rem, 1.3rem + 2vw, 3rem);
  --text-3xl: clamp(2.25rem, 1.5rem + 3.5vw, 4rem);
}
h1 { font-size: var(--text-3xl); }
h2 { font-size: var(--text-2xl); }
body { font-size: var(--text-base); }
```

## Scroll-Driven Animations

```css
/* Progress bar fills as user scrolls */
.scroll-progress {
  position: fixed; top: 0; left: 0; width: 100%; height: 3px;
  background: var(--color-primary); transform-origin: left;
  animation: grow linear; animation-timeline: scroll(root);
}
@keyframes grow { from { transform: scaleX(0); } to { transform: scaleX(1); } }
```

## Logical Properties

```css
/* Logical properties for automatic RTL/LTR support */
.card {
  margin-inline: auto;             /* margin-left/right */
  padding-block: var(--space-md);  /* padding-top/bottom */
  padding-inline: var(--space-lg); /* padding-left/right */
  border-block-end: 1px solid var(--color-border);
  inline-size: 100%;              /* width */
  max-inline-size: 40rem;         /* max-width */
}
```
