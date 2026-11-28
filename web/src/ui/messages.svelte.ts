// Short messages that float under the header and never push the page.

interface Toast {
  id: number;
  text: string;
  action?: { label: string; run: () => void };
}

const SHOWN_MS = 5000;
let next = 0;

export const toasts = $state<Toast[]>([]);

export function toast(text: string, action?: Toast["action"]) {
  const id = next++;
  toasts.push({ id, text, action });
  setTimeout(() => drop(id), action ? SHOWN_MS * 2 : SHOWN_MS);
}

export function drop(id: number) {
  const i = toasts.findIndex((t) => t.id === id);
  if (i >= 0) toasts.splice(i, 1);
}
