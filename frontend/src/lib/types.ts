// Generated from schema/events.schema.json
// Do not edit manually - run schema/generate.sh to regenerate

// Todo item projected from events
export interface Todo {
  id: string
  name: string
  createdAt: string
  completedAt: string | null
  sortOrder: number
  starred: boolean
  categoryId?: string | null
}

export interface Category {
  id: string
  name: string
  createdAt: string
  sortOrder: number
}

// Event types
export interface TodoCreated {
  type: "TodoCreated"
  id: string
  name: string
  createdAt: string
  sortOrder: number
  categoryId?: string | null
}

export interface TodoCompleted {
  type: "TodoCompleted"
  id: string
  completedAt: string
}

export interface TodoUncompleted {
  type: "TodoUncompleted"
  id: string
}

export interface TodoStarred {
  type: "TodoStarred"
  id: string
  sortOrder: number
}

export interface TodoUnstarred {
  type: "TodoUnstarred"
  id: string
}

export interface TodoReordered {
  type: "TodoReordered"
  id: string
  sortOrder: number
}

export interface TodoRenamed {
  type: "TodoRenamed"
  id: string
  name: string
}

export interface TodoCategorized {
  type: "TodoCategorized"
  id: string
  categoryId: string | null
}

export interface CategoryCreated {
  type: "CategoryCreated"
  id: string
  name: string
  createdAt: string
  sortOrder: number
}

export interface CategoryRenamed {
  type: "CategoryRenamed"
  id: string
  name: string
}

export interface CategoryDeleted {
  type: "CategoryDeleted"
  id: string
}

export interface CategoryReordered {
  type: "CategoryReordered"
  id: string
  sortOrder: number
}

export interface ListTitleChanged {
  type: "ListTitleChanged"
  title: string
}

// Command types (client -> server)
export interface CreateTodo {
  type: "CreateTodo"
  commandId: string
  id: string
  name: string
  sortOrder?: number
  categoryId?: string | null
}

export interface CategorizeTodo {
  type: "CategorizeTodo"
  commandId: string
  id: string
  categoryId: string | null
}

export interface CreateCategory {
  type: "CreateCategory"
  commandId: string
  id: string
  name: string
  sortOrder?: number
}

export interface RenameCategory {
  type: "RenameCategory"
  commandId: string
  id: string
  name: string
}

export interface DeleteCategory {
  type: "DeleteCategory"
  commandId: string
  id: string
}

export interface ReorderCategory {
  type: "ReorderCategory"
  commandId: string
  id: string
  sortOrder: number
}

export interface CompleteTodo {
  type: "CompleteTodo"
  commandId: string
  id: string
}

export interface UncompleteTodo {
  type: "UncompleteTodo"
  commandId: string
  id: string
}

export interface StarTodo {
  type: "StarTodo"
  commandId: string
  id: string
}

export interface UnstarTodo {
  type: "UnstarTodo"
  commandId: string
  id: string
}

export interface ReorderTodo {
  type: "ReorderTodo"
  commandId: string
  id: string
  sortOrder: number
}

export interface RenameTodo {
  type: "RenameTodo"
  commandId: string
  id: string
  name: string
}

export interface SetListTitle {
  type: "SetListTitle"
  commandId: string
  title: string
}

export interface StateRollup {
  type: "StateRollup"
  todos: Todo[]
  categories: Category[]
  listTitle: string
}

// Union types
export type Event =
  | TodoCreated
  | TodoCompleted
  | TodoUncompleted
  | TodoStarred
  | TodoUnstarred
  | TodoReordered
  | TodoRenamed
  | TodoCategorized
  | CategoryCreated
  | CategoryRenamed
  | CategoryDeleted
  | CategoryReordered
  | ListTitleChanged

export interface ClientCount {
  type: "ClientCount"
  count: number
}

// Autocomplete types
export interface AutocompleteRequest {
  type: "AutocompleteRequest"
  query: string
  requestId: string
}

export interface AutocompleteSuggestion {
  name: string
  categoryId: string | null
  categoryName: string | null
}

export interface AutocompleteResponse {
  type: "AutocompleteResponse"
  suggestions: AutocompleteSuggestion[]
  requestId: string
}

export interface CommandResponse {
  type: "CommandResponse"
  commandId: string
  success: boolean
  error?: string
}

export type ServerMessage =
  | Event
  | StateRollup
  | ClientCount
  | AutocompleteResponse
  | CommandResponse

export type Command =
  | CreateTodo
  | CompleteTodo
  | UncompleteTodo
  | StarTodo
  | UnstarTodo
  | ReorderTodo
  | RenameTodo
  | CategorizeTodo
  | CreateCategory
  | RenameCategory
  | DeleteCategory
  | ReorderCategory
  | SetListTitle

// Type guards
export function isTodoCreated(msg: ServerMessage): msg is TodoCreated {
  return msg.type === "TodoCreated"
}

export function isTodoCompleted(msg: ServerMessage): msg is TodoCompleted {
  return msg.type === "TodoCompleted"
}

export function isTodoUncompleted(msg: ServerMessage): msg is TodoUncompleted {
  return msg.type === "TodoUncompleted"
}

export function isTodoStarred(msg: ServerMessage): msg is TodoStarred {
  return msg.type === "TodoStarred"
}

export function isTodoUnstarred(msg: ServerMessage): msg is TodoUnstarred {
  return msg.type === "TodoUnstarred"
}

export function isTodoReordered(msg: ServerMessage): msg is TodoReordered {
  return msg.type === "TodoReordered"
}

export function isTodoRenamed(msg: ServerMessage): msg is TodoRenamed {
  return msg.type === "TodoRenamed"
}

export function isListTitleChanged(
  msg: ServerMessage
): msg is ListTitleChanged {
  return msg.type === "ListTitleChanged"
}

export function isStateRollup(msg: ServerMessage): msg is StateRollup {
  return msg.type === "StateRollup"
}

export function isEvent(msg: ServerMessage): msg is Event {
  return msg.type !== "StateRollup"
}

export function isClientCount(msg: ServerMessage): msg is ClientCount {
  return msg.type === "ClientCount"
}

export function isAutocompleteResponse(
  msg: ServerMessage
): msg is AutocompleteResponse {
  return msg.type === "AutocompleteResponse"
}
