// Learn: accuracy. Letters open one at a time; a round is 30 words.

import { get } from "../core/api";
import type { LearnResult, LearnStats } from "../core/api-types";
import { app } from "../core/state.svelte";
import { LetterStream, ROUND_WORDS } from "./letters";
import type { Source } from "./source";

export async function refreshLearn() {
  app.learn = await get<LearnStats>("/api/learn/stats");
}

export function learnSource(): Source {
  const stream = new LetterStream("learn", "");
  return {
    mode: "learn",
    oneAtATime: true,
    maxMistakes: 0,
    length: () => ({ kind: "words", n: ROUND_WORDS }),
    start: () => stream.start(),
    next: (k, r) => stream.next(k, r, ROUND_WORDS),
    async finish(r) {
      const res = await stream.send<LearnResult>(r, "words-30");
      await refreshLearn();
      if (!res.newLetter) return { show: "next" };
      const finish = app.learn?.letters.every((l) => l.open) ?? false;
      return { show: "letter", letter: res.newLetter, finish };
    },
  };
}
