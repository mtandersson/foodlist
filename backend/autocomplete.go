package main

// getAutocompleteSuggestions returns up to 4 autocomplete suggestions based on query
// It delegates the logic to the AutocompleteLogic struct.
func (s *Server) getAutocompleteSuggestions(query string) []AutocompleteSuggestion {
	activeTodos := s.state.GetActiveTodoNames()
	return s.state.autocomplete.GetSuggestions(query, activeTodos)
}
