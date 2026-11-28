<script lang="ts">
  import { drop, toasts } from "./messages.svelte";
</script>

<div class="pointer-events-none fixed inset-x-0 top-20 z-30 flex flex-col items-center gap-2" aria-live="polite">
  {#each toasts as t (t.id)}
    <div class="pointer-events-auto flex items-center gap-3 rounded-xl bg-ink px-4 py-2 text-sm text-paper shadow-lg">
      <span>{t.text}</span>
      {#if t.action}
        <button
          type="button"
          class="rounded-md bg-paper/15 px-2 py-0.5 font-medium"
          onclick={() => {
            t.action?.run();
            drop(t.id);
          }}>{t.action.label}</button
        >
      {/if}
    </div>
  {/each}
</div>
