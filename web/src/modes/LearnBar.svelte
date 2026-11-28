<script lang="ts">
  // Learn's bar over the text: every open letter as a chip filled by how
  // right it is, the newest one big with its bar to the mark where the next
  // letter opens, and the next letter waiting, dashed.
  import { app } from "../core/state.svelte";
  import { T } from "../core/texts";

  const stats = $derived(app.learn);
  const newest = $derived(stats ? stats.letters[stats.unlocked - 1] : undefined);
  const waiting = $derived(stats ? stats.letters[stats.unlocked] : undefined);
  const colorOf = (c: string) => {
    const key = app.layout?.rows.flat().find((k) => k.char === c);
    return (key && app.layout?.fingers[key.finger]?.color) ?? "#9aa3ad";
  };
</script>

{#if stats && newest}
  <div class="flex w-full max-w-230 flex-col gap-3">
    <div class="flex items-center gap-4">
      <span
        class="flex size-11 flex-none items-center justify-center rounded-[10px] font-mono text-[22px] font-medium uppercase text-ink shadow-[0_4px_0_var(--color-key-edge)]"
        style:background={colorOf(newest.char)}
        title={T.letter.accuracy(Math.round(newest.level))}>{newest.char}</span
      >
      <div
        class="relative h-2.5 flex-1 rounded-full bg-track"
        role="img"
        aria-label="{newest.char}: {Math.round(newest.level)}% right; the next letter opens at {stats.mastery}%"
      >
        <div
          class="h-2.5 rounded-full transition-[width] duration-500"
          style:width="{Math.min(100, newest.level)}%"
          style:background={colorOf(newest.char)}
        ></div>
        <div class="absolute -top-1.5 h-5.5 w-0.5 bg-ink" style:left="{stats.mastery}%"></div>
      </div>
      <span class="w-12 flex-none text-right font-mono font-semibold tabular-nums"
        >{Math.round(newest.level)}<span class="font-normal text-sub">%</span></span
      >
      {#if waiting}
        <span
          aria-label="{T.letter.next}: {waiting.char}"
          class="flex size-11 flex-none items-center justify-center rounded-[10px] border-2 border-dashed border-key-edge font-mono text-[22px] uppercase text-sub"
          >{waiting.char}</span
        >
      {/if}
    </div>
    <div class="flex flex-wrap gap-1.5" aria-label={T.stats.letters}>
      {#each stats.letters.slice(0, stats.unlocked) as l (l.char)}
        <span
          class="relative flex h-7 w-7 items-center justify-center overflow-hidden rounded-md bg-key font-mono text-sm uppercase shadow-[0_2px_0_var(--color-key-edge)]"
          title={T.letter.accuracy(Math.round(l.level))}
        >
          <span
            class="absolute inset-x-0 bottom-0 opacity-40"
            style:height="{Math.min(100, l.level)}%"
            style:background={colorOf(l.char)}
          ></span>
          <span class="relative {l.mastered ? 'font-semibold text-ink' : 'text-sub'}">{l.char}</span>
        </span>
      {/each}
    </div>
  </div>
{/if}
