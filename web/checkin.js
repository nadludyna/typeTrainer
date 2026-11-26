// The check-in when a mode is first opened: one question (Learn) or a
// 30-second sample (Reinforce, Freetype) sets where the mode starts. The
// rules live on the server (/checkin, docs/HOW-THE-MODES-WORK.md section 0.6);
// the sample's presses are sent nowhere.

import { t } from "./i18n.js";
import { S, store, setWordLevel, listLabel } from "./state.js";
import { postJSON } from "./api.js";
import { showCard } from "./result.js";
import { runProbe } from "./typing.js";
import { refreshStats } from "./stats.js";
import { setMode, setFreetypeList, refreshFreeplay } from "./modes.js";

const MODES = ["learn", "reinforce", "freetype"];
const SAMPLE = { reinforce: 200, freetype: 1000 };
const SAMPLE_SECONDS = 30;
const key = (mode) => `tt.checkin.${mode}`;
const markDone = (mode) => store.set(key(mode), "1");

// A mode needs its check-in once, and only while it has no rounds of its own.
export function needsCheckin(mode) {
  if (!MODES.includes(mode) || store.get(key(mode)) === "1") return false;
  const had = mode === "learn" ? (S.lastStats?.learnRounds ?? 0) > 0
    : mode === "reinforce" ? (S.lastStats?.sessions ?? []).some((s) => s.level > 0)
    : (S.freetype?.sessionCount ?? 0) > 0;
  if (had) { markDone(mode); return false; }
  return true;
}

const button = (act, text, primary = false) =>
  `<button type="button" class="${primary ? "primary" : ""}" data-act="${act}">${text}</button>`;
const skip = `<button type="button" class="link" data-act="skip">${t("ci.skip")}</button>`;
const card = (kick, title, text, acts) =>
  `<p class="ci-kick">${kick}</p><h2>${title}</h2><p>${text}</p><div class="fin-acts">${acts}</div>`;
const num = (v, unit) => `<span class="ci-big">${Math.round(v)}<small>${unit}</small></span>`;

// Runs the mode's check-in; true when it is done (answered or skipped),
// false when something else closed it, so it comes back next time.
export async function runCheckin(mode) {
  if (mode === "learn") return learnCheckin();
  return sampleCheckin(mode);
}

async function learnCheckin() {
  const a = await showCard(card(t("ci.kick"), t("ci.learnTitle"), t("ci.learnText"),
    button("no", t("ci.no")) + button("little", t("ci.little")) + button("yes", t("ci.yes")) + skip), "no");
  if (a === "skip" || a === "no") { markDone("learn"); return true; }
  if (a !== "little" && a !== "yes") return false;
  await postJSON("/checkin", { mode: "learn", answer: a });
  await refreshStats();
  markDone("learn");
  if (a === "yes") {
    const b = await showCard(card(t("ci.kick"), t("ci.yesTitle"), t("ci.yesText"),
      button("reinforce", t("ci.toReinforce"), true) + button("stay", t("ci.stayLearn"))), "reinforce");
    if (b === "reinforce") setMode("reinforce");
  }
  return true;
}

async function sampleCheckin(mode) {
  const name = t("mode." + mode), words = SAMPLE[mode];
  const a = await showCard(card(t("ci.sampleKick", { mode: name, s: SAMPLE_SECONDS }), t("ci.sampleTitle"),
    t("ci.sampleText", { n: words.toLocaleString("en"), s: SAMPLE_SECONDS }),
    button("start", t("ci.start"), true) + skip), "start");
  if (a === "skip") { markDone(mode); return true; }
  if (a !== "start") return false;
  for (;;) {
    const r = await runProbe(words, SAMPLE_SECONDS);
    if (!r) return false;
    const out = await postJSON("/checkin", { mode, wpm: Math.min(r.wpm, 400), accuracy: r.accuracy });
    let text;
    if (mode === "reinforce") {
      setWordLevel(out.wordLevel);
      await refreshStats();
      text = t("ci.reResult", { goal: Math.round(out.targetWpm), words: listLabel(S.lastStats?.wordLevels?.[out.wordLevel - 1] ?? 0) });
    } else {
      await setFreetypeList(out.list);
      if (r.consistency != null) store.set(`tt.rhythm.${S.lang}`, String(Math.round(r.consistency)));
      await refreshFreeplay();
      text = r.consistency != null
        ? t("ci.ftResult", { list: listLabel(out.list), rhythm: Math.round(r.consistency) })
        : t("ci.ftNoRhythm", { list: listLabel(out.list) });
    }
    const nums = num(r.wpm, "wpm") + num(r.accuracy, "%") + (mode === "freetype" && r.consistency != null ? num(r.consistency, t("ci.rhythm")) : "");
    const b = await showCard(`<div class="ci-pair">${nums}</div><p>${text}</p><div class="fin-acts">` +
      button("go", t("ci.go", { mode: name }), true) + button("again", t("ci.again")) + `</div>`, "go");
    if (b === "again") continue;
    markDone(mode);
    return b === "go" || b === "tab";
  }
}
