# Editable List Title Feature

## Feature Description
Added the ability to click on the list title (e.g., "🧀 Mat 2025") at the top of the todo list to edit and rename it. The title can include emojis as part of the name.

## Implementation

### Changes Made to `/Users/martin/gotodo/frontend/src/lib/TodoList.svelte`

#### 1. Added State Management
```typescript
// List title state
let listTitle = $state('🧀 Mat 2025');
let editingTitle = $state(false);
let titleInputValue = $state('');
```

#### 2. Added Title Editing Functions
```typescript
// Title editing
function startEditingTitle() {
  editingTitle = true;
  titleInputValue = listTitle;
}

function finishEditingTitle() {
  const newTitle = titleInputValue.trim();
  if (newTitle && newTitle !== listTitle) {
    listTitle = newTitle;
    // Persist to localStorage
    localStorage.setItem('listTitle', newTitle);
  }
  editingTitle = false;
}

function handleTitleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    finishEditingTitle();
  } else if (e.key === 'Escape') {
    editingTitle = false;
  }
}

// Load title from localStorage on mount
onMount(() => {
  const savedTitle = localStorage.getItem('listTitle');
  if (savedTitle) {
    listTitle = savedTitle;
  }
});
```

#### 3. Updated Template
Replaced the static `<h1>` with a conditional render:
```html
{#if editingTitle}
  <input
    type="text"
    class="title-input"
    bind:value={titleInputValue}
    onkeydown={handleTitleKeydown}
    onblur={finishEditingTitle}
    autofocus
  />
{:else}
  <h1 class="title" onclick={startEditingTitle} role="button" tabindex="0">
    {listTitle}
  </h1>
{/if}
```

#### 4. Added Styling
```css
.title {
  color: white;
  font-size: 32px;
  font-weight: 600;
  margin: 0 0 20px 0;
  cursor: pointer;
  transition: opacity 0.2s;
  padding: 8px;
  margin-left: -8px;
  border-radius: 8px;
}

.title:hover {
  opacity: 0.9;
  background: rgba(255, 255, 255, 0.1);
}

.title:focus {
  outline: 2px solid rgba(255, 255, 255, 0.5);
  outline-offset: 2px;
}

.title-input {
  color: white;
  font-size: 32px;
  font-weight: 600;
  margin: 0 0 20px 0;
  padding: 8px;
  margin-left: -8px;
  border: 2px solid rgba(255, 255, 255, 0.5);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.15);
  width: 100%;
  outline: none;
  transition: all 0.2s;
}

.title-input:focus {
  border-color: white;
  background: rgba(255, 255, 255, 0.2);
}
```

## Features

### ✅ Click to Edit
- Click on the title to enter edit mode
- Cursor automatically focuses on the input field

### ✅ Keyboard Navigation
- **Enter**: Save changes
- **Escape**: Cancel editing
- **Tab**: Can navigate to/from the title

### ✅ Visual Feedback
- Hover effect shows the title is clickable
- Input field has a visible border when editing
- Smooth transitions between states

### ✅ Data Persistence
- Title is saved to `localStorage`
- Persists across page refreshes
- Default title is "🧀 Mat 2025"

### ✅ Validation
- Empty titles are rejected (keeps previous title)
- Whitespace is trimmed

### ✅ Emoji Support
- Emojis are fully supported as part of the title
- Example: "🧀 Mat 2025", "🎄 Christmas Shopping 2025"

### ✅ Accessibility
- `role="button"` for screen readers
- `tabindex="0"` for keyboard navigation
- Proper focus management

## User Experience

1. **View Mode**: Title displays with subtle hover effect
2. **Click**: Switches to edit mode with input field
3. **Edit**: Type new title (including emojis)
4. **Save**: Press Enter or click away to save
5. **Cancel**: Press Escape to discard changes
6. **Persistence**: Title survives page refreshes

## Technical Details

- **Storage**: `localStorage` (client-side only)
- **Key**: `"listTitle"`
- **Default**: `"🧀 Mat 2025"`
- **Encoding**: UTF-8 (supports all Unicode including emojis)

## Future Enhancements (Not Implemented)

If you want to sync the list title across clients, you could:
1. Add a `ListTitleChanged` event type to the schema
2. Send it through WebSocket to the backend
3. Broadcast to all connected clients
4. Store in the event log

For now, the title is local to each browser (via localStorage), which is simpler and works well for personal use.

## Testing

✅ Tested functionality:
- Click to enter edit mode
- Type new title
- Save with Enter key
- Cancel with Escape key
- Blur to save
- Page refresh persistence
- Emoji support

All features working correctly!

