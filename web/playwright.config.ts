import { defineConfig } from "@playwright/test";

// The browser checks: real rounds typed in a real browser against the built
// page served by the Go server on :3100 (npx playwright test). TT_BROWSER=msedge
// or chrome uses an installed browser where Playwright's Chromium is missing.
const PORT = 3100;

export default defineConfig({
  testDir: "e2e",
  testMatch: "*.spec.ts",
  fullyParallel: true,
  retries: 0,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL: `http://localhost:${PORT}`,
    channel: process.env.TT_BROWSER || undefined,
    viewport: { width: 1366, height: 820 },
    trace: "retain-on-failure",
  },
  webServer: {
    command: `npm run build && go -C .. run ./cmd/typetrainer`,
    env: { ADDR: `localhost:${PORT}` },
    url: `http://localhost:${PORT}`,
    timeout: 180_000,
    reuseExistingServer: false,
  },
});
