// What every mode gives the typing screen: where its words come from and what
// happens at the end of a round. The typing screen knows no mode; a mode is
// one of these (learn.ts, reinforce.ts, freetype.ts), picked in App.svelte.

import type { Round } from "../core/round";
import type { Mode } from "../core/state.svelte";

/** A few words, the letter they lean on and why each came. */
export interface Piece {
  words: string[];
  focus?: string;
  why?: string[];
}

export interface Length {
  kind: "words" | "time";
  n: number;
}

/** What the end of a round leads to: the next round, or a result to look at first. */
export type After =
  | { show: "next" }
  | { show: "result"; result: Shown }
  | { show: "letter"; letter: string; finish: boolean };

/** A Freetype result as its card shows it. */
export interface Shown {
  wpm: number;
  accuracy: number;
  consistency: number | null;
  words: { word: string; wrong: boolean; wpm: number | null; why: string }[];
  seconds: number;
  best: boolean;
}

export interface Source {
  mode: Mode;
  length(): Length;
  /** Learn and Reinforce show one word more for each word typed; Freetype shows all. */
  oneAtATime: boolean;
  /** The round ends at this many mistakes (a clean round), else never. */
  maxMistakes: number;
  start(): Promise<Piece[]>;
  /** next is the words after portion k is typed; none when the round has all it needs. */
  next(k: number, r: Round): Promise<Piece[]>;
  finish(r: Round, why: string[]): Promise<After>;
}

/** lengthOf reads a length id: "words-30", "time-60". */
export function lengthOf(id: string): Length {
  const [kind, n] = id.split("-");
  return { kind: kind === "time" ? "time" : "words", n: Number(n) || 30 };
}

/** Freetype asks for this many words at the start of a time round, then a portion per portion typed. */
export const TIME_AHEAD_WORDS = 36;
