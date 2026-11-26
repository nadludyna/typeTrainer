// Learn's part of its stats (stats.js): every letter round by round, and the keys you mix up.

import { t } from "./i18n.js";
import { letterProgress } from "./letters.js";

const CELL_MIN = 6, CELL_MAX = 16, GAP = 2, LABEL = 28;

// learnGrid: one row per open letter, one cell per round, as wide as `width`.
export function learnGrid(rounds, order, width) {
  const open = rounds.at(-1).unlocked;
  const cell = Math.max(CELL_MIN, Math.min(CELL_MAX, Math.floor((width - LABEL + GAP) / rounds.length) - GAP));
  const fit = Math.max(10, Math.floor((width - LABEL + GAP) / (cell + GAP)));
  const shown = rounds.slice(-fit), first = rounds.length - shown.length + 1;
  const rows = order.slice(0, open).map((c, i) => {
    const cells = shown.map((r, j) => {
      const k = r.letters[i];
      if (!k) return `<i class="off"></i>`;
      const p = k.mastered ? 1 : letterProgress(k);
      const tip = t(k.mastered ? "ls.mastered" : "ls.cellTip", { n: first + j, l: c, p: Math.round(p * 100) });
      return `<i class="${k.mastered ? "done" : k.count ? "" : "fresh"}" style="--p:${p.toFixed(2)}" data-tip-text="${tip}"></i>`;
    }).join("");
    return `<b class="ls-l">${c}</b><div class="ls-row">${cells}</div>`;
  }).join("");
  const scope = shown.length === rounds.length ? t("chart.all") : t("chart.last", { n: shown.length });
  return { scope, html: `<div class="ls-grid" style="--cell:${cell}px;--gap:${GAP}px">${rows}</div>` };
}

export function mixUps(list) {
  if (!list.length) return `<p class="st-none">${t("ls.noMix")}</p>`;
  const key = (c) => (c === " " ? "␣" : c);
  return `<ul class="ls-mix">${list.map((c) =>
    `<li><b>${key(c.want)}</b><span>${t("ls.instead")}</span><b class="bad">${key(c.typed)}</b><small>${t("ls.times", { n: c.n })}</small></li>`).join("")}</ul>`;
}
