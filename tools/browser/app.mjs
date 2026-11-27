// Shared steps of the browser checks and the latency script.

export async function openApp(page, storage = {}) {
  await page.addInitScript((s) => {
    localStorage.setItem("tt.introDone", "1");
    for (const m of ["learn", "reinforce", "freetype"]) localStorage.setItem(`tt.checkin.${m}`, "1");
    for (const [k, v] of Object.entries(s)) localStorage.setItem(k, v);
  }, storage);
  await page.goto("/");
  await waitForText(page);
}

// The round's text is there and stands still: the keyboard is fitted after
// the text arrives and moves it, so a check that measures at once reads a
// screen that is still settling.
export async function waitForText(page) {
  await page.waitForFunction(
    () => document.querySelectorAll("#text > span").length > 20,
    null, { timeout: 20_000 });
  await page.waitForFunction(() => {
    const r = document.getElementById("text").getBoundingClientRect(), now = `${r.top} ${r.left} ${r.width}`;
    const still = window.__ttStill === now;
    window.__ttStill = now;
    return still;
  }, null, { timeout: 20_000, polling: 120 });
}

export function nextChar(page) {
  return page.evaluate(() => {
    if (document.body.classList.contains("tb-resulting")) return null;
    const s = document.querySelector("#text > span:not(.ok):not(.bad):not(.miss):not(.beyond):not(.extra)");
    return s ? s.textContent : "";
  });
}

// The untyped text up to the next space, with it: a word and its space.
export function nextWord(page) {
  return page.evaluate(() => {
    if (document.body.classList.contains("tb-resulting")) return null;
    let word = "";
    for (const s of document.querySelectorAll("#text > span:not(.ok):not(.bad):not(.miss):not(.beyond):not(.extra)")) {
      word += s.textContent;
      if (s.textContent === " ") break;
    }
    return word;
  });
}

// typeRound types until the round ends (or `done`), a word at a time with
// `delay` ms between keys.
export async function typeRound(page, { delay = 40, done } = {}) {
  for (let i = 0; i < 600; i++) {
    if (done && await done()) return;
    const word = await nextWord(page);
    if (word === null) return;
    if (word === "") { await page.waitForTimeout(50); continue; }
    await page.keyboard.type(word, { delay });
    await page.waitForTimeout(delay);
  }
  throw new Error("the round did not end after 600 words");
}
