// Cypress E2E support file
// Add custom commands and global configuration here

Cypress.Commands.add('clearEventStore', () => {
  // Clear the backend event store for clean tests
  // This would require a test-only endpoint on the backend
  // For now, we'll just ensure tests are independent
});

Cypress.Commands.add('getByPlaceholder', (placeholder) => {
  return cy.get(`input[placeholder="${placeholder}"]`);
});

Cypress.Commands.add('getTodoItem', (name) => {
  return cy.contains('.todo-item', name);
});

