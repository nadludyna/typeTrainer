// Shared steps of the browser checks and the latency script.

// The placement test is skipped unless `test` is set, so Learn opens on its
// usual words. A 404 means a server without the test.
export async function openApp(page, storage = {}, { test = false } = {}) {
  if (!test) {
    const res = await page.request.post("/api/v1/placement?lang=en", { data: { skip: true } });
    if (!res.ok() && res.status() !== 409 && res.status() !== 404) throw new Error("placement skip: " + res.status());
  }
  await page.addInitScript((s) => {
    localStorage.setItem("tt.introDone", "1");
    localStorage.setItem("tt.allModes", "1");
    for (const [k, v] of Object.entries(s)) localStorage.setItem(k, v);
  }, storage);
  await page.goto("/");
  await waitForText(page);
}

export async function waitForText(page) {
  await page.waitForFunction(
    () => document.querySelectorAll("#text > span").length > 20,
    null, { timeout: 20_000 });
}

export function nextChar(page) {
  return page.evaluate(() => {
    if (document.body.classList.contains("tb-resulting")) return null;
    const s = document.querySelector("#text > span:not(.ok):not(.bad):not(.miss):not(.beyond):not(.extra)");
    return s ? s.textContent : "";
  });
}

export async function typeRound(page, { delay = 70, afterKey, done } = {}) {
  for (let i = 0; i < 600; i++) {
    if (done && await done()) return;
    const ch = await nextChar(page);
    if (ch === null) return;
    if (ch === "") { await page.waitForTimeout(50); continue; }
    await page.keyboard.type(ch);
    if (afterKey) await afterKey();
    await page.waitForTimeout(delay);
  }
  throw new Error("the round did not end after 600 steps");
}
