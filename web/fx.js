// The toast.

// ---------- motion: opacity and a few pixels; off when the system asks for less motion ----------
const still = matchMedia("(prefers-reduced-motion: reduce)");
const EASE = "cubic-bezier(.2,.8,.2,1)";
export function fade(node, frames, ms) {
  if (still.matches || !node.animate) return Promise.resolve();
  return node.animate(frames, { duration: ms, easing: EASE }).finished.catch(() => {});
}

// One clock for a screen's charts: a chart drawn again while it grows picks up where it was.
let growStart = -1e9;
export function startGrow() { growStart = performance.now(); }
export function grow(nodes, frames, ms, step = 0) {
  if (still.matches) return;
  const t = performance.now() - growStart;
  if (t > ms + 400) return;
  nodes.forEach((n, i) => {
    const a = n.animate(frames, { duration: ms, delay: Math.min(i * step, 200), easing: EASE, fill: "backwards" });
    a.currentTime = t;
  });
}

export const FX = (() => {
  // ---------- messages: part of the interface, not pop-ups ----------
  const NOTE_MS = 3500;
  const INFO_ICON = '<circle cx="12" cy="12" r="8"/><path d="M12 11v5M12 8h.01"/>';
  const queue = [];
  let current = null;
  function toast(html, act) {
    queue.push({ html, act });
    if (!current) showNext();
  }
  function showNext() {
    const box = document.getElementById("notice");
    const item = queue.shift();
    if (!item) {
      current = null;
      return;
    }
    const node = document.createElement("div");
    node.className = "note";
    const mark = document.createElement("span");
    mark.className = "note-mark";
    mark.innerHTML = `<svg viewBox="0 0 24 24" aria-hidden="true">${INFO_ICON}</svg>`;
    node.innerHTML = `<span class="note-body">${item.html}</span>`;
    node.prepend(mark);
    if (item.act) node.querySelector(".note-act")?.addEventListener("click", item.act);
    node.addEventListener("click", () => closeNote(node));
    box.replaceChildren(node);
    fade(node, [{ opacity: 0, translate: "0 -4px" }, { opacity: 1, translate: "0 0" }], 180);
    current = node;
    setTimeout(() => closeNote(node), NOTE_MS);
  }
  function closeNote(node) {
    if (node !== current) return;
    current = null;
    fade(node, [{ opacity: 1 }, { opacity: 0, translate: "0 -4px" }], 120).then(() => {
      node.remove();
      if (!current) showNext();
    });
  }

  return { toast };
})();
