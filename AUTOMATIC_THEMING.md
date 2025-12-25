# Automatic Theming Documentation

## Overview

The GoTodo app now features automatic dark/light mode theming that respects the browser/system's color scheme preference. The theme automatically switches when the user changes their system preferences without requiring any manual intervention.

## Implementation Details

### 1. CSS Variables (app.css)

The theming system uses CSS custom properties (variables) defined in two contexts:

#### Light Mode (Default)
```css
:root {
  /* Light mode theme palette */
  --primary-gradient: linear-gradient(135deg, #6366f1 0%, #8b5cf6 50%, #a78bfa 100%);
  --primary-color: #6366f1;
  --card-bg: white;
  --text-primary: #1e293b;
  --text-secondary: #64748b;
  --text-muted: #94a3b8;
  /* ... and more */
}
```

#### Dark Mode
```css
@media (prefers-color-scheme: dark) {
  :root {
    /* Dark mode theme palette */
    --primary-gradient: linear-gradient(135deg, #1e1b4b 0%, #312e81 50%, #3730a3 100%);
    --primary-color: #818cf8;
    --card-bg: #1e293b;
    --text-primary: #f1f5f9;
    --text-secondary: #cbd5e1;
    --text-muted: #94a3b8;
    /* ... and more */
  }
}
```

### 2. Theme Detection (main.ts)

The application includes a theme detection system that:

- Detects the initial color scheme preference on load
- Listens for changes in the system's color scheme
- Updates a `data-theme` attribute on the HTML element for debugging purposes

```typescript
function setupThemeDetection() {
  const darkModeQuery = window.matchMedia('(prefers-color-scheme: dark)');
  
  function updateThemeAttribute() {
    const isDark = darkModeQuery.matches;
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light');
  }
  
  updateThemeAttribute();
  darkModeQuery.addEventListener('change', updateThemeAttribute);
}
```

### 3. Component Updates

All components (App.svelte, TodoList.svelte, TodoItem.svelte, CategoriesView.svelte, ModeSwitch.svelte) use CSS variables, so they automatically adapt to both themes without any code changes.

## How It Works

1. **Automatic Detection**: When the app loads, it checks the browser's `prefers-color-scheme` media query
2. **Dynamic Switching**: The `matchMedia` listener detects when the system theme changes and immediately applies the new theme
3. **CSS Variables**: All UI components reference CSS variables that are redefined based on the active theme
4. **Smooth Transitions**: The background gradient includes a transition for smooth theme switching

## Theme Color Schemes

### Light Mode
- **Background**: Purple-to-indigo gradient (#6366f1 → #8b5cf6 → #a78bfa)
- **Cards**: White (#ffffff)
- **Primary Text**: Dark slate (#1e293b)
- **Accent**: Indigo (#6366f1)

### Dark Mode
- **Background**: Deep indigo gradient (#1e1b4b → #312e81 → #3730a3)
- **Cards**: Dark slate (#1e293b)
- **Primary Text**: Light gray (#f1f5f9)
- **Accent**: Light indigo (#818cf8)

## Testing the Feature

### Manual Testing

1. **On macOS**:
   - Go to System Preferences → General → Appearance
   - Switch between "Light" and "Dark"
   - The app should instantly reflect the change

2. **On Windows**:
   - Go to Settings → Personalization → Colors
   - Under "Choose your color", select "Light" or "Dark"
   - The app should instantly reflect the change

3. **Browser DevTools**:
   - Open DevTools (F12)
   - Open the Command Palette (Cmd/Ctrl + Shift + P)
   - Type "Rendering"
   - Find "Emulate CSS prefers-color-scheme"
   - Toggle between "light" and "dark"

### Verification Points

✅ Background gradient changes from bright purple to deep indigo  
✅ Todo cards change from white to dark slate  
✅ Text color adapts for proper contrast  
✅ Buttons and interactive elements remain accessible  
✅ Transitions are smooth and not jarring  
✅ Theme persists across page reloads based on system preference  

## Browser Support

The automatic theming feature works in all modern browsers that support:
- CSS Custom Properties (CSS Variables)
- `prefers-color-scheme` media query
- `matchMedia` JavaScript API

**Supported browsers:**
- Chrome/Edge 76+
- Firefox 67+
- Safari 12.1+
- Opera 62+

## Future Enhancements

Possible future improvements:

1. **Manual Override**: Add a toggle to let users manually select light/dark mode regardless of system preference
2. **Theme Customization**: Allow users to customize colors
3. **Additional Themes**: Add more color schemes (e.g., high contrast, sepia)
4. **Transition Control**: Add user preference for enabling/disabling theme transitions
5. **Theme Persistence**: Store manual theme preference in localStorage

## Technical Notes

- The theme system is implemented purely with CSS and vanilla JavaScript
- No external theming libraries are required
- The implementation is lightweight and performant
- Theme changes are instant with no flash of unstyled content (FOUC)
- The system is compatible with Svelte's reactivity system

