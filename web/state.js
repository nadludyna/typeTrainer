// Shared state and helpers.

// ---------- page elements ----------
export const $ = (id) => document.getElementById(id);
export const el = {
  textBox: $("text-box"), text: $("text"), caret: $("caret"), input: $("hidden-input"),
  keyboard: $("keyboard"), hint: $("finger-hint"),
  live: $("live"), wpm: $("wpm"), acc: $("acc"), roundCount: $("round-count"),
  letters: $("letters"),
  tooltip: $("tooltip"),
  hands: $("hands"),
  intro: $("intro"),
  nav: $("nav-switch"), tabLearn: $("tab-learn"), tabFreetype: $("tab-freetype"), tabStats: $("tab-stats"),
  viewType: $("view-type"), viewStats: $("view-stats"),
  lengthFilter: $("length-filter"), tiles: $("tiles"), chart: $("chart"), chartNote: $("chart-note"),
  warn: $("layout-warn"), toasts: $("toasts"),
};

// ---------- state ----------
export const S = {
  languages: [],
  lang: "en",
  alphabet: new Set(),
  layout: null,
  keyEls: {},
  altBase: {},
  altgrEl: null,
  exercise: null,
  lastStats: null,
  roundSeconds: 0,
  text: "",
  pos: 0,
  busy: false,
  view: "type",
  mode: "learn",
  freetype: null,
  freetypeList: 0,
  freetypeVariant: "",
  settings: { length: "words-30", live: false, caret: "line", goal: 0, daily: 15,
    glide: "smooth",
    stop: false, nextKey: true, hand: true },
  kbHidden: false,
};

// ---------- small helpers ----------
export const store = {
  get(k) { try { return localStorage.getItem(k); } catch { return null; } },
  set(k, v) { try { localStorage.setItem(k, v); } catch {} },
};
export const introOpen = () => !el.intro.classList.contains("hidden") && !el.intro.classList.contains("closing");
export const listless = (v) => v === "sentences" || v === "custom";
export const roundList = () => listless(S.freetypeVariant) ? 0 : S.freetypeList;
export const listLabel = (n) => n >= 1000 && n % 1000 === 0 ? `${n / 1000}k` : String(n);
export const isTouch = () => matchMedia("(pointer: coarse)").matches;
const motionQuery = matchMedia("(prefers-reduced-motion: reduce)");
export const reducedMotion = () => motionQuery.matches;
export function boxState(on, ...names) {
  for (const n of names) {
    el.textBox.classList.toggle(n, on);
    document.body.classList.toggle("tb-" + n, on);
  }
}
export function focusText() {
  if (isTouch() && document.activeElement !== el.input) el.input.focus({ preventScroll: true });
  el.textBox.classList.remove("blurred");
}

// Freetype is hidden for now: its code and saved rounds stay.
// tt.allModes = "1" shows it (browser checks only).
export const SHOWN_MODES = store.get("tt.allModes") === "1" ? ["learn", "freetype"] : ["learn"];
export const shownMode = (m) => SHOWN_MODES.includes(m);

// Reinforce: Learn once every letter is open. The word level is this
// browser's choice per language.
export const reinforcing = () => S.mode === "learn" && !!S.lastStats?.keys?.length && S.lastStats.keys.every((k) => k.unlocked);
// The stats screen opens with Reinforce: while letters are learned it would only distract.
export const statsShown = () => SHOWN_MODES.length > 1 || S.mode !== "learn" || reinforcing();
export const wordLevel = () => Math.min(Math.max(Number(store.get(`tt.wordLevel.${S.lang}`)) || 1, 1), S.lastStats?.wordLevels?.length || 1);
export const setWordLevel = (n) => store.set(`tt.wordLevel.${S.lang}`, String(n));
