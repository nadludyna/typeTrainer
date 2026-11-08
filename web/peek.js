// A quick look at the stats of the mode you're in, in place of what's under the text.

import { t } from "./i18n.js";
import { $, el, S, isTouch, listLabel, listless } from "./state.js";
import { lengthFor, lengthLabel } from "./settings.js";
import { goalFor, summary, trendTag } from "./stats.js";
import { forecastText } from "./letters.js";
import { roundBars } from "./modebar.js";
import { resultOpen } from "./result.js";
import { roundStarted } from "./typing.js";

const box = $("peek");
const ROUNDS = 50, WEAK_LETTERS = 4, TOP_ROOM = 1.08, HOVER_MS = 200, GAP = 20;

const statsOfMode = () => (S.mode === "learn" ? S.lastStats : S.mode === "climb" ? S.climb : S.freetype);

// ---------- what it shows ----------
function title(length) {
  const parts = [t(`mode.${S.mode === "climb" ? "free" : S.mode}`)];
  if (S.mode === "freetype") {
    const v = S.freetypeVariant;
    if (v) parts.push(t(`ft.${v}`).toLowerCase());
    if (!listless(v)) parts.push(listLabel(S.freetypeList));
  }
  parts.push(lengthLabel(length));
  const goal = goalFor(S.mode);
  return `<p class="peek-title"><span>${parts.join(", ")}</span>${goal ? `<span>${t("stats.goalIs", { wpm: goal })}</span>` : ""}</p>`;
}

function numbers(s, sessions, length) {
  const [[rounds], , [wpm], [acc], [cons]] = summary(s);
  const best = s.bests?.[length]?.wpm;
  const cell = (value, label, trend = "", cls = "") =>
    `<div class="act-stat${cls}"><b>${value}${trend}</b><small>${t(label)}</small></div>`;
  return `<div class="peek-stats">` +
    cell(wpm, "tile.avgWpm", trendTag(sessions, (x) => x.wpm), " speed") +
    cell(acc, "tile.acc", trendTag(sessions, (x) => x.accuracy)) +
    cell(cons, "tile.cons") +
    cell(best == null ? "—" : Math.round(best), "tile.best") +
    cell(rounds, "tile.exercises") + `</div>`;
}

function chart(sessions) {
  const rounds = sessions.slice(-ROUNDS), goal = goalFor(S.mode);
  const top = Math.max(goal, ...rounds.map((r) => r.wpm)) * TOP_ROOM;
  const line = goal ? `<i class="peek-goal" style="bottom:${(goal / top * 100).toFixed(1)}%"></i>` : "";
  return `<div class="peek-chart"><div class="stand-chart">${roundBars(rounds, top)}</div>${line}</div>`;
}

const chips = (label, list) => list.length ?
  `<span>${label}</span>${list.map((x) => `<b class="peek-key">${x}</b>`).join("")}` : "";
function facts(s) {
  let left = "", right = "";
  if (S.mode === "learn") {
    const weak = s.keys.filter((k) => k.unlocked && !k.mastered && k.count && k.char !== " ")
      .sort((a, b) => b.difficulty - a.difficulty).slice(0, WEAK_LETTERS).map((k) => k.char);
    const f = forecastText(s.forecast, (s.sessions[0]?.unlocked ?? 0) >= s.order.length);
    left = chips(t("peek.weak"), weak);
    if (f) right = `${f.label}: ${f.value}`;
  }
  if (S.mode === "climb") {
    left = chips(t("climb.weak"), (s.weak ?? []).map((w) => w.pair));
    if (s.levels?.length) right = s.place?.level < s.levels.length ? t("climb.words", { up: s.passesUp, pass: s.passAccuracy }) : t("climb.top");
  }
  if (!left && !right) return "";
  return `<div class="peek-facts">${left}<span class="peek-end">${right}</span></div>`;
}

function draw(s) {
  const length = lengthFor(S.mode), sessions = [...s.sessions].reverse();
  box.innerHTML = title(length) + (sessions.length ?
    numbers(s, sessions, length) + chart(sessions) + facts(s) :
    `<p class="stand-none">${t("below.none")}</p>`);
}

// ---------- when it shows ----------
let shown = false, timer = 0;
export const peekShown = () => shown;

export function openPeek() {
  const s = statsOfMode();
  if (shown || !s || S.view !== "type" || isTouch() || resultOpen() || roundStarted()) return;
  shown = true;
  draw(s);
  const text = el.textBox.getBoundingClientRect(), foot = document.querySelector(".bottom").getBoundingClientRect();
  Object.assign(box.style, { left: `${text.left}px`, width: `${text.width}px`, top: `${text.bottom + GAP}px`, height: `${foot.top - text.bottom - 2 * GAP}px` });
  box.hidden = false;
  document.body.classList.add("peek-on");
}

export function closePeek() {
  clearTimeout(timer);
  if (!shown) return;
  shown = false;
  box.hidden = true;
  document.body.classList.remove("peek-on");
}

el.tabStats.addEventListener("pointerenter", () => { timer = setTimeout(openPeek, HOVER_MS); });
el.tabStats.addEventListener("pointerleave", closePeek);
addEventListener("pointerdown", closePeek, true);
addEventListener("blur", closePeek);
