// Freetype: rhythm. Words of a word list, by words or by seconds; the list
// follows the consistency of the rounds.

import { get, post, roundId, save } from "../core/api";
import type { FreetypePortions, FreetypeResult, FreetypeStats, Result } from "../core/api-types";
import { app, keep } from "../core/state.svelte";
import { T } from "../core/texts";
import { toast } from "../ui/messages.svelte";
import { lengthOf, type Source, TIME_AHEAD_WORDS } from "./source";

const SPRINT = "time-15";

export async function refreshFreetype() {
  app.freetype = await get<FreetypeStats>(`/api/freetype/stats?list=${app.list}`);
  if (app.freetype.list !== app.list) setList(app.freetype.list);
}

export function setList(list: number) {
  app.list = list;
  keep("list", list);
}

export function freetypeSource(): Source {
  const kind = app.kind.freetype;
  const list = app.list;
  const lengthId = kind === "sprint" ? SPRINT : app.length;
  const length = lengthOf(lengthId);
  let id = ""; // each round gets its own, at its start
  const ask = async (body: object) =>
    (await post<FreetypePortions>("/api/freetype/portion", { list, kind, recent: [], start: false, words: 0, ...body }))
      .portions;
  return {
    mode: "freetype",
    oneAtATime: false,
    maxMistakes: kind === "clean" ? 3 : 0,
    length: () => length,
    start: () => {
      id = roundId();
      return ask({ start: true, words: length.kind === "words" ? length.n : TIME_AHEAD_WORDS });
    },
    next: async (_k, r) => (length.kind === "time" ? ask({ recent: r.recent() }) : []),
    async finish(r, why) {
      const keys = r.typedSamples();
      const before = app.freetype?.history.bests[lengthId]?.wpm ?? 0;
      const body: Result = {
        id,
        samples: [],
        keys,
        chars: keys.length,
        wpm: Math.round(r.wpm() * 10) / 10,
        accuracy: Math.round(r.accuracy() * 10) / 10,
        perSecond: r.seconds(),
        length: lengthId,
        kind,
        level: 0,
        list,
      };
      const res = await save<FreetypeResult>("/api/freetype/results", body);
      await refreshFreetype();
      followList(list, kind);
      if (app.freetype?.tired) toast(T.tired);
      const words = r.wordSpeeds().map((w, i) => ({ ...w, why: why[i] ?? "" }));
      const seconds = r.startTime !== null && r.lastTime !== null ? (r.lastTime - r.startTime) / 1000 : 0;
      return {
        show: "result",
        result: {
          wpm: body.wpm,
          accuracy: body.accuracy,
          consistency: res.consistency,
          words,
          seconds,
          best: before > 0 && body.wpm > before,
        },
      };
    },
  };
}

// The word list follows its rounds as Reinforce's level does.
function followList(list: number, kind: string) {
  const s = app.freetype;
  if (!s || kind || !s.standing.moved || list !== app.list) return;
  const next = s.lists[s.standing.level - 1];
  if (!next || next === list) return;
  setList(next);
  toast(s.standing.moved > 0 ? T.levelUp(next) : T.levelDown(next));
}
