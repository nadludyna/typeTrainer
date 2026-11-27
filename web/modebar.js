// The round bar above the text.

import { t } from "./i18n.js";
import { $, S, listLabel, listless, reinforcing, learnDone, wordLevel, roundKind } from "./state.js";
import { LENGTHS, lengthOf, lengthLabel } from "./settings.js";
import { hasCustomText } from "./customtext.js";
import { ring, ringBtn } from "./minipop.js";

const bar = $("mode-bar"), stand = $("stand");

export function renderModeBar() {
  const cur = S.settings.length;
  const parts = [];
  bar.hidden = S.mode === "learn";
  document.body.classList.toggle("reinforce", reinforcing());
  if (bar.hidden) {
    bar.innerHTML = "";
    stand.innerHTML = "";
    return;
  }
  if (reinforcing()) {
    bar.innerHTML = reinforceBar();
    stand.innerHTML = "";
    return;
  }
  stand.innerHTML = freetypeStand();
  if (S.mode === "freetype") {
    const v = S.freetypeVariant;
    const kinds = ["", "long", ...(S.freetype?.sentences ? ["sentences"] : []), "custom"].map((k) =>
      ringBtn(`data-variant="${k}"`, t(k ? `ft.${k}` : "ft.words"), v === k, k && k !== "custom" ? t(`ft.${k}Tip`) : ""));
    if (v === "custom" && hasCustomText()) kinds.push(ringBtn(`data-edit="1"`, t("ft.edit"), false));
    parts.push(ring(t("ft.kind"), kinds));
    if (!listless(v)) parts.push(kindRing(S.freetype?.weakSpots));
    if (!listless(v)) parts.push(ring(t("freetype.list"), (S.freetype?.lists ?? []).map((n) =>
      ringBtn(`data-list="${n}"`, listLabel(n), n === S.freetypeList, t("freetype.listN", { n })))));
  }
  const kind = lengthOf(cur).kind;
  const end = [];
  end.push(ring(t("bar.kind"), ["words", "time"].map((k) =>
    ringBtn(`data-length="${k}-30"`, t(k === "words" ? "settings.words" : "settings.time"), k === kind))));
  end.push(ring(t("bar.length"), LENGTHS.filter((id) => lengthOf(id).kind === kind).map((id) =>
    ringBtn(`data-length="${id}"`, String(lengthOf(id).n), id === cur, lengthLabel(id)))));
  bar.innerHTML = `<span class="bar-side">${parts.join("")}</span><span class="bar-side">${end.join("")}</span>`;
}

// The kind of round: the usual one, or one built from your weak spots
// (spots: how many the server found; with none the choice is off).
function kindRing(spots) {
  const kind = roundKind(), none = !spots && kind !== "weak";
  // Each kind is one button that is on or off: the bar has no room for a button of the usual round.
  return ring(t("kind.label"), [
    ringBtn(`data-kind="${kind === "weak" ? "" : "weak"}" data-weak${none ? " disabled" : ""}`, t("kind.weak"), kind === "weak", t(none ? "kind.noWeak" : "kind.weakTip")),
    ringBtn(`data-kind="${kind === "clean" ? "" : "clean"}" data-clean`, t("kind.clean"), kind === "clean", t("kind.cleanTip")),
  ]);
}

// ---------- Reinforce: the word level and the speed goal ----------
function reinforceBar() {
  const s = S.lastStats, at = wordLevel();
  if (!s) return "";
  const levels = ring(t("re.words"), (s.wordLevels ?? []).map((n, i) =>
    ringBtn(`data-wlevel="${i + 1}"`, kLabel(n), i + 1 === at, t("re.wordsTip", { n: n.toLocaleString("en") }))));
  const step = s.lap ?? 0, last = s.lastLap ?? 0;
  const goal = `<span class="ring stepper" role="group" aria-label="${t("re.goal")}">` +
    `<button type="button" data-step="${step - 1}" aria-label="${t("re.down")}"${step > 0 ? "" : " disabled"}>−</button>` +
    `<output data-tip-text="${t("re.goalTip", { n: step + 1, all: last + 1 })}">${Math.round(s.targetWpm)} wpm</output>` +
    `<button type="button" data-step="${step + 1}" aria-label="${t("re.up")}"${step < last ? "" : " disabled"}>+</button></span>`;
  const left = s.keys.filter((k) => !k.unlocked).length;
  const note = learnDone() ? "" : `<p class="re-note">${t("re.learnLeft", { n: left })}</p>`;
  return `<span class="bar-side"><b class="bar-title">${t("re.title")}</b>${levels}</span><span class="bar-side">${kindRing(s.weakSpots)}${goal}</span>${note}`;
}

const kLabel = (n) => n >= 1000 ? `${Math.round(n / 500) / 2}k` : String(n);

// ---------- Freetype: the best and the last rounds of this list and length ----------
const LAST_ROUNDS = 20;
const roundBars = (rounds, top) => rounds.map((r) => `<i style="height:${Math.max(4, r.wpm / top * 100).toFixed(1)}%" data-tip-text="${t("result.wordWpm", { wpm: Math.round(r.wpm) })}"></i>`).join("");
function freetypeStand() {
  const f = S.freetype;
  if (!f) return "";
  const rounds = (f.sessions ?? []).slice(0, LAST_ROUNDS).reverse();
  if (!rounds.length) return `<p class="stand-none">${t("below.none")}</p>`;
  const best = f.bests?.[S.settings.length]?.wpm ?? Math.max(...rounds.map((r) => r.wpm));
  const top = Math.max(best, ...rounds.map((r) => r.wpm));
  return `<p class="stand-fact"><small>${t("below.best")}</small><b>${Math.round(best)}</b></p>` +
    `<div class="stand-chart" role="img" aria-label="${t("below.last", { n: rounds.length })}">${roundBars(rounds, top)}</div>`;
}

let pickListener = () => {};
export function onBarPick(fn) { pickListener = fn; }
const onClick = (e) => {
  const b = e.target.closest("button");
  if (!b) return;
  b.blur();
  if (b.dataset.edit) return pickListener({ edit: true });
  if (b.getAttribute("aria-pressed") === "true") return;
  if (b.dataset.variant !== undefined) pickListener({ variant: b.dataset.variant });
  else if (b.dataset.list) pickListener({ list: Number(b.dataset.list) });
  else if (b.dataset.wlevel) pickListener({ wordLevel: Number(b.dataset.wlevel) });
  else if (b.dataset.kind !== undefined) pickListener({ kind: b.dataset.kind });
  else if (b.dataset.step !== undefined) pickListener({ step: Number(b.dataset.step) });
  else if (b.dataset.length) pickListener({ length: b.dataset.length });
};
bar.addEventListener("click", onClick);
