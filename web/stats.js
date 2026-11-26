// The stats of the mode you're in, full screen (input.js opens and closes
// them with ~ or the Stats button). Each mode leads with its own measure:
// Learn accuracy, Reinforce speed against its goal, Freetype consistency.

import { i18nVars, t, applyI18n } from "./i18n.js";
import { $, el, S, isTouch, listLabel, listless, wordLevel } from "./state.js";
import { getJSON } from "./api.js";
import { lengthLabel, lengthFor, lengthOf } from "./settings.js";
import { ring, ringBtn } from "./minipop.js";
import { learnGrid, mixUps } from "./learnstats.js";
import { freetypeUrl } from "./modes.js";
import { grow } from "./fx.js";

// S.lastStats is the typing screen's: every round of Learn's stream.
let introVars = "";
export async function refreshStats() {
  S.lastStats = await getJSON(`/stats?length=${lengthFor("learn")}`);
  stale = true;
  const vars = [S.lastStats.targetWpm, S.lastStats.masteryMin, Math.round(S.lastStats.masteryLevel)];
  if (vars.join() !== introVars) {
    introVars = vars.join();
    [i18nVars.target, i18nVars.min, i18nVars.level] = vars;
    applyI18n(el.intro);
  }
  if (S.view === "stats") renderStats();
}

// ---------- what is shown: the mode's rounds of one length ----------
const sheet = $("sheet");
const shown = { mode: "", length: "", key: "", data: null, days: [], history: null };
let stale = true;

// resetStatsView: the stats open on the length you type now.
export function resetStatsView() { shown.mode = ""; }
const keyOf = () => [S.lang, S.mode, shown.length, S.freetypeList, S.freetypeVariant].join(" ");

export function renderStats() {
  if (shown.mode !== S.mode) Object.assign(shown, { mode: S.mode, length: S.mode === "learn" ? "" : lengthFor(S.mode) });
  const key = keyOf();
  if ((stale || shown.key !== key) && loading !== key) {
    loading = key;
    // A load cut short (a page left, the server away) says so on the sheet; the next open tries again.
    load(key).catch(() => {
      stale = true;
      if (keyOf() === key && S.view === "stats") sheet.innerHTML = header(null) + `<div class="panel st-empty"><h2>${t("st.failTitle")}</h2><p>${t("st.fail")}</p></div>`;
    }).finally(() => { if (loading === key) loading = ""; });
  }
  if (shown.key === key) draw();
  else if (loading === key) sheet.innerHTML = header(null) + `<div class="st-wait"></div>`;
}

let loading = "";   // the key of the stats being loaded
async function load(key) {
  stale = false;
  const mode = S.mode, tz = -new Date().getTimezoneOffset();
  const url = mode === "freetype" ? freetypeUrl(shown.length, S.freetypeList, S.freetypeVariant) : `/stats?length=${shown.length}&part=${mode}`;
  const [data, act, history] = await Promise.all([
    getJSON(url), getJSON(`/activity?tz=${tz}&part=${mode}`), mode === "learn" ? getJSON("/learn/history") : null,
  ]);
  if (key !== keyOf()) return;
  Object.assign(shown, { key, data, days: act.days, history });
  if (S.view === "stats") draw();
}

sheet.addEventListener("click", (e) => {
  const b = e.target.closest("[data-length]");
  if (!b || b.dataset.length === shown.length) return;
  b.blur();
  shown.length = b.dataset.length;
  renderStats();
});

// ---------- the sheet ----------
const BODY = { learn, reinforce, freetype };
function draw() {
  const s = shown.data, rounds = [...s.sessions].reverse();
  if (!s.lengths.length) {
    const list = S.mode === "freetype" && !listless(S.freetypeVariant) ? t("st.noneList", { n: S.freetypeList }) : "";
    sheet.innerHTML = header(s) + `<div class="panel st-empty"><h2>${t("st.noneTitle", { mode: t(`mode.${S.mode}`) })}</h2><p>${list || t("st.none")}</p></div>`;
    return;
  }
  const { tiles, panels, chart } = BODY[S.mode](s, rounds);
  // The server learns only from keyboard rounds (storage.OnKeyboard); on a phone the sheet says so.
  const phone = isTouch() && S.mode !== "freetype" ? `<p class="st-phone">${t("st.phone")}</p>` : "";
  sheet.innerHTML = header(s) + `<div class="st-tiles">${tiles.filter(Boolean).join("")}</div>` + phone + panels + calendar();
  barChart(sheet.querySelector(".st-chart"), rounds, s.sessionCount, chart);
}

const byLength = (a, b) => { const x = lengthOf(a), y = lengthOf(b); return x.kind.localeCompare(y.kind) || x.n - y.n; };
function header(s) {
  const mode = S.mode;
  const lens = mode === "learn" || !s ? [] : [...new Set([...s.lengths, shown.length])].sort(byLength).reverse();
  const pick = lens.length > 1 ? ring(t("settings.length"), lens.map((id) => ringBtn(`data-length="${id}"`, lengthLabel(id), id === shown.length))) : "";
  return `<div class="st-head"><h1>${t("st.title", { mode: t(`mode.${mode}`) })}</h1><span class="st-what">${t(`st.what.${mode}`)}</span>${pick}` +
    `<span class="st-close keys-only">${t("st.close")}</span><button type="button" class="btn st-back">${t("st.back")}</button></div>`;
}

// ---------- the numbers ----------
const avg = (arr, f) => arr.length ? arr.reduce((a, x) => a + f(x), 0) / arr.length : null;
const last10 = (rounds, f) => avg(rounds.slice(-10).filter((x) => f(x) != null), f);
const num = (v, unit = "") => v === null ? "—" : Math.round(v) + (unit && `<i>${unit}</i>`);
function trendTag(rounds, f) {
  const last = rounds.slice(-10), before = rounds.slice(-20, -10);
  const now = avg(last.filter((x) => f(x) != null), f), was = avg(before.filter((x) => f(x) != null), f);
  if (now === null || was === null || before.length < 5) return "";
  const d = Math.round(now - was);
  if (!d) return "";
  return `<span class="trend ${d > 0 ? "up" : "down"}" data-tip="tile.trend">${d > 0 ? "↑" : "↓"} ${Math.abs(d)}</span> `;
}
function tile(label, value, meta = "", { hero = false, part = null } = {}) {
  const bar = part === null ? "" : `<i class="goal-bar" style="--p:${Math.min(1, Math.max(0, part)).toFixed(3)}"></i>`;
  return `<div class="tile${hero ? " hero" : ""}"><b>${value}</b><small>${label}</small><span class="meta">${meta}</span>${bar}</div>`;
}
function roundsTile(count) {
  const sec = shown.days.reduce((a, d) => a + d.seconds, 0);
  return tile(t("tile.exercises"), count, sec ? t("st.typing", { time: duration(sec) }) : "");
}
function duration(sec) {
  sec = Math.round(sec);
  if (sec < 60) return t("time.s", { s: sec });
  const m = Math.floor(sec / 60);
  return m < 60 ? t("time.m", { m }) : t("time.h", { h: Math.floor(m / 60), m: m % 60 });
}
const panel = (title, note, body, cls = "") =>
  `<div class="panel${cls}"><div class="panel-head"><h2>${title}</h2><span class="panel-note">${note}</span></div>${body}</div>`;
const chartPanel = (title, note) => panel(title, note, `<div class="chart st-chart"></div>`);

// ---------- Learn: accuracy ----------
function learn(s, rounds) {
  const acc = last10(rounds, (x) => x.accuracy);
  const open = s.keys.filter((k) => k.unlocked).length, all = s.order.length, next = s.order[open];
  const f = s.forecast?.known ? s.forecast : null;
  const weak = s.keys.filter((k) => k.unlocked && !k.mastered && k.count && k.char !== " ")
    .sort((a, b) => b.difficulty - a.difficulty)[0];
  const tiles = [
    tile(t("st.acc10"), num(acc, "%"), trendTag(rounds, (x) => x.accuracy), { hero: true, part: acc === null ? 0 : acc / 100 }),
    tile(t("st.open"), `${open}<i>/ ${all}</i>`, next ? t("st.nextIs", { l: next }) : t("st.allOpen"), { part: open / all }),
    tile(t(next ? "fc.next" : "fc.lap"), f ? (f.more ? `${f.rounds}+` : `≈ ${f.rounds}`) : "—", f ? t("st.rounds") : t("st.fcLater")),
    roundsTile(s.sessionCount),
    weak ? tile(t("st.weakest"), weak.char, t("st.right", { p: Math.floor(weak.level) })) : null,
  ];
  const h = shown.history, rounds2 = h?.rounds ?? [];
  let grid = "";
  if (rounds2.length) {
    const g = learnGrid(rounds2, h.order, sheet.clientWidth - 36);
    grid = panel(t("ls.grid"), t("st.gridNote", { scope: g.scope }), g.html);
  }
  const panels = `<div class="st-two">${chartPanel(t("st.accChart"), t("st.accLine", { p: Math.round(s.masteryLevel) }))}` +
    panel(t("ls.mix"), t("ls.mixNote"), mixUps(h?.confusions ?? [])) + `</div>` + grid;
  return { tiles, panels, chart: { value: (x) => x.accuracy, unit: "%", line: s.masteryLevel, percent: true, order: s.order } };
}

// ---------- Reinforce: speed against your goal ----------
function reinforce(s, rounds) {
  const wpm = last10(rounds, (x) => x.wpm), acc = last10(rounds, (x) => x.accuracy), goal = s.targetWpm;
  const words = s.wordLevels?.[wordLevel() - 1];
  const best = s.bests?.[shown.length];
  const tiles = [
    tile(t("st.wpm10"), num(wpm, "wpm"), trendTag(rounds, (x) => x.wpm) + t("st.goalIs", { w: goal }), { hero: true, part: wpm === null ? 0 : wpm / goal }),
    tile(t("tile.acc"), num(acc, "%"), trendTag(rounds, (x) => x.accuracy) + t("tile.last10")),
    words ? tile(t("re.words"), listLabel(words), levelNote(s.standing, s.wordLevels)) : null,
    tile(t("tile.best"), best ? num(best.wpm, "wpm") : "—", best ? t("st.bestAcc", { p: Math.round(best.accuracy) }) : ""),
    roundsTile(s.sessionCount),
  ];
  const slowKeys = s.keys.filter((k) => k.unlocked && k.count && k.avgMs > 0 && k.char !== " ")
    .sort((a, b) => b.avgMs - a.avgMs).slice(0, 4).map((k) => [k.char, 12000 / k.avgMs]);
  const slowPairs = (s.pairs ?? []).map((p) => [p.pair, 12000 / p.cost]);
  const chips = (list) => list.length ? `<div class="st-chips">${list.map(([c, w]) =>
    `<span class="st-chip"><b>${c}</b><small class="${w < goal ? "bad" : ""}">${Math.round(w)} wpm</small></span>`).join("")}</div>` : `<p class="st-none">${t("st.noSlow")}</p>`;
  const slow = panel(t("st.slowKeys"), t("st.goalIs", { w: goal }), chips(slowKeys) +
    `<div class="panel-head st-sub"><h2>${t("st.slowPairs")}</h2></div>` + chips(slowPairs));
  const panels = `<div class="st-two">${chartPanel(t("stats.speed"), t("st.goalLine"))}${slow}</div>`;
  return { tiles, panels, chart: { value: (x) => x.wpm, unit: "wpm", line: goal } };
}

// How close the next move of the word level is (levels.Standing on the server).
function levelNote(st, levels) {
  const words = levels[wordLevel() - 1];
  if (!st || st.level !== wordLevel()) return t("st.common", { n: words });
  if (st.passes && levels[st.level]) return t("st.toUp", { n: st.passes, of: st.passesUp, next: listLabel(levels[st.level]) });
  if (st.fails && st.level > 1) return t("st.toDown", { n: st.fails, of: st.failsDown, prev: listLabel(levels[st.level - 2]) });
  return t("st.common", { n: words });
}

// How close the next move of Freetype's word list is (as levelNote).
function listNote(st, lists) {
  const n = S.freetypeList, at = (lists ?? []).indexOf(n) + 1;
  if (!st || st.level !== at) return t("st.common", { n });
  if (st.passes && lists[at]) return t("st.toUp", { n: st.passes, of: st.passesUp, next: listLabel(lists[at]) });
  if (st.fails && at > 1) return t("st.toDown", { n: st.fails, of: st.failsDown, prev: listLabel(lists[at - 2]) });
  return t("st.common", { n });
}

// ---------- Freetype: rhythm ----------
function freetype(s, rounds) {
  const cons = last10(rounds, (x) => x.consistency), wpm = last10(rounds, (x) => x.wpm), acc = last10(rounds, (x) => x.accuracy);
  const goal = S.settings.goal, v = S.freetypeVariant;
  const tiles = [
    tile(t("st.cons10"), num(cons, "%"), trendTag(rounds, (x) => x.consistency) + t("st.consMeans"), { hero: true, part: cons === null ? 0 : cons / 100 }),
    tile(t("tile.wpm"), num(wpm, "wpm"), trendTag(rounds, (x) => x.wpm) + (goal ? t("st.goalIs", { w: goal }) : t("tile.last10")), goal ? { part: wpm === null ? 0 : wpm / goal } : {}),
    tile(t("tile.acc"), num(acc, "%"), trendTag(rounds, (x) => x.accuracy) + t("tile.last10")),
    tile(t("re.words"), listless(v) ? t(`ft.${v}`) : listLabel(S.freetypeList), listless(v) ? "" : listNote(s.standing, s.lists)),
    roundsTile(s.sessionCount),
  ];
  const rh = s.rhythm, pairs = rh?.pairs ?? [];
  const body = !rh?.threshold ? `<p class="st-none">${t("st.noSlow")}</p>` : !pairs.length ? `<p class="st-none">${t("st.noStumbles")}</p>` :
    `<div class="st-chips">${pairs.map((p) => `<span class="st-chip"><b>${p.pair}</b><small>${Math.round(p.rate * 100)}%</small></span>`).join("")}</div>`;
  const stumbles = panel(t("st.stumbles"), rh?.threshold ? t("st.stumbleNote", { ms: Math.round(rh.threshold) }) : "", body);
  const panels = `<div class="st-two">${chartPanel(t("st.consChart"), t("st.consNote"))}${stumbles}</div>`;
  return { tiles, panels, chart: { value: (x) => x.consistency, unit: "%", percent: true } };
}

// ---------- the chart: one bar per round, the mode's measure ----------
function barChart(node, rounds, total, o) {
  if (!rounds.length) { node.innerHTML = `<div class="empty">${t("chart.empty")}</div>`; return; }
  const vals = rounds.map(o.value), have = vals.filter((v) => v != null);
  const first = total - rounds.length + 1;
  const W = Math.max(280, node.clientWidth || 860), H = Math.max(120, node.clientHeight || 200);
  const padL = 34, padR = 8, padT = 10, padB = 22;
  const plotW = W - padL - padR, plotH = H - padT - padB;
  const lo = o.percent ? Math.max(0, Math.floor((Math.min(...have, o.line ?? 100) - 5) / 20) * 20) : 0;
  const hi = o.percent ? 100 : niceMax(Math.max(o.line ?? 0, ...have));
  const y = (v) => padT + plotH - ((v - lo) / (hi - lo)) * plotH;
  const slot = plotW / Math.max(rounds.length, 10);
  const bw = Math.max(3, Math.min(28, slot - 2));
  const shift = plotW - slot * rounds.length;
  const cx = (i) => padL + shift + (i + 0.5) * slot;

  let svg = `<svg viewBox="0 0 ${W} ${H}" role="img">` +
    `<text class="axis axis-name" x="${shift > slot ? padL : padL + plotW / 2}" y="${H - 6}" text-anchor="${shift > slot ? "start" : "middle"}">${t("chart.axisRound")}</text>`;
  for (let i = 0; i <= 4; i++) {
    const v = lo + ((hi - lo) / 4) * i, yy = y(v);
    svg += `<line class="grid" x1="${padL}" x2="${W - padR}" y1="${yy}" y2="${yy}"/>`;
    svg += `<text class="axis" x="${padL - 6}" y="${yy + 4}" text-anchor="end">${Math.round(v)}</text>`;
  }
  vals.forEach((v, i) => {
    if (v == null) return;
    const bx = padL + shift + i * slot + (slot - bw) / 2, top = y(v), base = y(lo);
    const r = Math.min(4, bw / 2, base - top);
    const d = `M${bx},${base} V${top + r} Q${bx},${top} ${bx + r},${top} H${bx + bw - r} Q${bx + bw},${top} ${bx + bw},${top + r} V${base} Z`;
    svg += `<path class="bar${i === rounds.length - 1 ? " last" : ""}${o.line != null && v < o.line ? " under" : ""}" data-i="${i}" d="${d}"/>`;
  });
  const opened = rounds.map((x, i) => o.order && i && x.unlocked > (rounds[i - 1].unlocked || x.unlocked)
    ? o.order.slice(rounds[i - 1].unlocked, x.unlocked) : []);
  opened.forEach((letters, i) => {
    if (!letters.length) return;
    svg += `<text class="opened" x="${cx(i)}" y="${padT + 9}" text-anchor="middle">${letters.join("")}</text>` +
      `<line class="opened" x1="${cx(i)}" x2="${cx(i)}" y1="${padT + 13}" y2="${y(vals[i] ?? lo) - 3}"/>`;
  });
  if (o.line != null) svg += `<line class="target" x1="${padL}" x2="${W - padR}" y1="${y(o.line)}" y2="${y(o.line)}"/>`;
  if (have.length >= 3) {
    const pts = [];
    vals.forEach((v, i) => {
      const part = vals.slice(Math.max(0, i - 9), i + 1).filter((x) => x != null);
      if (v != null) pts.push(`${cx(i).toFixed(1)},${y(part.reduce((a, x) => a + x, 0) / part.length).toFixed(1)}`);
    });
    svg += `<polyline class="avg" points="${pts.join(" ")}"/>`;
  }
  svg += `<text class="axis" x="${cx(0)}" y="${H - 6}" text-anchor="middle">#${first}</text>`;
  if (rounds.length > 1) svg += `<text class="axis" x="${cx(rounds.length - 1)}" y="${H - 6}" text-anchor="middle">#${total}</text>`;
  rounds.forEach((_, i) => { svg += `<rect class="hit" data-i="${i}" x="${padL + shift + i * slot}" y="${padT}" width="${slot}" height="${plotH}"/>`; });
  node.innerHTML = svg + `</svg>`;
  // The chart is named by its panel's title, set as text, never parsed as HTML.
  node.firstElementChild.setAttribute("aria-label", node.closest(".panel").querySelector("h2").textContent);

  const bars = new Map([...node.querySelectorAll(".bar")].map((b) => [Number(b.dataset.i), b]));
  grow([...bars.values()], [{ transform: "scaleY(0)" }, { transform: "scaleY(1)" }], 300, 6);
  grow(node.querySelectorAll(".avg, .target"), [{ opacity: 0 }, { opacity: 1 }], 300);
  node.querySelectorAll(".hit").forEach((h) => {
    const i = Number(h.dataset.i), x = rounds[i];
    h.addEventListener("pointerenter", () => bars.get(i)?.classList.add("hover"));
    h.addEventListener("pointerleave", () => { bars.get(i)?.classList.remove("hover"); el.tooltip.classList.remove("show"); });
    h.addEventListener("pointermove", (e) => {
      const cons = x.consistency != null ? ` <span>·</span> ${t("chart.cons", { pct: Math.round(x.consistency) })}` : "";
      const news = opened[i].length ? ` <span>·</span> ${t("chart.opened", { l: opened[i].join(", ") })}` : "";
      el.tooltip.innerHTML = `<span>#${first + i}</span> ${Math.round(x.wpm)} wpm <span>·</span> ${Math.round(x.accuracy)}%${cons}${news}`;
      el.tooltip.style.left = e.clientX + "px";
      el.tooltip.style.top = e.clientY + "px";
      el.tooltip.classList.remove("below");
      el.tooltip.classList.add("show");
    });
  });
}
function niceMax(v) {
  const steps = [5, 10, 15, 20, 25, 30, 40, 50, 60, 75, 100, 125, 150, 200, 250];
  const step = steps.find((s) => s * 4 >= v * 1.05) ?? Math.ceil(v * 1.05 / 400) * 100;
  return step * 4;
}

// ---------- the days you practised this mode ----------
const WEEKS_MAX = 53, CELL_MIN = 10, CELL_GAP = 3, DAY_LABELS = 34;
const dayKey = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
function dayStreak(byDate) {
  const d = new Date();
  if (!byDate.has(dayKey(d))) d.setDate(d.getDate() - 1);
  let n = 0;
  while (byDate.has(dayKey(d))) { n++; d.setDate(d.getDate() - 1); }
  return n;
}
function calendar() {
  const days = shown.days, byDate = new Map(days.map((x) => [x.date, x]));
  const width = sheet.clientWidth - 36 - DAY_LABELS;
  const weeks = Math.max(8, Math.min(WEEKS_MAX, Math.floor((width + CELL_GAP) / (CELL_MIN + CELL_GAP))));
  const today = new Date(), start = new Date(today);
  start.setDate(today.getDate() - ((today.getDay() + 6) % 7) - (weeks - 1) * 7);
  const dates = [];
  for (const d = new Date(start); d <= today; d.setDate(d.getDate() + 1)) dates.push(dayKey(d));
  const most = Math.max(1, ...dates.map((k) => byDate.get(k)?.seconds ?? 0));
  const lang = document.documentElement.lang || undefined;
  const fmt = new Intl.DateTimeFormat(lang, { weekday: "short", month: "short", day: "numeric" });
  const cells = dates.map((k) => {
    const x = byDate.get(k), date = fmt.format(new Date(k + "T12:00"));
    const level = x ? Math.max(1, Math.ceil((x.seconds / most) * 4)) : 0;
    const tip = !x ? t("act.dayNone", { date }) : t(x.rounds === 1 ? "act.dayOne" : "act.day", { date, time: duration(x.seconds), n: x.rounds });
    return `<i class="l${level}" data-tip-text="${tip}"></i>`;
  }).join("");
  const monthName = new Intl.DateTimeFormat(lang, { month: "short" });
  let months = "";
  for (let w = 0, last = -1; w < weeks; w++) {
    const d = new Date(start);
    d.setDate(start.getDate() + w * 7);
    if (d.getMonth() !== last && (w > 0 || d.getDate() <= 7)) months += `<span style="grid-column:${w + 1} / span 4">${monthName.format(d)}</span>`;
    last = d.getMonth();
  }
  const dayName = new Intl.DateTimeFormat(lang, { weekday: "short" });
  const dayNames = [0, 2, 4].map((i) => `<span style="grid-row:${i + 1}">${dayName.format(new Date(2026, 9, 19 + i))}</span>`).join("");
  const note = t("st.days", { n: days.filter((d) => d.rounds).length, streak: dayStreak(byDate) });
  return panel(t("st.daysTitle", { mode: t(`mode.${S.mode}`) }), note,
    `<div class="act-cal" style="--weeks:${weeks}"><div class="act-months">${months}</div>` +
    `<div class="act-days">${dayNames}</div><div class="act-grid">${cells}</div></div>`, " activity");
}
