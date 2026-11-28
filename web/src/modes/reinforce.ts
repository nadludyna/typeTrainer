// Reinforce: speed. Real words of a word level; the speed goal grows in
// steps, and the word level follows the rounds.

import { get } from "../core/api";
import type { ReinforceResult, ReinforceStats } from "../core/api-types";
import { app, keep } from "../core/state.svelte";
import { T } from "../core/texts";
import { toast } from "../ui/messages.svelte";
import { LetterStream, ROUND_WORDS } from "./letters";
import { lengthOf, type Source } from "./source";

const SPRINT = "time-15";

export async function refreshReinforce() {
  app.reinforce = await get<ReinforceStats>("/api/reinforce/stats");
}

export function setLevel(level: number) {
  app.level = level;
  keep("level", level);
}

export function reinforceSource(): Source {
  const kind = app.kind.reinforce;
  const level = app.level;
  const length = kind === "sprint" ? SPRINT : "words-30";
  const stream = new LetterStream("reinforce", kind, level);
  return {
    mode: "reinforce",
    oneAtATime: kind !== "sprint",
    maxMistakes: kind === "clean" ? 3 : 0,
    length: () => lengthOf(length),
    start: () => stream.start(),
    next: (k, r) => (kind === "sprint" ? stream.next(k, r, Number.MAX_SAFE_INTEGER) : stream.next(k, r, ROUND_WORDS)),
    async finish(r) {
      const res = await stream.send<ReinforceResult>(r, length);
      await refreshReinforce();
      if (res.newGoal) toast(T.goal.wpm(res.newGoal));
      followLevel(level, kind);
      if (app.reinforce?.tired) toast(T.tired);
      return { show: "next" };
    },
  };
}

// The word level follows its rounds, but only right after a usual round at
// the level in use, so a level picked by hand stays until rounds move it.
function followLevel(level: number, kind: string) {
  const s = app.reinforce;
  if (!s || kind || !s.standing.moved || level !== app.level || s.standing.level === level) return;
  const words = s.wordLevels[s.standing.level - 1];
  if (!words) return;
  setLevel(s.standing.level);
  toast(s.standing.moved > 0 ? T.levelUp(words) : T.levelDown(words));
}
