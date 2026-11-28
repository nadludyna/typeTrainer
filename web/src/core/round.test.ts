import { describe, expect, it } from "vitest";
import { Mark, Round } from "./round";

function newRound(words = 0) {
  const marks: Record<number, Mark> = {};
  const extras: Record<number, string[]> = {};
  const r = new Round({ mark: (i, m) => (marks[i] = m), extras: (at, l) => (extras[at] = l) }, words);
  return { r, marks, extras };
}

/** typeAll types s, one press every step ms from start. */
function typeAll(r: Round, s: string, start = 1000, step = 200) {
  let last = r.type(s[0] ?? "", start);
  [...s.slice(1)].forEach((ch, i) => {
    last = r.type(ch, start + (i + 1) * step);
  });
  return last;
}

describe("Round", () => {
  it("times every press but the first, and counts right characters", () => {
    const { r } = newRound();
    r.append([["ab", "cd"]]);
    typeAll(r, "ab c");
    expect(r.typedSamples().map((s) => s.ms)).toEqual([0, 200, 200, 200]);
    expect(r.rightChars).toBe(4);
    expect(r.accuracy()).toBe(100);
  });

  it("moves on after a wrong letter and keeps the wrong key", () => {
    const { r, marks } = newRound();
    r.append([["ab"]]);
    typeAll(r, "xb");
    expect(marks[0]).toBe(Mark.Wrong);
    expect(r.samples[0]).toEqual({ char: "a", errors: 1, ms: 0, typed: "x" });
    expect(r.accuracy()).toBe(50);
    expect(r.mistyped(0)).toEqual(["ab"]);
  });

  it("skips the rest of a word on a space, and counts each skipped letter as a miss", () => {
    const { r, marks } = newRound();
    r.append([["abc", "de"]]);
    typeAll(r, "a d");
    expect([marks[1], marks[2]]).toEqual([Mark.Skipped, Mark.Skipped]);
    expect(r.pos).toBe(5);
    expect(r.presses).toBe(5);
    expect(r.typedSamples().map((s) => s.char)).toEqual(["a", " ", "d"]);
  });

  it("keeps letters typed where a space belongs as extras, errors of that space", () => {
    const { r, extras } = newRound();
    r.append([["ab", "c"]]);
    typeAll(r, "abxy ");
    expect(extras[2]).toEqual(["x", "y"]);
    expect(r.samples[2]?.errors).toBe(2);
    r.backspace(false);
    r.backspace(false);
    expect(extras[2]).toEqual(["x"]);
  });

  it("ignores a space at the start of a word", () => {
    const { r } = newRound();
    r.append([["ab"]]);
    expect(r.type(" ", 0)).toEqual({ wordDone: false, portionDone: -1, roundDone: false });
    expect(r.pos).toBe(0);
  });

  it("goes back one word at most, and a fixed mistake still counts", () => {
    const { r, marks } = newRound();
    r.append([["ab", "cd", "ef"]]);
    typeAll(r, "xb cd ");
    r.backspace(false); // into "cd"
    r.backspace(true); // the whole of "cd"
    r.backspace(false); // not further back
    expect(r.pos).toBe(3);
    expect(marks[3]).toBe(Mark.Untyped);
    expect(r.samples[0]?.errors).toBe(1);
    expect(r.wordsTyped).toBe(1);
  });

  it("does not time the press after a Backspace", () => {
    const { r } = newRound();
    r.append([["abc"]]);
    r.type("a", 0);
    r.type("x", 100);
    r.backspace(false);
    r.type("b", 900);
    expect(r.samples[1]).toEqual({ char: "b", errors: 1, ms: 100, typed: "x" });
    r.type("c", 1000);
    expect(r.samples[2]?.ms).toBe(100);
  });

  it("a word typed again clean after going back does not come back", () => {
    const { r } = newRound();
    r.append([["ab", "cd"]]);
    typeAll(r, "ab c");
    r.backspace(false);
    r.backspace(false);
    typeAll(r, " cd ", 5000);
    expect(r.mistyped(0)).toEqual([]);
  });

  it("tells when a portion and a words round are done", () => {
    const { r } = newRound(3);
    r.append([
      ["ab", "cd"],
      ["ef", "gh"],
    ]);
    expect(r.endAt).toBe(8);
    expect(typeAll(r, "ab cd ").portionDone).toBe(0);
    const last = typeAll(r, "ef", 9000);
    expect(last.roundDone).toBe(true);
    expect(r.wordsTyped).toBe(3);
  });

  it("takes the time of a pause out of the round", () => {
    const { r } = newRound();
    r.append([["abcd"]]);
    r.type("a", 0);
    r.type("b", 1000);
    r.pause(1000);
    r.type("c", 61_000);
    expect(r.samples[2]?.ms).toBe(0);
    expect(r.lastTime).toBe(61_000);
    expect(r.startTime).toBe(60_000);
    expect(r.wpm()).toBeCloseTo(3 / 5 / (1 / 60));
  });

  it("gives each word its speed", () => {
    const { r } = newRound();
    r.append([["ab", "cd"]]);
    typeAll(r, "ab cd", 0, 100);
    expect(r.wordSpeeds()).toEqual([
      { word: "ab", wrong: false, wpm: 120 },
      { word: "cd", wrong: false, wpm: 120 },
    ]);
  });

  it("counts the keypresses of every full second", () => {
    const { r } = newRound();
    r.append([["abcdef"]]);
    typeAll(r, "abcdef", 0, 400);
    expect(r.seconds()).toEqual([3, 2]);
  });
});
