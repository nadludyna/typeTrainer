// Browser checks of the typing screen.
import { test, expect } from "@playwright/test";
import { openApp, typeRound, waitForText, nextChar } from "./app.mjs";

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

async function expectCaretInLine(page) {
  const m = await page.evaluate(() => {
    const caret = document.getElementById("caret").getBoundingClientRect();
    const box = document.getElementById("text-box").getBoundingClientRect();
    const letter = [...document.querySelectorAll("#text > span")].find((s) => s.textContent.trim())
      .getBoundingClientRect();
    return { h: caret.height, line: letter.height, top: caret.top, bottom: caret.bottom, boxTop: box.top, boxBottom: box.bottom };
  });
  expect(m.h, "caret height vs a letter").toBeLessThan(m.line * 1.3);
  expect(m.h, "caret height vs a letter").toBeGreaterThan(m.line * 0.5);
  expect(m.top, "caret top inside the text box").toBeGreaterThanOrEqual(m.boxTop - 2);
  expect(m.bottom, "caret bottom inside the text box").toBeLessThanOrEqual(m.boxBottom + 2);
}

async function expectResult(page) {
  await expect(page.locator("body")).toHaveClass(/\btb-resulting\b/);
  await expect(page.locator("#result")).toBeVisible();
  await expect(page.locator("#result")).toContainText(/\d/);
}

// Learn's /stats as if 12 rounds were typed: speeds newest first, the last 4 at 97%.
async function withHistory(page) {
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
async function withFreetypeHistory(page) {
  await page.route("**/api/v1/freetype/stats?*", async (route) => {
    const res = await route.fetch(), body = await res.json();
    body.sessions = Array.from({ length: 12 }, (_, i) => ({ wpm: 50 - i, accuracy: 96, consistency: 80 - i, length: "words-30" }));
    body.sessionCount = 12;
    if (!body.lengths?.length) body.lengths = ["words-30"];
    return route.fulfill({ response: res, json: body });
  });
}

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

const expectedOnly = (info, path) => {
  const rest = info.problems.filter((p) => !p.includes(path) && !p.startsWith("console:"));
  info.problems.splice(0, info.problems.length, ...rest);
};

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

test("Live stats is on for a new user and doesn't switch itself", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  await expect(page.locator("body")).toHaveAttribute("data-live", "on");
  await page.evaluate(() => localStorage.setItem("tt.live", "0"));
  await page.reload();
  await waitForText(page);
  await expect(page.locator("body")).toHaveAttribute("data-live", "off");
  await typeRound(page);
  await expectResult(page);
  expect(await page.evaluate(() => localStorage.getItem("tt.live")), "a finished round leaves the setting alone").toBe("0");
  await expect(page.locator("body")).toHaveAttribute("data-live", "off");
});

test("a held or hammered shortcut fires once: tab, ~, esc", async ({ page }) => {
  await openApp(page, {});
  const text = () => page.evaluate(() => document.getElementById("text").textContent);
  const held = (key, extra = {}) => page.evaluate(([key, extra]) => {
    for (let i = 0; i < 5; i++) document.activeElement.dispatchEvent(new KeyboardEvent("keydown", { key, code: key, repeat: true, bubbles: true, cancelable: true, ...extra }));
  }, [key, extra]);
  const before = await text();
  await page.keyboard.press("Tab");
  // A new round clears the text before its words arrive: wait for the words.
  const fresh = async () => { const t = await text(); return t && t !== before ? t : ""; };
  await expect.poll(fresh).not.toBe("");
  const once = await text();
  await held("Tab");
  await page.keyboard.press("Tab");
  await page.waitForTimeout(150);
  expect(await text(), "repeats and a second press within the cooldown do nothing").toBe(once);
  await page.waitForTimeout(250);
  await page.keyboard.press("Tab");
  await expect.poll(text, "a press after the cooldown works").not.toBe(once);

  await page.keyboard.press("Backquote");
  await expect(page.locator("#view-stats")).toBeVisible();
  await held("Backquote");
  await page.waitForTimeout(100);
  await expect(page.locator("#view-stats"), "still on the stats").toBeVisible();
  await page.waitForTimeout(300);
  await page.keyboard.press("Backquote");
  await expect(page.locator("#view-type")).toBeVisible();

  await page.waitForTimeout(350);
  await page.keyboard.press("Escape");
  await expect(page.locator("#settings-win")).toBeVisible();
  await held("Escape");
  await page.waitForTimeout(100);
  await expect(page.locator("#settings-win"), "a held esc doesn't close what it opened").toBeVisible();
});

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

test("the caret is still at rest, spaces have a dot unless you switch it off", async ({ page }) => {
  await openApp(page, { "tt.caret": "outline", "tt.blink": "pulse" });
  await expect(page.locator("body")).toHaveAttribute("data-caret", "line");
  await page.waitForTimeout(1500);
  const caretAnims = await page.evaluate(() => document.getElementById("caret").getAnimations().length);
  expect(caretAnims, "animations on the caret at rest").toBe(0);
  const dot = () => page.evaluate(() => getComputedStyle(document.querySelector("#text span[data-sp]")).backgroundImage);
  expect(await dot(), "a space has its dot").toContain("radial-gradient");
  await page.keyboard.press("Escape");
  await expect(page.getByText("Caret at rest")).toHaveCount(0);
  const win = page.locator("#settings-win");
  await win.locator('button[data-key="ws"][data-value="none"]').click();
  expect(await page.evaluate(() => localStorage.getItem("tt.ws"))).toBe("none");
  expect(await dot(), "no dot").toBe("none");
  const preview = win.locator(".sw-preview");
  await win.locator('#settings-chips button[data-cat="caret"]').click();
  await expect(preview, "the preview next to Caret").toBeVisible();
  await page.keyboard.press("ArrowDown");
  await expect(preview, "the preview next to Caret movement").toBeVisible();
  await win.locator('#settings-chips button[data-cat="typing"]').click();
  await expect(preview, "no preview on Stop on a mistake").toBeHidden();
});

test("settings: one window with every setting, a search and categories", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype" });
  await page.keyboard.press("Escape");
  const win = page.locator("#settings-win"), q = win.locator("#settings-q");
  await expect(q, "the search waits for typing").toBeFocused();
  const names = () => win.locator(".sw-name > span:first-child").allTextContents();
  expect(await names()).toEqual(["Space between words", "Live stats", "Caret style", "Caret movement",
    "Stop on a mistake", "Round length", "Show keyboard", "Next key", "Hand",
    "Daily goal, minutes", "Speed goal, wpm", "Progress data", "Start Learn over", "Reset progress"]);
  await expect(win.locator(".sw-row", { hasText: "Round length" }).locator(".sw-tag"), "a mode's own setting says whose").toHaveText("Freetype");

  await page.keyboard.type("cursor");
  expect(await names(), "a search finds other words for a setting too").toEqual(["Caret style", "Caret movement"]);
  await page.keyboard.press("ArrowRight");
  await expect(page.locator("body")).toHaveAttribute("data-caret", "block");
  await page.keyboard.press("Escape");
  await expect(q, "the first esc clears the search").toHaveValue("");
  await expect(win).toBeVisible();
  await page.keyboard.type("zzzz");
  await expect(win.locator(".sw-none")).toBeVisible();
  await page.keyboard.press("Escape");

  await win.locator('#settings-chips button[data-cat="goals"]').click();
  expect(await names()).toEqual(["Daily goal, minutes", "Speed goal, wpm"]);
  await win.locator('button[data-key="goal"][data-step="1"]').click();
  await expect(win.locator(".stepper output")).toHaveText("40");
  expect(await page.evaluate(() => localStorage.getItem("tt.goal"))).toBe("40");

  await win.locator('#settings-chips button[data-cat="typing"]').click();
  await win.locator('button[data-key="length"][data-value="words-10"]').click();
  await expect(page.locator('#mode-bar button[data-length="words-10"]'), "the round bar follows").toHaveAttribute("aria-pressed", "true");
  await page.keyboard.press("Escape");
  await expect(win).toBeHidden();
  await expect(page.locator("#round-count"), "a new round of the new length").toHaveText("0 / 10");
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

test("Learn at 200% zoom with round history: the line stays inside the letter row", async ({ page }) => {
  await withHistory(page);
  await page.setViewportSize({ width: 960, height: 540 });
  await openApp(page, { "tt.mode.en": "learn", "tt.goal": "15" });
  await expect(page.locator("#lp-cols .lp-last")).toBeVisible();
  const edge = await page.evaluate(() => ({
    row: document.querySelector("#letters").getBoundingClientRect().right,
    ring: document.querySelector("#round-count").getBoundingClientRect().right,
    page: document.documentElement.scrollWidth - innerWidth,
  }));
  expect(edge.ring, "the count ends at the letter row's edge").toBeLessThanOrEqual(edge.row + 0.5);
  expect(edge.page, "no sideways scroll").toBeLessThanOrEqual(0);
});

test("text can't be selected with the mouse, also in windows; only fields can", async ({ page }) => {
  await openApp(page, {});
  await page.keyboard.press("Escape");
  await expect(page.locator("#settings-win")).toBeVisible();
  const odd = await page.evaluate(() => [...document.querySelectorAll("body, dialog, [popover], dialog *")]
    .filter((e) => !e.matches("input, textarea") && getComputedStyle(e).userSelect !== "none")
    .map((e) => e.id || e.className || e.tagName).slice(0, 5));
  expect(odd, "these can be selected").toEqual([]);
  const fields = await page.evaluate(() => [...document.querySelectorAll("textarea, input[type=text], input[type=password]")]
    .filter((e) => getComputedStyle(e).userSelect === "none").length);
  expect(fields, "fields stay selectable").toBe(0);
});

test("browser zoom 200% keeps the desktop layout and nothing overlaps", async ({ browser }) => {
  const ctx = await browser.newContext({ baseURL: "http://localhost:3000", viewport: { width: 960, height: 500 }, deviceScaleFactor: 2 });
  const page = await ctx.newPage();
  const check = async (state) => {
    const m = await page.evaluate(() => {
      const nodes = [...document.querySelectorAll("#lp-cols > .lp-cell"), document.getElementById("round-count")];
      const cells = nodes.map((n) => n.getBoundingClientRect());
      return {
        sideways: document.documentElement.scrollWidth > innerWidth,
        railAtBottom: document.getElementById("rail").getBoundingClientRect().top > innerHeight / 2,
        overlaps: cells.slice(1).filter((r, i) => r.left < cells[i].right - 1).length,
        spills: nodes.filter((n) => n.scrollWidth > n.clientWidth + 1).map((n) => n.className),
      };
    });
    expect(m, state).toEqual({ sideways: false, railAtBottom: false, overlaps: 0, spills: [] });
  };
  await openApp(page, { "tt.mode.en": "learn" });
  await check("first visit");
  await ctx.close();
});

test("the keyboard shows only while Learn opens letters", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "learn", "tt.kbHidden": "1" });
  const kb = page.locator("#keyboard");
  await expect(kb).toBeVisible();
  const card = await page.locator(".keyboard > .hands").evaluate((h) => {
    const c = getComputedStyle(h);
    return { bg: c.backgroundColor, shadow: c.boxShadow };
  });
  expect(card, "no card around the hand").toEqual({ bg: "rgba(0, 0, 0, 0)", shadow: "none" });
  await page.locator(".kb-gear").click();
  const pop = page.locator("#mini-pop");
  await expect(pop.locator(".mp-name", { hasText: /^Next key$/ })).toBeVisible();
  await expect(page.locator("#keyboard .key.next")).toHaveCount(1);
  await pop.locator('button[data-key="nextKey"][data-value=""]').click();
  await expect(page.locator("#keyboard .key.next")).toHaveCount(0);
  expect(await page.evaluate(() => localStorage.getItem("tt.nextKey"))).toBe("0");
  await page.keyboard.press("Escape");
  await expect(pop).toBeHidden();
  await page.locator("#tab-freetype").click();
  await expect(kb).toBeHidden();
  await page.locator("#tab-learn").click();
  await expect(kb).toBeVisible();
  await page.request.post("/api/v1/progress?lang=en", { data: { unlockAll: true } });
  await page.reload();
  await waitForText(page);
  await expect(kb).toBeHidden();
});

test("alt+k hides and shows the keyboard, and each mode keeps its own choice", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "learn" });
  const kb = page.locator("#keyboard");
  await expect(kb, "Learn shows it by default").not.toHaveClass(/\boff\b/);
  await page.keyboard.press("Alt+k");
  await expect(kb).toHaveClass(/\boff\b/);
  expect(await page.evaluate(() => localStorage.getItem("tt.kb.learn"))).toBe("off");
  await page.reload();
  await waitForText(page);
  await expect(kb, "the choice is kept").toHaveClass(/\boff\b/);
  await openApp(page, { "tt.mode.en": "freetype" });
  await expect(kb, "Freetype hides it by default").toHaveClass(/\boff\b/);
  await page.keyboard.press("Alt+k");
  await expect(kb, "and shows it on alt+k").not.toHaveClass(/\boff\b/);
});

test("the text stands at the same place for the same keyboard state", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "learn" });
  const top = async () => Math.round((await page.locator("#text-box").boundingBox()).y);
  const learn = await top();
  let noKeyboard;
  for (const m of ["freetype", "learn"]) {
    await page.locator(`#tab-${m}`).click();
    await waitForText(page);
    await expect(page.locator("#keyboard")).toHaveClass(m === "learn" ? /^(?!.*\boff\b)/ : /\boff\b/);
    await page.waitForFunction(() => !document.getAnimations().some((a) => a.playState === "running" && a.effect?.target?.closest?.("#keyboard, #text-box, .letters")));
    if (m === "learn") expect(await top(), "the text in Learn").toBe(learn);
    else expect(await top(), `the text in ${m}`).toBe(noKeyboard ??= await top());
  }
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

test("one grid in all modes: the text stands at one place and the round's count ends at the text", async ({ page }) => {
  for (const [width, height] of [[1100, 900], [1600, 900]]) {
    await page.setViewportSize({ width, height });
    const tops = [];
    for (const mode of ["learn", "freetype"]) {
      await openApp(page, { "tt.mode.en": mode });
      const m = await page.evaluate(() => {
        const text = document.getElementById("text").getBoundingClientRect();
        return { top: text.top, textR: text.right, countR: document.getElementById("round-count").getBoundingClientRect().right };
      });
      tops.push(m.top);
      expect(m.countR, `${mode}: the count ends at the text`).toBeCloseTo(m.textR, 0);
    }
    expect(Math.max(...tops) - Math.min(...tops), `${width}: one place for the text`).toBeLessThanOrEqual(1);
  }
});

// Everything visible in the given parts that stands outside the text's left and right edges.
const textEdges = (page) => page.evaluate(() => {
  const r = document.getElementById("text").getBoundingClientRect();
  return { left: r.left, right: r.right };
});
const outsideColumn = async (page, parts, text) => page.evaluate(([parts, text]) => {
  const out = [];
  for (const part of parts) {
    for (const e of [document.querySelector(part), ...document.querySelectorAll(`${part} *`)]) {
      if (!e) continue;
      const r = e.getBoundingClientRect();
      if (r.width && r.height && getComputedStyle(e).visibility !== "hidden" && (r.left < text.left - 1 || r.right > text.right + 1)) {
        out.push(`${e.tagName.toLowerCase()}.${e.getAttribute("class") ?? ""} ${Math.round(r.left)}-${Math.round(r.right)} (text ${Math.round(text.left)}-${Math.round(text.right)})`);
      }
    }
  }
  return out;
}, [parts, text ?? await textEdges(page)]);

for (const width of [1280, 1600]) {
  test(`one column at ${width}: the letters, the line above the text, the keyboard and its gear, the result stand between the text edges`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    for (const mode of ["learn", "freetype"]) {
      await openApp(page, { "tt.mode.en": mode });
      expect(await outsideColumn(page, ["#view-type > .letters", "#keyboard"]), mode).toEqual([]);
    }
    await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
    const text = await textEdges(page);
    await typeRound(page);
    await expectResult(page);
    await page.waitForTimeout(800);
    expect(await outsideColumn(page, ["#result"], text), "result").toEqual([]);
  });
}

test("stats: the title, the tiles and the panels share one left edge", async ({ page }) => {
  await withHistory(page);
  await openApp(page, { "tt.mode.en": "learn" });
  await page.keyboard.press("Backquote");
  await expect(page.locator(".st-chart svg")).toBeVisible();
  const lefts = await page.evaluate(() => [".st-head h1", ".st-tiles", ".st-two", ".activity"]
    .map((q) => Math.round(document.querySelector(q).getBoundingClientRect().left)));
  expect(new Set(lefts).size, `left edges ${lefts}`).toBe(1);
});

for (const [w, h] of [[1920, 1080], [1366, 768]]) {
  test(`Freetype at ${w}: the switches stand on one line at the text edges, and the count shows the round`, async ({ page }) => {
    await page.setViewportSize({ width: w, height: h });
    await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
    const m = () => page.evaluate(() => {
      const text = document.getElementById("text").getBoundingClientRect();
      const sides = [...document.querySelectorAll("#mode-bar > .bar-side")].map((e) => e.getBoundingClientRect());
      return { textL: text.left, textR: text.right, sides, done: document.getElementById("round-count").textContent };
    });
    const at = await m();
    expect(at.sides[0].left, "the left switches start at the text").toBeCloseTo(at.textL, 0);
    expect(at.sides[1].right, "the right switches end at the text").toBeCloseTo(at.textR, 0);
    expect(at.sides[0].top, "one line").toBe(at.sides[1].top);
    expect(at.done).toBe("0 / 10");
    for (let i = 0; i < 15; i++) { const c = await nextChar(page); if (!c) break; await page.keyboard.type(c); }
    await page.waitForTimeout(400);
    expect((await m()).done, "the count moves while typing").not.toBe("0 / 10");
  });
}

test("the mouse pointer hides while you type and comes back when the mouse moves", async ({ page }) => {
  await openApp(page, {});
  const box = await page.locator(".logo").boundingBox();
  const x = box.x + box.width / 2, y = box.y + box.height / 2;
  const under = () => page.evaluate(([x, y]) => {
    const e = document.elementFromPoint(x, y);
    return getComputedStyle(e, e === document.documentElement ? "::after" : null).cursor;
  }, [x, y]);
  await page.mouse.move(x, y);
  expect(await under()).toBe("pointer");
  await page.keyboard.type(await nextChar(page));
  expect(await under(), "hidden while typing").toBe("none");
  await page.mouse.move(x + 2, y + 1);
  expect(await under(), "back after a move").toBe("pointer");
});

test("the round's count follows each word and goes back when a word is undone", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  const count = page.locator("#round-count");
  for (;;) { const c = await nextChar(page); await page.keyboard.type(c); if (c === " ") break; }
  await expect(count).toHaveText("1 / 10");
  await page.keyboard.press("Backspace");
  await expect(count).toHaveText("0 / 10");
});

test("no text on the typing, result, settings and stats screens is smaller than 14px", async ({ page }) => {
  await page.setViewportSize({ width: 1366, height: 768 });
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-10" });
  const small = () => page.evaluate(() => {
    const out = new Set();
    const w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    for (let n; (n = w.nextNode());) {
      const el = n.parentElement;
      if (!n.textContent.trim() || !el.checkVisibility({ visibilityProperty: true }) || el.closest("#keyboard, .logo")) continue;
      const fs = parseFloat(getComputedStyle(el).fontSize);
      if (fs < 13.95) out.add(`${el.tagName.toLowerCase()}.${el.className} ${fs}px "${n.textContent.trim().slice(0, 20)}"`);
    }
    return [...out];
  });
  expect(await small(), "typing").toEqual([]);
  await typeRound(page);
  await expectResult(page);
  expect(await small(), "result").toEqual([]);
  await page.locator("#tab-stats").click();
  await expect(page.locator(".st-tiles > *").first()).toBeVisible();
  expect(await small(), "stats").toEqual([]);
  await openApp(page, { "tt.mode.en": "freetype" });
  await page.keyboard.press("Escape");
  await expect(page.locator("dialog[open]").first()).toBeVisible();
  expect(await small(), "settings").toEqual([]);
});

test("the hand over the keyboard shows the finger of the next key", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "learn" });
  const fingerOf = await page.evaluate(async () => {
    const layout = await (await fetch("/api/v1/layout?lang=en")).json();
    return Object.fromEntries(layout.rows.flat().map((k) => [k.char, k.finger]));
  });
  const side = (id) => (id === "thl" || id[0] === "l" ? "l" : "r");
  for (let i = 0; i < 25; i++) {
    const ch = await nextChar(page);
    if (!ch) break;
    const ids = fingerOf[ch] === "th" ? ["thl", "thr"] : [fingerOf[ch]];
    const sides = [...new Set(ids.map(side))];
    await expect(page.locator("#hands")).toHaveAttribute("data-show", sides.length === 2 ? "lr" : sides[0]);
    for (const s of sides) {
      await expect(page.locator(`.keyboard .hand.${s === "l" ? "left" : "right"} > img`))
        .toHaveAttribute("data-f", ids.filter((id) => side(id) === s).join(" "));
    }
    await page.keyboard.type(ch);
  }
});

test("a space that ends a line has no width, so its dot can't hang at the edge", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-50" });
  const ends = await page.evaluate(() => {
    const spans = [...document.querySelectorAll("#text > span")];
    return spans.flatMap((s, i) => s.hasAttribute("data-sp") && i > 0 && spans[i + 1] && spans[i + 1].offsetTop > spans[i - 1].offsetTop
      ? [s.getBoundingClientRect().width] : []);
  });
  expect(ends.length, "the text wraps at least once").toBeGreaterThan(0);
  for (const w of ends) expect(w).toBe(0);
});

test("the caret stays one line tall with space marks across line wraps", async ({ page }) => {
  await openApp(page, { "tt.mode.en": "freetype", "tt.length.freetype": "words-30" });
  await typeRound(page, { afterKey: () => expectCaretInLine(page) });
  await expectResult(page);
});

test("gold text reads on the page: 4.5:1, big numbers 3:1; an old light-theme choice is ignored", async ({ page }) => {
  await openApp(page, { "tt.theme": "light" });
  await expect(page.locator("html")).not.toHaveAttribute("data-theme", /./);
  const ratio = await page.evaluate(() => {
    const rgb = (v) => {
      const probe = document.createElement("i");
      probe.style.color = `var(${v})`;
      document.body.append(probe);
      const c = getComputedStyle(probe).color.match(/\d+(\.\d+)?/g).slice(0, 3).map(Number);
      probe.remove();
      return c;
    };
    const lum = (c) => c.map((x) => x / 255).map((x) => x <= .03928 ? x / 12.92 : ((x + .055) / 1.055) ** 2.4)
      .reduce((s, x, i) => s + x * [.2126, .7152, .0722][i], 0);
    const on = (v) => {
      const [a, b] = [lum(rgb(v)), lum(rgb("--bg"))].sort((x, y) => y - x);
      return (a + .05) / (b + .05);
    };
    return { text: on("--accent-text"), big: on("--accent-big") };
  });
  expect(ratio.text, "--accent-text on --bg").toBeGreaterThanOrEqual(4.5);
  expect(ratio.big, "--accent-big on --bg").toBeGreaterThanOrEqual(3);
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
  await expect(page.locator(".mascot"), "one click doesn't wake the mascot").toHaveCount(0);
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

test("Learn: a keyboard that can't fit leaves one line in its place", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 480 });
  await openApp(page, { "tt.mode.en": "learn" });
  await expect(page.locator("body")).toHaveClass(/\bkb-nofit\b/);
  await expect(page.locator("#keyboard")).toBeHidden();
  await expect(page.locator("#kb-away")).toHaveText("Zoom out to see the keyboard.");
  await page.setViewportSize({ width: 1280, height: 900 });
  await expect(page.locator("#keyboard")).toBeVisible();
  await expect(page.locator("#kb-away")).toBeHidden();
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

test("every mode has its tab: Learn, Reinforce and Freetype", async ({ page }) => {
  await openApp(page);
  for (const id of ["tab-welcome", "tab-learn", "tab-reinforce", "tab-freetype"]) await expect(page.locator("#" + id)).toBeVisible();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "learn");
  await expect(page.locator("#mode-bar"), "Learn has no round bar").toBeHidden();
  await page.locator("#tab-reinforce").click();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "reinforce");
  await waitForText(page);
  await expect(page.locator("#mode-bar")).toContainText("Reinforce");
  await expect(page.locator("#mode-bar .re-note"), "Reinforce says Learn is not finished").toContainText("letters to open");
  await page.locator("#tab-freetype").click();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "freetype");
  await waitForText(page);
  await page.reload();
  await waitForText(page);
  await expect(page.locator("body"), "the mode is remembered").toHaveAttribute("data-mode", "freetype");
});
test("Start Learn over in the settings: Learn's rounds go, the first lesson comes back", async ({ page }) => {
  await openApp(page);
  const keys = Object.fromEntries([..."abcdefghijklmnopqrstuvwxyz"].map((c) => [c, { count: 50, errEma: 0.05, timeEma: 250, timedCount: 50 }]));
  await page.evaluate((k) => fetch("/api/v1/import?lang=en", { method: "POST", body: JSON.stringify({ unlocked: 26, keys: k }) }), keys);
  await page.reload();
  await waitForText(page);
  await page.keyboard.press("Escape");
  await page.locator("#settings-q").fill("start over");
  await page.locator("#btn-relearn").click();
  await expect(page.locator("#reset-title")).toHaveText("Start Learn over?");
  await page.locator("#reset-ok").click();
  await page.waitForLoadState("load");
  await waitForText(page);
  const s = await page.evaluate(() => fetch("/api/v1/stats?lang=en").then((r) => r.json()));
  expect(s.keys.filter((k) => k.unlocked).length).toBe(6);
});

test("Reinforce: word levels and speed goal steps on its own tab", async ({ page }) => {
  await openApp(page);
  const keys = Object.fromEntries([..."abcdefghijklmnopqrstuvwxyz"].map((c) => [c, { count: 50, errEma: 0.05, timeEma: 250, timedCount: 50 }]));
  await page.evaluate((k) => fetch("/api/v1/import?lang=en", { method: "POST", body: JSON.stringify({ unlocked: 26, keys: k }) }), keys);
  await page.reload();
  await waitForText(page);
  await expect(page.locator("#mode-bar"), "Learn keeps no Reinforce bar, even with every letter open").toBeHidden();
  await page.locator("#tab-reinforce").click();
  await expect(page.locator("#mode-bar")).toContainText("Reinforce");
  await expect(page.locator("#mode-bar .re-note"), "Learn is finished: no line about it").toHaveCount(0);
  await page.locator('#mode-bar button[data-wlevel="1"]').waitFor();
  const top = await page.evaluate(() => fetch("/api/v1/stats?lang=en").then((r) => r.json()).then((s) => s.wordLevels));
  expect(top[0]).toBe(200);
  await page.locator('#mode-bar button[data-wlevel="10"]').click();
  await expect(page.locator('#mode-bar button[data-wlevel="10"]')).toHaveAttribute("aria-pressed", "true");
  await page.locator('#mode-bar button[data-step="1"]').click();
  await expect(page.locator("#mode-bar output")).toHaveText("40 wpm");
  const asked = page.waitForRequest((r) => r.url().includes("/api/v1/portion") && r.postDataJSON()?.portionIndex === 1);
  await page.locator("#text-box").click();
  for (let i = 0; i < 80 && !(await page.evaluate(() => document.querySelectorAll("#text > span.ok").length > 60)); i++) {
    const ch = await nextChar(page);
    if (ch) await page.keyboard.type(ch);
  }
  expect((await asked).postDataJSON().level, "the next words come from the chosen level").toBe(10);
});

test("Learn: the round that opens the last letter ends on the finish screen; Enter goes to Reinforce", async ({ page }) => {
  await openApp(page);
  const order = await page.evaluate(() => fetch("/api/v1/stats?lang=en").then((r) => r.json()).then((s) => s.order));
  const keys = Object.fromEntries(order.map((c) => [c, { count: 200, errEma: 0.01, timeEma: 200, timedCount: 200, hits: "1".repeat(100) }]));
  await page.evaluate((k) => fetch("/api/v1/import?lang=en", { method: "POST", body: JSON.stringify({ unlocked: k.n, keys: k.keys }) }), { n: order.length - 1, keys });
  await page.reload();
  await waitForText(page);
  await typeRound(page, { delay: 30 });
  const fin = page.locator("#result .fin");
  await expect(fin).toBeVisible();
  await expect(fin.locator("h2")).toHaveText("Every letter is open!");
  await expect(fin).toContainText("26 letters");
  await page.keyboard.press("Enter");
  await expect(page.locator("body")).toHaveAttribute("data-mode", "reinforce");
  await waitForText(page);
  await expect(page.locator("#mode-bar")).toContainText("Reinforce");
});
test("Welcome: a first visit opens it; the home row, the finger tour, then Learn; the tab brings it back", async ({ page }) => {
  await openApp(page, { "tt.introDone": "" });
  const step = () => page.locator("#view-welcome .intro-step.active").getAttribute("data-step");
  await expect(page.locator("body")).toHaveAttribute("data-view", "welcome");
  await expect(page.locator("#tab-welcome")).toHaveAttribute("aria-selected", "true");
  await expect(page.locator("#view-type")).toBeHidden();
  await page.keyboard.press("x");
  expect(await step(), "any key leaves the first step").toBe("1");
  for (const k of ["a", "s", "d", "f", "j", "k", "l", ";"]) await page.keyboard.press(k);
  expect(await step(), "the home row alone is not enough").toBe("1");
  await page.keyboard.press("Space");
  await expect.poll(step).toBe("2");
  await expect(page.locator("#tour-of")).toHaveText("Finger 1 of 8 · left pinky");
  await expect(page.locator("#tour-map .key.next .lbl")).toHaveText("q");
  await page.keyboard.press("w");
  await expect(page.locator("#tour-map .key.next .lbl"), "a wrong key changes nothing").toHaveText("q");
  for (let i = 0; i < 300 && await step() === "2"; i++) {
    const next = page.locator("#tour-map .key.next .lbl");
    if (await next.count()) await page.keyboard.press(await next.textContent());
    else await page.waitForTimeout(100);
  }
  expect(await step(), "eight fingers lead to the start").toBe("3");
  await expect(page.locator("#wl-chips .wl-chip.done")).toHaveCount(3);
  await page.keyboard.press("Enter");
  await expect(page.locator("body")).toHaveAttribute("data-view", "type");
  await expect(page.locator("body")).toHaveAttribute("data-mode", "learn");
  await waitForText(page);
  expect(await page.evaluate(() => localStorage.getItem("tt.introDone"))).toBe("1");
  await page.locator("#tab-welcome").click();
  await expect(page.locator("body")).toHaveAttribute("data-view", "welcome");
  expect(await step(), "Welcome opens at its start again").toBe("0");
  await page.keyboard.press("ArrowRight");
  await page.keyboard.press("ArrowRight");
  expect(await step(), "the arrows walk the steps").toBe("2");
  await page.locator("#wl-start").click();
  await expect(page.locator("body")).toHaveAttribute("data-view", "type");
});

test("Welcome: a returning visitor goes straight to Learn, and nothing typed in Welcome is sent", async ({ page }) => {
  await openApp(page);
  await expect(page.locator("body")).toHaveAttribute("data-view", "type");
  const sent = [];
  page.on("request", (r) => { if (/\/api\/v1\/(portion|results|abort)/.test(r.url())) sent.push(r.url()); });
  await page.locator("#tab-welcome").click();
  for (const k of ["x", "a", "s", "d", "f", "Space"]) await page.keyboard.press(k);
  await page.waitForTimeout(300);
  expect(sent, "Welcome sends no presses").toEqual([]);
});

test("check-in, Learn: the first visit asks once; \"A little\" opens ten letters", async ({ page }) => {
  await openApp(page, { "tt.checkin.learn": "" });
  await page.locator("#tab-learn").click();
  const card = page.locator("#result .fin");
  await expect(card.locator("h2")).toHaveText("Have you learned touch typing before?");
  await card.locator('[data-act="little"]').click();
  await expect(card).toBeHidden();
  await waitForText(page);
  const s = await page.evaluate(() => fetch("/api/v1/stats?lang=en").then((r) => r.json()));
  expect(s.keys.filter((k) => k.unlocked).length).toBe(10);
  expect(await page.evaluate(() => localStorage.getItem("tt.checkin.learn"))).toBe("1");
  await page.locator("#tab-learn").click();
  await expect(page.locator("#result .fin"), "asked once").toBeHidden();
});

test("check-in, Reinforce: a 30-second sample sends no presses and sets the goal and the word level", async ({ page }) => {
  await openApp(page, { "tt.checkin.reinforce": "" });
  await page.locator("#tab-reinforce").click();
  const card = page.locator("#result .fin");
  await expect(card.locator("h2")).toHaveText("Let's find where you start");
  const sent = [];
  page.on("request", (r) => { if (/\/api\/v1\/(portion|results|abort)/.test(r.url())) sent.push(r.url()); });
  await page.keyboard.press("Enter");
  await waitForText(page);
  await typeRound(page, { delay: 60, done: () => page.locator("#result .fin .ci-pair").isVisible() });
  await expect(card.locator(".ci-pair")).toBeVisible({ timeout: 40_000 });
  expect(sent, "the sample sends nothing to the letter model").toEqual([]);
  await expect(card).toContainText("Your goal starts at");
  const goal = Number((await card.locator("p b").first().textContent()).match(/\d+/)[0]);
  await page.keyboard.press("Enter");
  await expect(page.locator("body")).toHaveAttribute("data-mode", "reinforce");
  await waitForText(page);
  const s = await page.evaluate(() => fetch("/api/v1/stats?lang=en").then((r) => r.json()));
  expect(s.targetWpm, "the goal the card named is kept").toBe(goal);
  expect(await page.evaluate(() => localStorage.getItem("tt.checkin.reinforce"))).toBe("1");
});

test("check-in, Freetype: Skip starts the round at once; Welcome brings the check-ins back", async ({ page }) => {
  await openApp(page, { "tt.checkin.freetype": "" });
  await page.locator("#tab-freetype").click();
  await page.locator('#result .fin [data-act="skip"]').click();
  await expect(page.locator("#result .fin")).toBeHidden();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "freetype");
  await waitForText(page);
  expect(await page.evaluate(() => localStorage.getItem("tt.checkin.freetype"))).toBe("1");
  await page.locator("#tab-welcome").click();
  for (let i = 0; i < 3; i++) await page.keyboard.press("ArrowRight");
  await page.locator("#wl-checkins").click();
  expect(await page.evaluate(() => ["learn", "reinforce", "freetype"].map((m) => localStorage.getItem(`tt.checkin.${m}`)))).toEqual(["", "", ""]);
});
