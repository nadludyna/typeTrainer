// The words of Learn and Reinforce: portions the server builds from the
// letter model, one request per portion typed, each with the round's id so
// a request sent twice gets the same answer.

import { post, roundId, save } from "../core/api";
import type { PortionRequest, Portions, Result } from "../core/api-types";
import type { Round } from "../core/round";
import type { Kind } from "../core/state.svelte";
import type { Piece } from "./source";

export const ROUND_WORDS = 30;

// One stream gives every round of a source: each round starts with a new id.
export class LetterStream {
  private id = "";
  private received = 0;

  constructor(
    private path: "learn" | "reinforce",
    private kind: Kind,
    private level = 0,
  ) {}

  private async ask(req: Partial<PortionRequest>): Promise<Piece[]> {
    const body: PortionRequest = {
      round: this.id,
      index: 0,
      ack: this.received,
      start: false,
      last: false,
      recent: [],
      retry: [],
      kind: this.kind,
      level: this.level,
      ...req,
    };
    const out = await post<Portions>(`/api/${this.path}/portion`, body);
    this.received += out.portions.length;
    return out.portions;
  }

  start() {
    this.id = roundId();
    this.received = 0;
    return this.ask({ start: true });
  }

  next(k: number, r: Round, words: number) {
    const last = r.wordsLoaded >= words;
    return this.ask({ index: k + 1, recent: r.recent(), retry: r.mistyped(k), last });
  }

  /** result is the round's end as the server takes it. */
  result(r: Round, length: string): Result {
    const keys = r.typedSamples();
    return {
      id: this.id,
      samples: [],
      keys,
      chars: keys.length,
      wpm: Math.round(r.wpm() * 10) / 10,
      accuracy: Math.round(r.accuracy() * 10) / 10,
      perSecond: r.seconds(),
      length,
      kind: this.kind,
      level: this.level,
      list: 0,
    };
  }

  send<T>(r: Round, length: string) {
    return save<T>(`/api/${this.path}/results`, this.result(r, length));
  }
}
