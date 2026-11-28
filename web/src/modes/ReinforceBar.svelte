<script lang="ts">
  // Reinforce's bar: the word level, the speed goal with − and +, the kind
  // of round, and how far the rounds are from moving the level.
  import { post } from "../core/api";
  import type { Step } from "../core/api-types";
  import { app } from "../core/state.svelte";
  import { T } from "../core/texts";
  import Ring from "../ui/Ring.svelte";
  import KindPicker from "./KindPicker.svelte";
  import { refreshReinforce, setLevel } from "./reinforce";

  const s = $derived(app.reinforce);

  async function step(by: number) {
    if (!s) return;
    await post<Step>("/api/reinforce/step", { step: Math.min(s.lastStep, Math.max(0, s.step + by)) });
    await refreshReinforce();
  }
</script>

{#if s}
  <div class="flex w-full max-w-230 flex-wrap items-center justify-between gap-3">
    <Ring
      label={T.level.label}
      value={app.level}
      pick={setLevel}
      options={s.wordLevels.map((n, i) => ({ value: i + 1, label: T.words(n), tip: T.level.tip(n) }))}
    />
    <div class="flex items-center gap-3">
      <KindPicker mode="reinforce" spots={s.weakSpots} />
      <div class="flex items-center gap-1 font-mono text-sm" role="group" aria-label={T.goal.label}>
        <button
          type="button"
          class="size-8 rounded-md bg-track text-sub hover:text-ink disabled:opacity-40"
          aria-label={T.goal.down}
          disabled={s.step === 0}
          onclick={() => step(-1)}>−</button
        >
        <span class="w-20 text-center font-semibold tabular-nums">{T.goal.wpm(s.goal)}</span>
        <button
          type="button"
          class="size-8 rounded-md bg-track text-sub hover:text-ink disabled:opacity-40"
          aria-label={T.goal.up}
          disabled={s.step === s.lastStep}
          onclick={() => step(1)}>+</button
        >
      </div>
      <div class="flex gap-1" role="img" aria-label={T.standing(s.standing.passes, s.standing.passesUp)}>
        {#each Array(s.standing.passesUp) as _, i (i)}
          <span class="size-2.5 rounded-full {i < s.standing.passes ? 'bg-good' : 'bg-track'}"></span>
        {/each}
      </div>
    </div>
  </div>
{/if}
