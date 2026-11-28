// One round as it is typed, with no screen: the text, where the typist is,
// what each position became and the press measured there. The typing screen
// draws it; the tests drive it with times of their own.
//
// The rules: typing never stops on a mistake (a wrong letter turns red and
// the caret moves on); space in the middle of a word skips its rest; letters
// typed where a space belongs are extra; Backspace goes back inside the word,
// and at most one word back. Every typed position keeps one sample: its
// errors (a mistake fixed with Backspace still counts) and the time from the
// previous press to its first press. The first press of a round, after a
// Backspace and after a pause is not timed (0).

import type { Sample } from "./api-types";

export const Mark = { Untyped: 0, Right: 1, Wrong: 2, Skipped: 3 } as const;
export type Mark = (typeof Mark)[keyof typeof Mark];

/** Called when a position's mark or the extra letters after a word change. */
export interface Draw {
  mark(i: number, m: Mark): void;
  extras(at: number, letters: string[]): void;
}

/** What a press did. */
export interface Typed {
  wordDone: boolean; // a word was finished
  portionDone: number; // the portion whose last word was finished, -1 for none
  roundDone: boolean; // the round's last word was finished
}

const MAX_EXTRA = 10;

interface Portion {
  start: number;
  end: number; // the position of the space after its last word
}

export class Round {
  text = "";
  pos = 0;
  marks: Mark[] = [];
  samples: (Sample | null)[] = [];
  portions: Portion[] = [];
  wordsTyped = 0;
  wordsLoaded = 0;
  /** The position after the round's last letter, -1 until the words are there. */
  endAt = -1;

  startTime: number | null = null;
  lastTime: number | null = null;
  presses = 0;
  rightPresses = 0;
  rightChars = 0;
  mistakes = 0;
  perSecond: number[] = [];

  private extraLetters = new Map<number, string[]>();
  private untimed = false;
  private pausedAt: number | null = null;
  private backWord = -1;
  private backErrors = 0;
  private fixed = new Set<number>();

  constructor(
    private draw: Draw,
    private words = 0, // a words round: its length; 0 for a time round
  ) {}

  /** append adds portions of words to the text, each followed by a space. */
  append(portions: string[][]) {
    for (const words of portions) {
      if (!words.length) continue;
      const start = this.text.length;
      const body = words.join(" ");
      this.text += `${body} `;
      for (let i = start; i < this.text.length; i++) {
        this.marks.push(Mark.Untyped);
        this.samples.push(null);
      }
      this.portions.push({ start, end: start + body.length });
      if (this.words && this.endAt < 0 && this.wordsLoaded + words.length >= this.words) {
        this.endAt = start + words.slice(0, this.words - this.wordsLoaded).join(" ").length;
      }
      this.wordsLoaded += words.length;
    }
  }

  get expected(): string | undefined {
    return this.text[this.pos];
  }

  get done(): boolean {
    return this.pos >= this.text.length;
  }

  wordStart(i: number) {
    while (i > 0 && this.text[i - 1] !== " ") i--;
    return i;
  }

  wordEnd(i: number) {
    while (i < this.text.length && this.text[i] !== " ") i++;
    return i;
  }

  /** type is one press of ch, a letter of the alphabet or a space, at now (ms). */
  type(ch: string, now: number): Typed {
    const out: Typed = { wordDone: false, portionDone: -1, roundDone: false };
    const expected = this.expected;
    if (expected === undefined) return out;
    if (ch === " " && expected !== " " && this.pos === this.wordStart(this.pos)) return out;
    if (this.pausedAt !== null) this.resume(now);
    if (this.startTime === null) this.startTime = now;
    this.presses++;
    const second = Math.floor((now - this.startTime) / 1000);
    this.perSecond[second] = (this.perSecond[second] ?? 0) + 1;

    if (expected === " " && ch !== " ") {
      this.record(this.pos, true, now, ch);
      const list = this.extraLetters.get(this.pos) ?? [];
      if (list.length < MAX_EXTRA) {
        list.push(ch);
        this.extraLetters.set(this.pos, list);
        this.draw.extras(this.pos, [...list]);
      }
    } else {
      if (ch === " " && expected !== " ") {
        const end = this.wordEnd(this.pos);
        this.presses += end - this.pos;
        while (this.pos < end) this.setMark(this.pos++, Mark.Skipped);
      }
      const ok = ch === this.text[this.pos];
      this.record(this.pos, !ok, now, ch);
      this.setMark(this.pos, ok ? Mark.Right : Mark.Wrong);
      if (ok) this.rightPresses++;
      const space = this.text[this.pos] === " ";
      if (space) {
        this.leaveBackWord();
        this.endWord(out);
      }
      this.pos++;
      const at = this.pos - 1;
      if (space && at === this.endAt) out.roundDone = true;
      else if (space) out.portionDone = this.portions.findIndex((p) => p.end === at);
      else if (this.pos === this.endAt) {
        this.endWord(out);
        out.roundDone = true;
      }
    }
    this.lastTime = now;
    this.untimed = false;
    return out;
  }

  /** backspace takes back one letter, or with whole the word so far. */
  backspace(whole: boolean) {
    if (this.startTime === null) return;
    if (this.pausedAt !== null) this.resume(this.pausedAt);
    if (this.pos === this.wordStart(this.pos)) {
      const space = this.pos - 1;
      if (space < 0 || this.backWord >= 0) return;
      this.setMark(space, Mark.Untyped);
      this.pos = space;
      while (this.pos > 0 && this.marks[this.pos - 1] === Mark.Skipped) this.setMark(--this.pos, Mark.Untyped);
      this.backWord = this.wordStart(space);
      this.backErrors = this.errorsIn(this.backWord, space);
      this.wordsTyped--;
      if (whole) this.clearWord();
    } else if (whole) {
      this.clearWord();
    } else if (this.extraLetters.get(this.pos)?.length) {
      const list = this.extraLetters.get(this.pos) ?? [];
      list.pop();
      this.draw.extras(this.pos, [...list]);
    } else {
      this.setMark(--this.pos, Mark.Untyped);
    }
    this.untimed = true;
  }

  /** pause stops the clock at since; the next press takes the time away out. */
  pause(since: number) {
    if (this.pausedAt === null && this.startTime !== null) this.pausedAt = since;
  }

  get paused() {
    return this.pausedAt !== null;
  }

  private resume(now: number) {
    const away = now - (this.pausedAt ?? now);
    if (this.startTime !== null) this.startTime += away;
    if (this.lastTime !== null) this.lastTime += away;
    this.pausedAt = null;
    this.untimed = true;
  }

  private record(i: number, wrong: boolean, now: number, typed: string) {
    let s = this.samples[i];
    if (!s) {
      const ms = this.lastTime === null || this.untimed ? 0 : Math.round(now - this.lastTime);
      s = { char: this.text[i] ?? "", errors: 0, ms };
      this.samples[i] = s;
    }
    if (wrong) {
      this.mistakes++;
      s.errors++;
      if (s.typed === undefined) s.typed = typed;
    }
  }

  private setMark(i: number, m: Mark) {
    if (this.marks[i] === Mark.Right) this.rightChars--;
    if (m === Mark.Right) this.rightChars++;
    this.marks[i] = m;
    this.draw.mark(i, m);
  }

  private endWord(out: Typed) {
    this.wordsTyped++;
    out.wordDone = true;
  }

  private errorsIn(start: number, end: number) {
    let n = 0;
    for (let j = start; j <= end; j++) n += this.samples[j]?.errors ?? 0;
    return n;
  }

  private hasError(start: number, end: number) {
    for (let j = start; j < end; j++) if ((this.marks[j] ?? 0) >= Mark.Wrong) return true;
    return (this.extraLetters.get(end)?.length ?? 0) > 0;
  }

  // A word gone back into and typed again with no new mistake is practised:
  // it does not come back as a mistyped word.
  private leaveBackWord() {
    if (this.backWord < 0 || this.wordStart(this.pos) !== this.backWord) return;
    if (this.errorsIn(this.backWord, this.pos) === this.backErrors && !this.hasError(this.backWord, this.pos)) {
      this.fixed.add(this.backWord);
    }
    this.backWord = -1;
  }

  private clearWord() {
    if (this.extraLetters.get(this.pos)?.length) {
      this.extraLetters.set(this.pos, []);
      this.draw.extras(this.pos, []);
    }
    const start = this.wordStart(this.pos);
    while (this.pos > start) this.setMark(--this.pos, Mark.Untyped);
  }

  // ---------- what the round sends and shows ----------

  /** samples of the positions from..to (to excluded) that were typed. */
  typedSamples(from = 0, to = this.samples.length): Sample[] {
    return this.samples.slice(from, to).filter((s): s is Sample => s !== null);
  }

  /** mistyped is the words of a portion typed with a mistake, to come back. */
  mistyped(k: number): string[] {
    const p = this.portions[k];
    if (!p) return [];
    const out: string[] = [];
    for (let i = p.start; i <= p.end; ) {
      const end = this.wordEnd(i);
      let bad = false;
      for (let j = i; j <= end; j++) {
        const s = this.samples[j];
        if (s ? s.errors > 0 : j < end) bad = true;
      }
      if (bad && !this.fixed.has(i)) out.push(this.text.slice(i, end));
      i = end + 1;
    }
    return out;
  }

  recent(n = 40): string[] {
    return this.text.trim().split(" ").slice(-n);
  }

  wpm(end = this.lastTime): number {
    if (this.startTime === null || end === null) return 0;
    const minutes = (end - this.startTime) / 60000;
    return minutes > 0 ? this.rightChars / 5 / minutes : 0;
  }

  accuracy(): number {
    return this.presses ? (this.rightPresses / this.presses) * 100 : 100;
  }

  /** seconds is every full second of the round's keypresses, a quiet one as 0. */
  seconds(end = this.lastTime): number[] {
    const n = this.startTime === null || end === null ? 0 : Math.floor((end - this.startTime) / 1000);
    return Array.from({ length: n }, (_, i) => this.perSecond[i] ?? 0);
  }

  /** words is each typed word with its letters and its speed, for the result. */
  wordSpeeds(): { word: string; wrong: boolean; wpm: number | null }[] {
    const out = [];
    for (let start = 0; start < this.pos; ) {
      const end = this.wordEnd(start);
      let ms = 0;
      let n = 0;
      let wrong = false;
      for (let i = start; i <= Math.min(end, this.pos - 1); i++) {
        const s = this.samples[i];
        if (i < end && (s?.errors ?? 0) > 0) wrong = true;
        if (s && s.ms > 0) {
          ms += s.ms;
          n++;
        }
      }
      if (start < Math.min(end, this.pos)) {
        out.push({ word: this.text.slice(start, end), wrong, wpm: n ? Math.round(12000 / (ms / n)) : null });
      }
      start = end + 1;
    }
    return out;
  }
}
