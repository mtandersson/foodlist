/**
 * PWA Update Management
 *
 * Enhances vite-plugin-pwa's auto-update with periodic checking for faster updates.
 * The plugin handles registration automatically, this adds aggressive update checking.
 */

let updateCheckInterval: number | null = null;

/**
 * Check for service worker updates and apply immediately if available
 */
export async function checkForUpdates(): Promise<boolean> {
  if (!("serviceWorker" in navigator)) {
    return false;
  }

  try {
    const registration = await navigator.serviceWorker.getRegistration();
    if (!registration) {
      return false;
    }

    // Force check for updates
    await registration.update();

    // Check if there's a waiting service worker
    if (registration.waiting) {
      // Service worker is waiting, trigger skipWaiting
      registration.waiting.postMessage({ type: "SKIP_WAITING" });
      return true;
    }

    return false;
  } catch (error) {
    console.error("Error checking for updates:", error);
    return false;
  }
}

/**
 * Start periodic update checking
 * Checks every 30 seconds for new versions (aggressive checking)
 */
export function startUpdateChecking(intervalMs: number = 30000): void {
  if (!("serviceWorker" in navigator)) {
    return;
  }

  // Check immediately on start
  checkForUpdates();

  // Clear any existing interval
  if (updateCheckInterval) {
    clearInterval(updateCheckInterval);
  }

  // Set up periodic checking
  updateCheckInterval = window.setInterval(async () => {
    await checkForUpdates();
  }, intervalMs);
}

/**
 * Stop periodic update checking
 */
export function stopUpdateChecking(): void {
  if (updateCheckInterval) {
    clearInterval(updateCheckInterval);
    updateCheckInterval = null;
  }
}

/**
 * Initialize PWA update checking
 * Works with vite-plugin-pwa's auto-registration
 */
export function initPWAUpdates(): void {
  if (!("serviceWorker" in navigator)) {
    return;
  }

  // Wait for service worker to be ready
  if (document.readyState === "loading") {
    window.addEventListener("load", () => {
      setupUpdateListeners();
    });
  } else {
    setupUpdateListeners();
  }
}

/**
 * Set up update listeners for the service worker
 */
function setupUpdateListeners(): void {
  navigator.serviceWorker.getRegistration().then((registration) => {
    if (!registration) {
      return;
    }

    // Listen for updates found
    registration.addEventListener("updatefound", () => {
      const newWorker = registration.installing;
      if (!newWorker) return;

      newWorker.addEventListener("statechange", () => {
        if (
          newWorker.state === "installed" &&
          navigator.serviceWorker.controller
        ) {
          // New service worker installed, it will activate automatically
          // due to skipWaiting: true in config
          console.log(
            "New service worker installed, update will apply on next reload",
          );
        }
      });
    });

    // Listen for controller change (service worker takeover)
    navigator.serviceWorker.addEventListener("controllerchange", () => {
      console.log("Service worker updated, reloading page...");
      window.location.reload();
    });
  });

  // Start aggressive periodic checking (every 30 seconds)
  startUpdateChecking(30000);
}
