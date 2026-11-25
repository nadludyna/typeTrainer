// The Welcome screen: touch typing, the home row and space, the finger tour
// (docs/HOW-THE-MODES-WORK.md, section 14.1), then Learn. Its presses are
// never sent anywhere.

import { $, el, S, store } from "./state.js";
import { t } from "./i18n.js";
import { makeKey } from "./keyboard.js";

const HOME_CODES = ["KeyA", "KeyS", "KeyD", "KeyF", "KeyJ", "KeyK", "KeyL", "Semicolon"];
const HOME_INDEX = [0, 1, 2, 3, 6, 7, 8, 9];
const STEP_WHAT = 0, STEP_HOME = 1, STEP_TOUR = 2, STEP_START = 3;
const FINGERS = ["lp", "lr", "lm", "li", "ri", "rm", "rr", "rp"];
const TOUR_ROWS = 3, TOUR_COLS = 10;   // the letter block: q…p, a…;, z…/
const NEXT_FINGER_MS = 350;

let step = 0;
let homeKeys = {};
let homePressed = new Set();
let spacePressed = false;
let tourKeys = [];        // per finger: its keys, column by column, top to bottom
let tourEls = {};         // char -> key element on the tour map
let finger = 0, at = 0;
let onStart = () => {};

export function onWelcomeStart(fn) { onStart = fn; }

// ---------- drawing the keys (on a language load) ----------
export function drawIntroKeys() {
  const rows = S.layout.rows;
  const home = $("intro-home");
  home.innerHTML = "";
  homeKeys = {};
  HOME_INDEX.forEach((idx, i) => {
    const key = makeKey(rows[1][idx]);
    if (i === 4) home.appendChild(Object.assign(document.createElement("span"), { className: "gap" }));
    home.appendChild(key);
    homeKeys[HOME_CODES[i]] = key;
  });
  drawMap($("intro-map"));
  tourEls = drawMap($("tour-map"));
  const main = rows.slice(0, TOUR_ROWS).map((r) => r.slice(0, TOUR_COLS));
  tourKeys = FINGERS.map((f) => fingerKeys(main, f));
  $("tour-fingers").innerHTML = FINGERS.map((f) =>
    `<i style="--fc:${S.layout.fingers[f].color}"></i>`).join("");
  resetWelcome();
}

function drawMap(box) {
  box.innerHTML = "";
  const els = {};
  S.layout.rows.slice(0, TOUR_ROWS).forEach((r, i) => {
    const rowEl = document.createElement("div");
    rowEl.className = "kb-row";
    rowEl.style.paddingLeft = `calc(var(--u) * ${S.layout.offsets?.[i] ?? 0})`;
    for (const k of r) {
      const key = makeKey(k);
      key.dataset.f = k.finger;
      els[k.char] = key;
      rowEl.appendChild(key);
    }
    box.appendChild(rowEl);
  });
  return els;
}

// A finger's keys in the order the tour asks for them: column by column,
// each from the top row down (r f v, then t g b for the left index).
export function fingerKeys(rows, f) {
  const cols = rows.map((r) => r.filter((k) => k.finger === f).map((k) => k.char));
  const out = [];
  for (let c = 0; c < Math.max(0, ...cols.map((x) => x.length)); c++) {
    for (const col of cols) if (col[c] !== undefined) out.push(col[c]);
  }
  return out;
}

// ---------- the steps ----------
export function resetWelcome() {
  step = 0;
  homePressed = new Set();
  spacePressed = false;
  finger = 0; at = 0;
  Object.values(homeKeys).forEach((k) => k.classList.remove("hit"));
  $("intro-space").classList.remove("hit");
  renderWelcome();
}

export function renderWelcome() {
  const info = S.languages.find((l) => l.id === S.lang);
  $("intro-lang").textContent = info ? " · " + info.name : "";
  el.intro.querySelectorAll(".intro-step").forEach((s, i) => s.classList.toggle("active", i === step));
  el.intro.querySelectorAll("#wl-chips .wl-chip").forEach((c, i) => {
    c.classList.toggle("on", i === step);
    c.classList.toggle("done", i < step);
    c.setAttribute("aria-current", i === step ? "step" : "false");
  });
  if (step === STEP_TOUR) renderTour();
}

function goTo(n) {
  step = Math.max(STEP_WHAT, Math.min(STEP_START, n));
  renderWelcome();
}

function renderTour() {
  const done = finger >= FINGERS.length;
  const f = FINGERS[Math.min(finger, FINGERS.length - 1)];
  const name = S.layout.fingers[f].name.toLowerCase();
  const keys = tourKeys[Math.min(finger, FINGERS.length - 1)] ?? [];
  $("tour-fingers").querySelectorAll("i").forEach((d, i) => {
    d.classList.toggle("done", i < finger);
    d.classList.toggle("on", i === finger);
  });
  for (const [ch, key] of Object.entries(tourEls)) {
    const mine = !done && key.dataset.f === f;
    key.classList.toggle("col-on", mine);
    key.classList.toggle("next", mine && ch === keys[at]);
    key.classList.toggle("hit", mine && keys.indexOf(ch) < at);
  }
  $("tour-of").textContent = done ? t("tour.done") : t("tour.of", { n: finger + 1, all: FINGERS.length, name });
  $("tour-keys").innerHTML = done ? "" : keys.map((ch, i) =>
    `<span class="${i < at ? "hit" : i === at ? "next" : ""}">${ch.toUpperCase()}${i < at ? " ✓" : ""}</span>`).join("");
  const home = S.layout.rows[1].find((k) => k.finger === f)?.char ?? "";
  $("tour-text").innerHTML = done ? "" : t("tour.p", { name, home: home.toUpperCase() });
  $("tour-do").textContent = done ? "" : t("tour.do", { key: (keys[at] ?? "").toUpperCase() });
}

function tourKey(ch) {
  const keys = tourKeys[finger];
  if (!keys) return;
  if (ch !== keys[at]) {
    tourEls[keys[at]]?.animate?.([{ scale: "1" }, { scale: "1.12" }, { scale: "1" }], { duration: 240, easing: "ease-out" });
    return;
  }
  at++;
  if (at < keys.length) { renderTour(); return; }
  const was = finger;
  renderTour();
  setTimeout(() => {
    if (step !== STEP_TOUR || finger !== was) return;
    finger++; at = 0;
    if (finger >= FINGERS.length) goTo(STEP_START);
    else renderTour();
  }, NEXT_FINGER_MS);
}

function homeKey(e) {
  if (e.code === "Space") {
    spacePressed = true;
    $("intro-space").classList.add("hit");
  } else {
    const key = homeKeys[e.code];
    if (!key) return;
    key.classList.add("hit");
    homePressed.add(e.code);
  }
  if (homePressed.size === HOME_CODES.length && spacePressed) {
    const box = $("intro-home");
    const a = box.animate?.([{ scale: "1" }, { scale: "1.06" }, { scale: "1" }], { duration: 260, easing: "ease-out" });
    const go = () => { if (step === STEP_HOME) goTo(STEP_TOUR); };
    if (a) a.finished.then(go, go); else go();
  }
}

// ---------- keys, while Welcome is shown ----------
export function welcomeKey(e) {
  if (e.key === "Enter") { e.preventDefault(); startLearn(); return; }
  if (e.key === "ArrowLeft") { e.preventDefault(); goTo(step - 1); return; }
  if (e.key === "ArrowRight") { e.preventDefault(); goTo(step + 1); return; }
  if (e.key === "Tab" || e.ctrlKey || e.metaKey || e.altKey) return;
  if (e.key.length !== 1) return;
  e.preventDefault();
  if (e.repeat) return;
  if (step === STEP_WHAT) goTo(STEP_HOME);
  else if (step === STEP_HOME) homeKey(e);
  else if (step === STEP_TOUR) tourKey(e.key.toLowerCase());
}

export function startLearn() {
  store.set("tt.introDone", "1");
  onStart();
}

el.intro.querySelectorAll("#wl-chips .wl-chip").forEach((c) =>
  c.addEventListener("click", (e) => { e.currentTarget.blur(); goTo(Number(c.dataset.step)); }));
$("wl-start").addEventListener("click", (e) => { e.currentTarget.blur(); startLearn(); });
