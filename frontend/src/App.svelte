<script lang="ts">
  import TodoList from './lib/TodoList.svelte';
</script>

<main>
  <TodoList />
</main>

<style>
  main {
    min-height: 100vh;
    min-height: 100dvh;
    display: flex;
    justify-content: center;
    padding: var(--spacing-lg) 0;
  }

  @media (max-width: 768px) {
    main {
      padding: 0;
    }
  }
</style>
