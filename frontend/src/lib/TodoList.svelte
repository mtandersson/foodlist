<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { flip } from 'svelte/animate';
  import { fade, slide } from 'svelte/transition';
  import { v4 as uuidv4 } from 'uuid';
  import { createTodoStore } from './store';
  import TodoItem from './TodoItem.svelte';
  import ModeSwitch from './ModeSwitch.svelte';
  import CategoriesView from './CategoriesView.svelte';
  import CollapsibleSection from './CollapsibleSection.svelte';
  import CheckboxRing from './CheckboxRing.svelte';
  import CategoryBadge from './CategoryBadge.svelte';
  import AboutModal from './AboutModal.svelte';
  import { getStoredTheme, setTheme, type ThemeMode } from './theme';
  import type { Todo, AutocompleteSuggestion } from './types';
  import { smartSplit as smartSplitUtil, extractVoiceInput } from './voiceInputParser';

  // Determine WebSocket URL
  // Check for VITE_BACKEND_URL environment variable first (for testing remote servers)
  // Otherwise use the same path as the current page (to support secret paths like /secret-123/ws)
  let wsUrl: string;
  if (import.meta.env.VITE_BACKEND_URL) {
    // Use explicit backend URL from environment
    // Convert http/https to ws/wss and append /ws
    const backendUrl = import.meta.env.VITE_BACKEND_URL;
    const wsProtocol = backendUrl.startsWith('https') ? 'wss:' : 'ws:';
    const urlWithoutProtocol = backendUrl.replace(/^https?:\/\//, '');
    const [host, ...pathParts] = urlWithoutProtocol.split('/');
    const path = pathParts.length > 0 ? '/' + pathParts.join('/') : '';
    wsUrl = `${wsProtocol}//${host}${path}/ws`;
  } else {
    // Extract the base path from the current location (everything before the last segment)
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const basePath = window.location.pathname.endsWith('/') 
      ? window.location.pathname 
      : window.location.pathname + '/';
    wsUrl = `${wsProtocol}//${window.location.host}${basePath}ws`;
  }

  const store = createTodoStore(wsUrl);
  const { activeTodosCollapsed, completedTodos, completedTodosCollapsed, categories, activeTodosByCategoryCollapsed, categoryLookup, connectionState, userCount, listTitle, autocompleteSuggestions, errorMessage, isSynced, serverVersion } = store;
  
  // Get client version from build-time injection
  const clientVersion = import.meta.env.VITE_APP_VERSION || 'dev';

  // Watch for error messages related to category operations
  $effect(() => {
    const error = $errorMessage;
    if (error && creatingCategory) {
      categoryError = error;
      store.clearError(); // Clear global error since we're showing it in the dialog
    }
  });

  // Watch for successful category creation to close dialog
  let lastCategoryCount = $state(0);
  $effect(() => {
    const currentCount = $categories.length;
    if (creatingCategory && currentCount > lastCategoryCount) {
      // New category was created successfully
      creatingCategory = false;
      newCategoryName = '';
      categoryError = null;
    }
    lastCategoryCount = currentCount;
  });

  let newTodoName = $state('');
  let completedExpanded = $state(typeof localStorage !== 'undefined' && localStorage.getItem('completedExpanded') !== null ? localStorage.getItem('completedExpanded') === 'true' : true);
  let viewMode: 'normal' | 'categories' = $state((typeof localStorage !== 'undefined' && (localStorage.getItem('viewMode') as 'normal' | 'categories')) || 'normal');

  // Expanded categories state
  let expandedCategories = $state<Set<string | null>>(new Set());
  let isExpandedCategoriesInitialized = false;

  // Load expanded state from localStorage once on mount
  if (typeof localStorage !== 'undefined' && !isExpandedCategoriesInitialized) {
    const stored = localStorage.getItem('expandedCategories');
    if (stored) {
      try {
        const parsed = JSON.parse(stored);
        expandedCategories = new Set(parsed);
      } catch (e) {
        console.error('Failed to parse expandedCategories from localStorage', e);
      }
    }
    isExpandedCategoriesInitialized = true;
  }

  function saveExpandedCategories() {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('expandedCategories', JSON.stringify(Array.from(expandedCategories)));
    }
  }

  function handleToggleCategory(categoryId: string | null) {
    const newSet = new Set(expandedCategories);
    if (newSet.has(categoryId)) {
      newSet.delete(categoryId);
    } else {
      newSet.add(categoryId);
    }
    expandedCategories = newSet;
    saveExpandedCategories();
  }

  // Cleanup deleted categories from expanded set
  $effect(() => {
    if (!$isSynced) return;
    
    const currentCategories = $categories;
    const categoryIds = new Set(currentCategories.map(c => c.id));
    let changed = false;
    const newSet = new Set(expandedCategories);
    
    newSet.forEach(id => {
      if (id !== null && !categoryIds.has(id)) {
        newSet.delete(id);
        changed = true;
      }
    });
    
    if (changed) {
      expandedCategories = newSet;
      saveExpandedCategories();
    }
  });

  let pendingCategoryId: string | null = $state(null);
  let menuOpen = $state(false);
  let showAboutModal = $state(false);
  
  // Theme state
  let currentTheme: ThemeMode = $state(getStoredTheme());
  
  // List title state
  let editingTitle = $state(false);
  let titleInputValue = $state('');
  
  // New category state
  let creatingCategory = $state(false);
  let newCategoryName = $state('');
  let categoryError = $state<string | null>(null);

  // Autocomplete state
  let showAutocomplete = $state(false);
  let selectedAutocompleteIndex = $state(-1);
  let inputFocused = $state(false);
  let todoInputEl: HTMLInputElement | null = null;
  let inlineSuggestion = $state('');

  $effect(() => {
    const suggestions = $autocompleteSuggestions;
    const currentInput = newTodoName.toLowerCase();
    
    if (showAutocomplete && suggestions.length > 0 && currentInput) {
      const topSuggestion = suggestions[0].name.toLowerCase();
      if (topSuggestion.startsWith(currentInput)) {
        inlineSuggestion = suggestions[0].name.substring(newTodoName.length);
      } else {
        inlineSuggestion = '';
      }
    } else {
      inlineSuggestion = '';
    }
  });

  // Function to predict which category will be used for the current input
  function getPredictedCategory(): string | null {
    if (pendingCategoryId) {
      return getCategoryName(pendingCategoryId);
    }
    
    const name = (newTodoName + inlineSuggestion).trim();
    if (!name) return null;
    
    // Get the best category suggestion from autocomplete
    const suggestions = $autocompleteSuggestions;
    if (suggestions.length > 0) {
      const lowerName = name.toLowerCase();
      const similarSuggestions = suggestions.filter(s => 
        s.name.toLowerCase().includes(lowerName) || 
        lowerName.includes(s.name.toLowerCase()) ||
        s.name.toLowerCase().startsWith(lowerName.substring(0, 3))
      );
      
      if (similarSuggestions.length > 0) {
        const bestMatch = similarSuggestions[0];
        return getCategoryName(bestMatch.categoryId);
      }
    }
    
    // If still no category, try to find category from existing todos with similar names
    const activeTodos = $activeTodosCollapsed;
    const lowerName = name.toLowerCase();
    const similarTodos = activeTodos.filter(entry => {
      const todoName = entry.todo.name.toLowerCase();
      return todoName.includes(lowerName.substring(0, Math.min(3, lowerName.length))) ||
             lowerName.includes(todoName.substring(0, Math.min(3, todoName.length)));
    });
    
    if (similarTodos.length > 0) {
      const categoryCount = new Map<string | null, number>();
      similarTodos.forEach(entry => {
        const catId = entry.todo.categoryId ?? null;
        categoryCount.set(catId, (categoryCount.get(catId) || 0) + 1);
      });
      
      let maxCount = 0;
      let mostCommonCategory: string | null = null;
      categoryCount.forEach((count, catId) => {
        if (count > maxCount) {
          maxCount = count;
          mostCommonCategory = catId;
        }
      });
      
      return getCategoryName(mostCommonCategory);
    }
    
    return null;
  }

  function handleAddTodo() {
    const name = (newTodoName + inlineSuggestion).trim();
    if (name) {
      let categoryId = pendingCategoryId;
      
      // If no pending category, try to intelligently detect it
      if (!categoryId) {
        // Get the best category suggestion from autocomplete
        const suggestions = $autocompleteSuggestions;
        if (suggestions.length > 0) {
          // Find suggestions that match or are similar to our input
          const lowerName = name.toLowerCase();
          const similarSuggestions = suggestions.filter(s => 
            s.name.toLowerCase().includes(lowerName) || 
            lowerName.includes(s.name.toLowerCase()) ||
            s.name.toLowerCase().startsWith(lowerName.substring(0, 3))
          );
          
          if (similarSuggestions.length > 0) {
            // Use category from the most similar suggestion
            const bestMatch = similarSuggestions[0];
            categoryId = bestMatch.categoryId;
          }
        }
        
        // If still no category, try to find category from existing todos with similar names
        if (!categoryId) {
          // Get all active todos and find ones with similar names
          const activeTodos = $activeTodosCollapsed;
          const lowerName = name.toLowerCase();
          const similarTodos = activeTodos.filter(entry => {
            const todoName = entry.todo.name.toLowerCase();
            return todoName.includes(lowerName.substring(0, Math.min(3, lowerName.length))) ||
                   lowerName.includes(todoName.substring(0, Math.min(3, todoName.length)));
          });
          
          if (similarTodos.length > 0) {
            // Find the most common category among similar todos
            const categoryCount = new Map<string | null, number>();
            similarTodos.forEach(entry => {
              const catId = entry.todo.categoryId ?? null;
              categoryCount.set(catId, (categoryCount.get(catId) || 0) + 1);
            });
            
            // Get the most frequent category
            let maxCount = 0;
            let mostCommonCategory: string | null = null;
            categoryCount.forEach((count, catId) => {
              if (count > maxCount) {
                maxCount = count;
                mostCommonCategory = catId;
              }
            });
            
            categoryId = mostCommonCategory;
          }
        }
      }
      
      store.createTodo(name, categoryId);
      newTodoName = '';
      inlineSuggestion = '';
      pendingCategoryId = null;
      selectedAutocompleteIndex = -1;

      // Keep suggestions open for fast multi-add if the input is still focused.
      if (inputFocused) {
        store.requestAutocomplete(newTodoName);
        showAutocomplete = true;
        queueMicrotask(() => todoInputEl?.focus());
      } else {
        hideAutocomplete();
      }
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    const suggestions = $autocompleteSuggestions;
    
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (suggestions.length > 0) {
        selectedAutocompleteIndex = Math.min(selectedAutocompleteIndex + 1, suggestions.length - 1);
        updateScroll();
      }
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (suggestions.length > 0) {
        selectedAutocompleteIndex = Math.max(selectedAutocompleteIndex - 1, 0);
        updateScroll();
      }
    } else if (e.key === 'Enter') {
        e.preventDefault();
        if (inlineSuggestion) {
          const name = newTodoName + inlineSuggestion;
          const suggestion = $autocompleteSuggestions.find(s => s.name.toLowerCase() === name.toLowerCase());
          store.createTodo(name, suggestion?.categoryId ?? null);
          newTodoName = '';
          inlineSuggestion = '';
          pendingCategoryId = null;
          selectedAutocompleteIndex = -1;
          if (inputFocused) {
            store.requestAutocomplete(newTodoName);
            showAutocomplete = true;
            queueMicrotask(() => todoInputEl?.focus());
          } else {
            hideAutocomplete();
          }
        } else if (selectedAutocompleteIndex >= 0 && selectedAutocompleteIndex < suggestions.length) {
          selectSuggestion(suggestions[selectedAutocompleteIndex]);
        } else {
          handleAddTodo();
        }
    } else if (e.key === 'Tab' || e.key === 'ArrowRight') {
      if (inlineSuggestion && todoInputEl && todoInputEl.selectionStart === newTodoName.length) {
        e.preventDefault();
        newTodoName += inlineSuggestion;
        inlineSuggestion = '';
      }
    } else if (e.key === 'Escape') {
      hideAutocomplete();
    }
  }

  function updateScroll() {
    const list = document.querySelector('.autocomplete-suggestions');
    const item = document.querySelector('.suggestion-item.selected');
    if (list && item) {
      const listRect = list.getBoundingClientRect();
      const itemRect = item.getBoundingClientRect();
      if (itemRect.bottom > listRect.bottom) {
        list.scrollTop += itemRect.bottom - listRect.bottom;
      } else if (itemRect.top < listRect.top) {
        list.scrollTop -= listRect.top - itemRect.top;
      }
    }
  }

  function handleInput() {
    // Request autocomplete on every keystroke
    store.requestAutocomplete(newTodoName);
    showAutocomplete = true;
    selectedAutocompleteIndex = -1;
    pendingCategoryId = null;
  }

  function handleInputFocus() {
    inputFocused = true;
    // Request autocomplete when focusing even if empty
    store.requestAutocomplete(newTodoName);
    showAutocomplete = true;
  }

  function handleInputBlur() {
    inputFocused = false;
    // Delay hiding to allow click on suggestion
    setTimeout(() => {
      if (!inputFocused) {
        hideAutocomplete();
      }
    }, 150);
  }

  function selectSuggestion(suggestion: AutocompleteSuggestion) {
    // Immediately add the todo (without closing the dropdown)
    store.createTodo(suggestion.name, suggestion.categoryId ?? null);
    newTodoName = '';
    inlineSuggestion = '';
    pendingCategoryId = null;
    selectedAutocompleteIndex = -1;

    if (inputFocused) {
      store.requestAutocomplete(newTodoName);
      showAutocomplete = true;
      queueMicrotask(() => todoInputEl?.focus());
    } else {
      hideAutocomplete();
    }
  }

  function hideAutocomplete() {
    showAutocomplete = false;
    selectedAutocompleteIndex = -1;
    inlineSuggestion = '';
    store.clearAutocomplete();
  }

  function toggleCompletedSection() {
    completedExpanded = !completedExpanded;
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('completedExpanded', String(completedExpanded));
    }
  }

  function handleModeChange(mode: 'normal' | 'categories') {
    viewMode = mode;
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('viewMode', mode);
    }
  }

  function getCategoryName(categoryId: string | null | undefined): string | null {
    if (!categoryId) return null;
    const cat = $categoryLookup.get(categoryId);
    return cat ? cat.name : null;
  }

  function handleCategorize(id: string, categoryId: string | null) {
    store.categorizeTodo(id, categoryId);
  }

  function handleCreateCategory(name: string) {
    const id = uuidv4();
    
    // Expand immediately (Optimistic update in store will ensure it exists)
    const newSet = new Set(expandedCategories);
    newSet.add(id);
    expandedCategories = newSet;
    saveExpandedCategories();

    store.createCategory(name, id).then(
      () => {
        // Success - dialog will be closed by $effect watching categories
        categoryError = null;
      },
      (error: string) => {
        // Error - show in dialog
        categoryError = error;
      }
    );
  }

  function handleRenameCategory(id: string, name: string) {
    store.renameCategory(id, name);
  }

  function handleDeleteCategory(id: string) {
    store.deleteCategory(id);
  }

  function handleReorderCategory(id: string, sortOrder: number) {
    store.reorderCategory(id, sortOrder);
  }

  // Title editing
  function startEditingTitle() {
    editingTitle = true;
    titleInputValue = $listTitle;
  }

  function finishEditingTitle() {
    const newTitle = titleInputValue.trim();
    if (newTitle && newTitle !== $listTitle) {
      store.setListTitle(newTitle);
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

  // Menu state
  function toggleMenu() {
    menuOpen = !menuOpen;
  }

  function closeMenu() {
    menuOpen = false;
  }

  function handleAboutClick() {
    showAboutModal = true;
    closeMenu();
  }

  function handleAboutClose() {
    showAboutModal = false;
  }

  function handleNewCategory() {
    creatingCategory = true;
    newCategoryName = '';
    categoryError = null;
    closeMenu();
    // Focus the input after it renders
    setTimeout(() => {
      const input = document.querySelector('.new-category-input') as HTMLInputElement;
      input?.focus();
    }, 0);
  }

  function finishCreatingCategory() {
    const name = newCategoryName.trim();
    if (name) {
      handleCreateCategory(name);
      // Don't close immediately - wait for server response
    }
  }

  function cancelCreatingCategory() {
    creatingCategory = false;
    newCategoryName = '';
    categoryError = null;
  }

  function handleNewCategoryKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      finishCreatingCategory();
    } else if (e.key === 'Escape') {
      cancelCreatingCategory();
    }
  }

  function handleThemeChange(theme: ThemeMode) {
    currentTheme = theme;
    setTheme(theme);
    closeMenu();
  }

  function getThemeLabel(theme: ThemeMode): string {
    switch (theme) {
      case 'light': return '☀️ Ljust';
      case 'dark': return '🌙 Mörkt';
      case 'auto': return '⚙️ Auto';
    }
  }

  function getThemeIcon(theme: ThemeMode): string {
    switch (theme) {
      case 'light': return '☀️';
      case 'dark': return '🌙';
      case 'auto': return '⚙️';
    }
  }

  // Drag and drop state
  let draggedId: string | null = $state(null);
  let dropTargetId: string | null = $state(null);
  let dropPosition: 'above' | 'below' | null = $state(null);
  let isDragging = $state(false);

  function computeSortOrder(todos: Array<{ sortOrder: number }>, newIndex: number): number {
    if (todos.length === 0) return 0;
    if (newIndex <= 0) return todos[0].sortOrder + 1000;
    if (newIndex >= todos.length) return todos[todos.length - 1].sortOrder - 1000;
    const above = todos[newIndex - 1].sortOrder;
    const below = todos[newIndex].sortOrder;
    return Math.floor((above + below) / 2);
  }

  function handleDragStart(e: DragEvent, todo: Todo) {
    draggedId = todo.id;
    dropTargetId = null;
    dropPosition = null;
    isDragging = true;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', todo.id);
    }
  }

  function wouldMove(targetId: string, position: 'above' | 'below'): boolean {
    const entries = $activeTodosCollapsed;
    const todos = entries.map((e) => e.todo);
    const draggedIndex = todos.findIndex((t) => t.id === draggedId);
    const targetIndex = todos.findIndex((t) => t.id === targetId);
    if (draggedIndex === -1 || targetIndex === -1) return false;
    const newIndex = position === 'below' ? targetIndex + 1 : targetIndex;
    // No-op if dropping in same or adjacent slot that keeps order
    return !(newIndex === draggedIndex || newIndex === draggedIndex + 1);
  }

  function handleDragOverItem(e: DragEvent, todo: Todo) {
    e.preventDefault();
    if (!draggedId || draggedId === todo.id) return;

    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const position: 'above' | 'below' = e.clientY < rect.top + rect.height / 2 ? 'above' : 'below';

    if (!wouldMove(todo.id, position)) {
      dropTargetId = null;
      dropPosition = null;
      return;
    }

    dropTargetId = todo.id;
    dropPosition = position;
  }

  function handleDragOverList(e: DragEvent) {
    e.preventDefault();
    if (!draggedId) return;

    const listEl = e.currentTarget as HTMLElement;
    const wrappers = Array.from(listEl.querySelectorAll('.todo-wrapper')) as HTMLElement[];
    const todos = $activeTodosCollapsed.map((e) => e.todo);

    if (!wrappers.length || !todos.length) {
      dropTargetId = null;
      dropPosition = null;
      return;
    }

    const firstRect = wrappers[0].getBoundingClientRect();
    const lastRect = wrappers[wrappers.length - 1].getBoundingClientRect();
    const { clientY } = e;

    // If near the very top, target before first
    if (clientY < firstRect.top + firstRect.height / 3 && wouldMove(todos[0].id, 'above')) {
      dropTargetId = todos[0].id;
      dropPosition = 'above';
      return;
    }

    // If near the very bottom, target after last
    if (clientY > lastRect.bottom - lastRect.height / 3 && wouldMove(todos[todos.length - 1].id, 'below')) {
      dropTargetId = todos[todos.length - 1].id;
      dropPosition = 'below';
      return;
    }
    // Otherwise let item-level handlers set precise position
  }

  function handleDragLeaveItem(e: DragEvent, todoId: string) {
    const relatedTarget = e.relatedTarget as Node | null;
    if (!relatedTarget || !(e.currentTarget as Node).contains(relatedTarget)) {
      if (dropTargetId === todoId) {
        dropTargetId = null;
        dropPosition = null;
      }
    }
  }

  function finishDrop() {
    if (!draggedId) return;

    // Only reorder if we have a valid drop target
    if (!dropTargetId || !dropPosition) {
      clearDragState();
      return;
    }

    const todos = $activeTodosCollapsed.map((e) => e.todo);
    const targetIndex = todos.findIndex((t) => t.id === dropTargetId);
    
    if (targetIndex === -1) {
      clearDragState();
      return;
    }

    const newIndex = dropPosition === 'below' ? targetIndex + 1 : targetIndex;
    const newSortOrder = computeSortOrder(todos, newIndex);
    store.reorder(draggedId, newSortOrder);
    clearDragState();
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    finishDrop();
  }

  function handleDragEnd() {
    clearDragState();
  }

  function clearDragState() {
    draggedId = null;
    dropTargetId = null;
    dropPosition = null;
    isDragging = false;
  }

  // Get all previous items from server (for smart matching)
  function getAllPreviousItems(): Promise<string[]> {
    return new Promise((resolve) => {
      let timeoutId: number | null = null;
      let unsubscribe: (() => void) | null = null;
      let suggestionsReceived = false;
      
      // Clear any existing suggestions first
      store.clearAutocomplete();
      
      // Use a small delay to ensure the clear has taken effect
      queueMicrotask(() => {
        // Set up a one-time listener for autocomplete response
        unsubscribe = store.autocompleteSuggestions.subscribe((suggestions) => {
          // Only process suggestions that arrive after our request
          if (!suggestionsReceived && suggestions.length > 0) {
            suggestionsReceived = true;
            
            // Extract item names from suggestions
            const items = suggestions.map(s => s.name);
            
            // Clean up
            if (timeoutId !== null) clearTimeout(timeoutId);
            if (unsubscribe) unsubscribe();
            store.clearAutocomplete();
            
            resolve(items);
          }
        });
        
        // Request autocomplete with empty query to get all items
        store.requestAutocomplete('');
      });
      
      // Timeout after 2 seconds - if no response, return empty array
      timeoutId = window.setTimeout(() => {
        if (!suggestionsReceived) {
          suggestionsReceived = true;
          if (unsubscribe) unsubscribe();
          store.clearAutocomplete();
          resolve([]);
        }
      }, 2000);
    });
  }

  // Extract queries from voice input (removes "handla" prefix and smart-splits into multiple items)
  async function extractQueries(text: string): Promise<string[]> {
    if (!text) return [];
    
    // Extract voice input (removes prefix)
    const itemsText = extractVoiceInput(text);
    
    // Get all previous items for smart matching
    const knownItems = await getAllPreviousItems();
    
    // Use smart splitting to match known items first
    return smartSplitUtil(itemsText, knownItems);
  }

  // Get best match from autocomplete using server suggestions
  function getBestMatch(query: string): Promise<string> {
    return new Promise((resolve) => {
      if (!query) {
        resolve('');
        return;
      }
      
      let timeoutId: number | null = null;
      let unsubscribe: (() => void) | null = null;
      let suggestionsReceived = false;
      
      // Clear any existing suggestions first
      store.clearAutocomplete();
      
      // Use a small delay to ensure the clear has taken effect
      queueMicrotask(() => {
        // Set up a one-time listener for autocomplete response
        unsubscribe = store.autocompleteSuggestions.subscribe((suggestions) => {
          // Only process suggestions that arrive after our request (ignore initial empty state)
          // Check if we have suggestions and haven't already resolved
          if (!suggestionsReceived && suggestions.length > 0) {
            suggestionsReceived = true;
            
            // Use the first suggestion (best match from server, sorted by frequency/score)
            const bestMatch = suggestions[0].name;
            
            // Clean up
            if (timeoutId !== null) clearTimeout(timeoutId);
            if (unsubscribe) unsubscribe();
            store.clearAutocomplete();
            
            resolve(bestMatch);
          }
        });
        
        // Request autocomplete after subscription is set up
        store.requestAutocomplete(query);
      });
      
      // Timeout after 2 seconds - if no response, use the query as-is
      timeoutId = window.setTimeout(() => {
        if (!suggestionsReceived) {
          suggestionsReceived = true;
          if (unsubscribe) unsubscribe();
          store.clearAutocomplete();
          resolve(query); // Fallback to original query
        }
      }, 2000);
    });
  }

  // Handle URL parameters for shortcuts
  async function handleShortcutParams() {
    const urlParams = new URLSearchParams(window.location.search);
    const action = urlParams.get('action');
    const text = urlParams.get('text');
    
    if (action === 'add' && text) {
      const voiceInput = decodeURIComponent(text);
      
      // Wait for store to sync before processing
      const checkSync = async () => {
        if ($isSynced) {
          // Extract queries using smart splitting (matches known items first)
          const queries = await extractQueries(voiceInput);
          
          if (queries.length > 0) {
            // Get best match for each item from server autocomplete
            // Process sequentially to avoid overwhelming the server
            for (const query of queries) {
              const itemName = await getBestMatch(query);
              if (itemName) {
                store.createTodo(itemName);
                // Small delay between items to ensure proper ordering
                await new Promise(resolve => setTimeout(resolve, 50));
              }
            }
          }
          // Clean up URL parameters
          const newUrl = window.location.pathname;
          window.history.replaceState({}, '', newUrl);
        } else {
          // Retry after a short delay
          setTimeout(checkSync, 100);
        }
      };
      checkSync();
    }
  }

  onMount(() => {
    handleShortcutParams();
  });

  onDestroy(() => {
    store.destroy();
  });
</script>

<div class="todo-list-container">
  <header class="header">
    {#if editingTitle}
      <input
        type="text"
        class="title-input"
        bind:value={titleInputValue}
        onkeydown={handleTitleKeydown}
        onblur={finishEditingTitle}
      />
    {:else}
      <button class="title" onclick={startEditingTitle} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); startEditingTitle(); } }}>
        {$listTitle}
      </button>
    {/if}
    <div class="header-right">
      <ModeSwitch value={viewMode} on:change={(e) => handleModeChange(e.detail)} />
      <span class="member-count" aria-label="Connected users">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
          <circle cx="9" cy="7" r="4"></circle>
          <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
          <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
        </svg>
        {$userCount}
      </span>
      <div class="menu-wrapper">
        <button class="menu-btn" aria-label="Menu" onclick={toggleMenu}>
          <svg viewBox="0 0 24 24" fill="currentColor">
            <circle cx="12" cy="5" r="2"></circle>
            <circle cx="12" cy="12" r="2"></circle>
            <circle cx="12" cy="19" r="2"></circle>
          </svg>
        </button>
        {#if menuOpen}
          <div class="menu-dropdown">
            <button class="menu-item" onclick={handleNewCategory}>
              <span class="menu-icon">➕</span>
              Ny kategori
            </button>
            <div class="menu-divider"></div>
            <div class="menu-section-title">Tema</div>
            <button 
              class="menu-item" 
              class:selected={currentTheme === 'light'}
              onclick={() => handleThemeChange('light')}
            >
              <span class="menu-icon">{getThemeIcon('light')}</span>
              Ljust
              {#if currentTheme === 'light'}
                <span class="menu-checkmark">✓</span>
              {/if}
            </button>
            <button 
              class="menu-item" 
              class:selected={currentTheme === 'dark'}
              onclick={() => handleThemeChange('dark')}
            >
              <span class="menu-icon">{getThemeIcon('dark')}</span>
              Mörkt
              {#if currentTheme === 'dark'}
                <span class="menu-checkmark">✓</span>
              {/if}
            </button>
            <button 
              class="menu-item" 
              class:selected={currentTheme === 'auto'}
              onclick={() => handleThemeChange('auto')}
            >
              <span class="menu-icon">{getThemeIcon('auto')}</span>
              Auto
              {#if currentTheme === 'auto'}
                <span class="menu-checkmark">✓</span>
              {/if}
            </button>
            <div class="menu-divider"></div>
            <button class="menu-item" onclick={handleAboutClick}>
              <span class="menu-icon">ℹ️</span>
              Om
            </button>
          </div>
          <button class="menu-backdrop" onclick={closeMenu} aria-label="Close menu"></button>
        {/if}
      </div>
    </div>
  </header>

  <div class="scrollable-content">
    <!-- Connection status indicator -->
    {#if $connectionState !== 'CONNECTED'}
      <div class="connection-status" transition:slide>
        {#if $connectionState === 'CONNECTING'}
          Ansluter...
        {:else if $connectionState === 'RECONNECTING'}
          Återansluter...
        {:else}
          Frånkopplad
        {/if}
      </div>
    {/if}

    <!-- New Category Input -->
    {#if creatingCategory}
      <div class="new-category-wrapper" transition:slide>
        <div class="new-category-input-container">
          <input
            type="text"
            class="new-category-input"
            class:error={categoryError}
            bind:value={newCategoryName}
            onkeydown={handleNewCategoryKeydown}
            onblur={finishCreatingCategory}
            placeholder="Namn på ny kategori..."
          />
          {#if categoryError}
            <div class="category-error" transition:slide>
              {categoryError}
            </div>
          {/if}
          <div class="new-category-actions">
            <button
              type="button"
              class="new-category-btn primary"
              onclick={finishCreatingCategory}
              disabled={!newCategoryName.trim()}
            >
              Skapa
            </button>
            <button
              type="button"
              class="new-category-btn"
              onclick={cancelCreatingCategory}
            >
              Avbryt
            </button>
          </div>
        </div>
      </div>
    {/if}

    {#if viewMode === 'normal'}
      <!-- Active todos -->
      <div
        class="todos-section"
        ondragover={handleDragOverList}
        ondrop={handleDrop}
        role="list"
      >
        {#each $activeTodosCollapsed as entry (entry.todo.id)}
          <div
            class="todo-container"
            class:spacer-top={dropTargetId === entry.todo.id && dropPosition === 'above'}
            class:spacer-bottom={dropTargetId === entry.todo.id && dropPosition === 'below'}
            animate:flip={{ duration: isDragging ? 0 : 300 }}
            transition:fade={{ duration: 200 }}
          >
            <div
              class="todo-wrapper"
              role="listitem"
              aria-grabbed={draggedId === entry.todo.id}
              class:drop-above={dropTargetId === entry.todo.id && dropPosition === 'above'}
              class:drop-below={dropTargetId === entry.todo.id && dropPosition === 'below'}
              draggable="true"
              ondragstart={(e) => handleDragStart(e, entry.todo)}
              ondragover={(e) => handleDragOverItem(e, entry.todo)}
              ondragleave={(e) => handleDragLeaveItem(e, entry.todo.id)}
              ondrop={handleDrop}
              ondragend={handleDragEnd}
              class:dragging={draggedId === entry.todo.id}
            >
              <TodoItem
                todo={entry.todo}
                duplicateCount={entry.count}
                categoryName={getCategoryName(entry.todo.categoryId ?? null)}
                onToggleComplete={store.toggleComplete}
                onToggleStar={store.toggleStar}
                onRename={store.rename}
              />
            </div>
          </div>
        {/each}
      </div>

      <!-- Completed section -->
      {#if $completedTodosCollapsed.length > 0}
        <CollapsibleSection
          title="Slutfört"
          expanded={completedExpanded}
          onToggle={toggleCompletedSection}
        >
          {#each $completedTodosCollapsed as collapsed (collapsed.todo.id)}
            <div
              class="todo-wrapper"
              animate:flip={{ duration: 300 }}
              transition:fade={{ duration: 200 }}
            >
              <TodoItem
                todo={collapsed.todo}
                duplicateCount={collapsed.count}
                categoryName={getCategoryName(collapsed.todo.categoryId ?? null)}
                onToggleComplete={store.toggleComplete}
                onToggleStar={store.toggleStar}
                onRename={store.rename}
              />
            </div>
          {/each}
        </CollapsibleSection>
      {/if}
    {:else}
      <CategoriesView
        categories={$categories}
        activeTodosByCategory={$activeTodosByCategoryCollapsed}
        completedTodos={$completedTodosCollapsed}
        getCategoryName={getCategoryName}
        onToggleComplete={store.toggleComplete}
        onToggleStar={store.toggleStar}
        onRename={store.rename}
        onDeleteCategory={handleDeleteCategory}
        onCategorizeTodo={handleCategorize}
        onReorderCategory={handleReorderCategory}
        onRenameCategory={handleRenameCategory}
        onReorder={store.reorder}
        completedExpanded={completedExpanded}
        onToggleCompletedSection={toggleCompletedSection}
        expandedCategories={expandedCategories}
        onToggleCategory={handleToggleCategory}
        viewMode={viewMode}
      />
    {/if}
  </div>

  <!-- Add todo input at bottom -->
  <div class="add-todo-wrapper">
    {#if showAutocomplete && $autocompleteSuggestions.length > 0}
      <div class="autocomplete-dropdown" transition:slide={{ duration: 150 }}>
        {#each $autocompleteSuggestions as suggestion, index}
          <button
            type="button"
            class="autocomplete-item"
            class:selected={index === selectedAutocompleteIndex}
            onmousedown={(e) => { e.preventDefault(); selectSuggestion(suggestion); }}
            onmouseenter={() => selectedAutocompleteIndex = index}
          >
            <div class="autocomplete-item-main">
              <span>{suggestion.name}</span>
              {#if suggestion.categoryName}
                <CategoryBadge name={suggestion.categoryName!} />
              {/if}
            </div>
          </button>
        {/each}
      </div>
    {/if}
    <form class="add-todo-bottom" onsubmit={(e) => { e.preventDefault(); handleAddTodo(); }}>
      <div class="add-todo-icon">
        <svg class="icon-plus" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        <div class="icon-ring">
          <CheckboxRing size="small" />
        </div>
      </div>
      <div class="input-wrapper">
        <input
          type="text"
          placeholder="Lägg till en uppgift"
          bind:this={todoInputEl}
          bind:value={newTodoName}
          onkeydown={handleKeydown}
          oninput={handleInput}
          onfocus={handleInputFocus}
          onblur={handleInputBlur}
          aria-label="Ny uppgift"
          autocomplete="off"
          class:has-category={newTodoName.trim() && getPredictedCategory()}
        />
        {#if inlineSuggestion}
          <div class="inline-suggestion">{newTodoName}<span class="suggestion-text">{inlineSuggestion}</span></div>
        {/if}
        {#if newTodoName.trim()}
          {@const predictedCategory = getPredictedCategory()}
          {#if predictedCategory}
            <div class="predicted-category">
              <CategoryBadge name={predictedCategory!} showIcon={true} size="small" />
            </div>
          {/if}
        {/if}
      </div>
    </form>
  </div>
</div>

{#if showAboutModal}
  <AboutModal 
    clientVersion={clientVersion}
    serverVersion={$serverVersion}
    onCancel={handleAboutClose}
  />
{/if}

<style>
  .todo-list-container {
    width: min(var(--container-viewport-width), var(--container-max-width));
    max-width: var(--container-max-width);
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    height: calc(100dvh - var(--spacing-lg));
    padding: 0 var(--spacing-2xl);
    padding-bottom: var(--spacing-lg);
  }

  @media (max-width: 768px) {
    .todo-list-container {
      width: 100%;
      padding: env(safe-area-inset-top) var(--spacing-md) var(--spacing-lg);
      height: 100dvh;
      /* Prevent layout shift when iOS keyboard appears */
      position: relative;
    }
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--spacing-2xl) 0 var(--spacing-xl);
    flex-shrink: 0;
  }

  @media (max-width: 768px) {
    .header {
      padding: var(--spacing-lg) 0 var(--spacing-md);
    }
  }

  .scrollable-content {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    padding-right: var(--spacing-xs);
    min-height: 0;
    /* Force scrollbar to always be visible for consistent layout */
    scrollbar-gutter: stable;
    /* Prevent iOS bounce scrolling from interfering with layout */
    -webkit-overflow-scrolling: touch;
  }

  .scrollable-content::-webkit-scrollbar {
    width: var(--scrollbar-width);
  }

  .scrollable-content::-webkit-scrollbar-track {
    background: transparent;
  }

  .scrollable-content::-webkit-scrollbar-thumb {
    background: var(--surface-muted);
    border-radius: var(--spacing-xs);
  }

  .scrollable-content::-webkit-scrollbar-thumb:hover {
    background: var(--surface-muted-strong);
  }

  @media (max-width: 768px) {
    .scrollable-content {
      scrollbar-width: none; /* Firefox */
      -ms-overflow-style: none; /* IE and Edge */
      /* Ensure content can scroll properly when keyboard appears */
      padding-bottom: var(--spacing-md);
    }

    .scrollable-content::-webkit-scrollbar {
      display: none; /* Chrome, Safari, Opera */
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: var(--spacing-lg);
  }

  .member-count {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    color: var(--text-on-primary);
    font-size: var(--font-size-sm);
  }

  .member-count svg {
    width: var(--icon-sm);
    height: var(--icon-sm);
  }

  .menu-btn {
    background: none;
    border: none;
    color: var(--text-on-primary);
    cursor: pointer;
    padding: var(--spacing-xs);
  }

  .menu-btn svg {
    width: var(--icon-md);
    height: var(--icon-md);
  }

  .menu-wrapper {
    position: relative;
  }

  .menu-dropdown {
    position: absolute;
    top: calc(100% + var(--spacing-sm));
    right: 0;
    background: var(--card-bg);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-2xl);
    min-width: var(--menu-min-width);
    z-index: var(--z-index-menu);
    overflow: hidden;
  }

  .menu-item {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    width: 100%;
    padding: var(--spacing-md) var(--spacing-lg);
    border: none;
    background: transparent;
    color: var(--text-primary);
    font-size: var(--font-size-sm);
    text-align: left;
    cursor: pointer;
    transition: background var(--transition-normal);
    position: relative;
  }

  .menu-item:hover {
    background: var(--surface-muted);
  }

  .menu-item.selected {
    background: var(--surface-light);
  }

  .menu-icon {
    font-size: var(--font-size-xl);
  }

  .menu-checkmark {
    margin-left: auto;
    font-size: var(--font-size-base);
    color: var(--primary-color);
    font-weight: var(--font-weight-bold);
  }

  .menu-divider {
    height: 1px;
    background: var(--border-color);
    margin: var(--spacing-sm) 0;
  }

  .menu-section-title {
    padding: var(--spacing-sm) var(--spacing-lg);
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-semibold);
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .menu-backdrop {
    position: fixed;
    inset: 0;
    background: transparent;
    border: none;
    cursor: default;
    z-index: var(--z-index-dropdown);
  }

  .title {
    color: var(--text-on-primary);
    font-size: var(--font-size-2xl);
    font-weight: var(--font-weight-semibold);
    margin: 0;
    cursor: pointer;
    transition: opacity var(--transition-normal);
    padding: var(--spacing-sm);
    border-radius: var(--radius-sm);
    background: transparent;
    border: none;
    text-align: left;
    font-family: inherit;
  }

  .title:hover {
    opacity: var(--opacity-hover);
    background: var(--surface-light);
  }

  .title:focus {
    outline: var(--stroke-thin) solid rgba(var(--card-bg-rgb), 0.5);
    outline-offset: var(--stroke-thin);
  }

  .title-input {
    color: var(--text-on-primary);
    font-size: var(--font-size-2xl);
    font-weight: var(--font-weight-semibold);
    margin: 0;
    padding: var(--spacing-sm);
    border: var(--stroke-thin) solid var(--text-on-primary);
    border-radius: var(--radius-sm);
    background: var(--surface-muted);
    width: 100%;
    outline: none;
    transition: all var(--transition-normal);
  }

  .title-input:focus {
    border-color: var(--text-on-primary);
    background: var(--surface-muted-strong);
  }

  .connection-status {
    background: var(--surface-muted);
    color: var(--text-on-primary);
    padding: var(--spacing-sm) var(--spacing-lg);
    border-radius: var(--radius-sm);
    text-align: center;
    margin-bottom: var(--spacing-lg);
    font-size: var(--font-size-sm);
  }

  .new-category-wrapper {
    margin-bottom: var(--spacing-lg);
  }

  .new-category-input-container {
    background: var(--card-bg);
    border-radius: var(--radius-md);
    padding: var(--spacing-lg);
    box-shadow: var(--shadow-lg);
  }

  .new-category-input {
    width: 100%;
    padding: var(--spacing-md) var(--spacing-lg);
    font-size: var(--font-size-base);
    border: var(--stroke-thin) solid var(--surface-muted);
    border-radius: var(--radius-sm);
    background: var(--surface-muted);
    color: var(--text-primary);
    outline: none;
    font-family: inherit;
    transition: all var(--transition-normal);
    margin-bottom: var(--spacing-md);
  }

  .new-category-input.error {
    border-color: var(--danger);
  }

  .new-category-input:focus {
    border-color: var(--text-primary);
    background: var(--surface-muted-strong);
  }

  .new-category-input.error:focus {
    border-color: var(--danger);
  }

  .new-category-input::placeholder {
    color: var(--text-muted);
  }

  .category-error {
    color: var(--danger);
    font-size: var(--font-size-sm);
    margin-bottom: var(--spacing-md);
    padding: var(--spacing-sm) var(--spacing-md);
    background: rgba(var(--danger-rgb), 0.1);
    border-radius: var(--radius-sm);
    border-left: 3px solid var(--danger);
  }

  .new-category-actions {
    display: flex;
    gap: var(--spacing-sm);
    justify-content: flex-end;
  }

  .new-category-btn {
    padding: var(--spacing-sm) var(--spacing-lg);
    font-size: var(--font-size-sm);
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
    font-family: inherit;
    font-weight: var(--font-weight-medium);
    transition: all var(--transition-normal);
    background: var(--surface-muted);
    color: var(--text-primary);
  }

  .new-category-btn:hover:not(:disabled) {
    background: var(--surface-muted-strong);
    transform: translateY(-1px);
  }

  .new-category-btn.primary {
    background: var(--text-on-primary);
    color: var(--primary-bg);
  }

  .new-category-btn.primary:hover:not(:disabled) {
    opacity: var(--opacity-hover);
  }

  .new-category-btn:disabled {
    opacity: var(--opacity-disabled);
    cursor: not-allowed;
  }

  .add-todo-bottom {
    display: flex;
    align-items: center;
    gap: var(--spacing-lg);
    padding: var(--font-size-xl) var(--spacing-xl);
    background: rgba(255, 255, 255, 0.1); /* Fallback */
    background: var(--surface-muted);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-md);
    cursor: text;
    transition: all var(--transition-normal);
    /* Ensure proper touch target size on iOS */
    min-height: 44px;
  }

  .add-todo-bottom:hover {
    background: rgba(255, 255, 255, 0.15);
    background: var(--surface-muted-strong);
    box-shadow: var(--shadow-lg);
  }

  .add-todo-bottom:focus-within {
    background: var(--card-bg); /* Opaque when focused for readability */
    box-shadow: var(--shadow-focus);
  }

  .add-todo-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: var(--icon-xl);
    height: var(--icon-xl);
    flex-shrink: 0;
    color: var(--text-on-primary);
    opacity: 0.7;
    transition: color var(--transition-normal), opacity var(--transition-normal);
    position: relative;
  }

  .add-todo-icon .icon-plus {
    width: var(--icon-lg);
    height: var(--icon-lg);
    transition: opacity var(--transition-normal);
    opacity: 1;
    position: absolute;
  }

  .add-todo-icon .icon-ring {
    transition: opacity var(--transition-normal);
    opacity: 0;
  }

  .add-todo-bottom:focus-within .add-todo-icon {
    color: var(--text-primary);
    opacity: 1;
  }

  .add-todo-bottom:focus-within .add-todo-icon .icon-plus {
    opacity: 0;
  }

  .add-todo-bottom:focus-within .add-todo-icon .icon-ring {
    opacity: 1;
  }

  .add-todo-bottom input {
    flex: 1;
    border: none;
    outline: none;
    font-size: var(--font-size-lg);
    color: var(--text-on-primary);
    background: transparent;
    font-family: inherit;
    transition: color var(--transition-normal), opacity var(--transition-normal);
    opacity: 0.8;
    /* Prevent iOS zoom on focus */
    font-size: max(var(--font-size-lg), 16px);
    /* Ensure proper appearance on iOS */
    -webkit-appearance: none;
    appearance: none;
    /* Add padding-right when category is shown to prevent text overlap */
    padding-right: 0;
    width: 100%;
    /* Ensure input takes full height of wrapper */
    height: 100%;
    line-height: 1.4;
  }

  .add-todo-bottom input.has-category {
    padding-right: calc(var(--spacing-xl) + 80px); /* Space for category badge */
  }

  .add-todo-bottom input:focus {
    opacity: 1;
  }

  .add-todo-bottom input::placeholder {
    color: var(--text-on-primary);
    opacity: 0.6;
  }

  .add-todo-bottom:focus-within input {
    color: var(--text-primary);
  }

  .add-todo-bottom:focus-within input::placeholder {
    color: var(--text-secondary);
  }

  .input-wrapper {
    position: relative;
    flex: 1;
    display: flex;
    align-items: center;
    min-width: 0; /* Allow flex item to shrink */
    overflow: visible; /* Allow category badge to be visible */
    /* Ensure proper height matching input */
    height: 100%;
  }

  .predicted-category {
    position: absolute;
    right: var(--spacing-md);
    z-index: 3;
    pointer-events: none;
    transition: all var(--duration-instant);
    /* Ensure it's vertically centered with input (input is ~23px tall) */
    top: 50%;
    transform: translateY(-50%);
    /* Prevent it from overlapping with text */
    max-width: 35%;
  }

  /* Higher z-index when focused */
  .add-todo-bottom:focus-within .predicted-category {
    z-index: 4;
  }

  @media (max-width: 768px) {
    .predicted-category {
      right: var(--spacing-sm);
      max-width: 30%;
    }

    .add-todo-bottom input.has-category {
      padding-right: calc(var(--spacing-md) + 70px); /* Space for category badge on mobile */
    }
  }

  .inline-suggestion {
    position: absolute;
    top: 0;
    left: 0;
    font-size: var(--font-size-lg);
    color: transparent;
    pointer-events: none;
  }

  .suggestion-text {
    color: var(--text-muted);
  }

  .todos-section {
    margin-bottom: var(--spacing-lg);
  }

  .todo-container {
    position: relative;
  }

  .todo-container.spacer-top {
    margin-top: var(--drop-spacing);
  }

  .todo-container.spacer-bottom {
    margin-bottom: var(--drop-spacing);
  }

  .todo-wrapper {
    position: relative;
    margin-bottom: var(--spacing-sm);
    transition: transform var(--duration-instant);
  }

  @media (max-width: 768px) {
    .todo-wrapper {
      margin-bottom: var(--spacing-xs);
    }
  }

  .todo-wrapper.dragging {
    opacity: var(--opacity-dragging);
  }

  .todo-wrapper.drop-above,
  .todo-wrapper.drop-below {
    z-index: var(--z-index-drop-target);
  }

  .todo-wrapper.drop-above::before,
  .todo-wrapper.drop-below::after {
    content: '';
    display: block;
    height: var(--drop-indicator-height);
    border: var(--stroke-thin) dashed var(--text-on-primary);
    border-radius: var(--radius-md);
    opacity: var(--opacity-hover);
    animation: pulse var(--duration-pulse) ease-in-out infinite;
    position: absolute;
    width: 100%;
    z-index: var(--z-index-drop-indicator);
  }

  .todo-wrapper.drop-above::before {
    top: calc(-1 * var(--drop-spacing));
  }

  .todo-wrapper.drop-below::after {
    top: auto;
    bottom: calc(-1 * var(--drop-spacing));
  }

  /* 
    Remove the translateY on the item itself to avoid layout thrashing/jumping
    The pseudo-element provides the visual cue.
  */
  
  @keyframes pulse {
    0%, 100% {
      opacity: var(--opacity-pulse-min);
      border-width: var(--stroke-thin);
    }
    50% {
      opacity: var(--opacity-pulse-max);
      border-width: var(--stroke-bold);
    }
  }

  .add-todo-wrapper {
    position: relative;
    flex-shrink: 0;
    /* Match scrollable-content's scrollbar gutter + padding-right */
    padding-right: calc(var(--scrollbar-width) + var(--spacing-xs));
  }

  @media (max-width: 768px) {
    .add-todo-wrapper {
      padding-right: 0;
      /* Ensure input stays visible when keyboard appears on iOS */
      position: sticky;
      bottom: 0;
      z-index: 10;
      background: var(--app-bg-color);
      background-image: var(--app-bg-image);
      padding-top: var(--spacing-sm);
      margin-top: var(--spacing-sm);
      /* Smooth transition when keyboard appears */
      transition: transform var(--transition-normal);
    }
  }

  /* iOS-specific fixes for keyboard appearance */
  @supports (-webkit-touch-callout: none) {
    @media (max-width: 768px) {
      .add-todo-wrapper {
        /* Use sticky positioning to keep input visible */
        position: -webkit-sticky;
        position: sticky;
        /* Add background to prevent content showing through */
        background: var(--app-bg-color);
        background-image: var(--app-bg-image);
        /* Ensure it stays above content */
        z-index: 10;
      }

      .add-todo-bottom {
        /* Removed safe-area-inset-bottom to eliminate unnecessary spacing */
      }

      /* Adjust container when input is focused to account for keyboard */
      .todo-list-container {
        /* Use visual viewport height when available for better keyboard handling */
        height: 100vh;
        height: 100dvh;
      }
    }
  }

  .autocomplete-dropdown {
    position: absolute;
    bottom: 100%;
    left: 0;
    right: 0;
    background: var(--card-bg);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-menu);
    margin-bottom: var(--spacing-sm);
    overflow: hidden;
    z-index: var(--z-index-dropdown);
  }

  .autocomplete-item {
    display: block;
    width: 100%;
    padding: var(--font-size-sm) var(--spacing-xl);
    text-align: left;
    background: transparent;
    border: none;
    font-size: var(--font-size-base);
    color: var(--text-primary);
    cursor: pointer;
    transition: background var(--transition-fast);
    font-family: inherit;
  }

  .autocomplete-item:hover,
  .autocomplete-item.selected {
    background: var(--surface-muted);
  }

  .autocomplete-item-main {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-md);
  }


  .autocomplete-item:first-child {
    border-radius: var(--radius-md) var(--radius-md) 0 0;
  }

  .autocomplete-item:last-child {
    border-radius: 0 0 var(--radius-md) var(--radius-md);
  }

  .autocomplete-item:only-child {
    border-radius: var(--radius-md);
  }
</style>
