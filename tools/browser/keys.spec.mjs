// Browser checks: keys, the caret and the keyboard while typing.
import { test, expect } from "@playwright/test";
import { openApp, typeRound, waitForText, nextChar } from "./app.mjs";
import { watchPage, expectResult, withHistory, withFreetypeHistory, expectedOnly } from "./checks.mjs";

watchPage(test);

// The caret's place and the next key in one look at the page (null once the
// round has ended and the result covers the text).
function caretAndNext(page) {
  return page.evaluate(() => {
    if (document.body.classList.contains("tb-resulting")) return null;
    const caret = document.getElementById("caret").getBoundingClientRect();
    const box = document.getElementById("text-box").getBoundingClientRect();
    const letter = [...document.querySelectorAll("#text > span")].find((s) => s.textContent.trim())
      .getBoundingClientRect();
    const next = document.querySelector("#text > span:not(.ok):not(.bad):not(.miss):not(.beyond):not(.extra)");
    return { next: next ? next.textContent : "", h: caret.height, line: letter.height, top: caret.top, bottom: caret.bottom, boxTop: box.top, boxBottom: box.bottom };
  });
}

function expectCaretInLine(m) {
  expect(m.h, "caret height vs a letter").toBeLessThan(m.line * 1.3);
  expect(m.h, "caret height vs a letter").toBeGreaterThan(m.line * 0.5);
  expect(m.top, "caret top inside the text box").toBeGreaterThanOrEqual(m.boxTop - 2);
  expect(m.bottom, "caret bottom inside the text box").toBeLessThanOrEqual(m.boxBottom + 2);
}

test("a held or hammered shortcut fires once: tab, ~, esc", async ({ page }) => {
  await openApp(page, {});
  const held = (key, extra = {}) => page.evaluate(([key, extra]) => {
    for (let i = 0; i < 5; i++) document.activeElement.dispatchEvent(new KeyboardEvent("keydown", { key, code: key, repeat: true, bubbles: true, cancelable: true, ...extra }));
  }, [key, extra]);
  // A new round asks the server for its first words: count those requests.
  const starts = [];
  page.on("request", (r) => { if (r.url().includes("/api/v1/portion") && r.postDataJSON()?.start) starts.push(r.url()); });
  // Two presses inside the page in one go are surely within the cooldown,
  // however slow the machine is; a held key only sends repeats.
  await page.evaluate(() => {
    for (let i = 0; i < 2; i++) document.activeElement.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", code: "Tab", bubbles: true, cancelable: true }));
  });
  await held("Tab");
  await expect.poll(() => starts.length).toBe(1);
  await waitForText(page);
  await page.waitForTimeout(400);
  expect(starts.length, "repeats and a second press within the cooldown do nothing").toBe(1);
  await page.keyboard.press("Tab");
  await expect.poll(() => starts.length, "a press after the cooldown works").toBe(2);

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
  for (let i = 0; i < 1500; i++) {
    const m = await caretAndNext(page);
    if (!m) break;
    expectCaretInLine(m);
    // 40 ms a key is about 300 wpm: faster, and the server turns the round
    // down as impossible (over 400 wpm), on a fast machine it would be.
    if (m.next) await page.keyboard.type(m.next);
    await page.waitForTimeout(m.next ? 40 : 50);
  }
  await expectResult(page);
});
