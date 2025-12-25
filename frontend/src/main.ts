import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

// Set up automatic theme detection
function setupThemeDetection() {
  // Listen for changes in system color scheme preference
  const darkModeQuery = window.matchMedia('(prefers-color-scheme: dark)');
  
  // Optional: Add data attribute to HTML element for easier debugging
  function updateThemeAttribute() {
    const isDark = darkModeQuery.matches;
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light');
  }
  
  // Set initial theme
  updateThemeAttribute();
  
  // Listen for changes
  darkModeQuery.addEventListener('change', updateThemeAttribute);
}

// Initialize theme detection
setupThemeDetection();

const app = mount(App, {
  target: document.getElementById('app')!,
})

export default app
