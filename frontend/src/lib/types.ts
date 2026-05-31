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
  count?: number | null
  unit?: string | null
  originalInput?: string
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
  count?: number | null
  unit?: string | null
  originalInput?: string
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
  count?: number | null
  unit?: string | null
  originalInput?: string
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
  count?: number | null
  unit?: string | null
  originalInput?: string
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

export interface FeatureFlags {
  suggestions?: boolean
  recipes?: boolean
  recipesParse?: boolean
}

export interface StateRollup {
  type: "StateRollup"
  todos: Todo[]
  categories: Category[]
  listTitle: string
  version?: string
  featureFlags?: FeatureFlags
}

export interface Suggestion {
  id: string
  name: string
  categoryId?: string | null
  categoryName?: string | null
  lastPurchasedAt: string
  purchaseCount: number
  avgIntervalSeconds: number
}

export interface SuggestionsRollup {
  type: "SuggestionsRollup"
  suggestions: Suggestion[]
}

export interface SuggestionAdded {
  type: "SuggestionAdded"
  suggestion: Suggestion
}

export interface SuggestionRemoved {
  type: "SuggestionRemoved"
  id: string
}

// Recipe types - stored as JSON files on the backend, served via HTTP.
// They are NOT events: they don't appear in the event log and don't
// follow the WS command/event flow.
export interface Ingredient {
  amount?: number | null
  unit?: string
  name: string
}

export interface Recipe {
  id: string
  title: string
  ingredients: Ingredient[]
  instructions: string[]
  imageFilename: string
  imageMime: string
  createdAt: string
  updatedAt: string
}

export interface RecipeListItem {
  id: string
  title: string
  imageUrl: string
  createdAt: string
  updatedAt: string
}

export interface RecipeListResponse {
  recipes: RecipeListItem[]
}

export interface RecipeDetailResponse {
  recipe: Recipe
  imageUrl: string
}

export interface RecipeParseResponse {
  parsed: Recipe
}

// Cook* commands are sent over WebSocket but are NOT events: the server
// keeps the cook session in memory only and never persists them. They
// share the standard CommandResponse acknowledgment.
export interface CookCheckStep {
  type: "CookCheckStep"
  commandId: string
  recipeId: string
  stepIndex: number
}

export interface CookUncheckStep {
  type: "CookUncheckStep"
  commandId: string
  recipeId: string
  stepIndex: number
}

export interface CookReset {
  type: "CookReset"
  commandId: string
  recipeId: string
}

export interface CookStateChanged {
  type: "CookStateChanged"
  recipeId: string
  checkedSteps: number[]
}

export interface CookStateRollup {
  type: "CookStateRollup"
  sessions: Record<string, number[]>
}

export interface RecipeChanged {
  type: "RecipeChanged"
  id: string
  deleted: boolean
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
  | SuggestionsRollup
  | SuggestionAdded
  | SuggestionRemoved
  | CookStateChanged
  | CookStateRollup
  | RecipeChanged

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
  | CookCheckStep
  | CookUncheckStep
  | CookReset

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
