// Browser checks: where things stand: the grid, the widths, zoom, text sizes and contrast.
import { test, expect } from "@playwright/test";
import { openApp, typeRound, waitForText, nextChar } from "./app.mjs";
import { watchPage, expectResult, withHistory, withFreetypeHistory, expectedOnly } from "./checks.mjs";

watchPage(test);

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

// A phone's browser widens the whole page to its widest element, and the
// bottom navigation then slides off the screen: only a real phone emulation
// (isMobile) shows it, a narrow window does not.
test.describe("on a phone", () => {
  test.use({ viewport: { width: 360, height: 760 }, hasTouch: true, isMobile: true });
  test("every screen fits: the page keeps the screen's width, the navigation stays at the bottom, the switches stay above the text", async ({ page }) => {
    await withHistory(page);
    await withFreetypeHistory(page);
    const fits = (where) => page.evaluate((where) => {
      const out = [];
      if (innerWidth !== 360) out.push(`${where}: the page is ${innerWidth}px wide`);
      const rail = document.getElementById("rail").getBoundingClientRect();
      if (rail.bottom > innerHeight + 0.5 || rail.right > 360.5) out.push(`${where}: the navigation is off the screen`);
      for (const n of document.querySelectorAll(".top *, #rail *, .view.active *")) {
        const r = n.getBoundingClientRect();
        if (r.width && r.right > 360.5) out.push(`${where}: ${n.tagName.toLowerCase()}.${n.getAttribute("class")} ends at ${Math.round(r.right)}`);
      }
      const band = document.querySelector("#view-type > .letters")?.getBoundingClientRect(), box = document.getElementById("text-box").getBoundingClientRect();
      if (where !== "stats" && where !== "welcome" && band.bottom > box.top + 0.5) out.push(`${where}: the switches run into the text`);
      return out.slice(0, 6);
    }, where);
    for (const mode of ["learn", "reinforce", "freetype"]) {
      await openApp(page, { "tt.mode.en": mode });
      expect(await fits(mode)).toEqual([]);
    }
    await page.locator("#tab-stats").click();
    await expect(page.locator(".st-tiles .tile.hero")).toBeVisible();
    expect(await fits("stats")).toEqual([]);
    await page.locator("#tab-welcome").click();
    await expect(page.locator("#view-welcome")).toBeVisible();
    const steps = page.locator("#view-welcome .wl-chip");
    for (let i = 0; i < await steps.count(); i++) {
      await steps.nth(i).click();
      await page.waitForTimeout(350);
      expect(await fits("welcome"), `Welcome, step ${i + 1}`).toEqual([]);
    }
    await page.locator("#btn-settings").click();
    await expect(page.locator("#settings-win")).toBeVisible();
    const win = await page.evaluate(() => { const r = document.getElementById("settings-win").getBoundingClientRect(); return [innerWidth, Math.round(r.left), Math.round(r.right)]; });
    expect(win[0], "settings: the page keeps the screen's width").toBe(360);
    expect(win[1] >= 0 && win[2] <= 360, `settings window ${win[1]}..${win[2]}`).toBe(true);
  });
});
