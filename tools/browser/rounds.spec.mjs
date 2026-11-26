// Browser checks: a round in each mode: typed to the end, its numbers, its result, saving it.
import { test, expect } from "@playwright/test";
import { openApp, typeRound, waitForText, nextChar } from "./app.mjs";
import { watchPage, expectResult, withHistory, withFreetypeHistory, expectedOnly } from "./checks.mjs";

watchPage(test);

test("Learn: a round can be typed to the end and is saved", async ({ page }) => {
  let saved = false;
  page.on("response", (r) => {
    if (r.request().method() === "POST" && new URL(r.url()).pathname === "/api/v1/results" && r.ok()) saved = true;
  });
  await openApp(page, { "tt.mode.en": "learn", "tt.length.learn": "words-15" });
  await expect(page.locator("body")).toHaveAttribute("data-mode", "learn");
  await expect(page.locator("#round-count"), "one round length: an old 15 becomes 30").toHaveText("0 / 30");
  await expect(page.locator("#letters .locked.next").first(), "before the first round: how to open the next letter")
    .toHaveAttribute("data-tip-text", /Mastered means 90% accurate over your last 100 presses of the letter\.$/);
  await expect(page.locator("#lp-cols .lp-cell"), "the last round waits for the first round").toHaveCount(0);
  await typeRound(page, { done: async () => saved });
  await expect(page.locator("#round-count")).toHaveText("0 / 30");
  await expect(page.locator("#lp-cols .lp-last [data-n]").first()).toHaveAttribute("data-n", /^[1-9]/);
});

test("Learn: the line over the text holds the last round, the daily goal and the round's count", async ({ page }) => {
  await withHistory(page);
  await openApp(page, { "tt.mode.en": "learn" });
  const last = page.locator("#lp-cols .lp-last"), goal = page.locator("#goal-count");
  await expect(last.locator("[data-n]")).toHaveCount(2);
  await expect(last.locator("[data-n]").first()).toHaveAttribute("data-n", /^[1-9]\d*$/);
  await expect(last).toHaveAttribute("data-tip-text", /^Your last round\. .*faster/);
  await expect(goal, "minutes and seconds").toHaveText(/^(\d+:\d\d of 15:00|15:00 done)$/);
  await expect(goal).toHaveAttribute("data-tip-text", /Daily goal/);
  await expect(page.locator("#letters .locked.next").first()).toHaveAttribute("data-tip-text", /rounds? away\. Mastered means/);
  await goal.click();
  await expect(page.locator("#mini-pop button[data-key=\"daily\"]").first(), "the number changes the daily goal").toBeVisible();
});

test("Learn: the daily goal counts typed time only and never steps back", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "learn" });
  const goal = page.locator("#goal-count");
  const seconds = async () => {
    const [m, s] = (await goal.textContent()).split(" ")[0].split(":").map(Number);
    return m * 60 + s;
  };
  for (let i = 0; i < 6; i++) { await page.keyboard.type(await nextChar(page)); await page.waitForTimeout(400); }
  await page.waitForTimeout(600);
  const typed = await seconds();
  expect(typed, "the first keys count").toBeGreaterThanOrEqual(1);
  await page.waitForTimeout(3000);
  expect(await seconds(), "time without keys does not count").toBe(typed);
  await page.keyboard.press("Tab");
  await waitForText(page);
  await page.waitForTimeout(800);
  expect(await seconds(), "a round dropped by Tab keeps its time").toBe(typed);
});

test("Learn: a best round's speed is gold, the letters change state without motion", async ({ page }) => {
  await withHistory(page);
  await openApp(page, { "tt.mode.en": "learn" });
  await page.waitForFunction(() => !document.getAnimations().some((a) => a.playState === "running"));
  await page.evaluate(async () => {
    const m = await import("/learnpanel.js");
    m.roundMilestone({ best: true, mastered: [] });
    m.renderPanel();
  });
  await expect(page.locator("#lp-cols .lp-last .lp-best"), "a best round's speed stands out").toHaveCount(1);
  expect(await page.evaluate(() => document.getAnimations().length),
    "nothing runs on the screen").toBe(0);
});

test("Learn: while typing this round's numbers take the last round's place, nothing moves", async ({ page }) => {
  await withHistory(page);
  await openApp(page, { "tt.mode.en": "learn", "tt.live": "1" });
  const last = page.locator("#lp-cols"), wpm = page.locator("#wpm"), count = page.locator("#round-count");
  const row = page.locator("#letters"), text = page.locator("#text");
  await expect(last).toBeVisible();
  await expect(page.locator(".live > .metric.num").first()).toHaveCSS("opacity", "0");
  const at = async () => Promise.all([row, count, text].map(async (l) => (await l.boundingBox()).y));
  const before = await at();
  await page.keyboard.type(await nextChar(page));
  await expect(last).toBeHidden();
  await expect(page.locator(".live > .metric.num").first()).toHaveCSS("opacity", "1");
  await expect(page.locator("#acc")).toBeVisible();
  await page.waitForTimeout(500);
  expect(await at(), "nothing moved").toEqual(before);
});

const MODES = [
  { name: "Freetype", storage: { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" } },
];

for (const mode of MODES) {
  test(`${mode.name}: a round can be typed to the end and shows the result`, async ({ page }) => {
    await openApp(page, mode.storage);
    await expect(page.locator("body")).toHaveAttribute("data-mode", mode.storage["tt.mode.en"]);
    await typeRound(page);
    await expectResult(page);
    await page.keyboard.press("Tab");
    await expect(page.locator("body")).not.toHaveClass(/\btb-resulting\b/);
    await waitForText(page);
  });

  test(`${mode.name}: typing neither dims nor moves the round switches`, async ({ page }) => {
    await openApp(page, mode.storage);
    const bar = page.locator("#mode-bar"), length = bar.locator(".bar-side:last-child > .ring").last();
    const px = (b) => ({ x: Math.round(b.x), y: Math.round(b.y), width: Math.round(b.width), height: Math.round(b.height) });
    const before = px(await bar.boundingBox());
    await page.keyboard.type(await nextChar(page));
    await expect(length).toHaveCSS("opacity", "1");
    expect(px(await bar.boundingBox()), "the bar stays where it was").toEqual(before);
  });
}

test("Freetype: a faster round shows New best, and the small button starts the next round", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  await typeRound(page, { delay: 160 });
  await expectResult(page);
  await expect(page.locator("#result .r-best")).toHaveCount(0);
  await expect(page.locator(".hints .result-only")).toBeVisible();
  await expect(page.locator(".hints .round-only").first()).toBeHidden();
  await page.locator("#result .r-next").click();
  await expect(page.locator("body")).not.toHaveClass(/\btb-resulting\b/);
  await waitForText(page);
  await typeRound(page, { delay: 40 });
  await expectResult(page);
  await expect(page.locator("#result .r-best")).toHaveText("New best");
  expect(await page.locator(".confetti").count(), "no confetti").toBe(0);
  await page.keyboard.press("Enter");
  await expect(page.locator("body")).not.toHaveClass(/\btb-resulting\b/);
});

test("Freetype: an unsaved round still shows its result and can be saved again", async ({ page }, info) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  let tries = 0;
  await page.route("**/api/v1/freetype/results*", (route) => { tries++; return route.abort(); });
  await typeRound(page);
  await expectResult(page);
  const line = page.locator("#result .r-unsaved");
  await expect(line).toContainText("isn't saved yet");
  expect(tries, "the save and its two retries").toBe(3);
  await page.unroute("**/api/v1/freetype/results*");
  await line.getByRole("button", { name: "Save again" }).click();
  await expect(line).toHaveCount(0);
  await expect(page.locator("#result .r-saved")).toBeVisible();
    expectedOnly(info, "/freetype/results");
});

test("Freetype: a round whose answer was lost is sent again and kept once", async ({ page }, info) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  let tries = 0;
  await page.route("**/api/v1/freetype/results*", async (route) => {
    if (++tries === 1) { await route.fetch(); return route.abort(); }
    return route.continue();
  });
  await typeRound(page);
  await expectResult(page);
  expect(tries).toBe(2);
  const count = await page.evaluate(async () =>
    (await (await fetch("/api/v1/freetype/stats?lang=en&length=words-10")).json()).sessionCount);
  expect(count, "rounds in the Freetype history").toBe(1);
  expectedOnly(info, "/freetype/results");
});

test("Freetype: a round the server turns down says so and is not tried again", async ({ page }, info) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  let tries = 0;
  await page.route("**/api/v1/freetype/results*", (route) => {
    tries++;
    return route.fulfill({ status: 400, contentType: "application/json", body: '{"error":"wpm or accuracy out of range"}' });
  });
  await typeRound(page);
  await expectResult(page);
  const line = page.locator("#result .r-unsaved");
  await expect(line).toContainText("looks impossible");
  await expect(line.getByRole("button")).toHaveCount(0);
  expect(tries).toBe(1);
  expectedOnly(info, "/freetype/results");
});

test("Learn: an unsaved round offers to save it again", async ({ page }, info) => {
  await openApp(page, { "tt.mode.en": "learn", "tt.length.learn": "words-15" });
  await page.route("**/api/v1/results*", (route) => route.abort());
  await typeRound(page, { done: async () => page.locator("#notice .note-act").isVisible() });
  await page.unroute("**/api/v1/results*");
  await page.locator("#notice .note-act").click();
  await expect(page.locator("#notice")).toContainText("Saved");
  expectedOnly(info, "/api/v1/results");
});

test("Freetype: the live numbers show from the first key, not as zeros at rest", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10", "tt.live": "1" });
  const wpm = page.locator(".live > .metric.num").first();
  await expect(wpm, "no zeros before the round").toHaveCSS("opacity", "0");
  await page.keyboard.type(await nextChar(page));
  await expect(wpm, "the numbers after the first key").toHaveCSS("opacity", "1");
  await typeRound(page);
  await expectResult(page);
  await page.keyboard.press("Tab");
  await waitForText(page);
  await expect(wpm, "the last round's numbers don't stay over the new text").toHaveCSS("opacity", "0");
});

test("Freetype: the result opens in place and the round switches do not move", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  const bar = page.locator("#mode-bar");
  const before = (await bar.boundingBox()).y;
  await typeRound(page);
  await expectResult(page);
  await page.waitForFunction(() => !document.getAnimations().some((a) => a.playState === "running" && a.effect?.target?.closest?.("#view-type")));
  expect((await bar.boundingBox()).y, "the bar on the result").toBeCloseTo(before, 0);
  const box = await page.locator("#result").boundingBox();
  expect(box.y + box.height, "the result fits the window").toBeLessThanOrEqual(page.viewportSize().height);
});

test("the round's count follows each word and goes back when a word is undone", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  const count = page.locator("#round-count");
  for (;;) { const c = await nextChar(page); await page.keyboard.type(c); if (c === " ") break; }
  await expect(count).toHaveText("1 / 10");
  await page.keyboard.press("Backspace");
  await expect(count).toHaveText("0 / 10");
});

test("Freetype shows the whole round at once; Learn shows its next word when one is typed", async ({ page }) => {
  const words = () => page.evaluate(() => [...document.querySelectorAll("#text > span:not(.beyond):not(.extra)")].map((s) => s.textContent).join("").trim().split(" ").length);
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-50" });
  expect(await words(), "all 50 words are there before the first key").toBe(50);
  await expect(page.locator("#text > span.later")).toHaveCount(0);
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "time-60" });
  expect(await words(), "a time round keeps lines ahead").toBeGreaterThanOrEqual(36);
  await expect(page.locator("#text > span.later")).toHaveCount(0);
  await openApp(page, { "tt.mode.en": "learn" });
  const later = await page.locator("#text > span.later").count();
  for (;;) { const c = await nextChar(page); await page.keyboard.type(c); if (c === " ") break; }
  expect(await page.locator("#text > span.later").count(), "a word came into view").toBeLessThan(later);
  expect(await page.locator("#text > span.later").count(), "Learn still shows 12 words ahead").toBeGreaterThan(0);
});

test("under the text: where you stand, a thin strip over the keyboard, and typing hides it", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  const below = page.locator("#stand");
  await expect(below).toContainText("Your rounds will show up here.");
  await typeRound(page);
  await expectResult(page);
  await expect(below, "the result stands alone").toBeHidden();
  await page.keyboard.press("Enter");
  await waitForText(page);
  await expect(below.locator(".stand-chart > i")).toHaveCount(1);
  await expect(below.locator(".stand-fact b")).toHaveText(/^\d+$/);
  await page.keyboard.press("Alt+k");
  await expect(page.locator("#keyboard")).toBeVisible();
  expect((await below.boundingBox()).height, "a strip over the keys").toBeLessThan(50);
  await expect(page.locator("#keyboard .key.next"), "the next key lights at once").toHaveCount(1);
  await page.waitForTimeout(350);
  await page.keyboard.press("Alt+k");
  await expect(below).toBeVisible();
  await page.keyboard.type(await nextChar(page));
  await expect(below, "typing leaves only the text").toBeHidden();
  await page.locator("#tab-learn").hover();
  await expect(page.locator("#tooltip")).toHaveClass(/\bshow\b/);
  expect((await page.locator("#tooltip").boundingBox()).height, "a tooltip under the header stays one line").toBeLessThan(50);
});
