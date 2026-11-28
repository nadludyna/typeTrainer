// Typing latency: node e2e/perf.ts [--mode freetype] [--runs 3] [--slow 6]
// [--bare] [--against main], from web/. Three numbers: style recalc per key
// at full speed, key → caret start and the share of slow frames with the CPU
// slowed down. Without --against it measures the server on :3000; with it,
// it builds that git ref and this code, runs each on :3000 in turn (which
// must be free), alternating for --rounds rounds, and prints both side by
// side: the slow frames move a lot between runs, so only numbers measured
// side by side compare.

import { type ChildProcess, execFileSync, spawn } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { type Browser, chromium } from "@playwright/test";

const args: Record<string, string> = Object.fromEntries(
  process.argv
    .slice(2)
    .join(" ")
    .split("--")
    .filter(Boolean)
    .map((a) => a.trim().split(/\s+/))
    .map(([k = "", v]) => [k, v ?? "1"]),
);
const opt = {
  url: "http://localhost:3000",
  mode: args.mode ?? "learn",
  slow: Number(args.slow ?? 6),
  delay: Number(args.delay ?? 80),
  keys: Number(args.keys ?? 50),
  runs: Number(args.runs ?? 3),
  bare: "bare" in args,
  against: args.against,
  rounds: Number(args.rounds ?? 2),
};

interface Probe {
  keys: number[];
  moves: number[];
  frames: number[];
}

function setupProbes() {
  const W: Probe = { keys: [], moves: [], frames: [] };
  (window as unknown as { __perf: Probe }).__perf = W;
  addEventListener("keydown", (e) => W.keys.push(e.timeStamp), true);
  const caret = document.querySelector("[data-text] > span[aria-hidden]");
  if (caret) new MutationObserver(() => W.moves.push(performance.now())).observe(caret, { attributes: true });
  let last = 0;
  const frame = (t: number) => {
    if (last) W.frames.push(t - last);
    last = t;
    requestAnimationFrame(frame);
  };
  requestAnimationFrame(frame);
}

const median = (a: number[]) => {
  const s = [...a].sort((x, y) => x - y);
  return s.length ? (s[s.length >> 1] ?? Number.NaN) : Number.NaN;
};

async function oneRun(browser: Browser, cpu: number) {
  const ctx = await browser.newContext({ baseURL: opt.url, viewport: { width: 1920, height: 1080 } });
  const keyboard = { learn: !opt.bare, reinforce: false, freetype: false };
  await ctx.addInitScript(
    ([mode, kb]) => {
      localStorage.setItem("tt.screen", JSON.stringify(mode));
      localStorage.setItem("tt.checkedIn", "true");
      localStorage.setItem("tt.keyboard", JSON.stringify(kb));
    },
    [opt.mode, keyboard] as const,
  );
  const page = await ctx.newPage();
  await page.goto("/");
  await page.waitForFunction((n) => document.querySelectorAll("[data-text] > span[data-m]").length > n, opt.keys + 1);
  await page.waitForTimeout(800);
  const text = await page.evaluate(() =>
    [...document.querySelectorAll("[data-text] > span[data-m]")].map((s) => s.textContent).join(""),
  );
  const cdp = await ctx.newCDPSession(page);
  await cdp.send("Performance.enable");
  await cdp.send("Emulation.setCPUThrottlingRate", { rate: cpu });
  await page.evaluate(setupProbes);
  const recalc = async () =>
    (await cdp.send("Performance.getMetrics")).metrics.find((m) => m.name === "RecalcStyleDuration")?.value ?? 0;
  const before = await recalc();
  for (const ch of text.slice(0, opt.keys)) {
    await page.keyboard.press(ch === " " ? "Space" : ch);
    await page.waitForTimeout(opt.delay);
  }
  await page.waitForTimeout(400);
  const after = await recalc();
  const W = await page.evaluate(() => (window as unknown as { __perf: Probe }).__perf);
  await ctx.close();
  const lags: number[] = [];
  let j = 0;
  for (const k of W.keys) {
    while (j < W.moves.length && (W.moves[j] ?? 0) < k) j++;
    if (j < W.moves.length) lags.push((W.moves[j] ?? 0) - k);
  }
  return {
    recalc: ((after - before) * 1000) / opt.keys,
    caret: median(lags),
    slow: W.frames.filter((f) => f > 25).length / W.frames.length,
  };
}

type Runs = { fast: Awaited<ReturnType<typeof oneRun>>[]; slow: Awaited<ReturnType<typeof oneRun>>[] };

const browser = await chromium.launch({ channel: process.env.TT_BROWSER || undefined });

async function measure(): Promise<Runs> {
  const out: Runs = { fast: [], slow: [] };
  for (let i = 0; i < opt.runs; i++) out.fast.push(await oneRun(browser, 1));
  for (let i = 0; i < opt.runs; i++) out.slow.push(await oneRun(browser, opt.slow));
  return out;
}

const numbers = ({ fast, slow }: Runs) => ({
  recalc: median(fast.map((r) => r.recalc)),
  caret: median(slow.map((r) => r.caret)),
  slow: median(slow.map((r) => r.slow)) * 100,
});
const fmt = (x: number, d = 1) => x.toFixed(d);
const title = `mode ${opt.mode}${opt.bare ? " (bare: no keyboard)" : ""}, ${opt.keys} keys every ${opt.delay} ms`;

async function serving() {
  try {
    return (await fetch(`${opt.url}/api/layout`)).ok;
  } catch {
    return false;
  }
}

async function until(ok: () => Promise<boolean>) {
  for (let i = 0; i < 300; i++) {
    if (await ok()) return;
    await new Promise((r) => setTimeout(r, 200));
  }
  throw new Error("perf: the server did not start or stop in a minute");
}

if (!opt.against) {
  const n = numbers(await measure());
  console.log(`${title}, median of ${opt.runs} runs`);
  console.log(`  style recalc per key   ${fmt(n.recalc)} ms   (full speed)`);
  console.log(`  key -> caret start     ${fmt(n.caret, 0)} ms    (CPU ${opt.slow}x slower)`);
  console.log(`  slow frames (>25 ms)   ${fmt(n.slow, 0)}%    (CPU ${opt.slow}x slower)`);
} else {
  if (await serving()) {
    console.error("perf: something already serves :3000. Stop it and run again: a busy machine reads slow.");
    process.exit(1);
  }
  const repo = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
  const dir = mkdtempSync(join(tmpdir(), "tt-perf-"));
  const exe = process.platform === "win32" ? ".exe" : "";
  const npm = process.platform === "win32" ? "npm.cmd" : "npm";
  const base = {
    name: opt.against,
    tree: join(dir, "base"),
    bin: join(dir, `base${exe}`),
    runs: { fast: [], slow: [] } as Runs,
  };
  const here = { name: "this code", tree: repo, bin: join(dir, `here${exe}`), runs: { fast: [], slow: [] } as Runs };
  try {
    execFileSync("git", ["worktree", "add", "--detach", base.tree, opt.against], { cwd: repo, stdio: "ignore" });
    for (const side of [base, here]) {
      const web = join(side.tree, "web");
      if (side === base) execFileSync(npm, ["ci"], { cwd: web, stdio: "inherit", shell: true });
      execFileSync(npm, ["run", "build"], { cwd: web, stdio: "inherit", shell: true });
      execFileSync("go", ["build", "-o", side.bin, "./cmd/typetrainer"], { cwd: side.tree, stdio: "inherit" });
    }
    for (let r = 0; r < opt.rounds; r++) {
      for (const side of r % 2 ? [here, base] : [base, here]) {
        const server: ChildProcess = spawn(side.bin, [], { cwd: side.tree, stdio: "ignore" });
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
    try {
      execFileSync("git", ["worktree", "remove", "--force", base.tree], { cwd: repo, stdio: "ignore" });
    } catch {}
    rmSync(dir, { recursive: true, force: true });
  }
  const a = numbers(base.runs);
  const b = numbers(here.runs);
  const row = (name: string, x: number, y: number, d: number, unit: string) =>
    console.log(
      `  ${name.padEnd(22)} ${(fmt(x, d) + unit).padStart(9)} ${(fmt(y, d) + unit).padStart(11)} ${((y >= x ? "+" : "") + fmt(y - x, d) + unit).padStart(9)}`,
    );
  console.log(`\n${title}, median of ${opt.runs * opt.rounds} runs each, ${opt.rounds} rounds taking turns`);
  console.log(`  ${"".padEnd(22)} ${opt.against.padStart(9)} ${"this code".padStart(11)} ${"change".padStart(9)}`);
  row("style recalc per key", a.recalc, b.recalc, 1, " ms");
  row(`key -> caret (CPU ${opt.slow}x)`, a.caret, b.caret, 0, " ms");
  row(`slow frames (CPU ${opt.slow}x)`, a.slow, b.slow, 0, "%");
}
await browser.close();
