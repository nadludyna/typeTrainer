<script lang="ts">
  // The stats of the mode you are in, over the typing screen: its own
  // measure first (Learn accuracy, Reinforce speed against the goal,
  // Freetype consistency), one bar per round, then what to work on.
  import type { Round } from "../core/api-types";
  import { app, type Mode } from "../core/state.svelte";
  import { T } from "../core/texts";

  let { mode }: { mode: Mode } = $props();

  const history = $derived(
    mode === "learn" ? app.learn?.history : mode === "reinforce" ? app.reinforce?.history : app.freetype?.history,
  );
  const measure = (r: Round) => (mode === "learn" ? r.accuracy : mode === "reinforce" ? r.wpm : (r.consistency ?? 0));
  const unit = $derived(mode === "reinforce" ? T.live.wpm : "%");
  const line = $derived(mode === "learn" ? app.learn?.mastery : mode === "reinforce" ? app.reinforce?.goal : undefined);

  const rounds = $derived([...(history?.rounds ?? [])].reverse()); // oldest first
  const values = $derived(rounds.map(measure));
  const top = $derived(Math.max(line ?? 0, ...values, 1));
  const last10 = $derived(values.slice(-10));
  const average = $derived(last10.length ? Math.round(last10.reduce((a, v) => a + v, 0) / last10.length) : 0);

  const slowest = $derived(
    [...(app.reinforce?.letters ?? [])].filter((l) => l.count > 0).sort((a, b) => b.avgMs - a.avgMs).slice(0, 6),
  );
</script>

<svelte:window
  onkeydown={(e) => {
    if (e.key === "Escape" || e.code === "Backquote") {
      e.preventDefault();
      app.statsOpen = false;
    }
  }}
/>

<section class="flex w-full max-w-230 flex-col gap-8" aria-label="{T.nav[mode]} {T.nav.stats}">
  <header class="flex items-baseline justify-between">
    <h2 class="text-2xl font-semibold">{T.nav[mode]}</h2>
    <span class="text-sub">{T.what[mode]}</span>
  </header>

  {#if !history || history.count === 0}
    <p class="py-10 text-center text-sub">{T.stats.none}</p>
  {:else}
    <div class="flex items-end gap-10">
      <div class="flex items-baseline gap-2">
        <b class="font-mono text-6xl font-semibold tabular-nums">{average}</b>
        <span class="text-sub">{unit} · {T.stats.average}</span>
      </div>
      <div class="flex items-baseline gap-2">
        <b class="font-mono text-3xl font-semibold tabular-nums">{history.count}</b><span class="text-sub">{T.stats.rounds}</span>
      </div>
    </div>

    <div class="relative flex h-40 items-end justify-end gap-1" role="img" aria-label={T.stats.perRound[mode]}>
      {#each values as v, i (i)}
        <div
          class="max-w-4 min-w-1 flex-1 rounded-t-sm {line !== undefined && v >= line ? 'bg-good' : 'bg-accent/70'}"
          style:height="{(v / top) * 100}%"
          title="{Math.round(v)}{unit === '%' ? '%' : ` ${unit}`}"
        ></div>
      {/each}
      {#if line !== undefined}
        <div class="absolute inset-x-0 border-t-2 border-dashed border-ink/50" style:bottom="{(line / top) * 100}%"></div>
      {/if}
    </div>

    <div class="grid gap-8 sm:grid-cols-2">
      {#if mode === "learn" && app.learn}
        <div>
          <h3 class="mb-3 text-sm text-sub">{T.stats.letters}</h3>
          <div class="flex flex-wrap gap-1.5">
            {#each app.learn.letters.filter((l) => l.open) as l (l.char)}
              <span
                class="flex w-10 flex-col items-center rounded-md bg-key py-1 font-mono shadow-[0_2px_0_var(--color-key-edge)]"
                title={T.letter.accuracy(Math.round(l.level))}
              >
                <span class="uppercase">{l.char}</span>
                <span class="text-xs {l.mastered ? 'text-good' : 'text-sub'}">{Math.round(l.level)}</span>
              </span>
            {/each}
          </div>
        </div>
        <div>
          <h3 class="mb-3 text-sm text-sub">{T.stats.mixed}</h3>
          <ul class="flex flex-col gap-1 font-mono">
            {#each app.learn.confusions as c (c.want + c.typed)}
              <li><b class="uppercase">{c.want}</b> <span class="text-sub">{T.stats.instead}</span> <b class="uppercase text-bad">{c.typed}</b> <span class="text-sub">× {c.n}</span></li>
            {:else}
              <li class="text-sub">—</li>
            {/each}
          </ul>
        </div>
      {:else if mode === "reinforce" && app.reinforce}
        <div>
          <h3 class="mb-3 text-sm text-sub">{T.stats.slowLetters}</h3>
          <ul class="flex flex-col gap-1 font-mono">
            {#each slowest as l (l.char)}
              <li class="flex items-center gap-3">
                <b class="w-4 uppercase">{l.char}</b>
                <span class="h-2 rounded-full bg-accent/70" style:width="{Math.min(100, l.avgMs / 6)}%"></span>
                <span class="text-sm text-sub tabular-nums">{Math.round(12000 / Math.max(1, l.avgMs))} {T.live.wpm}</span>
              </li>
            {/each}
          </ul>
        </div>
        <div>
          <h3 class="mb-3 text-sm text-sub">{T.stats.slowPairs}</h3>
          <ul class="flex flex-col gap-1 font-mono">
            {#each app.reinforce.pairs as p (p.pair)}
              <li><b>{p.pair}</b> <span class="text-sm text-sub">{Math.round(p.cost)} ms</span></li>
            {:else}
              <li class="text-sub">—</li>
            {/each}
          </ul>
        </div>
      {:else if mode === "freetype" && app.freetype}
        <div>
          <h3 class="mb-3 text-sm text-sub">{T.stats.stumbles}</h3>
          <ul class="flex flex-col gap-1 font-mono">
            {#each app.freetype.rhythm.pairs as p (p.pair)}
              <li><b>{p.pair}</b> <span class="text-sm text-sub">{Math.round(p.rate * 100)}%</span></li>
            {:else}
              <li class="text-sub">—</li>
            {/each}
          </ul>
        </div>
        <div>
          <h3 class="mb-3 text-sm text-sub">{T.stats.best}</h3>
          <ul class="flex flex-col gap-1 font-mono">
            {#each Object.entries(app.freetype.history.bests) as [len, r] (len)}
              <li><span class="text-sub">{T.len(len)}</span> <b>{Math.round(r.wpm)}</b> <span class="text-sub">{T.live.wpm}</span></li>
            {/each}
          </ul>
        </div>
      {/if}
    </div>
  {/if}
</section>
