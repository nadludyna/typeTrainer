<script lang="ts">
  // Learn's last letter is open: on to Reinforce, or keep practising here.
  import { app } from "../core/state.svelte";
  import { T } from "../core/texts";

  let { done }: { done: () => void } = $props();
</script>

<svelte:window
  onkeydown={(e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      app.go("reinforce");
    }
  }}
/>

<section class="flex w-full max-w-230 flex-col items-center gap-6 py-10">
  <h2 class="text-3xl font-semibold">{T.finish.title}</h2>
  <div class="flex flex-wrap justify-center gap-1.5">
    {#each app.learn?.letters ?? [] as l (l.char)}
      <span class="flex size-9 items-center justify-center rounded-lg bg-key font-mono uppercase shadow-[0_3px_0_var(--color-key-edge)]"
        >{l.char}</span
      >
    {/each}
  </div>
  <div class="flex gap-3">
    <button type="button" class="rounded-xl bg-ink px-5 py-3 text-paper" onclick={() => app.go("reinforce")}
      >{T.finish.reinforce} ↵</button
    >
    <button type="button" class="rounded-xl bg-track px-5 py-3" onclick={done}>{T.finish.stay}</button>
  </div>
</section>
