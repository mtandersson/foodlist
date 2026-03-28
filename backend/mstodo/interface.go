package mstodo

// TodoProvider is a generic interface for todo list providers
// This allows adding other providers (like Google Tasks, Apple Reminders) in the future
type TodoProvider interface {
	GetLists() ([]TodoList, error)
	GetTasks(listID string) ([]TodoTask, error)
}
