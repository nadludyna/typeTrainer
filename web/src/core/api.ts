// The API: JSON over HTTP to the Go server, or on GitHub Pages to the same Go
// program running in the page (pages.ts). The shapes are api-types.ts.

import { pagesCall } from "./pages";

declare const __PAGES__: boolean;

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
  }
  /** A try again may work: no answer, or the server failed (5xx); never a 4xx. */
  get retryable() {
    return this.status === 0 || this.status >= 500;
  }
}

async function call<T>(method: "GET" | "POST", path: string, body?: unknown): Promise<T> {
  const text = body === undefined ? "" : JSON.stringify(body);
  let status: number;
  let answer: string;
  if (__PAGES__) {
    ({ status, body: answer } = await pagesCall(method, path, text));
  } else {
    let res: Response;
    try {
      res = await fetch(path, { method, headers: { "Content-Type": "application/json" }, body: text || undefined });
    } catch {
      throw new ApiError(0, "no connection");
    }
    status = res.status;
    answer = await res.text();
  }
  if (status < 200 || status >= 300) {
    let message = `HTTP ${status}`;
    try {
      message = JSON.parse(answer).error ?? message;
    } catch {}
    throw new ApiError(status, message);
  }
  return JSON.parse(answer) as T;
}

export const get = <T>(path: string) => call<T>("GET", path);
export const post = <T>(path: string, body: unknown = {}) => call<T>("POST", path, body);

const TRIES = 3;

/**
 * save sends the end of a round, again twice when the server was not reached
 * or failed: the round's id makes a round that got through once kept once.
 */
export async function save<T>(path: string, body: unknown): Promise<T> {
  for (let i = 1; ; i++) {
    try {
      return await post<T>(path, body);
    } catch (err) {
      if (!(err instanceof ApiError) || !err.retryable || i >= TRIES) throw err;
      await new Promise((r) => setTimeout(r, 400 * i));
    }
  }
}

/** roundId is a new random id for a round, the same in every try. */
export const roundId = () =>
  Array.from(crypto.getRandomValues(new Uint8Array(16)), (b) => b.toString(16).padStart(2, "0")).join("");
