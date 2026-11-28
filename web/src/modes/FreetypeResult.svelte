<script lang="ts">
  // A Freetype round's result: speed and accuracy big, each word's speed as a
  // line, the words coloured by speed (red is for mistakes only), then why
  // the round's words came. Enter or Tab goes on.
  import { T } from "../core/texts";
  import type { Shown } from "./source";

  let { result, done }: { result: Shown; done: () => void } = $props();

  const W = 640;
  const H = 96;
  const speeds = $derived(result.words.map((w) => w.wpm).filter((v): v is number => v !== null));
  const top = $derived(Math.max(1, ...speeds));
  const points = $derived(
    speeds.map((v, i) => `${(i / Math.max(1, speeds.length - 1)) * W},${H - (v / top) * (H - 8) - 4}`).join(" "),
  );
  // Five shades from slow to fast, by the round's own spread.
  const lo = $derived(Math.min(...speeds));
  const shade = (v: number | null) => {
    if (v === null || top === lo) return "text-ink";
    const k = Math.min(4, Math.floor(((v - lo) / (top - lo)) * 5));
    return ["text-muted", "text-sub", "text-ink/70", "text-ink/85", "text-ink"][k];
  };
  const reasons = $derived(
    [...new Set(result.words.map((w) => w.why).filter(Boolean))].map((w) => {
      const [kind = "", what = ""] = w.split(":");
      return T.result.why[kind]?.(what) ?? "";
    }),
  );
</script>

<svelte:window
  onkeydown={(e) => {
    if (e.key === "Enter" || e.key === "Tab") {
      e.preventDefault();
      done();
    }
  }}
/>

<section class="flex w-full max-w-230 flex-col gap-6" aria-label="Result">
  <div class="flex items-end gap-10">
    <div class="flex items-baseline gap-2">
      <b class="font-mono text-6xl font-semibold tabular-nums">{Math.round(result.wpm)}</b><span class="text-sub"
        >{T.live.wpm}</span
      >
      {#if result.best}<span class="ml-2 rounded-md bg-good px-2 py-0.5 text-sm text-white">{T.result.best}</span>{/if}
    </div>
    <div class="flex items-baseline gap-2">
      <b class="font-mono text-3xl font-semibold tabular-nums">{Math.round(result.accuracy)}%</b><span class="text-sub"
        >{T.live.acc}</span
      >
    </div>
    {#if result.consistency !== null}
      <div class="flex items-baseline gap-2">
        <b class="font-mono text-3xl font-semibold tabular-nums">{Math.round(result.consistency)}%</b><span
          class="text-sub">{T.live.cons}</span
        >
      </div>
    {/if}
    <span class="ml-auto font-mono text-sub tabular-nums">{Math.round(result.seconds)} s</span>
  </div>

  {#if speeds.length > 1}
    <svg viewBox="0 0 {W} {H}" preserveAspectRatio="none" class="h-24 w-full" role="img" aria-label="Speed of each word">
      <polyline
        {points}
        fill="none"
        stroke="var(--color-accent)"
        stroke-width="2"
        stroke-linejoin="round"
        vector-effect="non-scaling-stroke"
      />
    </svg>
  {/if}

  <p class="font-mono text-xl leading-relaxed">
    {#each result.words as w, i (i)}<span class={w.wrong ? "text-bad" : shade(w.wpm)} title={w.wpm ? `${w.wpm} wpm` : ""}
        >{w.word}</span
      >{" "}{/each}
  </p>

  <div class="flex items-center justify-between gap-4 text-sm text-sub">
    {#if reasons.length}<span>{T.result.built}: <b class="text-ink">{reasons.join(" · ")}</b></span>{:else}<span></span>{/if}
    <button type="button" class="rounded-lg bg-ink px-4 py-2 text-paper" onclick={done}>{T.result.next} ↵</button>
  </div>
</section>
