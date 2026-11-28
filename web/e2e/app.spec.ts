// What a person does on the site, typed in a real browser. Every test is a
// new browser, so a new guest with no rounds.

import { expect, type Page, test } from "@playwright/test";

const DELAY = 70; // ms between presses: about 170 wpm, under the server's 400

/** untyped is the visible text from the caret on. */
async function untyped(page: Page) {
  return page.locator("[data-text] > span[data-m='0']:not(.invisible)").allTextContents();
}

/** typeRound types what is shown until the round is over (its counter starts again or a card shows). */
async function typeRound(page: Page, wrongAt = -1) {
  let n = 0;
  for (let tries = 0; tries < 40; tries++) {
    const chars = await untyped(page);
    if (!chars.length) break;
    for (const c of chars) {
      await page.keyboard.press(n === wrongAt ? "x" : c === " " ? "Space" : c, { delay: 0 });
      n++;
      await page.waitForTimeout(DELAY);
    }
    if ((await page.getByRole("status").count()) || (await page.getByRole("region", { name: "Result" }).count())) break;
    if ((await counter(page)).startsWith("0 /")) break;
  }
  return n;
}

const counter = async (page: Page) =>
  ((await page.locator("section[aria-label='Text to type'] .tabular-nums").last().textContent()) ?? "")
    .replace(/\s+/g, " ")
    .trim();

async function toLearn(page: Page) {
  await page.goto("/");
  await page.keyboard.press("Enter");
  await page.getByRole("button", { name: "No", exact: true }).click();
  await expect(page.locator("[data-text]")).toBeVisible();
}

test("Welcome walks the fingers and Enter starts Learn with its question", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByText("1 / 8 · left pinky")).toBeVisible();
  for (const k of ["q", "a", "z"]) await page.keyboard.press(k);
  await expect(page.getByText("2 / 8 · left ring")).toBeVisible();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("heading", { name: "Have you learned touch typing before?" })).toBeVisible();
  await page.getByRole("button", { name: "Yes", exact: true }).click();
  await expect(page.getByLabel("Letters").locator(":scope > span")).toHaveCount(10);
});

test("a Learn round is typed, saved and the next one starts", async ({ page }) => {
  await toLearn(page);
  await expect(page.getByLabel("Keyboard")).toBeVisible();
  await typeRound(page);
  await expect.poll(() => counter(page)).toBe("0 / 30");
  const stats = await page.request.get("/api/learn/stats").then((r) => r.json());
  expect(stats.history.count).toBe(1);
});

test("a wrong letter is marked, the caret moves on and Backspace takes it back", async ({ page }) => {
  await toLearn(page);
  const first = (await untyped(page))[0] ?? "";
  const wrong = first === "x" ? "q" : "x";
  await page.keyboard.press(wrong);
  const marked = page.locator("[data-text] > span[data-m='2']");
  await expect(marked).toHaveCount(1);
  await page.keyboard.press("Backspace");
  await expect(marked).toHaveCount(0);
});

test("Tab starts the round again", async ({ page }) => {
  await toLearn(page);
  const before = (await untyped(page)).join("");
  await page.keyboard.press(before[0] ?? "a");
  await page.keyboard.press("Tab");
  await expect(page.locator("[data-text] > span[data-m='1']")).toHaveCount(0);
});

test("the text shows three lines and the caret one line tall", async ({ page }) => {
  await toLearn(page);
  const { lines, caret, line } = await page.evaluate(() => {
    const text = document.querySelector("[data-text]")?.parentElement as HTMLElement;
    const letter = document.querySelector("[data-text] > span[data-m]") as HTMLElement;
    const lineH = parseFloat(getComputedStyle(letter).lineHeight);
    const caretEl = document.querySelector("[data-text] > span[aria-hidden]") as HTMLElement;
    return { lines: text.clientHeight / lineH, caret: caretEl.offsetHeight, line: lineH };
  });
  expect(Math.round(lines)).toBe(3);
  expect(caret).toBeLessThanOrEqual(line);
});

test("Reinforce: a word level is picked and a clean round ends at the third mistake", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Reinforce" }).click();
  await page.getByRole("button", { name: "500" }).click();
  await expect(page.getByRole("button", { name: "500" })).toHaveAttribute("aria-pressed", "true");
  await page.getByRole("button", { name: "Clean" }).click();
  for (let i = 0; i < 3; i++) {
    const c = (await untyped(page))[0] ?? "";
    await page.keyboard.press(c === "x" ? "q" : "x");
    await page.waitForTimeout(DELAY);
  }
  await expect.poll(() => counter(page)).toBe("0 / 30");
  const stats = await page.request.get("/api/reinforce/stats").then((r) => r.json());
  expect(stats.history.rounds[0]).toMatchObject({ kind: "clean", level: 2 });
});

test("Freetype: a 10-word round ends on its result, and Enter goes on", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Freetype" }).click();
  await page.getByRole("button", { name: "10", exact: true }).click();
  await expect.poll(() => counter(page)).toBe("0 / 10");
  await expect(page.locator("[data-text] > span:not(.invisible)").filter({ hasText: /\S/ })).not.toHaveCount(0);
  await typeRound(page);
  const result = page.getByRole("region", { name: "Result" });
  await expect(result).toBeVisible();
  await expect(result.getByText("wpm", { exact: true })).toBeVisible();
  await page.keyboard.press("Enter");
  await expect(result).toBeHidden();
  await expect.poll(() => counter(page)).toBe("0 / 10");
});

test("Freetype: a time round ends when its time is up", async ({ page }) => {
  test.setTimeout(60_000);
  await page.goto("/");
  await page.getByRole("button", { name: "Freetype" }).click();
  await page.getByRole("button", { name: "Time" }).click();
  await page.getByRole("button", { name: "15", exact: true }).click();
  const result = page.getByRole("region", { name: "Result" });
  const until = Date.now() + 25_000;
  while (!(await result.count()) && Date.now() < until) {
    await page.keyboard.press((await untyped(page))[0] ?? "a");
    await page.waitForTimeout(250);
  }
  await expect(result).toBeVisible();
});

test("stats open on the key left of 1, whatever the layout, and Escape closes them", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Learn", exact: true }).click();
  await page.evaluate(() => window.dispatchEvent(new KeyboardEvent("keydown", { key: "ё", code: "Backquote" })));
  await expect(page.getByRole("region", { name: "Learn Stats" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("region", { name: "Learn Stats" })).toBeHidden();
});

test("settings: the keyboard of a mode, export, and starting Learn over", async ({ page }) => {
  await toLearn(page);
  await typeRound(page);
  await page.keyboard.press("Escape");
  const dialog = page.getByRole("dialog", { name: "Settings" });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("checkbox").uncheck();
  const [download] = await Promise.all([
    page.waitForEvent("download"),
    dialog.getByRole("button", { name: "Export" }).click(),
  ]);
  expect(download.suggestedFilename()).toMatch(/^typetrainer-.*\.json$/);
  await dialog.getByRole("button", { name: "Start Learn over" }).click();
  await dialog.getByRole("button", { name: "Start Learn over" }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByLabel("Keyboard")).toBeHidden();
  const stats = await page.request.get("/api/learn/stats").then((r) => r.json());
  expect(stats.history.count).toBe(0);
});
