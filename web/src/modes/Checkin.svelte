<script lang="ts">
  // Learn's first visit: one question sets the first letters.
  import { post } from "../core/api";
  import type { Checkin } from "../core/api-types";
  import { app, keep } from "../core/state.svelte";
  import { T } from "../core/texts";
  import { refreshLearn } from "./learn";

  async function answer(a: "no" | "little" | "yes") {
    await post<Checkin>("/api/learn/checkin", { answer: a });
    await refreshLearn();
    app.checkedIn = true;
    keep("checkedIn", true);
  }
</script>

<section class="flex w-full max-w-230 flex-col items-center gap-6 py-10">
  <h2 class="text-2xl font-semibold">{T.checkin.title}</h2>
  <div class="flex gap-3">
    {#each ["no", "little", "yes"] as const as a (a)}
      <button
        type="button"
        class="min-w-28 rounded-xl bg-key px-5 py-3 text-lg shadow-[0_4px_0_var(--color-key-edge)] hover:translate-y-0.5 hover:shadow-[0_2px_0_var(--color-key-edge)]"
        onclick={() => answer(a)}>{T.checkin[a]}</button
      >
    {/each}
  </div>
</section>
