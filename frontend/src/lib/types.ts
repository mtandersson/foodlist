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
}

// Event types
export interface TodoCreated {
  type: "TodoCreated"
  id: string
  name: string
  createdAt: string
  sortOrder: number
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

export interface ListTitleChanged {
  type: "ListTitleChanged"
  title: string
}

// Command types (client -> server)
export interface CreateTodo {
  type: "CreateTodo"
  id: string
  name: string
  sortOrder?: number
}

export interface CompleteTodo {
  type: "CompleteTodo"
  id: string
}

export interface UncompleteTodo {
  type: "UncompleteTodo"
  id: string
}

export interface StarTodo {
  type: "StarTodo"
  id: string
}

export interface UnstarTodo {
  type: "UnstarTodo"
  id: string
}

export interface ReorderTodo {
  type: "ReorderTodo"
  id: string
  sortOrder: number
}

export interface RenameTodo {
  type: "RenameTodo"
  id: string
  name: string
}

export interface SetListTitle {
  type: "SetListTitle"
  title: string
}

export interface StateRollup {
  type: "StateRollup"
  todos: Todo[]
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

export interface AutocompleteResponse {
  type: "AutocompleteResponse"
  suggestions: string[]
  requestId: string
}

export type ServerMessage = Event | StateRollup | ClientCount | AutocompleteResponse

export type Command =
  | CreateTodo
  | CompleteTodo
  | UncompleteTodo
  | StarTodo
  | UnstarTodo
  | ReorderTodo
  | RenameTodo
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

export function isListTitleChanged(msg: ServerMessage): msg is ListTitleChanged {
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

export function isAutocompleteResponse(msg: ServerMessage): msg is AutocompleteResponse {
  return msg.type === "AutocompleteResponse"
}
