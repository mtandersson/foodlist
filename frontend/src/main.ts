import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";
import { initTheme } from "./lib/theme";
import { initPWAUpdates } from "./lib/pwa-update";

// Initialize theme (handles auto/light/dark modes and localStorage)
initTheme();

// Initialize PWA update checking (works with vite-plugin-pwa's auto-registration)
// Checks for updates every 30 seconds and applies them immediately
initPWAUpdates();

const app = mount(App, {
  target: document.getElementById("app")!,
});

export default app;
