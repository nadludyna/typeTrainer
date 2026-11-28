<script lang="ts">
  // A new letter opened: the letter on its key, its finger, and a line that
  // runs out in a few seconds before the next round. Hold keeps it; any key goes on.
  import { app } from "../core/state.svelte";
  import { T } from "../core/texts";

  let { letter, done }: { letter: string; done: () => void } = $props();

  const SHOWN_MS = 5000;
  const key = $derived(app.layout?.rows.flat().find((k) => k.char === letter));
  const color = $derived((key && app.layout?.fingers[key.finger]?.color) ?? "#9aa3ad");
  let held = $state(false);

  $effect(() => {
    if (held) return;
    const t = setTimeout(done, SHOWN_MS);
    return () => clearTimeout(t);
  });
</script>

<svelte:window
  onkeydown={(e) => {
    if (e.key.length === 1 || e.key === "Enter") {
      e.preventDefault();
      done();
    }
  }}
/>

<div class="flex w-full max-w-230 flex-col items-center gap-6 py-6" role="status" aria-label="{T.letter.newTitle}: {letter}">
  <span class="text-sm tracking-wide text-sub uppercase">{T.letter.newTitle}</span>
  <span
    class="flex size-28 items-center justify-center rounded-3xl font-mono text-6xl font-semibold uppercase text-ink shadow-[0_8px_0_var(--color-key-edge)]"
    style:background={color}>{letter}</span
  >
  <span class="font-mono text-lg text-sub">{key ? T.fingers[key.finger] : ""}</span>
  <div class="flex w-80 items-center gap-3">
    <div class="h-1 flex-1 overflow-hidden rounded-full bg-track">
      {#if !held}
        <div class="h-1 rounded-full bg-ink" style:animation="tt-run {SHOWN_MS}ms linear forwards"></div>
      {/if}
    </div>
    <button type="button" class="rounded-md bg-track px-3 py-1 text-sm text-sub hover:text-ink" onclick={() => (held = true)}
      >{T.letter.hold}</button
    >
  </div>
</div>

<style>
  @keyframes -global-tt-run {
    from {
      width: 100%;
    }
    to {
      width: 0;
    }
  }
</style>
