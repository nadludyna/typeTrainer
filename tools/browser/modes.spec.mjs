// Browser checks: the tabs, Welcome, the check-ins, the settings, Learn's finish screen.
import { test, expect } from "@playwright/test";
import { openApp, typeRound, waitForText, nextChar } from "./app.mjs";
import { watchPage, expectResult, withHistory, withFreetypeHistory, expectedOnly } from "./checks.mjs";

watchPage(test);

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
  // Starting over reloads the page: wait for that load, not the one before it.
  const reloaded = page.waitForEvent("load");
  await page.locator("#reset-ok").click();
  await reloaded;
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
  await typeRound(page, { delay: 40 }); // about 300 wpm: the server turns down a round over 400
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

test("Reinforce: the word level follows the rounds, and the stats say how close the next move is", async ({ page }) => {
  let saved = false;
  // The round is marked saved before its answer reaches the page, so the next /stats sees it.
  await page.route("**/api/v1/results?*", async (route) => {
    const res = await route.fetch();
    if (res.ok()) saved = true;
    return route.fulfill({ response: res });
  });
  // The server's standing: one pass at the first level, then the round just saved moved it up.
  await page.route("**/api/v1/stats?*", async (route) => {
    const res = await route.fetch(), body = await res.json();
    body.standing = saved ? { level: 2, passes: 0, fails: 0, moved: 1, passesUp: 3, failsDown: 2 }
      : { level: 1, passes: 1, fails: 0, moved: 0, passesUp: 3, failsDown: 2 };
    if (!body.lengths?.length) body.lengths = ["words-30"];
    return route.fulfill({ response: res, json: body });
  });
  await openApp(page, { "tt.mode.en": "reinforce", "tt.wordLevel.en": "1" });
  await page.keyboard.press("Backquote");
  await expect(page.locator(".st-tiles"), "how close the next level is").toContainText("1 of 3 rounds to 500");
  await page.waitForTimeout(350);
  await page.keyboard.press("Backquote");
  await typeRound(page, { done: async () => saved });
  await expect(page.locator("#toasts")).toContainText("now the 500 most common words");
  await expect.poll(() => page.evaluate(() => localStorage.getItem("tt.wordLevel.en")), { message: "the level followed" }).toBe("2");
});
