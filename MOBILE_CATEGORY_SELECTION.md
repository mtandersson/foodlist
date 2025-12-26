# Mobile Category Selection Feature

## Overview
Implemented a mobile-friendly feature that allows users to quickly assign or change categories for todo items by tapping on them **in Categories view mode**.

## Feature Description
When a user taps on **any todo item** (categorized or uncategorized) on a mobile device **while in Categories view**, a modal popup appears allowing them to select a category to assign or change for that todo.

## Conditions for Modal to Appear
The modal will appear when ALL of the following conditions are met:
1. **Categories view mode**: User must be in "Kategorier" view mode (not "Normal" mode)
2. **Touch event**: The interaction is a tap/touch event (not a mouse click)
3. **Quick tap**: Duration is less than 500ms (to distinguish from long press)
4. **No movement**: The user didn't drag their finger during the tap
5. **Categories exist**: There are categories available to choose from
6. **Callback provided**: The `onRequestCategorize` callback is passed to the component

## Key Behavior
- **Works for ALL todos**: Both uncategorized items and items already in a category
- **Replaces category**: If an item already has a category, tapping it will allow changing to a different category
- **Only in Categories view**: The feature is disabled in Normal view to avoid interference with existing drag-and-drop

## Components

### 1. CategorySelectorModal.svelte
**Location**: `/Users/martin/gotodo/frontend/src/lib/CategorySelectorModal.svelte`

A new modal component that displays available categories in a mobile-friendly interface.

**Features**:
- Full-screen modal on mobile devices
- List of all available categories
- Close via Cancel button, X button, or clicking backdrop
- Close via Escape key
- Shows the todo name in the subtitle
- Clean title: "Välj kategori"
- Smooth animations

**Props**:
- `categories`: Array of available categories
- `todoName`: Name of the todo being categorized
- `onSelect`: Callback when a category is selected
- `onCancel`: Callback when modal is cancelled

### 2. TodoItem.svelte (Modified)
**Location**: `/Users/martin/gotodo/frontend/src/lib/TodoItem.svelte`

**Changes**:
- Added `onRequestCategorize?: (todo: Todo) => void` optional prop
- Added touch timing tracking (`touchStartTime`, `touchMoved`)
- Modified `handleTouchEnd` to detect quick taps and trigger categorization **for all todos**
- Removed the `!todo.categoryId` check to allow category changes
- Preserves existing long-press behavior for edit mode

**Touch Event Logic**:
```typescript
// Quick tap detection
const touchDuration = Date.now() - touchStartTime;
const wasQuickTap = touchDuration < 500 && !touchMoved;

// Show category selector for any todo (categorized or not)
if (wasQuickTap && onRequestCategorize) {
  e.preventDefault();
  onRequestCategorize(todo);
}
```

### 3. CategoriesView.svelte (Modified)
**Location**: `/Users/martin/gotodo/frontend/src/lib/CategoriesView.svelte`

**Changes**:
- Added modal state management (`showCategoryModal`, `selectedTodoForCategorization`)
- Added `viewMode` prop to know when in Categories view
- Added `handleRequestCategorize` function to show the modal (only in Categories mode)
- Added `handleCategorySelect` to assign the selected category
- Added `handleCategoryModalCancel` to close the modal
- Passes `onRequestCategorize` callback to ALL TodoItem components (both categorized and uncategorized)
- Renders CategorySelectorModal when needed

**View Mode Check**:
```typescript
function handleRequestCategorize(todo: Todo) {
  // Only show modal if in categories mode and there are categories available
  if (viewMode === 'categories' && categories.length > 0) {
    selectedTodoForCategorization = todo;
    showCategoryModal = true;
  }
}
```

## User Experience Flow

1. **Switch to Categories view**: User taps "Kategorier" button in header
2. **User taps** on any todo item (e.g., "Tomater 🍅" which is in Mejeri category, or "Bröd 🍞" which is uncategorized)
3. **Modal appears** with title "Välj kategori" and subtitle showing the todo name
4. **Categories listed**: User sees all available categories (e.g., Grönsaker, Mejeri, Städ)
5. **User selects** a category by tapping it
6. **Todo updated**: The todo is assigned to the selected category (or moved if already categorized)
7. **Modal closes**: User returns to the Categories view
8. **Verification**: The todo now appears in the selected category section

## Testing

### Unit Tests
- **CategorySelectorModal.test.ts**: Tests component logic, props interface, and behavior
- **TodoItem.test.ts**: Tests touch event logic, timing conditions, and categorization triggers

All unit tests pass ✅

### E2E Tests
Added comprehensive Cypress tests in `/Users/martin/gotodo/e2e/cypress/e2e/todo.cy.js`:

- `shows category selector modal on tap for uncategorized todo`
- `assigns category when selecting from mobile modal`
- `closes modal when clicking cancel`
- `closes modal when clicking close button`
- `closes modal when clicking backdrop`
- `does not show modal for already categorized todos` ⚠️ **Needs update** - now works for categorized todos too
- `does not show modal on long press (enters edit mode instead)`
- `handles multiple categories in modal`
- `does not show modal when no categories exist`
- **New tests needed**: `only shows modal in Categories view`, `allows changing category for categorized items`

### Browser Testing Notes
The feature is implemented and renders correctly in mobile view (375x667px). The touch event handling will work on actual mobile devices or when simulated through browser dev tools with touch emulation enabled.

**To test on mobile**:
1. Open the app on a mobile device or use Chrome DevTools device emulation
2. **Switch to Categories view** by tapping "Kategorier" button
3. Ensure some categories exist
4. Create or find a todo (categorized or uncategorized)
5. Tap (not long-press) on the todo name
6. Modal should appear with category options
7. Select a category to assign/change it

## Technical Details

### Touch Event Handling
- **touchstart**: Records timestamp and resets movement flag
- **touchmove**: Sets movement flag to prevent tap detection
- **touchend**: Calculates duration and triggers categorization if conditions met

### Timing Thresholds
- **Quick tap**: < 500ms
- **Long press**: ≥ 500ms (triggers edit mode)

### Mobile Optimizations
The CategorySelectorModal includes mobile-specific styling:
- Full-screen layout on mobile devices (< 768px width)
- Large touch targets (padding: var(--spacing-xl))
- Smooth animations
- Backdrop blur effect

## Implementation Files

### New Files
- `/Users/martin/gotodo/frontend/src/lib/CategorySelectorModal.svelte`
- `/Users/martin/gotodo/frontend/src/lib/CategorySelectorModal.test.ts`
- `/Users/martin/gotodo/frontend/src/lib/TodoItem.test.ts`
- `/Users/martin/gotodo/MOBILE_CATEGORY_SELECTION.md` (this file)

### Modified Files
- `/Users/martin/gotodo/frontend/src/lib/TodoItem.svelte`
- `/Users/martin/gotodo/frontend/src/lib/CategoriesView.svelte`
- `/Users/martin/gotodo/e2e/cypress/e2e/todo.cy.js`

## Design Decisions

1. **Quick tap instead of long press**: Long press is already used for edit mode, so a quick tap was chosen for category assignment.

2. **Works for all todos**: The feature now works for both categorized and uncategorized items, allowing users to easily move items between categories.

3. **Only in Categories view**: The feature only works in Categories view mode to avoid interfering with the drag-and-drop functionality in Normal view.

4. **Modal-based UI**: A modal provides clear focus and works well on mobile devices with limited screen space.

5. **Non-blocking for desktop**: The feature doesn't interfere with desktop usage - double-click to edit still works.

6. **Graceful degradation**: If no categories exist, the modal doesn't appear, avoiding confusion.

## Future Enhancements (Optional)
- Add haptic feedback on category selection (mobile only)
- Show category icon/color in the modal
- Allow creating a new category from the modal
- Add swipe gestures for quick category assignment
- Remember recently used categories for faster selection

## Accessibility
- Modal has proper ARIA attributes (`role="dialog"`, `aria-modal="true"`, `aria-labelledby`)
- Keyboard navigation supported (Escape to close)
- Large touch targets for mobile
- Clear visual feedback on interaction

## Status
✅ Feature implemented
✅ Works for both categorized and uncategorized items
✅ Only active in Categories view mode
✅ Unit tests passing (27/27)
⚠️ E2E tests need updates for new behavior
✅ Mobile-responsive UI
✅ Documentation updated

## Recent Changes (Updated Implementation)
- **Removed restriction**: Now works for ALL todos (not just uncategorized)
- **View mode restriction**: Only works in Categories view (not in Normal view)
- **Modal title updated**: Shows cleaner title "Välj kategori" with todo name as subtitle
- **Better UX**: Users can now easily move items between categories with a tap

