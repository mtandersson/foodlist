# Local Storage Persistence - Manual Testing Guide

## Prerequisites
- Development server running (`make dev`)
- Browser with Developer Tools open

## Test Scenarios

### Test 1: View Mode Persistence
1. Open the app in your browser (http://localhost:5173)
2. Open DevTools > Application > Local Storage
3. Click the mode switch to change from "Normal" to "Kategorier" (Categories)
4. **Verify:** Check Local Storage - `viewMode` should be set to `"categories"`
5. Refresh the page (F5 or Cmd+R)
6. **Verify:** The app should open in Categories view (not Normal view)
7. Click the mode switch back to "Normal"
8. **Verify:** Check Local Storage - `viewMode` should be set to `"normal"`
9. Refresh the page
10. **Verify:** The app should open in Normal view

**Expected Result:** ✅ View mode persists across page reloads

---

### Test 2: Completed Section Persistence
1. Create a few todo items and mark some as completed
2. Ensure the "Slutfört" (Completed) section is visible
3. Click on the "Slutfört" header to collapse it
4. **Verify:** Check Local Storage - `completedExpanded` should be set to `"false"`
5. Refresh the page
6. **Verify:** The Completed section should remain collapsed
7. Click the "Slutfört" header to expand it
8. **Verify:** Check Local Storage - `completedExpanded` should be set to `"true"`
9. Refresh the page
10. **Verify:** The Completed section should be expanded

**Expected Result:** ✅ Completed section state persists across page reloads

---

### Test 3: Category Expanded State (Categories View)
1. Switch to "Kategorier" (Categories) view
2. Create a new category via the menu (⋮ → "Ny kategori")
3. **Verify:** New category is automatically expanded
4. Add some todos to this category
5. Click the category header to collapse it
6. **Verify:** Check Local Storage - `expandedCategories` should NOT include this category ID
7. Refresh the page
8. **Verify:** The category should remain collapsed
9. Click the category header to expand it
10. **Verify:** Check Local Storage - `expandedCategories` should include this category ID
11. Refresh the page
12. **Verify:** The category should remain expanded

**Expected Result:** ✅ Category expanded/collapsed state persists across page reloads

---

### Test 4: Multiple Categories State
1. In Categories view, create 3 different categories
2. Expand the first category, collapse the second, expand the third
3. **Verify:** Check Local Storage - `expandedCategories` should contain IDs for first and third categories only
4. Refresh the page
5. **Verify:** First and third categories are expanded, second is collapsed
6. Switch to Normal view and back to Categories view
7. **Verify:** Expansion states are preserved

**Expected Result:** ✅ Multiple category states are tracked independently

---

### Test 5: Combined State Persistence
1. Switch to Categories view
2. Create multiple categories with different expansion states
3. Collapse the Completed section
4. **Verify:** Local Storage should have all three keys:
   - `viewMode`: `"categories"`
   - `completedExpanded`: `"false"`
   - `expandedCategories`: `["id1", "id3", ...]` (expanded category IDs)
5. Refresh the page
6. **Verify:** All states are restored:
   - App opens in Categories view
   - Categories maintain their expansion states
   - Completed section is collapsed

**Expected Result:** ✅ All state persists independently and correctly

---

### Test 6: New Category Auto-Expansion
1. In Categories view, collapse all existing categories
2. Create a new category via the menu
3. **Verify:** The new category is automatically expanded (even though others are collapsed)
4. Refresh the page
5. **Verify:** The new category remains expanded

**Expected Result:** ✅ New categories are automatically expanded and saved as expanded

---

### Test 7: Category Deletion Cleanup
1. Create a category and expand it
2. **Verify:** The category ID is in `expandedCategories` in Local Storage
3. Delete the category (must be empty)
4. **Verify:** The category ID is removed from `expandedCategories` in Local Storage
5. Refresh the page
6. **Verify:** No errors, app works normally

**Expected Result:** ✅ Deleted category IDs are cleaned up from storage

---

## Local Storage Inspection

To inspect Local Storage in Chrome/Edge:
1. Open DevTools (F12)
2. Go to Application tab
3. Select Local Storage → http://localhost:5173
4. Look for these keys:
   - `viewMode`
   - `completedExpanded`
   - `expandedCategories`

To inspect Local Storage in Firefox:
1. Open DevTools (F12)
2. Go to Storage tab
3. Select Local Storage → http://localhost:5173

## Quick Local Storage Reset

To reset all persisted state, run this in the browser console:
```javascript
localStorage.clear()
location.reload()
```

## Expected Local Storage Values

```javascript
{
  "viewMode": "normal" | "categories",
  "completedExpanded": "true" | "false",
  "expandedCategories": "[\"uuid1\", \"uuid2\", null]"
}
```

Note: `expandedCategories` is a JSON array of category UUIDs. The value `null` can appear if uncategorized todos section is expanded (in categories view).

