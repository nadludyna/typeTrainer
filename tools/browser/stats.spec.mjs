// Browser checks: the stats of each mode.
import { test, expect } from "@playwright/test";
import { openApp, typeRound, waitForText, nextChar } from "./app.mjs";
import { watchPage, expectResult, withHistory, withFreetypeHistory, expectedOnly } from "./checks.mjs";

watchPage(test);

test("the stats of the mode you're in: ~ or the Stats button opens them, ~, Esc or Tab closes them", async ({ page }) => {
  await withHistory(page);
  await withFreetypeHistory(page);
  await openApp(page, { "tt.mode.en": "learn" });
  const view = page.locator("#view-stats"), sheet = page.locator("#sheet");
  await page.keyboard.press("Backquote");
  await expect(view).toBeVisible();
  await expect(sheet.locator("h1")).toHaveText("Learn stats");
  await expect(sheet.locator(".tile.hero"), "Learn leads with accuracy").toContainText("Accuracy, last 10 rounds");
  await expect(sheet.locator(".st-chart svg")).toBeVisible();
  await expect(sheet.locator(".st-phone"), "a keyboard's rounds teach the trainer: no phone note").toHaveCount(0);
  await expect(page.locator("#tab-stats")).toHaveAttribute("aria-selected", "true");
  await page.waitForTimeout(350);
  await page.keyboard.press("Escape");
  await expect(page.locator("#view-type"), "Esc goes back to typing").toBeVisible();
  await expect(page.locator("#settings-win"), "and opens no settings").toBeHidden();

  await page.locator("#tab-reinforce").click();
  await page.locator("#tab-stats").click();
  await expect(sheet.locator("h1")).toHaveText("Reinforce stats");
  await expect(sheet.locator(".tile.hero"), "Reinforce leads with speed").toContainText("Speed, last 10 rounds");
  await expect(sheet.locator(".chart .target"), "and its goal").toHaveCount(1);
  await page.locator("#tab-stats").click();
  await expect(page.locator("#view-type"), "the button closes them too").toBeVisible();
  await waitForText(page);

  await page.locator("#tab-freetype").click();
  await page.keyboard.press("Backquote");
  await expect(sheet.locator("h1")).toHaveText("Freetype stats");
  await expect(sheet.locator(".tile.hero"), "Freetype leads with consistency").toContainText("Consistency, last 10 rounds");
  await expect(sheet.locator(".ring"), "no mode switch inside the stats").toHaveCount(0);
  await page.waitForTimeout(350);
  await page.keyboard.press("Tab");
  await expect(page.locator("#view-type"), "Tab: back to a new round").toBeVisible();
  await expect(page.locator("#text"), "the key types nothing").not.toContainText("`");
});

test("stats with no rounds say so, and Freetype's say each word list counts apart", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype" });
  await page.keyboard.press("Backquote");
  await expect(page.locator(".st-empty h2")).toHaveText("No Freetype rounds yet");
  await expect(page.locator(".st-empty p")).toContainText("Each word list keeps its own stats");
  await expect(page.locator(".st-tiles")).toHaveCount(0);
});

test("stats: Freetype shows your own speed goal, Reinforce its goal step", async ({ page }) => {
  await withHistory(page);
  await withFreetypeHistory(page);
  await openApp(page, { "tt.mode.en": "freetype", "tt.goal": "100" });
  await page.keyboard.press("Backquote");
  await expect(page.locator(".st-tiles .tile").nth(1)).toContainText("goal 100 wpm");
  await openApp(page, { "tt.mode.en": "freetype", "tt.goal": "0" });
  await page.keyboard.press("Backquote");
  await expect(page.locator(".st-tiles .tile").nth(1), "no goal, no goal text").not.toContainText("goal");
  await openApp(page, { "tt.mode.en": "reinforce" });
  await page.keyboard.press("Backquote");
  const target = await page.evaluate(() => fetch("/api/v1/stats?length=words-30&part=reinforce").then((r) => r.json()).then((s) => s.targetWpm));
  await expect(page.locator(".st-tiles .tile.hero")).toContainText(`goal ${target} wpm`);
});

test("stats: the title, the tiles and the panels share one left edge", async ({ page }) => {
  await withHistory(page);
  await openApp(page, { "tt.mode.en": "learn" });
  await page.keyboard.press("Backquote");
  await expect(page.locator(".st-chart svg")).toBeVisible();
  const lefts = await page.evaluate(() => [".st-head h1", ".st-tiles", ".st-two", ".activity"]
    .map((q) => Math.round(document.querySelector(q).getBoundingClientRect().left)));
  expect(new Set(lefts).size, `left edges ${lefts}`).toBe(1);
});

test("stats on a phone: nothing is wider than the screen", async ({ page }) => {
  await withHistory(page);
  for (const width of [390, 375]) {
    await page.setViewportSize({ width, height: 800 });
    await openApp(page, { "tt.mode.en": "learn" });
    await page.locator("#tab-stats").click();
    await expect(page.locator(".st-chart svg")).toBeVisible();
    const wide = await page.evaluate(() => [...document.querySelectorAll("#view-stats *")]
      .filter((n) => n.getBoundingClientRect().right > innerWidth + 0.5)
      .map((n) => `${n.tagName.toLowerCase()}#${n.id}.${n.getAttribute("class")}`));
    expect(wide, `elements past the right edge at ${width}px`).toEqual([]);
    expect(await page.evaluate(() => document.documentElement.scrollWidth), "no sideways scroll").toBe(width);
  }
});

test.describe("on a phone", () => {
  test.use({ viewport: { width: 390, height: 800 }, hasTouch: true });
  test("the stats say a phone's rounds don't move letters or the word level", async ({ page }) => {
    await withHistory(page);
    await withFreetypeHistory(page);
    for (const mode of ["learn", "reinforce", "freetype"]) {
      await openApp(page, { "tt.mode.en": mode });
      await page.locator("#tab-stats").click();
      await expect(page.locator(".st-tiles .tile.hero")).toBeVisible();
      await expect(page.locator(".st-phone"), `${mode}: Freetype learns nothing yet, so it needs no note`).toHaveCount(mode === "freetype" ? 0 : 1);
    }
  });
});

test("stats on a desktop: every tile holds its own text", async ({ page }) => {
  await page.route(/\/api\/v1\/(freetype\/)?stats\?/, async (route) => {
    const res = await route.fetch(), body = await res.json();
    body.sessions = Array.from({ length: 20 }, (_, i) => ({ ...(body.sessions?.[0] ?? {}), wpm: 52 - i, accuracy: 96, consistency: 80 }));
    body.sessionCount = 128;
    if (!body.lengths?.length) body.lengths = ["words-30"];
    return route.fulfill({ response: res, json: body });
  });
  for (const mode of ["freetype", "learn", "reinforce"]) for (const [width, height] of [[1024, 768], [1366, 768], [1920, 1080]]) {
    await page.setViewportSize({ width, height });
    await openApp(page, { "tt.mode.en": mode, "tt.goal": "50" });
    await page.locator("#tab-stats").click();
    await expect(page.locator(".st-tiles .tile.hero")).toBeVisible();
    const past = () => page.evaluate(() => [...document.querySelectorAll(".st-tiles .tile")].flatMap((tile) => {
      const edge = tile.getBoundingClientRect().right - parseFloat(getComputedStyle(tile).paddingRight);
      return [...tile.querySelectorAll("small, b, .meta, .meta *")]
        .filter((n) => n.getBoundingClientRect().right > edge + 0.5 || n.scrollWidth > n.clientWidth + 1)
        .map((n) => `${tile.querySelector("small").textContent}: "${n.textContent}" ${n.scrollWidth}/${n.clientWidth}`);
    }));
    await expect.poll(past, { message: `text past its tile, ${mode} at ${width}px`, timeout: 3000 }).toEqual([]);
  }
});

test("a click on the logo goes home: from the stats and from a result", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  await page.locator("#tab-stats").click();
  await expect(page.locator("#view-stats")).toBeVisible();
  await page.locator("#logo").click();
  await expect(page.locator("#view-type")).toBeVisible();
  await typeRound(page);
  await expectResult(page);
  await page.locator("#logo").click();
  await expect(page.locator("body")).not.toHaveClass(/\btb-resulting\b/);
});

test("the stats screen opens after a round and shows it", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  await typeRound(page);
  await expectResult(page);
  await page.locator("#tab-stats").click();
  await expect(page.locator("#view-stats")).toBeVisible();
  await expect(page.locator("#view-type")).toBeHidden();
  await expect(page.locator(".st-tiles .tile").last(), "the round counts").toContainText("1");
  await expect(page.locator(".st-chart .hit")).toHaveCount(1);
  const [bar, svg] = await Promise.all([page.locator(".st-chart .hit").boundingBox(), page.locator(".st-chart svg").boundingBox()]);
  expect(svg.x + svg.width - (bar.x + bar.width), "with one round the bar stands at the right edge").toBeLessThan(svg.width * 0.1);
});
