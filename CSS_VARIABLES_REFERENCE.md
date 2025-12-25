# CSS Variables Reference

Complete reference of all CSS custom properties available in the application.

All variables are defined in `frontend/src/app.css` under `:root`.

---

## 🎨 Colors

### Theme Colors
| Variable | Light Mode | Dark Mode | Purpose |
|----------|-----------|-----------|---------|
| `--primary-color` | #6366f1 | #818cf8 | Primary brand color |
| `--primary-color-rgb` | 99, 102, 241 | 129, 140, 248 | RGB values for alpha |
| `--star-color` | #fbbf24 | #fbbf24 | Star/favorite indicator |

### Text Colors
| Variable | Light Mode | Dark Mode |
|----------|-----------|-----------|
| `--text-on-primary` | #ffffff | #f1f5f9 |
| `--text-primary` | #1e293b | #f1f5f9 |
| `--text-secondary` | #64748b | #cbd5e1 |
| `--text-muted` | #94a3b8 | #94a3b8 |

### Surface Colors
| Variable | Light Mode | Dark Mode |
|----------|-----------|-----------|
| `--card-bg` | white | #1e293b |
| `--card-bg-rgb` | 255, 255, 255 | 30, 41, 59 |
| `--surface-light` | rgba(255,255,255,0.1) | rgba(255,255,255,0.05) |
| `--surface-muted` | rgba(255,255,255,0.2) | rgba(255,255,255,0.1) |
| `--surface-muted-strong` | rgba(255,255,255,0.3) | rgba(255,255,255,0.15) |

### Border Colors
| Variable | Light Mode | Dark Mode |
|----------|-----------|-----------|
| `--border-color` | #e2e8f0 | #334155 |
| `--checkbox-border` | #cbd5e1 | #475569 |

### Special Colors
| Variable | Value | Purpose |
|----------|-------|---------|
| `--color-delete` | #ff3b30 | Delete button text |
| `--color-delete-bg` | rgba(255, 59, 48, 0.1) | Delete button background |

---

## 📏 Spacing

Consistent spacing scale based on 4px increments.

| Variable | Value | Common Use |
|----------|-------|------------|
| `--spacing-xs` | 4px | Tight padding, small gaps |
| `--spacing-sm` | 8px | Compact spacing, list items |
| `--spacing-md` | 12px | Default gap, comfortable spacing |
| `--spacing-lg` | 16px | Section margins, padding |
| `--spacing-xl` | 20px | Large padding |
| `--spacing-2xl` | 24px | Section separation |
| `--spacing-3xl` | 32px | Major sections |
| `--spacing-4xl` | 48px | Large separation |
| `--spacing-5xl` | 64px | Drag & drop spacing |

---

## 🔘 Border Radius

| Variable | Value | Common Use |
|----------|-------|------------|
| `--radius-sm` | 8px | Buttons, inputs |
| `--radius-md` | 12px | Cards, containers |
| `--radius-lg` | 16px | Large cards |
| `--radius-full` | 999px | Pills, badges, circles |

---

## 📝 Typography

### Font Sizes
| Variable | Value | Common Use |
|----------|-------|------------|
| `--font-size-xs` | 12px | Badges, small labels |
| `--font-size-sm` | 14px | Secondary text, counts |
| `--font-size-base` | 16px | Body text, inputs |
| `--font-size-lg` | 17px | Emphasized text |
| `--font-size-xl` | 18px | Large text |
| `--font-size-2xl` | 32px | Page titles |

### Font Weights
| Variable | Value | Common Use |
|----------|-------|------------|
| `--font-weight-normal` | 400 | Body text |
| `--font-weight-medium` | 500 | Emphasized text |
| `--font-weight-semibold` | 600 | Headings, titles |
| `--font-weight-bold` | 700 | Strong emphasis |

### Line Heights
| Variable | Value | Common Use |
|----------|-------|------------|
| `--line-height-tight` | 1 | Icons, compact text |
| `--line-height-normal` | 1.4 | Body text, badges |

---

## 🖼️ Icons & Components

### Icon Sizes
| Variable | Value | Common Use |
|----------|-------|------------|
| `--icon-xs` | 16px | Small icons, chevrons |
| `--icon-sm` | 20px | Standard icons |
| `--icon-md` | 24px | Menu icons |
| `--icon-lg` | 28px | Large icons |
| `--icon-xl` | 32px | Extra large icons |

### Component Sizes
| Variable | Value | Purpose |
|----------|-------|---------|
| `--checkbox-size` | 28px | Checkbox dimensions |
| `--button-height-sm` | 32px | Small button height |
| `--button-height-md` | 40px | Medium button height |
| `--drop-indicator-height` | 56px | Drag drop placeholder |
| `--drop-spacing` | 64px | Space for drop target |
| `--min-drop-zone-height` | 20px | Minimum draggable area |

---

## 🌫️ Shadows

| Variable | Value | Common Use |
|----------|-------|------------|
| `--shadow-sm` | 0 1px 3px rgba(0,0,0,0.08) | Subtle cards |
| `--shadow-md` | 0 2px 6px rgba(0,0,0,0.05) | Elevated elements |
| `--shadow-lg` | 0 2px 8px rgba(0,0,0,0.1) | Prominent cards |
| `--shadow-xl` | 0 4px 16px rgba(0,0,0,0.15) | Modals, popovers |
| `--shadow-2xl` | 0 4px 16px rgba(0,0,0,0.2) | Dropdowns |
| `--shadow-menu` | 0 -4px 16px rgba(0,0,0,0.15) | Upward menus |
| `--shadow-inset` | inset 0 1px 2px rgba(0,0,0,0.05) | Inset elements |
| `--shadow-focus` | 0 1px 3px rgba(0,0,0,0.08) | Focus state |

---

## ⏱️ Transitions & Animations

### Transition Speeds
| Variable | Value | Common Use |
|----------|-------|------------|
| `--transition-fast` | 0.15s ease | Quick responses |
| `--transition-normal` | 0.2s ease | Standard transitions |
| `--transition-slow` | 0.3s ease | Slow, smooth changes |

### Animation Durations
| Variable | Value | Purpose |
|----------|-------|---------|
| `--duration-instant` | 0.1s | Very fast |
| `--duration-fast` | 0.15s | Fast interactions |
| `--duration-normal` | 0.2s | Standard transitions |
| `--duration-slow` | 0.3s | Slow transitions |
| `--duration-animation` | 300ms | Svelte animations |
| `--duration-flip-instant` | 0ms | Instant flip (drag) |
| `--duration-pulse` | 1s | Pulsing animations |
| `--duration-long-press` | 500ms | Mobile long press |
| `--duration-auto-expand` | 500ms | Auto-expand delay |
| `--duration-blur-delay` | 150ms | Blur event delay |

---

## 👻 Opacity

| Variable | Value | Common Use |
|----------|-------|------------|
| `--opacity-disabled` | 0.5 | Disabled elements |
| `--opacity-completed` | 0.7 | Completed todos |
| `--opacity-dragging` | 0.5 | Element being dragged |
| `--opacity-hover` | 0.9 | Hover state |
| `--opacity-subtle` | 0.6 | Subtle effects |
| `--opacity-pulse-min` | 0.8 | Pulse animation min |
| `--opacity-pulse-max` | 1 | Pulse animation max |

---

## 🗂️ Z-Index

Organized z-index scale to prevent conflicts.

| Variable | Value | Purpose |
|----------|-------|---------|
| `--z-index-dropdown` | 100 | Dropdown menus |
| `--z-index-drop-indicator` | 11 | Drop target indicator |
| `--z-index-drop-target` | 10 | Drop target element |
| `--z-index-menu` | 150 | Context menus |

---

## 📐 Layout

### Widths
| Variable | Value | Purpose |
|----------|-------|---------|
| `--container-max-width` | 820px | Max content width |
| `--container-viewport-width` | 90vw | Responsive width |
| `--scrollbar-width` | 8px | Custom scrollbar |
| `--menu-min-width` | 200px | Dropdown min width |

### Heights
| Variable | Value | Purpose |
|----------|-------|---------|
| `--viewport-height-offset` | 48px | Viewport adjustment |

---

## ✏️ Stroke Widths

| Variable | Value | Common Use |
|----------|-------|------------|
| `--stroke-thin` | 2px | Borders, lines |
| `--stroke-medium` | 2.5px | SVG strokes |
| `--stroke-bold` | 3px | Emphasized borders |

---

## 📍 Positioning

| Variable | Value | Purpose |
|----------|-------|---------|
| `--position-third` | 33.333% | One third position |
| `--position-half` | 50% | Center position |

---

## 🎯 Usage Examples

### Simple Component
```css
.button {
  padding: var(--spacing-md) var(--spacing-lg);
  font-size: var(--font-size-base);
  border-radius: var(--radius-sm);
  background: var(--primary-color);
  color: var(--text-on-primary);
  transition: all var(--transition-normal);
  box-shadow: var(--shadow-sm);
}

.button:hover {
  opacity: var(--opacity-hover);
}

.button:disabled {
  opacity: var(--opacity-disabled);
}
```

### Card with Spacing
```css
.card {
  padding: var(--spacing-lg) var(--spacing-xl);
  margin-bottom: var(--spacing-2xl);
  border-radius: var(--radius-md);
  background: var(--card-bg);
  box-shadow: var(--shadow-lg);
}
```

### Icon Button
```css
.icon-btn {
  width: var(--icon-xl);
  height: var(--icon-xl);
  padding: var(--spacing-sm);
  border-radius: var(--radius-sm);
  transition: all var(--transition-normal);
}

.icon-btn svg {
  width: var(--icon-sm);
  height: var(--icon-sm);
}
```

### Drag & Drop
```css
.drop-target::before {
  content: '';
  height: var(--drop-indicator-height);
  top: calc(-1 * var(--drop-spacing));
  border: var(--stroke-thin) dashed var(--text-on-primary);
  border-radius: var(--radius-md);
  opacity: var(--opacity-hover);
  animation: pulse var(--duration-pulse) ease-in-out infinite;
}
```

---

## 📋 Quick Reference by Use Case

### Creating a New Button
```css
padding: var(--spacing-sm) var(--spacing-lg);
border-radius: var(--radius-sm);
font-size: var(--font-size-base);
font-weight: var(--font-weight-medium);
transition: all var(--transition-normal);
```

### Creating a Card
```css
padding: var(--spacing-lg) var(--spacing-xl);
border-radius: var(--radius-md);
box-shadow: var(--shadow-sm);
background: var(--card-bg);
```

### Adding Icons
```css
width: var(--icon-md);
height: var(--icon-md);
```

### Setting Text Styles
```css
font-size: var(--font-size-base);
color: var(--text-primary);
font-weight: var(--font-weight-normal);
```

### Creating Gaps/Spacing
```css
gap: var(--spacing-md);
margin-bottom: var(--spacing-lg);
```

---

## ⚠️ Important Rules

1. **Never use magic numbers** - Always use a CSS variable
2. **Check existing constants first** - Don't create duplicates
3. **Use semantic names** - `--spacing-lg` not `--size-16`
4. **Group by category** - Keep related constants together
5. **Document new additions** - Add comments for non-obvious values
6. **Maintain consistency** - Follow the existing naming patterns

---

## 🔄 Updating Variables

All variables are in `frontend/src/app.css`:

```css
:root {
  /* Add new variables here */
  --my-new-constant: value;
}
```

Changes will automatically apply to all components using that variable.

---

**Last Updated:** December 2024  
**Total Variables:** 100+  
**Files Using Variables:** All CSS in components

