// The latency script: node perf.mjs [--mode freetype] [--runs 5] [--slow 4]
// [--bare] [--against main].
// Without --against it measures the server already running on :3000.
// --against <git ref> builds that ref and this code, starts each in turn on
// :3000 (which must be free), alternates them for --rounds 2 rounds and
// prints both side by side (docs/DEVELOPMENT.md).
import { chromium } from "@playwright/test";
import { spawn, execFileSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { openApp, waitForText } from "./app.mjs";

const args = Object.fromEntries(process.argv.slice(2).join(" ").split("--").filter(Boolean)
  .map((a) => a.trim().split(/\s+/)).map(([k, v]) => [k, v ?? "1"]));
const opt = {
  url: args.url ?? "http://localhost:3000",
  mode: args.mode ?? "learn",
  slow: Number(args.slow ?? 6),
  delay: Number(args.delay ?? 80),
  keys: Number(args.keys ?? 50),
  runs: Number(args.runs ?? 3),
  bare: "bare" in args,
  against: args.against,
  rounds: Number(args.rounds ?? 2),
};

function setupProbes() {
  const W = window.__perf = { keys: [], moves: [], frames: [] };
  addEventListener("keydown", (e) => W.keys.push(e.timeStamp), true);
  new MutationObserver(() => W.moves.push(performance.now()))
    .observe(document.getElementById("caret"), { attributes: true, attributeFilter: ["style", "class"] });
  let last = 0;
  const frame = (t) => { if (last) W.frames.push(t - last); last = t; requestAnimationFrame(frame); };
  requestAnimationFrame(frame);
}

const median = (a) => { const s = [...a].sort((x, y) => x - y); return s.length ? s[s.length >> 1] : NaN; };

async function oneRun(browser, cpu) {
  const ctx = await browser.newContext({ baseURL: opt.url, viewport: { width: 1920, height: 1080 } });
  const page = await ctx.newPage();
  const bare = opt.bare ? { [`tt.kb.${opt.mode}`]: "off" } : {};
  await openApp(page, { "tt.mode.en": opt.mode, "tt.length.learn": "words-30", [`tt.length.${opt.mode}`]: "words-30", ...bare });
  if (opt.bare && opt.mode === "learn") {
    await page.request.post("/api/v1/progress?lang=en", { data: { unlockAll: true } });
    await page.reload();
    await waitForText(page);
  }
  await page.waitForFunction((n) => document.querySelectorAll("#text > span").length > n, opt.keys + 1, { timeout: 20_000 });
  await page.waitForTimeout(800);
  const text = await page.evaluate(() => [...document.querySelectorAll("#text > span")].map((s) => s.textContent).join(""));

  const cdp = await ctx.newCDPSession(page);
  await cdp.send("Performance.enable");
  await cdp.send("Emulation.setCPUThrottlingRate", { rate: cpu });
  await page.evaluate(setupProbes);
  const recalc = async () => (await cdp.send("Performance.getMetrics")).metrics
    .find((m) => m.name === "RecalcStyleDuration").value;
  const before = await recalc();
  for (const ch of text.slice(0, opt.keys)) {
    await page.keyboard.type(ch);
    await page.waitForTimeout(opt.delay);
  }
  await page.waitForTimeout(400);
  const after = await recalc();
  const W = await page.evaluate(() => window.__perf);
  await ctx.close();

  const lags = [];
  let j = 0;
  for (const k of W.keys) {
    while (j < W.moves.length && W.moves[j] < k) j++;
    if (j < W.moves.length) lags.push(W.moves[j] - k);
  }
  return {
    recalc: ((after - before) * 1000) / opt.keys,
    caret: median(lags),
    slow: W.frames.filter((f) => f > 25).length / W.frames.length,
  };
}

// TT_BROWSER=msedge measures in the installed Edge, as the browser checks do.
const browser = await chromium.launch({ channel: process.env.TT_BROWSER || undefined });
async function measure() {
  const fast = [], slow = [];
  for (let i = 0; i < opt.runs; i++) fast.push(await oneRun(browser, 1));
  for (let i = 0; i < opt.runs; i++) slow.push(await oneRun(browser, opt.slow));
  return { fast, slow };
}
const numbers = ({ fast, slow }) => ({
  recalc: median(fast.map((r) => r.recalc)),
  caret: median(slow.map((r) => r.caret)),
  slow: median(slow.map((r) => r.slow)) * 100,
});
const fmt = (x, d = 1) => x.toFixed(d);
const title = `mode ${opt.mode}${opt.bare ? " (bare: no background, no keyboard)" : ""}, ${opt.keys} keys every ${opt.delay} ms`;

if (!opt.against) {
  const n = numbers(await measure());
  console.log(`${title}, median of ${opt.runs} runs`);
  console.log(`  style recalc per key   ${fmt(n.recalc)} ms   (full speed)`);
  console.log(`  key -> caret start     ${fmt(n.caret, 0)} ms    (CPU ${opt.slow}x slower)`);
  console.log(`  slow frames (>25 ms)   ${fmt(n.slow, 0)}%    (CPU ${opt.slow}x slower)`);
} else {
  if (await serving()) {
    console.error("perf: something already serves :3000. Stop it (and any other test server) and run again: a busy machine reads slow.");
    process.exit(1);
  }
  const repo = resolve(dirname(fileURLToPath(import.meta.url)), "../..");   // from any folder it is run in
  const dir = mkdtempSync(join(tmpdir(), "tt-perf-"));
  const exe = process.platform === "win32" ? ".exe" : "";
  const base = { name: opt.against, tree: join(dir, "base"), bin: join(dir, "base" + exe), runs: { fast: [], slow: [] } };
  const here = { name: "this code", tree: repo, bin: join(dir, "here" + exe), runs: { fast: [], slow: [] } };
  try {
    execFileSync("git", ["worktree", "add", "--detach", base.tree, opt.against], { cwd: repo, stdio: "ignore" });
    for (const side of [base, here]) execFileSync("go", ["build", "-o", side.bin, "."], { cwd: side.tree, stdio: "inherit" });
    for (let r = 0; r < opt.rounds; r++) {
      for (const side of r % 2 ? [here, base] : [base, here]) {
        const server = spawn(side.bin, [], { cwd: side.tree, stdio: "ignore" });
        try {
          await until(serving);
          const m = await measure();
          side.runs.fast.push(...m.fast);
          side.runs.slow.push(...m.slow);
          console.log(`round ${r + 1}: ${side.name} measured`);
        } finally {
          server.kill();
          await until(async () => !(await serving()));
        }
      }
    }
  } finally {
    await browser.close();
    try { execFileSync("git", ["worktree", "remove", "--force", base.tree], { cwd: repo, stdio: "ignore" }); } catch {}
    rmSync(dir, { recursive: true, force: true });
  }
  const a = numbers(base.runs), b = numbers(here.runs);
  const row = (name, x, y, d, unit) => console.log(`  ${name.padEnd(22)} ${(fmt(x, d) + unit).padStart(9)} ${(fmt(y, d) + unit).padStart(11)} ${((y >= x ? "+" : "") + fmt(y - x, d) + unit).padStart(9)}`);
  console.log(`\n${title}, median of ${opt.runs * opt.rounds} runs each, ${opt.rounds} rounds taking turns`);
  console.log(`  ${"".padEnd(22)} ${opt.against.padStart(9)} ${"this code".padStart(11)} ${"change".padStart(9)}`);
  row("style recalc per key", a.recalc, b.recalc, 1, " ms");
  row(`key -> caret (CPU ${opt.slow}x)`, a.caret, b.caret, 0, " ms");
  row(`slow frames (CPU ${opt.slow}x)`, a.slow, b.slow, 0, "%");
  process.exit(0);
}
await browser.close();

async function serving() {
  try { return (await fetch(opt.url + "/api/v1/languages")).ok; } catch { return false; }
}
async function until(ok) {
  for (let i = 0; i < 300; i++) {
    if (await ok()) return;
    await new Promise((r) => setTimeout(r, 200));
  }
  throw new Error("perf: the server did not start or stop in a minute");
}
