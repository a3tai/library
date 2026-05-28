import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import wails from "@wailsio/runtime/plugins/vite";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte(), wails("./bindings")],
  // Pre-declare every external dep we import directly so Vite optimizes
  // them at server startup instead of on first page load. Without this,
  // Vite's "discover on first request" optimizer races against the
  // Wails webview, which loads the page immediately — Vite then needs
  // to re-optimize and briefly closes its port, causing the Go-side
  // asset proxy to return "connection refused" and (on Wails alpha.89)
  // the app to terminate.
  optimizeDeps: {
    include: [
      "@wailsio/runtime",
      "@lucide/svelte",
      "marked",
      "dompurify",
    ],
  },
  build: {
    rolldownOptions: {
      checks: {
        pluginTimings: false,
      },
    },
  },
  server: {
    // Wails proxies asset requests through the dev server. Keep the
    // port stable so the proxy can always find us.
    strictPort: true,
    // Wails alpha.79+ forces tcp4 for the dev-mode reverse proxy. Vite's
    // default "localhost" binding can resolve to ::1 (IPv6) on macOS,
    // which makes the IPv4 dial fail with "connection refused" even
    // though the browser (which dual-stacks) connects fine. Binding to
    // 127.0.0.1 explicitly forces an IPv4 listener.
    host: "127.0.0.1",
  },
});
