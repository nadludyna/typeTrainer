// The letters row.

import { t } from "./i18n.js";
import { el, S } from "./state.js";
import { learning } from "./modes.js";
import { renderModeBar } from "./modebar.js";

let builtFor = "";
let chipOf = {};
export function renderLetters() {
  if (!learning()) {
    builtFor = "";
    renderFreeLetters();
    return;
  }
  if (!S.exercise) return;
  const info = {};
  for (const k of S.lastStats?.keys ?? []) info[k.char] = k;
  const focus = S.exercise.focus;

  const set = JSON.stringify([S.lang, S.exercise.unlocked, S.exercise.locked,
    S.exercise.unlocked.filter((c) => info[c]?.mastered)]);
  if (set !== builtFor) {
    builtFor = set;
    chipOf = {};
    const frag = document.createDocumentFragment();
    const add = (c, cls) => {
      const chip = document.createElement("span");
      chip.className = "chip " + cls;
      chip.textContent = c;
      if (!cls.includes("locked")) chip.appendChild(document.createElement("i")).className = "prog";
      chipOf[c] = chip;
      frag.appendChild(chip);
    };
    for (const c of S.exercise.unlocked) add(c, info[c]?.mastered ? "mastered" : "");
    for (const c of S.exercise.locked) add(c, c === S.exercise.locked[0] ? "locked next" : "locked");
    el.letters.replaceChildren(frag);

    const open = new Set([...S.exercise.unlocked, " "]);
    const known = new Set(S.exercise.unlocked.filter((c) => info[c]?.mastered));
    const altOf = Object.fromEntries(Object.entries(S.altBase).map(([a, b]) => [b, a]));
    for (const [c, key] of Object.entries(S.keyEls)) {
      key.classList.toggle("locked", !open.has(c) && !open.has(altOf[c]));
      key.classList.toggle("known", known.has(c) && (!altOf[c] || known.has(altOf[c])));
    }
    if (S.altgrEl) S.altgrEl.classList.toggle("locked", !S.exercise.unlocked.some((c) => S.altBase[c]));
  }
  const needs = [];
  for (const c of S.exercise.unlocked) {
    const k = info[c], chip = chipOf[c];
    if (!chip) continue;
    chip.style.setProperty("--p", k?.mastered ? 1 : letterProgress(k).toFixed(3));
    chip.classList.toggle("fresh", !k?.count);
    chip.dataset.tipText = letterTip(k);
    if (!k?.mastered) needs.push(c);
  }
  const next = chipOf[S.exercise.locked[0]];
  if (next) {
    const f = forecastText(S.lastStats?.forecast, false);
    next.dataset.tipText = (needs.length
      ? t("tip.nextNeeds", { l: S.exercise.locked[0], list: needs.join(" ") })
      : t("tip.next")) + (f ? " " + t("tip.nextIn", { when: f.value }) : "") +
      " " + t("tip.rule", { min: rule().min, level: rule().level, wpm: S.lastStats?.targetWpm || 30 });
  }
  for (const c of S.exercise.unlocked) {
    chipOf[c]?.classList.toggle("focus", c === focus);
  }

}

function renderFreeLetters() {
  el.letters.replaceChildren();
  for (const key of Object.values(S.keyEls)) key.classList.remove("locked", "known");
  if (S.altgrEl) S.altgrEl.classList.remove("locked");
  if (S.mode === "climb") renderModeBar();
}

// ---------- one letter's progress ----------
export const rule = () => {
  const s = S.lastStats ?? {};
  return { min: s.masteryMin ?? 40, level: s.masteryLevel ?? 80, ms: 12000 / (s.targetWpm || 30) };
};
// Part of the way to mastered: the presses and the level, whichever is behind.
export function letterProgress(k) {
  if (!k?.count) return 0;
  const r = rule();
  return Math.max(0, Math.min(1, k.count / r.min, (k.level ?? 0) / r.level));
}
// What holds a letter back: a mistake costs more than a slow press (Learn's
// difficulty weighs them 0.6 and 0.4), so mistakes over 5% come first.
const CLEAN_ENOUGH = 0.05;
export function letterNeeds(k) {
  if (k?.mastered) return [];
  if (!k?.count) return ["need.start"];
  const r = rule(), out = [];
  if ((k.level ?? 0) < r.level) {
    if (k.errorRate > CLEAN_ENOUGH) out.push("need.cleaner");
    if (!k.avgMs || k.avgMs > r.ms) out.push("need.faster");
  }
  if (k.count < r.min || !out.length) out.push("need.more");
  return out;
}
function letterTip(k) {
  if (!k?.count) return t("letter.noData");
  const parts = [t("letter.level", { n: Math.round(k.level ?? 0) }), t("heat.count", { n: k.count }), t("letter.err", { e: Math.round(k.errorRate * 100) })];
  if (k.avgMs) parts.push(t("letter.ms", { wpm: Math.round(12000 / k.avgMs) }));
  const needs = letterNeeds(k);
  return `${parts.join(", ")}. ${k.mastered ? t("letter.mastered") : t("letter.needs", { list: needs.map((n) => t(n)).join(", ") })}.`;
}

// ---------- the forecast ----------
export function forecastText(f, allOpen) {
  if (!f?.known) return null;
  const label = t(allOpen ? "fc.lap" : "fc.next");
  const value = f.more ? t("fc.more", { n: f.rounds }) : t(f.rounds === 1 ? "fc.one" : "fc.rounds", { n: f.rounds });
  return { label, value, tip: t("fc.tip", { l: f.letter }) };
}
