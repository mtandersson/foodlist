# Suggestions (Förslag) tab

Surface groceries the user is likely to buy soon, based on completed-todo
history.

## What the user sees

A third tab next to **Normal** and **Edit** named **Förslag** (the lightbulb
icon, 💡). The tab is only visible when the backend reports
`featureFlags.suggestions === true` in the initial `StateRollup`, which
happens when both `GEMINI_API_KEY` and `SUGGESTIONS_ENABLED` are set.

Each row shows:

- The (cluster-canonical) product name.
- An optional category badge, using the most common category the user has
  historically assigned the cluster to.
- A short relative date for the last purchase, e.g. `2d`, `3v`, `2mån`.
- A `+` button that creates a regular todo on the active list and removes
  the row from the suggestions tab.

The list is sorted by purchase frequency, most-frequent first.

## How the engine decides

`backend/suggestions.go` rebuilds the list with the following pipeline:

1. Group completed todos by `normalizeName(todo.Name)`.
2. Cluster groups by cosine similarity of their text embeddings using
   `SUGGESTIONS_DEDUP_SIMILARITY` as the threshold. The cluster's canonical
   name is the most popular member.
3. Drop clusters with fewer than `SUGGESTIONS_MIN_PURCHASES` total
   completions.
4. Compute `avgIntervalSeconds` from the gaps between the last
   `SUGGESTIONS_RECENT_LIMIT` completions. Drop clusters where the average
   interval exceeds `SUGGESTIONS_MAX_INTERVAL_DAYS`.
5. Keep only clusters that are "due", i.e. where
   `(now - lastPurchasedAt) / avgInterval >= SUGGESTIONS_DUE_FRACTION`.
6. Dedupe against the active (uncompleted) todo list both by normalized
   name and by embedding similarity, so e.g. an active "2l mjölk" hides a
   "Mjölk" suggestion.
7. Assign the most-common historical category to the cluster.

The recompute runs:

- Once on startup, after the initial event replay.
- On every `TodoCreated`, `TodoCompleted`, `TodoUncompleted`,
  `TodoRenamed`, `TodoCategorized`, `CategoryCreated`, `CategoryRenamed`,
  or `CategoryDeleted` event.
- Periodically every `SUGGESTIONS_RECOMPUTE_HOURS` hours as a safety net.

## Wire protocol

A new client receives:

1. The usual `StateRollup`, now carrying
   `featureFlags: { suggestions: true }`.
2. A `SuggestionsRollup` with the full snapshot.

After that, deltas are pushed:

- `SuggestionAdded { suggestion }` when a cluster becomes due.
- `SuggestionRemoved { id }` when a cluster is no longer due (or was
  added to the active list).

A `TodoCreated` event optimistically pushes a `SuggestionRemoved` for any
matching cluster so the UI feels instant; the full recompute reconciles
afterward.

Delta sends are non-blocking — if the broadcast channel is saturated the
delta is dropped and the next periodic recompute or reconnect will resync
clients.

## MCP

Two MCP surfaces expose the same data to AI agents:

- Tool `foodlist_suggestions` returns the current list as Markdown.
- Resource `foodlist://suggestions` returns the list as JSON.

Both return an empty list (and a message explaining the feature is off)
when the engine is not initialized.

## Configuration

See [`backend/CONFIG.md`](../../backend/CONFIG.md#suggestions-förslag-tab)
for the full list of environment variables and their defaults.

## Manual testing scenarios

| Scenario                                              | Expected                                                                  |
| ----------------------------------------------------- | ------------------------------------------------------------------------- |
| Fresh DB, no completions                              | Förslag tab visible (if `GEMINI_API_KEY` set), shows empty state          |
| `GEMINI_API_KEY` unset                                | Förslag tab is hidden in the frontend                                     |
| Complete the same item 4× over ~4 weeks               | Item appears in Förslag once the due fraction is reached                  |
| Click `+` on a suggestion                             | Item moves to the active list; row disappears immediately                 |
| Rename a previously-purchased item to something new   | Old cluster's count drops, suggestion may disappear                       |
| Categorize a previously-purchased item                | Suggestion picks up the new category on the next render                   |
| Add `2l mjölk` to active list while `Mjölk` suggested | `Mjölk` is filtered out (embedding similarity above dedup threshold)      |
