// The hidden tuning panel (Alt+P): Learn's main numbers on sliders, for the owner.

import { S, store, focusText } from "./state.js";
import { getJSON, postJSON } from "./api.js";
import { refreshStats } from "./stats.js";
import { renderLetters } from "./letters.js";

const KEY = "tt.tune";
const ROWS = {
  TargetWPM: ["Speed goal", "wpm", "The speed a letter aims for. Its level counts speed up to here."],
  MasteryLevel: ["Level to master", "", "A letter is mastered at this level. Lower: letters open sooner."],
  MasteryMinSamples: ["Presses to master", "", "How many presses a letter needs before it can be mastered."],
  ErrMemory: ["Mistakes remembered", "presses", "More: one slip moves the level less, and it takes longer to show a real problem."],
  TimeMemory: ["Speed remembered", "presses", "More: the speed changes slowly; fewer: it follows your last presses."],
  SpeedWeight: ["Speed in the level", "", "0: only mistakes count. 1: only speed counts."],
  Push: ["Push on weak letters", "", "More: the weakest letter comes in almost every word. Fewer: more variety."],
  RankPenalty: ["Rare words", "", "Lower: almost only common words. 1: rare words as often as common ones."],
};

const box = document.createElement("aside");
box.className = "tune";
box.hidden = true;
box.setAttribute("aria-label", "Tuning");
document.body.appendChild(box);

let params = [];
const saved = () => { try { return JSON.parse(store.get(KEY) || "{}"); } catch { return {}; } };
const own = () => Object.fromEntries(params.filter((p) => p.override).map((p) => [p.name, p.value]));

function draw() {
  box.innerHTML = `<header><b>Tuning</b><small>Alt+P</small></header>` + params.map((p) => {
    const [name, unit, tip] = ROWS[p.name] ?? [p.name, "", ""];
    return `<label class="tune-row${p.override ? " own" : ""}" title="${tip}">` +
      `<span class="tune-name">${name}</span><output>${p.value}${unit ? " " + unit : ""}</output>` +
      `<input type="range" data-name="${p.name}" min="${p.min}" max="${p.max}" step="${p.step}" value="${p.value}">` +
      `<small>${tip} Default ${p.default}.</small></label>`;
  }).join("") +
    `<footer><button type="button" class="btn" data-act="copy">Copy</button><button type="button" class="btn" data-act="reset">Defaults</button></footer>`;
}

let timer = 0;
async function send(values) {
  params = (await postJSON("/tune", { values })).params;
  store.set(KEY, JSON.stringify(own()));
  draw();
  await refreshStats();
  renderLetters();
}

box.addEventListener("input", (e) => {
  const input = e.target.closest("input[data-name]");
  if (!input) return;
  const p = params.find((x) => x.name === input.dataset.name);
  p.value = Number(input.value);
  p.override = true;
  input.previousElementSibling.textContent = `${p.value}${ROWS[p.name]?.[1] ? " " + ROWS[p.name][1] : ""}`;
  clearTimeout(timer);
  timer = setTimeout(() => send(own()).catch(console.error), 250);
});
box.addEventListener("click", (e) => {
  const act = e.target.closest("button")?.dataset.act;
  if (act === "reset") send({}).catch(console.error);
  if (act === "copy") {
    const line = params.map((p) => `${p.name}=${p.value}`).join(" ");
    navigator.clipboard?.writeText(line).catch(console.error);
    e.target.closest("button").textContent = "Copied";
  }
});

async function toggle() {
  box.hidden = !box.hidden;
  if (box.hidden) { focusText(); return; }
  params = (await getJSON("/tune")).params;
  draw();
}
document.addEventListener("keydown", (e) => {
  if (e.altKey && !e.ctrlKey && !e.metaKey && e.code === "KeyP") {
    e.preventDefault();
    if (!e.repeat) toggle().catch(console.error);
  }
});

// The server keeps the values in memory only: send this browser's copy again.
export async function restoreTune() {
  const values = saved();
  if (S.lang && Object.keys(values).length) await postJSON("/tune", { values }).catch(() => store.set(KEY, "{}"));
}
