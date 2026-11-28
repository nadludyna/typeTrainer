import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

// npm run dev: the page with hot reload on :5173, /api/ sent to the Go
// server on :3000. npm run build: the page into dist/page, built into the server.
// --mode pages: the GitHub Pages build, which talks to the Go program running
// in the page as WebAssembly instead (src/core/pages.ts).
export default defineConfig(({ mode }) => ({
  base: mode === "pages" ? "./" : "/",
  plugins: [svelte(), tailwindcss()],
  define: { __PAGES__: JSON.stringify(mode === "pages") },
  server: { proxy: { "/api": "http://localhost:3000" } },
  build: { outDir: "dist/page", emptyOutDir: true, target: "es2023" }, // dist/.keep stays for go:embed
  test: { environment: "node", include: ["src/**/*.test.ts"] },
}));
