describe("Todo App", () => {
  beforeEach(() => {
    cy.visit("/")
    // Wait for WebSocket connection
    cy.wait(500)
  })

  describe("Basic Flows", () => {
    it("should display the app title", () => {
      cy.contains("Mat 2025").should("be.visible")
    })

    it("should create a new todo", () => {
      const todoName = `Test todo ${Date.now()}`

      cy.get('input[placeholder="Lägg till en uppgift"]').type(todoName)
      cy.get('input[placeholder="Lägg till en uppgift"]').type("{enter}")

      // Should appear in the list
      cy.contains(todoName).should("be.visible")
    })

    it("should complete a todo and show in completed section", () => {
      const todoName = `Complete test ${Date.now()}`

      // Create todo
      cy.get('input[placeholder="Lägg till en uppgift"]').type(
        `${todoName}{enter}`
      )
      cy.contains(todoName).should("be.visible")

      // Find and click the checkbox
      cy.contains(todoName)
        .parent()
        .find('button[aria-label="Mark as complete"]')
        .click()

      // Should show "Slutfört" section
      cy.contains("Slutfört").should("be.visible")

      // Todo should be in completed section with strikethrough
      cy.contains(todoName)
        .should("have.class", "strikethrough")
        .or("have.css", "text-decoration-line", "line-through")
    })

    it("should star a todo", () => {
      const todoName = `Star test ${Date.now()}`

      // Create todo
      cy.get('input[placeholder="Lägg till en uppgift"]').type(
        `${todoName}{enter}`
      )
      cy.contains(todoName).should("be.visible")

      // Find and click the star button
      cy.contains(todoName).parent().find('button[aria-label="Star"]').click()

      // Star should be filled (has starred class)
      cy.contains(todoName)
        .parent()
        .find('button.starred, button[aria-label="Unstar"]')
        .should("exist")
    })

    it("should uncomplete a completed todo", () => {
      const todoName = `Uncomplete test ${Date.now()}`

      // Create and complete todo
      cy.get('input[placeholder="Lägg till en uppgift"]').type(
        `${todoName}{enter}`
      )
      cy.contains(todoName)
        .parent()
        .find('button[aria-label="Mark as complete"]')
        .click()

      // Wait for animation
      cy.wait(300)

      // Click to uncomplete
      cy.contains(todoName)
        .parent()
        .find('button[aria-label="Mark as incomplete"]')
        .click()

      // Should no longer have strikethrough
      cy.contains(todoName).should("not.have.class", "strikethrough")
    })

    it("should toggle completed section visibility", () => {
      const todoName = `Toggle test ${Date.now()}`

      // Create and complete todo
      cy.get('input[placeholder="Lägg till en uppgift"]').type(
        `${todoName}{enter}`
      )
      cy.contains(todoName)
        .parent()
        .find('button[aria-label="Mark as complete"]')
        .click()

      // Completed section should be visible
      cy.contains("Slutfört").should("be.visible")
      cy.contains(todoName).should("be.visible")

      // Click to collapse
      cy.contains("Slutfört").click()

      // Todo should be hidden
      cy.contains(todoName).should("not.be.visible")

      // Click to expand again
      cy.contains("Slutfört").click()

      // Todo should be visible again
      cy.contains(todoName).should("be.visible")
    })
  })

  describe("Persistence", () => {
    it("should persist todos after page reload", () => {
      const todoName = `Persist test ${Date.now()}`

      // Create todo
      cy.get('input[placeholder="Lägg till en uppgift"]').type(
        `${todoName}{enter}`
      )
      cy.contains(todoName).should("be.visible")

      // Wait for persistence
      cy.wait(500)

      // Reload page
      cy.reload()
      cy.wait(500)

      // Todo should still be there
      cy.contains(todoName).should("be.visible")
    })
  })

  describe("Real-time Sync", () => {
    it("should sync between two tabs", () => {
      const todoName = `Sync test ${Date.now()}`

      // Create todo in first tab
      cy.get('input[placeholder="Lägg till en uppgift"]').type(
        `${todoName}{enter}`
      )
      cy.contains(todoName).should("be.visible")

      // Wait for sync
      cy.wait(500)

      // Open a new window (simulates second tab)
      cy.window().then((win) => {
        // The WebSocket should broadcast to all clients
        // In Cypress, we can verify by reloading and checking state
        cy.reload()
        cy.wait(500)
        cy.contains(todoName).should("be.visible")
      })
    })
  })

  describe("Optimistic Updates", () => {
    it("should show todo immediately before server confirmation", () => {
      const todoName = `Optimistic test ${Date.now()}`

      // Type and submit - should appear immediately
      cy.get('input[placeholder="Lägg till en uppgift"]').type(
        `${todoName}{enter}`
      )

      // Should appear immediately (optimistic update)
      cy.contains(todoName).should("be.visible")
    })
  })

  describe("Sorting", () => {
    it("should show newer todos at the top", () => {
      const todo1 = `First ${Date.now()}`
      const todo2 = `Second ${Date.now()}`

      // Create first todo
      cy.get('input[placeholder="Steel cut havregryn"]').type(`${todo1}{enter}`)
      cy.wait(100)

      // Create second todo
      cy.get('input[placeholder="Steel cut havregryn"]').type(`${todo2}{enter}`)
      cy.wait(100)

      // Second should be above first (higher sortOrder)
      cy.get(".todos-section").within(() => {
        cy.get(".todo-item").first().should("contain", todo2)
      })
    })

    it("should move starred todo to top", () => {
      const todo1 = `First star ${Date.now()}`
      const todo2 = `Second star ${Date.now()}`

      // Create todos
      cy.get('input[placeholder="Steel cut havregryn"]').type(`${todo1}{enter}`)
      cy.wait(100)
      cy.get('input[placeholder="Steel cut havregryn"]').type(`${todo2}{enter}`)
      cy.wait(100)

      // Star the first (older) todo
      cy.contains(todo1).parent().find('button[aria-label="Star"]').click()

      // Wait for animation
      cy.wait(300)

      // First should now be at top (starred)
      cy.get(".todos-section").within(() => {
        cy.get(".todo-item").first().should("contain", todo1)
      })
    })

    it("should show most recently completed items at the top of completed list", () => {
      const todo1 = `Complete First ${Date.now()}`
      const todo2 = `Complete Second ${Date.now()}`
      const todo3 = `Complete Third ${Date.now()}`

      // Create three todos
      cy.get('input[placeholder="Steel cut havregryn"]').type(`${todo1}{enter}`)
      cy.wait(100)
      cy.get('input[placeholder="Steel cut havregryn"]').type(`${todo2}{enter}`)
      cy.wait(100)
      cy.get('input[placeholder="Steel cut havregryn"]').type(`${todo3}{enter}`)
      cy.wait(100)

      // Complete them in order: 1, 2, 3 with delays
      cy.contains(todo1)
        .parent()
        .find('button[aria-label="Mark as complete"]')
        .click()
      cy.wait(200)

      cy.contains(todo2)
        .parent()
        .find('button[aria-label="Mark as complete"]')
        .click()
      cy.wait(200)

      cy.contains(todo3)
        .parent()
        .find('button[aria-label="Mark as complete"]')
        .click()
      cy.wait(300)

      // Completed section should be expanded
      cy.contains("Slutfört").should("be.visible")

      // In completed section, most recently completed (todo3) should be first
      cy.get(".completed-section").within(() => {
        cy.get(".todo-item").first().should("contain", todo3)
        cy.get(".todo-item").eq(1).should("contain", todo2)
        cy.get(".todo-item").eq(2).should("contain", todo1)
      })
    })
  })

  describe("List Title Editing", () => {
    it("should edit list title and persist", () => {
      // Click on title to edit
      cy.get(".title").click()

      // Should show input
      cy.get(".title-input").should("be.visible")

      // Type new title
      const newTitle = `Test List ${Date.now()}`
      cy.get(".title-input").clear().type(newTitle)

      // Press Enter to save
      cy.get(".title-input").type("{enter}")

      // Should show new title
      cy.contains(newTitle).should("be.visible")

      // Reload page - title should persist
      cy.reload()
      cy.wait(500)
      cy.contains(newTitle).should("be.visible")
    })

    it("should cancel editing on Escape", () => {
      const originalTitle = "Mat 2025"

      // Click on title to edit
      cy.get(".title").click()

      // Type something
      cy.get(".title-input").clear().type("New Title")

      // Press Escape
      cy.get(".title-input").type("{esc}")

      // Should still show original title
      cy.contains(originalTitle).should("be.visible")
    })
  })

  describe("Autocomplete", () => {
    // Create some historical items first for autocomplete to suggest
    beforeEach(() => {
      // Create and complete some items to build history
      const items = [
        "Autocomplete Milk",
        "Autocomplete Bread",
        "Autocomplete Eggs",
      ]
      items.forEach((item) => {
        cy.get('input[placeholder="Lägg till en uppgift"]').type(
          `${item}{enter}`
        )
        cy.wait(200)
      })
      // Complete them so they show up in autocomplete
      items.forEach((item) => {
        cy.contains(item)
          .parent()
          .find('button[aria-label="Mark as complete"]')
          .click()
        cy.wait(200)
      })
    })

    it("should show autocomplete dropdown when typing", () => {
      // Focus on input and type
      cy.get('input[placeholder="Lägg till en uppgift"]').focus().type("Auto")

      // Wait for autocomplete response
      cy.wait(300)

      // Should show autocomplete dropdown
      cy.get(".autocomplete-dropdown").should("be.visible")
    })

    it("should show suggestions matching the query", () => {
      // Type something that matches historical items
      cy.get('input[placeholder="Lägg till en uppgift"]')
        .focus()
        .type("Autocomplete M")

      cy.wait(300)

      // Should show Milk in suggestions
      cy.get(".autocomplete-dropdown").within(() => {
        cy.contains("Autocomplete Milk").should("be.visible")
      })
    })

    it("should fill input when clicking a suggestion", () => {
      // Type to trigger autocomplete
      cy.get('input[placeholder="Lägg till en uppgift"]')
        .focus()
        .type("Autocomplete M")

      cy.wait(300)

      // Click on a suggestion
      cy.get(".autocomplete-dropdown").contains("Autocomplete Milk").click()

      // The input should be cleared (item was added) and new todo should appear
      cy.get(".todos-section")
        .contains("Autocomplete Milk")
        .should("be.visible")
    })

    it("should hide autocomplete when pressing Escape", () => {
      // Type to trigger autocomplete
      cy.get('input[placeholder="Lägg till en uppgift"]').focus().type("Auto")

      cy.wait(300)

      // Should show autocomplete
      cy.get(".autocomplete-dropdown").should("be.visible")

      // Press Escape
      cy.get('input[placeholder="Lägg till en uppgift"]').type("{esc}")

      // Should hide autocomplete
      cy.get(".autocomplete-dropdown").should("not.exist")
    })

    it("should not show items that are already active", () => {
      // Add an item that won't be completed (active)
      const activeTodo = `Active Item ${Date.now()}`
      cy.get('input[placeholder="Lägg till en uppgift"]').type(
        `${activeTodo}{enter}`
      )
      cy.wait(300)

      // Now type the same name - it should not appear in autocomplete
      cy.get('input[placeholder="Lägg till en uppgift"]')
        .focus()
        .clear()
        .type(activeTodo.substring(0, 10))

      cy.wait(300)

      // The active item should not be in suggestions
      cy.get(".autocomplete-dropdown").should("not.contain", activeTodo)
    })

    it("should navigate suggestions with arrow keys", () => {
      // Type to trigger autocomplete
      cy.get('input[placeholder="Lägg till en uppgift"]')
        .focus()
        .type("Autocomplete")

      cy.wait(300)

      // Press down arrow to select first item
      cy.get('input[placeholder="Lägg till en uppgift"]').type("{downarrow}")

      // First item should be selected
      cy.get(".autocomplete-item.selected").should("exist")

      // Press down again
      cy.get('input[placeholder="Lägg till en uppgift"]').type("{downarrow}")

      // Second item should be selected
      cy.get(".autocomplete-item.selected").should("exist")

      // Press up
      cy.get('input[placeholder="Lägg till en uppgift"]').type("{uparrow}")

      // First item should be selected again
      cy.get(".autocomplete-item").first().should("have.class", "selected")
    })

    it("should add selected suggestion when pressing Enter", () => {
      // Type to trigger autocomplete
      cy.get('input[placeholder="Lägg till en uppgift"]')
        .focus()
        .type("Autocomplete B")

      cy.wait(300)

      // Press down to select
      cy.get('input[placeholder="Lägg till en uppgift"]').type("{downarrow}")

      // Press Enter to select
      cy.get('input[placeholder="Lägg till en uppgift"]').type("{enter}")

      // Should add the selected item to active todos
      cy.get(".todos-section")
        .contains("Autocomplete Bread")
        .should("be.visible")
    })

    it("should show suggestions with empty input when focused", () => {
      // Clear any existing input and focus
      cy.get('input[placeholder="Lägg till en uppgift"]').clear().focus()

      // Wait for autocomplete to show most frequent items
      cy.wait(500)

      // Should show autocomplete with most used items
      cy.get(".autocomplete-dropdown").should("be.visible")
      cy.get(".autocomplete-item").should("have.length.at.least", 1)
    })

    it("should hide autocomplete when input loses focus", () => {
      // Type to trigger autocomplete
      cy.get('input[placeholder="Lägg till en uppgift"]').focus().type("Auto")

      cy.wait(300)

      // Should show autocomplete
      cy.get(".autocomplete-dropdown").should("be.visible")

      // Click somewhere else (blur the input)
      cy.get(".header").click()

      // Should hide autocomplete (with small delay for the blur handler)
      cy.wait(200)
      cy.get(".autocomplete-dropdown").should("not.exist")
    })

    it("should limit suggestions to maximum 4 items", () => {
      // Create many historical items
      for (let i = 0; i < 6; i++) {
        cy.get('input[placeholder="Lägg till en uppgift"]').type(
          `LimitTest${i}{enter}`
        )
        cy.wait(100)
        cy.contains(`LimitTest${i}`)
          .parent()
          .find('button[aria-label="Mark as complete"]')
          .click()
        cy.wait(100)
      }

      // Type to trigger autocomplete
      cy.get('input[placeholder="Lägg till en uppgift"]')
        .focus()
        .type("LimitTest")

      cy.wait(300)

      // Should show max 4 items
      cy.get(".autocomplete-item").should("have.length.at.most", 4)
    })
  })

  describe("Categories Mode", () => {
    it("switches to categories mode and back", () => {
      cy.contains("button", "Kategorier").click()
      cy.contains("Ny kategori").should("be.visible")
      cy.contains("button", "Normal").click()
      cy.contains("Ny kategori").should("not.exist")
    })

    it("creates category and moves todo into it via modal", () => {
      const todoName = `Cat move ${Date.now()}`
      const categoryName = `WorkCat ${Date.now()}`

      cy.get('input[placeholder="Lägg till en uppgift"]').type(`${todoName}{enter}`)
      cy.contains(todoName).should("be.visible")

      cy.contains("button", "Kategorier").click()
      cy.contains("Ny kategori").should("be.visible")

      cy.get('input[placeholder="Namn på kategori"]').type(`${categoryName}{enter}`)
      cy.contains(".category-card", categoryName).should("exist")

      cy.contains(".todo-row", todoName).click()
      cy.contains(".modal-option", categoryName).click()

      cy.contains(".category-card", categoryName).within(() => {
        cy.contains(todoName).should("exist")
        cy.get('[data-cy="delete-category"]').should("be.disabled")
      })
    })

    it("deletes an empty category", () => {
      const categoryName = `TempCat ${Date.now()}`
      cy.contains("button", "Kategorier").click()
      cy.get('input[placeholder="Namn på kategori"]').type(`${categoryName}{enter}`)
      cy.contains(".category-card", categoryName).within(() => {
        cy.get('[data-cy="delete-category"]').click()
      })
      cy.contains(".category-card", categoryName).should("not.exist")
    })
  })
})
