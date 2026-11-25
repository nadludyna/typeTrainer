// The modes and their tabs.

import { el, S, store, shownMode, statsShown } from "./state.js";
import { getJSON } from "./api.js";
import { autoKb } from "./keyboard.js";
import { lengthFor } from "./settings.js";
import { renderModeBar } from "./modebar.js";
import { hasCustomText } from "./customtext.js";

const MODES = ["learn", "reinforce", "freetype"];

// Learn and Reinforce share the letter model and its routes (/portion, /results).
export const learning = () => S.mode === "learn" || S.mode === "reinforce";

export const freetypeUrl = (length, list, variant) =>
  `/freetype/stats?length=${length}${list ? `&list=${list}` : ""}${variant ? `&variant=${variant}` : ""}`;

export async function refreshFreeplay() {
  const length = lengthFor("freetype");
  const list = Number(store.get(`tt.list.${S.lang}`)) || 0;
  let variant = store.get(`tt.variant.${S.lang}`) ?? "";
  if (variant === "custom" && !hasCustomText()) variant = "";
  if (variant === "sentences" && S.freetype && !S.freetype.sentences) variant = "";
  const freetype = await getJSON(freetypeUrl(length, list, variant))
    .catch(() => getJSON(freetypeUrl(length, 0, "")));
  S.freetype = freetype;
  S.freetypeList = freetype.list || S.freetypeList || list || freetype.lists?.[0] || 0;
  S.freetypeVariant = freetype.variant ?? "";
  renderNav();
  renderModeBar();
}

export async function setFreetypeList(n) {
  S.freetypeList = n;
  store.set(`tt.list.${S.lang}`, String(n));
  S.freetype = await getJSON(freetypeUrl(lengthFor("freetype"), n, S.freetypeVariant));
}

export async function setFreetypeVariant(v) {
  S.freetypeVariant = v;
  store.set(`tt.variant.${S.lang}`, v);
  S.freetype = await getJSON(freetypeUrl(lengthFor("freetype"), S.freetypeList, v));
}

export function restoreMode() {
  const saved = store.get(`tt.mode.${S.lang}`);
  S.mode = shownMode(saved) ? saved : "learn";
  S.settings.length = lengthFor(S.mode);
  renderModeBar();
  renderNav();
  autoKb();
}

export function setMode(m) {
  S.mode = m;
  S.settings.length = lengthFor(m);
  renderModeBar();
  store.set(`tt.mode.${S.lang}`, m);
  renderNav();
  autoKb();
}


export function renderNav() {
  el.tabStats.hidden = !statsShown();
  const tabs = [el.tabWelcome, el.tabLearn, el.tabReinforce, el.tabFreetype, el.tabStats];
  const mine = S.view === "stats" ? el.tabStats : S.view === "welcome" ? el.tabWelcome : tabs[MODES.indexOf(S.mode) + 1];
  const shown = tabs.filter((b) => !b.hidden);
  el.nav.style.setProperty("--n", shown.length);
  el.nav.style.setProperty("--i", shown.indexOf(mine));
  tabs.forEach((b) => b.setAttribute("aria-selected", String(b === mine)));
  document.body.dataset.mode = S.mode;
}
