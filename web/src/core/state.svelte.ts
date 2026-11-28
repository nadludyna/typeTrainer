// What the whole page shares: the mode, the choices kept in the browser,
// the layout and each mode's stats. One object, so any part can read it and
// Svelte redraws exactly what reads a changed field.

import type { FreetypeStats, Layout, LearnStats, ReinforceStats } from "./api-types";

export type Mode = "learn" | "reinforce" | "freetype";
export type Screen = "welcome" | Mode;
export type Kind = "" | "weak" | "clean" | "sprint";

/** kept reads a choice kept in the browser (storage may be blocked: then def). */
function kept<T>(key: string, def: T): T {
  try {
    const v = localStorage.getItem(`tt.${key}`);
    return v === null ? def : (JSON.parse(v) as T);
  } catch {
    return def;
  }
}

export function keep(key: string, value: unknown) {
  try {
    localStorage.setItem(`tt.${key}`, JSON.stringify(value));
  } catch {}
}

class State {
  screen = $state<Screen>(kept("screen", "welcome"));
  layout = $state<Layout | null>(null);
  learn = $state<LearnStats | null>(null);
  reinforce = $state<ReinforceStats | null>(null);
  freetype = $state<FreetypeStats | null>(null);
  statsOpen = $state(false);
  settingsOpen = $state(false);

  /** keyboard: shown in each mode, by default only in Learn. */
  keyboard = $state<Record<Mode, boolean>>(kept("keyboard", { learn: true, reinforce: false, freetype: false }));
  /** kind: the kind of round of each mode; Learn has only the usual one. */
  kind = $state<Record<Mode, Kind>>(kept("kind", { learn: "", reinforce: "", freetype: "" }));
  level = $state<number>(kept("level", 1)); // Reinforce's word level, 1..
  list = $state<number>(kept("list", 200)); // Freetype's word list
  length = $state<string>(kept("length", "words-30")); // Freetype's round length
  checkedIn = $state<boolean>(kept("checkedIn", false)); // Learn's first question is answered
  welcomed = $state<boolean>(kept("welcomed", false));

  go(screen: Screen) {
    this.screen = screen;
    this.statsOpen = false;
    keep("screen", screen);
  }
}

export const app = new State();
