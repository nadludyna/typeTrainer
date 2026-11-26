// Keys and buttons, routed to the modules that handle them.

import { $, el, S, isTouch, focusText, setWordLevel } from "./state.js";
import { requestFit, highlightNextKey, toggleKb, autoKb } from "./keyboard.js";
import { refreshStats, renderStats, resetStatsView } from "./stats.js";
import { postJSON } from "./api.js";
import { fitText, placeCaret, handleChar, backspace, restartRound, setTyping, loadExercise, roundStarted, pauseRound, checkLayout, cancelProbe } from "./typing.js";
import { checkCaps } from "./warn.js";
import { welcomeKey, onWelcomeStart, resetWelcome } from "./intro.js";
import { profileOpen, closeProfile } from "./profile.js";
import { resultOpen, closeResult } from "./result.js";
import { settingsOpen, openSettings, onSettingsClose, onSetting, setLength } from "./settings.js";
import { setMode, refreshFreetype, renderNav, setFreetypeList, setFreetypeVariant } from "./modes.js";
import { openCustomText, hasCustomText } from "./customtext.js";
import { onBarPick, renderModeBar } from "./modebar.js";
import { needsCheckin, runCheckin } from "./checkin.js";
import { renderLetters } from "./letters.js";
import { renderPanel } from "./learnpanel.js";
import { startGrow } from "./fx.js";

window.addEventListener("resize", () => {
  fitText();
  requestFit();
  placeCaret(true);
  if (S.view === "stats") renderStats();
});

// ---------- screens: welcome / typing / stats ----------
function setView(name) {
  if (S.view === "type" && name !== "type" && roundStarted()) pauseRound();
  S.view = name;
  document.body.dataset.view = name;
  const stats = name === "stats", welcome = name === "welcome";
  renderNav();
  el.viewType.hidden = stats || welcome;
  el.viewStats.hidden = !stats;
  el.intro.hidden = !welcome;
  if (welcome) {
    setTyping(false);
    resetWelcome();
  } else if (stats) {
    startGrow();
    resetStatsView();
    setTyping(false);
    renderStats();
  } else {
    fitText();
    requestFit();
    placeCaret(true);
    if (!resultOpen()) focusText();
  }
}
el.tabStats.addEventListener("click", (e) => { e.currentTarget.blur(); setView(S.view === "stats" ? "type" : "stats"); });
el.viewStats.addEventListener("click", (e) => { if (e.target.closest(".st-back")) setView("type"); });
el.tabWelcome.addEventListener("click", (e) => { e.currentTarget.blur(); if (S.view !== "welcome") setView("welcome"); });
export function openWelcome() { setView("welcome"); }

// ---------- modes: learn / freetype ----------
function chooseMode(m) {
  if (m !== S.mode) {
    cancelProbe();
    if (resultOpen()) {
      setMode(m);
      closeResult();
    } else if (S.busy) {
      return;
    } else {
      setMode(m);
      startMode().catch(console.error);
    }
  } else if (needsCheckin(m) && !resultOpen() && !S.busy) {
    startMode().catch(console.error);
  }
  if (S.view !== "type") setView("type");
}
// A mode's first round, after its check-in when it has not had one.
async function startMode() {
  for (let i = 0; i < 3 && needsCheckin(S.mode); i++) {
    if (!(await runCheckin(S.mode))) break;
  }
  await loadExercise();
}
el.tabLearn.addEventListener("click", (e) => { e.currentTarget.blur(); chooseMode("learn"); });
el.tabReinforce.addEventListener("click", (e) => { e.currentTarget.blur(); chooseMode("reinforce"); });
onWelcomeStart(() => chooseMode("learn"));
el.tabFreetype.addEventListener("click", (e) => { e.currentTarget.blur(); chooseMode("freetype"); });
document.getElementById("logo").addEventListener("click", () => {
  if (S.view !== "type") setView("type");
  else if (resultOpen()) closeResult();
});
async function barPick(pick) {
  if (S.busy && !resultOpen()) return;
  if (pick.edit || (pick.variant === "custom" && !hasCustomText())) {
    openCustomText(() => barPick({ variant: "custom" }).catch(console.error));
    return;
  }
  if (pick.variant !== undefined) await setFreetypeVariant(pick.variant);
  if (pick.list) await setFreetypeList(pick.list);
  if (pick.wordLevel) setWordLevel(pick.wordLevel);
  if (pick.step !== undefined) {
    await postJSON("/learn/step", { step: pick.step });
    await refreshStats();
    renderLetters();
  }
  if (pick.length) {
    setLength(S.mode, pick.length);
    await Promise.all([refreshStats(), refreshFreetype()]);
  }
  renderModeBar();
  if (resultOpen()) closeResult();
  else loadExercise().catch(console.error);
}
onBarPick(barPick);

onSetting((id, value) => {
  if (id === "nextKey" || id === "hand") {
    fitText();
    requestFit();
    placeCaret(true);
    highlightNextKey();
  } else if (id === "kb") {
    autoKb();
    highlightNextKey();
  } else if (id === "daily" || id === "goal") {
    renderPanel();
    if (S.view === "stats") renderStats();
  } else if (id === "length") {
    barPick({ length: value }).catch(console.error);
  }
});

// ---------- input ----------
const COOLDOWN_MS = 300;
const lastPress = {};
function shortcut(e, name) {
  e.preventDefault();
  const now = performance.now();
  if (e.repeat || now - (lastPress[name] ?? -Infinity) < COOLDOWN_MS) return false;
  lastPress[name] = now;
  return true;
}

// ~ opens the stats of the mode you're in and closes them; Esc and Tab close them too.
const STATS_KEY = "Backquote";
const plain = (e) => !e.ctrlKey && !e.metaKey && !e.altKey;

document.addEventListener("keydown", (e) => {
  if (document.querySelector("dialog[open]")) return;
  if (profileOpen()) {
    if (e.key === "Escape") { e.preventDefault(); closeProfile(); }
    return;
  }
  if (S.view === "stats" && plain(e) && (e.code === STATS_KEY || e.key === "Escape" || e.key === "Tab")) {
    if (!shortcut(e, "view")) return;
    setView("type");
    if (e.key === "Tab") { if (resultOpen()) closeResult("tab"); else restartRound(); }
    return;
  }
  if (e.key === "Escape") {
    if (!shortcut(e, "esc")) return;
    if (S.view === "type" && !resultOpen() && roundStarted()) pauseRound();
    openSettings();
    return;
  }
  if (S.view === "welcome") { welcomeKey(e); return; }
  const altGr = e.getModifierState?.("AltGraph");
  if (e.altKey && !e.ctrlKey && !e.metaKey && !altGr && e.code === "KeyK") {
    if (shortcut(e, "kb") && S.view === "type") toggleKb();
    return;
  }
  if (e.code === STATS_KEY && plain(e) && S.view === "type") {
    if (shortcut(e, "view")) setView("stats");
    return;
  }
  if (resultOpen()) {
    if (e.key === "Enter" || e.key === "Tab") { if (shortcut(e, "tab")) closeResult(e.key === "Enter" ? "enter" : "tab"); }
    else if (e.key.length === 1) e.preventDefault();
    return;
  }
  if (S.view !== "type") return;
  if (e.key === "Tab") { if (shortcut(e, "tab")) restartRound(); return; }
  if (e.key === "Backspace") {
    e.preventDefault(); focusText(); hidePointer(); backspace(e.ctrlKey || e.altKey);
    keyBackspace = true;
    requestAnimationFrame(() => { keyBackspace = false; });
    return;
  }
  if (e.repeat) { e.preventDefault(); return; }
  if ((e.ctrlKey || e.metaKey || e.altKey) && !altGr) return;
  if (e.key.length !== 1) return;
  e.preventDefault();
  focusText();
  hidePointer();
  handleChar(e.key);
});

// ---------- mouse pointer: hidden while typing, back on a real move ----------
let pointerHidden = false, lastX = -1, lastY = -1;
function hidePointer() {
  if (pointerHidden) return;
  pointerHidden = true;
  document.documentElement.classList.add("no-pointer");
}
addEventListener("pointermove", (e) => {
  if (e.screenX === lastX && e.screenY === lastY) return;
  lastX = e.screenX; lastY = e.screenY;
  if (!pointerHidden) return;
  pointerHidden = false;
  document.documentElement.classList.remove("no-pointer");
}, { passive: true });

addEventListener("blur", pauseRound);
document.addEventListener("visibilitychange", () => { if (document.hidden) pauseRound(); else checkLayout(); });
addEventListener("focus", checkLayout);
for (const ev of ["keydown", "keyup", "mousemove", "mousedown"]) addEventListener(ev, checkCaps, { passive: true });

const SENTINEL = " ";
let keyBackspace = false;
function resetInput() {
  el.input.value = SENTINEL;
  try { el.input.setSelectionRange(SENTINEL.length, SENTINEL.length); } catch {}
}
el.input.addEventListener("input", (e) => {
  const v = el.input.value;
  if (v.length < SENTINEL.length) {
    if (!keyBackspace) backspace(e.inputType === "deleteWordBackward");
  } else {
    for (const ch of v.startsWith(SENTINEL) ? v.slice(SENTINEL.length) : v) handleChar(ch);
  }
  keyBackspace = false;
  resetInput();
});
el.input.addEventListener("focus", resetInput);
resetInput();

let tapClosedPanel = false;
onSettingsClose(() => { lastPress.esc = performance.now(); if (S.view === "type" && !resultOpen() && !isTouch()) focusText(); });
document.addEventListener("pointerdown", () => {
  tapClosedPanel = settingsOpen() || profileOpen();
}, true);
el.textBox.addEventListener("click", () => {
  if (resultOpen() || tapClosedPanel) return;
  el.input.focus({ preventScroll: true }); focusText();
});
el.input.addEventListener("blur", () => { if (isTouch()) el.textBox.classList.add("blurred"); });
