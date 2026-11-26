// What the browser checks share: an error in the page fails a check, and
// the made-up histories some checks start from.
import { expect } from "@playwright/test";

// watchPage: any page error, console error or failed API call fails the check
// (a check that expects one removes it from info.problems, expectedOnly).
export function watchPage(test) {
  test.beforeEach(async ({ page }, info) => {
    const problems = [];
    page.on("pageerror", (e) => problems.push(`page error: ${e.message}`));
    page.on("console", (m) => { if (m.type() === "error") problems.push(`console: ${m.text()}`); });
    page.on("response", (r) => { if (r.url().includes("/api/") && r.status() >= 400) problems.push(`HTTP ${r.status()} ${r.url()}`); });
    info.problems = problems;
  });
  test.afterEach(async ({}, info) => {
    expect(info.problems, "errors in the page").toEqual([]);
  });
}

export async function expectResult(page) {
  await expect(page.locator("body")).toHaveClass(/\btb-resulting\b/);
  await expect(page.locator("#result")).toBeVisible();
  await expect(page.locator("#result")).toContainText(/\d/);
}

// Learn's /stats as if 12 rounds were typed: speeds newest first, the last 4 at 97%.
export async function withHistory(page) {
  const wpms = [52, 51, 48, 50, 49, 47, 48, 45, 46, 44, 45, 43];
  await page.route("**/api/v1/stats?*", async (route) => {
    const res = await route.fetch(), body = await res.json();
    body.sessions = wpms.map((wpm, i) => ({ ...(body.sessions?.[0] ?? {}), wpm, accuracy: i < 4 ? 97 : 90 }));
    body.forecast = { known: true, rounds: 4, letter: "e" };
    if (!body.lengths?.length) body.lengths = ["words-30"];
    body.streaks = [{ min: 95, current: 4, longest: 6 }, { min: 97, current: 4, longest: 4 }, { min: 100, current: 0, longest: 1 }];
    return route.fulfill({ response: res, json: body });
  });
}

// Freetype's /freetype/stats as if 12 rounds were typed, the newest first.
export async function withFreetypeHistory(page) {
  await page.route("**/api/v1/freetype/stats?*", async (route) => {
    const res = await route.fetch(), body = await res.json();
    body.sessions = Array.from({ length: 12 }, (_, i) => ({ wpm: 50 - i, accuracy: 96, consistency: 80 - i, length: "words-30" }));
    body.sessionCount = 12;
    if (!body.lengths?.length) body.lengths = ["words-30"];
    return route.fulfill({ response: res, json: body });
  });
}

export const expectedOnly = (info, path) => {
  const rest = info.problems.filter((p) => !p.includes(path) && !p.startsWith("console:"));
  info.problems.splice(0, info.problems.length, ...rest);
};
