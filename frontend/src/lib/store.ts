import {writable, derived, get} from "svelte/store"
import {v4 as uuidv4} from "uuid"
import {TodoWebSocket, ConnectionState} from "./websocket"
import type {
  Todo,
  Category,
  Event,
  ServerMessage,
  TodoCreated,
  TodoCompleted,
  TodoUncompleted,
  TodoStarred,
  TodoUnstarred,
  TodoReordered,
  TodoRenamed,
  TodoCategorized,
  CategoryCreated,
  CategoryRenamed,
  CategoryDeleted,
  CategoryReordered,
  ListTitleChanged,
  Command,
  CreateTodo,
  CreateCategory,
  RenameCategory,
  DeleteCategory,
  ReorderCategory,
  CategorizeTodo,
  CompleteTodo,
  UncompleteTodo,
  StarTodo,
  UnstarTodo,
  ReorderTodo,
  RenameTodo,
  SetListTitle,
  AutocompleteResponse,
  AutocompleteSuggestion,
} from "./types"

export interface TodoStore {
  todos: ReturnType<typeof writable<Todo[]>>
  activeTodos: ReturnType<
    typeof derived<ReturnType<typeof writable<Todo[]>>, Todo[]>
  >
  completedTodos: ReturnType<
    typeof derived<ReturnType<typeof writable<Todo[]>>, Todo[]>
  >
  categories: ReturnType<
    typeof derived<
      ReturnType<typeof writable<Map<string, Category>>>,
      Category[]
    >
  >
  categoryLookup: ReturnType<
    typeof derived<
      ReturnType<typeof writable<Map<string, Category>>>,
      Map<string, Category>
    >
  >
  activeTodosByCategory: ReturnType<
    typeof derived<
      ReturnType<typeof writable<Todo[]>>,
      Map<string | null, Todo[]>
    >
  >
  connectionState: ReturnType<typeof writable<ConnectionState>>
  userCount: ReturnType<typeof writable<number>>
  listTitle: ReturnType<typeof writable<string>>
  autocompleteSuggestions: ReturnType<typeof writable<AutocompleteSuggestion[]>>
  errorMessage: ReturnType<typeof writable<string | null>>
  isSynced: ReturnType<typeof writable<boolean>>
  createTodo: (name: string, categoryId?: string | null) => void
  createCategory: (name: string, id?: string) => Promise<string>
  renameCategory: (id: string, name: string) => Promise<void>
  deleteCategory: (id: string) => void
  reorderCategory: (id: string, newSortOrder: number) => void
  categorizeTodo: (id: string, categoryId: string | null) => void
  toggleComplete: (id: string) => void
  toggleStar: (id: string) => void
  reorder: (id: string, newSortOrder: number) => void
  rename: (id: string, name: string) => void
  setListTitle: (title: string) => void
  requestAutocomplete: (query: string) => void
  clearAutocomplete: () => void
  clearError: () => void
  destroy: () => void
}

export function createTodoStore(wsUrl: string): TodoStore {
  const todosMap = writable<Map<string, Todo>>(new Map())
  const categoriesMap = writable<Map<string, Category>>(new Map())
  const connectionState = writable<ConnectionState>(ConnectionState.CONNECTING)
  const userCount = writable<number>(0)
  const listTitle = writable<string>("My Todo List")
  const autocompleteSuggestions = writable<AutocompleteSuggestion[]>([])
  const errorMessage = writable<string | null>(null)
  const isSynced = writable<boolean>(false)
  let errorTimeout: number | null = null

  // Track pending autocomplete request to match responses
  let pendingRequestId: string | null = null

  // Track pending commands to match responses
  type PendingCommand = {
    resolve: () => void
    reject: (error: string) => void
  }
  const pendingCommands = new Map<string, PendingCommand>()

  // Derived store that sorts todos by sortOrder descending
  const todos = derived(todosMap, ($map) => {
    const arr = Array.from($map.values())
    return arr.sort((a, b) => b.sortOrder - a.sortOrder)
  })

  // Active todos (not completed), sorted by sortOrder descending (highest first)
  const activeTodos = derived(todos, ($todos) =>
    $todos
      .filter((t) => t.completedAt === null)
      .sort((a, b) => b.sortOrder - a.sortOrder)
  )

  // Completed todos, sorted by completedAt descending (most recently completed first)
  const completedTodos = derived(todos, ($todos) =>
    $todos
      .filter((t) => t.completedAt !== null)
      .sort((a, b) => {
        // Sort by completedAt descending - most recent first
        if (a.completedAt && b.completedAt) {
          return b.completedAt.localeCompare(a.completedAt)
        }
        return 0
      })
  )

  const categories = derived(categoriesMap, ($map) => {
    const arr = Array.from($map.values())
    return arr.sort((a, b) => b.sortOrder - a.sortOrder)
  })

  const categoryLookup = derived(categoriesMap, ($map) => new Map($map))

  const activeTodosByCategory = derived(activeTodos, ($activeTodos) => {
    const grouped = new Map<string | null, Todo[]>()
    for (const todo of $activeTodos) {
      const key = todo.categoryId ?? null
      const list = grouped.get(key) ?? []
      list.push(todo)
      grouped.set(key, list)
    }
    // Sort each category's todos by sortOrder descending
    for (const [key, list] of grouped) {
      list.sort((a, b) => b.sortOrder - a.sortOrder)
      grouped.set(key, list)
    }
    return grouped
  })

  // WebSocket connection
  const ws = new TodoWebSocket(wsUrl)

  ws.onConnectionChange((state) => {
    connectionState.set(state)
  })

  ws.onMessage((message: ServerMessage) => {
    handleMessage(message)
  })

  ws.onAutocomplete((response: AutocompleteResponse) => {
    // Only update if this is the response we're waiting for
    if (response.requestId === pendingRequestId) {
      autocompleteSuggestions.set(response.suggestions)
    }
  })

  function handleMessage(message: ServerMessage) {
    if (message.type === "StateRollup") {
      // Initialize state from rollup
      const map = new Map<string, Todo>()
      for (const todo of message.todos) {
        map.set(todo.id, todo)
      }
      todosMap.set(map)
      const catMap = new Map<string, Category>()
      if ("categories" in message && Array.isArray(message.categories)) {
        for (const cat of message.categories) {
          catMap.set(cat.id, cat)
        }
      }
      categoriesMap.set(catMap)
      listTitle.set(message.listTitle)
      isSynced.set(true)
      return
    }

    if (message.type === "ClientCount") {
      userCount.set(message.count)
      return
    }

    if (message.type === "CommandResponse") {
      // Handle command response
      const pending = pendingCommands.get(message.commandId)
      if (pending) {
        if (message.success) {
          pending.resolve()
        } else {
          pending.reject(message.error || "Command failed")
        }
        pendingCommands.delete(message.commandId)
      }
      return
    }

    // Handle events
    applyEvent(message as Event)
  }

  function showError(message: string) {
    // Clear any existing timeout
    if (errorTimeout !== null) {
      clearTimeout(errorTimeout)
    }

    errorMessage.set(message)

    // Auto-clear after 5 seconds
    errorTimeout = window.setTimeout(() => {
      errorMessage.set(null)
      errorTimeout = null
    }, 5000)
  }

  function clearError() {
    if (errorTimeout !== null) {
      clearTimeout(errorTimeout)
      errorTimeout = null
    }
    errorMessage.set(null)
  }

  function applyEvent(event: Event) {
    todosMap.update((map) => {
      const newMap = new Map(map)

      switch (event.type) {
        case "TodoCreated": {
          const e = event as TodoCreated
          newMap.set(e.id, {
            id: e.id,
            name: e.name,
            createdAt: e.createdAt,
            completedAt: null,
            sortOrder: e.sortOrder,
            starred: false,
            categoryId: e.categoryId ?? null,
          })
          break
        }

        case "TodoCompleted": {
          const e = event as TodoCompleted
          const todo = newMap.get(e.id)
          if (todo) {
            newMap.set(e.id, {...todo, completedAt: e.completedAt})
          }
          break
        }

        case "TodoUncompleted": {
          const e = event as TodoUncompleted
          const todo = newMap.get(e.id)
          if (todo) {
            newMap.set(e.id, {...todo, completedAt: null})
          }
          break
        }

        case "TodoStarred": {
          const e = event as TodoStarred
          const todo = newMap.get(e.id)
          if (todo) {
            newMap.set(e.id, {...todo, starred: true, sortOrder: e.sortOrder})
          }
          break
        }

        case "TodoUnstarred": {
          const e = event as TodoUnstarred
          const todo = newMap.get(e.id)
          if (todo) {
            newMap.set(e.id, {...todo, starred: false})
          }
          break
        }

        case "TodoReordered": {
          const e = event as TodoReordered
          const todo = newMap.get(e.id)
          if (todo) {
            newMap.set(e.id, {...todo, sortOrder: e.sortOrder})
          }
          break
        }

        case "TodoRenamed": {
          const e = event as TodoRenamed
          const todo = newMap.get(e.id)
          if (todo) {
            newMap.set(e.id, {...todo, name: e.name})
          }
          break
        }

        case "TodoCategorized": {
          const e = event as TodoCategorized
          const todo = newMap.get(e.id)
          if (todo) {
            newMap.set(e.id, {...todo, categoryId: e.categoryId ?? null})
          }
          break
        }

        case "CategoryCreated": {
          const e = event as CategoryCreated
          categoriesMap.update((catMap) => {
            const mapCopy = new Map(catMap)
            mapCopy.set(e.id, {
              id: e.id,
              name: e.name,
              createdAt: e.createdAt,
              sortOrder: e.sortOrder,
            })
            return mapCopy
          })
          break
        }

        case "CategoryRenamed": {
          const e = event as CategoryRenamed
          categoriesMap.update((catMap) => {
            const mapCopy = new Map(catMap)
            const cat = mapCopy.get(e.id)
            if (cat) {
              mapCopy.set(e.id, {...cat, name: e.name})
            }
            return mapCopy
          })
          break
        }

        case "CategoryDeleted": {
          const e = event as CategoryDeleted
          categoriesMap.update((catMap) => {
            const mapCopy = new Map(catMap)
            mapCopy.delete(e.id)
            return mapCopy
          })
          // Clear categoryId for todos that referenced this category (shouldn't happen if validated server-side)
          newMap.forEach((todo, id) => {
            if (todo.categoryId === e.id) {
              newMap.set(id, {...todo, categoryId: null})
            }
          })
          break
        }

        case "CategoryReordered": {
          const e = event as CategoryReordered
          categoriesMap.update((catMap) => {
            const mapCopy = new Map(catMap)
            const cat = mapCopy.get(e.id)
            if (cat) {
              mapCopy.set(e.id, {...cat, sortOrder: e.sortOrder})
            }
            return mapCopy
          })
          break
        }

        case "ListTitleChanged": {
          const e = event as ListTitleChanged
          listTitle.set(e.title)
          break
        }
      }

      return newMap
    })
  }

  function getHighestSortOrder(): number {
    const currentTodos = get(todos)
    if (currentTodos.length === 0) return 0
    return Math.max(...currentTodos.map((t) => t.sortOrder))
  }

  function getHighestCategorySortOrder(): number {
    const currentCategories = get(categories)
    if (currentCategories.length === 0) return 0
    return Math.max(...currentCategories.map((c) => c.sortOrder))
  }

  function sendCommand(
    command: Command,
    optimisticEvent?: Event
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      // Apply optimistically if provided
      if (optimisticEvent) {
        applyEvent(optimisticEvent)
      }

      // Track pending command
      const commandId = command.commandId
      pendingCommands.set(commandId, {resolve, reject})

      // Send to server
      ws.send(command)

      // Timeout after 10 seconds
      setTimeout(() => {
        if (pendingCommands.has(commandId)) {
          pendingCommands.delete(commandId)
          reject("Command timeout")
        }
      }, 10000)
    })
  }

  // Public actions

  function createTodo(name: string, categoryId: string | null = null) {
    const commandId = uuidv4()
    const id = uuidv4()
    const command: CreateTodo = {
      type: "CreateTodo",
      commandId,
      id,
      name,
      sortOrder: getHighestSortOrder() + 1000,
      categoryId,
    }
    const optimistic: TodoCreated = {
      type: "TodoCreated",
      id,
      name,
      createdAt: new Date().toISOString(),
      sortOrder: command.sortOrder ?? getHighestSortOrder() + 1000,
      categoryId,
    }
    sendCommand(command, optimistic)
  }

  function createCategory(name: string, id?: string): Promise<string> {
    const commandId = uuidv4()
    const newId = id || uuidv4()
    const command: CreateCategory = {
      type: "CreateCategory",
      commandId,
      id: newId,
      name,
      sortOrder: getHighestCategorySortOrder() + 1000,
    }
    const optimistic: CategoryCreated = {
      type: "CategoryCreated",
      id: newId,
      name,
      createdAt: new Date().toISOString(),
      sortOrder: command.sortOrder!,
    }
    return sendCommand(command, optimistic).then(() => newId)
  }

  function renameCategory(id: string, name: string): Promise<void> {
    const commandId = uuidv4()
    const command: RenameCategory = {
      type: "RenameCategory",
      commandId,
      id,
      name,
    }
    // No optimistic update - wait for server response
    return sendCommand(command)
  }

  function deleteCategory(id: string) {
    const commandId = uuidv4()
    const command: DeleteCategory = {type: "DeleteCategory", commandId, id}
    // Do not optimistically remove; wait for server validation
    sendCommand(command)
  }

  function reorderCategory(id: string, newSortOrder: number) {
    const commandId = uuidv4()
    const command: ReorderCategory = {
      type: "ReorderCategory",
      commandId,
      id,
      sortOrder: newSortOrder,
    }
    const optimistic: CategoryReordered = {
      type: "CategoryReordered",
      id,
      sortOrder: newSortOrder,
    }
    sendCommand(command, optimistic)
  }

  function categorizeTodo(id: string, categoryId: string | null) {
    const commandId = uuidv4()
    const command: CategorizeTodo = {
      type: "CategorizeTodo",
      commandId,
      id,
      categoryId,
    }
    const optimistic: TodoCategorized = {
      type: "TodoCategorized",
      id,
      categoryId,
    }
    sendCommand(command, optimistic)
  }

  function toggleComplete(id: string) {
    const currentTodos = get(todos)
    const todo = currentTodos.find((t) => t.id === id)
    if (!todo) return

    const commandId = uuidv4()
    if (todo.completedAt === null) {
      const command: CompleteTodo = {type: "CompleteTodo", commandId, id}
      const optimistic: TodoCompleted = {
        type: "TodoCompleted",
        id,
        completedAt: new Date().toISOString(),
      }
      sendCommand(command, optimistic)
    } else {
      const command: UncompleteTodo = {type: "UncompleteTodo", commandId, id}
      const optimistic: TodoUncompleted = {type: "TodoUncompleted", id}
      sendCommand(command, optimistic)
    }
  }

  function toggleStar(id: string) {
    const currentTodos = get(todos)
    const todo = currentTodos.find((t) => t.id === id)
    if (!todo) return

    const commandId = uuidv4()
    if (todo.starred) {
      const command: UnstarTodo = {type: "UnstarTodo", commandId, id}
      const optimistic: TodoUnstarred = {type: "TodoUnstarred", id}
      sendCommand(command, optimistic)
    } else {
      // When starring, move to top
      const sortOrder = getHighestSortOrder() + 1000
      const command: StarTodo = {type: "StarTodo", commandId, id}
      const optimistic: TodoStarred = {type: "TodoStarred", id, sortOrder}
      sendCommand(command, optimistic)
    }
  }

  function reorder(id: string, newSortOrder: number) {
    const commandId = uuidv4()
    const command: ReorderTodo = {
      type: "ReorderTodo",
      commandId,
      id,
      sortOrder: newSortOrder,
    }
    const optimistic: TodoReordered = {
      type: "TodoReordered",
      id,
      sortOrder: newSortOrder,
    }
    sendCommand(command, optimistic)
  }

  function rename(id: string, name: string) {
    const commandId = uuidv4()
    const command: RenameTodo = {type: "RenameTodo", commandId, id, name}
    const optimistic: TodoRenamed = {type: "TodoRenamed", id, name}
    sendCommand(command, optimistic)
  }

  function setListTitle(title: string) {
    const commandId = uuidv4()
    const command: SetListTitle = {type: "SetListTitle", commandId, title}
    const optimistic: ListTitleChanged = {type: "ListTitleChanged", title}
    sendCommand(command, optimistic)
  }

  function requestAutocomplete(query: string) {
    const requestId = uuidv4()
    pendingRequestId = requestId
    ws.sendAutocompleteRequest(query, requestId)
  }

  function clearAutocomplete() {
    pendingRequestId = null
    autocompleteSuggestions.set([])
  }

  function destroy() {
    ws.close()
  }

  return {
    todos: todos as any, // Cast to satisfy interface
    activeTodos: activeTodos as any,
    completedTodos: completedTodos as any,
    categories: categories as any,
    categoryLookup: categoryLookup as any,
    activeTodosByCategory: activeTodosByCategory as any,
    connectionState,
    userCount,
    listTitle,
    autocompleteSuggestions,
    errorMessage,
    isSynced,
    createTodo,
    createCategory,
    renameCategory,
    deleteCategory,
    reorderCategory,
    categorizeTodo,
    toggleComplete,
    toggleStar,
    reorder,
    rename,
    setListTitle,
    requestAutocomplete,
    clearAutocomplete,
    clearError,
    destroy,
  }
}
