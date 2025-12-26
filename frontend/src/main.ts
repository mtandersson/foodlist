import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'
import { initTheme } from './lib/theme'

// Initialize theme (handles auto/light/dark modes and localStorage)
initTheme();

const app = mount(App, {
  target: document.getElementById('app')!,
})

export default app
