<script lang="ts">
  import TodoList from './lib/TodoList.svelte';
</script>

<main>
  <TodoList />
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
    background: var(--primary-gradient);
    min-height: 100vh;
    transition: background 0.3s ease;
  }

  :global(*) {
    box-sizing: border-box;
  }

  main {
    min-height: 100vh;
    display: flex;
    justify-content: center;
    padding: 24px 0;
  }
</style>
