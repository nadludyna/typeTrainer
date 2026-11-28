<script lang="ts">
  // The typing screen every mode shares: the text, the caret, the live
  // numbers and the keyboard. Its words come from a Source (modes/); the
  // rules of a press are in core/round.ts.
  import { tick, untrack } from "svelte";
  import { ApiError } from "../core/api";
  import { Mark, Round } from "../core/round";
  import { app } from "../core/state.svelte";
  import { T } from "../core/texts";
  import type { After, Piece, Source } from "../modes/source";
  import { toast } from "../ui/messages.svelte";
  import Keyboard from "./Keyboard.svelte";

  let { source, open = null, after }: { source: Source; open?: Set<string> | null; after: (a: After) => Promise<void> } =
    $props();

  const LINES = 3;
  const AHEAD = 12; // words shown past the one being typed, one at a time
  const IDLE_MS = 10_000;

  let round: Round;
  let chars = $state<string[]>([]);
  let marks = $state<Mark[]>([]);
  let extras = $state<Record<number, string[]>>({});
  let wordOf: number[] = [];
  let why: string[] = [];
  let pos = $state(0);
  let endAt = $state(-1); // the position after the round's last letter, -1 until it is loaded
  let typed = $state(0);
  let started = $state(false);
  let paused = $state(false);
  let busy = $state(false);
  let live = $state({ wpm: 0, acc: 100 });
  let left = $state(0); // a time round's seconds left
  let queue: Promise<void> = Promise.resolve();
  let generation = 0;

  let box = $state<HTMLDivElement>();
  let spans: HTMLSpanElement[] = $state([]);
  let caret = $state({ x: 0, y: 0, lineH: 0, scroll: 0 });

  const length = $derived(source.length());
  const shownUpTo = $derived(source.oneAtATime ? typed + AHEAD : Number.POSITIVE_INFINITY);
  const next = $derived(chars[pos] ?? "");
  const progress = $derived(
    length.kind === "words" ? Math.min(1, typed / length.n) : started ? 1 - left / length.n : 0,
  );

  function add(pieces: Piece[]) {
    const from = round.text.length;
    round.append(pieces.map((p) => p.words));
    let word = wordOf.length ? (wordOf[wordOf.length - 1] ?? 0) + 1 : 0;
    for (let i = from; i < round.text.length; i++) {
      wordOf.push(word);
      if (round.text[i] === " ") word++;
    }
    for (const p of pieces) why.push(...p.words.map((_, i) => p.why?.[i] ?? ""));
    chars.push(...round.text.slice(from));
    marks.push(...round.marks.slice(from));
    endAt = round.endAt;
  }

  export async function load() {
    const id = ++generation;
    busy = true;
    const l = source.length();
    round = new Round(
      {
        mark: (i, m) => {
          marks[i] = m;
        },
        extras: (at, letters) => {
          extras[at] = letters;
        },
      },
      l.kind === "words" ? l.n : 0,
    );
    wordOf = [];
    why = [];
    try {
      const first = await source.start();
      if (id !== generation) return;
      chars = [];
      marks = [];
      extras = {};
      endAt = -1;
      add(first);
      pos = typed = 0;
      started = paused = false;
      live = { wpm: 0, acc: 100 };
      left = l.kind === "time" ? l.n : 0;
      queue = Promise.resolve();
    } catch {
      toast(T.server);
    } finally {
      if (id === generation) busy = false;
    }
    await tick();
    place();
  }

  // place puts the caret before the letter to type and scrolls the text so
  // its line is the second of the three: only the caret and the scroll move.
  function place() {
    requestAnimationFrame(() => {
      const s = spans[Math.min(pos, spans.length - 1)];
      if (!s) return;
      const end = pos >= spans.length;
      const lineH = parseFloat(getComputedStyle(s).lineHeight) || s.offsetHeight;
      const line = Math.round(s.offsetTop / lineH);
      caret = { x: s.offsetLeft + (end ? s.offsetWidth : 0), y: s.offsetTop, lineH, scroll: Math.max(0, line - 1) * lineH };
    });
  }

  function press(ch: string, now: number) {
    if (busy || round.done) return;
    if (length.kind === "time" && round.startTime !== null && now - round.startTime >= length.n * 1000) {
      finish();
      return;
    }
    const out = round.type(ch, now);
    started = round.startTime !== null;
    paused = false;
    pos = round.pos;
    if (out.wordDone) {
      typed = round.wordsTyped;
      live = { wpm: Math.round(round.wpm()), acc: Math.round(round.accuracy()) };
    }
    if (out.portionDone >= 0) more(out.portionDone);
    if (out.roundDone || (source.maxMistakes && round.mistakes >= source.maxMistakes)) finish();
    else place();
  }

  function more(k: number) {
    const id = generation;
    queue = queue.then(async () => {
      if (id !== generation) return;
      try {
        const pieces = await source.next(k, round);
        if (id === generation && pieces.length) add(pieces);
      } catch {
        toast(T.server);
      }
    });
  }

  async function finish() {
    if (busy) return;
    busy = true;
    const id = generation;
    live = { wpm: Math.round(round.wpm()), acc: Math.round(round.accuracy()) };
    try {
      await queue;
      if (id !== generation) return;
      if (!round.typedSamples().length) {
        busy = false;
        await load();
        return;
      }
      await after(await source.finish(round, why));
    } catch (err) {
      toast(err instanceof ApiError && !err.retryable ? err.message : T.save.failed);
    }
    busy = false;
    if (id === generation) await load();
  }

  function onKey(e: KeyboardEvent) {
    if (app.statsOpen || app.settingsOpen || e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;
    if (e.key === "Tab") {
      e.preventDefault();
      if (!busy) load();
      return;
    }
    if (e.altKey || e.metaKey || (e.ctrlKey && e.key !== "Backspace")) return;
    if (e.key === "Backspace") {
      e.preventDefault();
      if (busy) return;
      round.backspace(e.ctrlKey);
      pos = round.pos;
      typed = round.wordsTyped;
      place();
      return;
    }
    if (e.key.length !== 1 || e.repeat) return;
    if (e.key === " " || (e.target instanceof HTMLButtonElement && e.key === "Enter")) e.preventDefault();
    if (e.getModifierState?.("CapsLock")) {
      toast(T.warn.caps);
      return;
    }
    const ch = e.key;
    if (ch !== " " && !/^[a-z]$/.test(ch)) {
      if (/\p{L}/u.test(ch)) toast(T.warn.layout);
      return;
    }
    if (document.activeElement instanceof HTMLElement && document.activeElement !== document.body) {
      document.activeElement.blur();
    }
    press(ch, e.timeStamp || performance.now());
  }

  // The clock: a time round ends when its time is up; ten seconds without a
  // key, or leaving the page, pause the round and take that time out.
  $effect(() => {
    const timer = setInterval(() => {
      if (!round || round.startTime === null || busy || round.paused) return;
      const now = performance.now();
      if (length.kind === "time") {
        left = Math.max(0, Math.ceil(length.n - (now - round.startTime) / 1000));
        if (left === 0) finish();
      }
      if (round.lastTime !== null && now - round.lastTime > IDLE_MS) {
        round.pause(round.lastTime);
        paused = true;
      }
    }, 250);
    const away = () => {
      if (round?.startTime !== null && !busy) {
        round.pause(performance.now());
        paused = true;
      }
    };
    window.addEventListener("blur", away);
    return () => {
      clearInterval(timer);
      window.removeEventListener("blur", away);
    };
  });

  // A new source (another mode, kind, level or list) starts a new round.
  $effect(() => {
    source;
    untrack(() => load());
  });

  $effect(() => {
    const onResize = () => place();
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  });

  function cls(i: number, m: Mark) {
    if ((wordOf[i] ?? 0) >= shownUpTo || (endAt >= 0 && i >= endAt)) return "invisible";
    switch (m) {
      case Mark.Right:
        return "text-ink";
      case Mark.Wrong:
        return chars[i] === " " ? "bg-bad-soft" : "text-bad bg-bad-soft rounded-sm";
      case Mark.Skipped:
        return "text-muted underline decoration-bad decoration-2 underline-offset-4";
      default:
        return "text-muted";
    }
  }
</script>

<svelte:window onkeydown={onKey} />

<section class="flex w-full max-w-230 flex-col gap-4" aria-label="Text to type">
  <div class="flex items-baseline justify-between font-mono text-sm text-sub" aria-live="off">
    <span class="flex gap-5 tabular-nums transition-opacity {started ? 'opacity-100' : 'opacity-0'}">
      <span><b class="text-lg font-semibold text-ink">{live.wpm}</b> {T.live.wpm}</span>
      <span><b class="text-lg font-semibold text-ink">{live.acc}%</b> {T.live.acc}</span>
    </span>
    <span class="tabular-nums">
      {#if length.kind === "time"}{started ? left : length.n} s{:else}{typed} / {length.n}{/if}
    </span>
  </div>

  <div
    bind:this={box}
    class="relative overflow-hidden font-mono text-[32px] leading-[1.7] tracking-[0.01em]"
    style:height="{LINES * 1.7}em"
  >
    <div class="relative transition-transform duration-150" style:translate="0 {-caret.scroll}px" data-text>
      {#each chars as ch, i (i)}{#if extras[i]?.length}<span class="text-bad bg-bad-soft">{extras[i]?.join("")}</span
          >{/if}<span
          bind:this={spans[i]}
          class={cls(i, marks[i] ?? Mark.Untyped)}
          data-m={marks[i]}>{ch}</span
        >{/each}
      <span
        aria-hidden="true"
        class="absolute top-0 left-0 w-[3px] rounded-sm bg-accent transition-[translate] duration-75 ease-out {started
          ? ''
          : 'animate-pulse'}"
        style:height="{caret.lineH * 0.66}px"
        style:translate="{caret.x - 1.5}px {caret.y + caret.lineH * 0.17}px"
      ></span>
    </div>
    {#if paused}
      <div class="absolute inset-0 flex items-center justify-center bg-paper/80 font-ui text-base text-sub">
        {T.pause}
      </div>
    {/if}
  </div>

  <div class="h-1 rounded-full bg-track" role="img" aria-label="{Math.round(progress * 100)}% of the round">
    <div class="h-1 rounded-full bg-accent transition-[width] duration-300" style:width="{progress * 100}%"></div>
  </div>
</section>

{#if app.layout && app.keyboard[source.mode]}
  <Keyboard layout={app.layout} {next} {open} />
{/if}
