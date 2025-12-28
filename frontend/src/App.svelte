<script lang="ts">
  import TodoList from './lib/TodoList.svelte';
</script>

<main>
  <TodoList />
</main>

<style>
  main {
    height: 100vh;
    height: 100dvh;
    display: flex;
    justify-content: center;
    padding-top: var(--spacing-lg);
    overflow: hidden;
  }

  @media (max-width: 768px) {
    main {
      padding-top: 0;
    }
  }
</style>
