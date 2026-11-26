// Playwright settings for the browser checks.
import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  testMatch: "*.spec.mjs",
  // Every check opens the site as a guest of its own (a new browser context),
  // so the checks share the server without sharing data and run side by side.
  fullyParallel: true,
  workers: process.env.TT_WORKERS ? Number(process.env.TT_WORKERS) : "50%",
  timeout: 90_000,
  // No retries: a check that fails now and then has found a race worth fixing.
  retries: 0,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: "http://localhost:3000",
    viewport: { width: 1600, height: 900 },
    // TT_BROWSER=msedge runs the checks in the installed Edge (or chrome in
    // Chrome), for a machine where `npx playwright install chromium` won't do.
    channel: process.env.TT_BROWSER || undefined,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  webServer: {
    command: "go run .",
    cwd: "../..",
    url: "http://localhost:3000/api/v1/languages",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
