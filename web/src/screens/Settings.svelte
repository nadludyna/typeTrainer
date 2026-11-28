<script lang="ts">
  // The few things that concern the whole site: the keyboard of this mode,
  // and the progress data (export, import, start Learn over, reset).
  import { get, post } from "../core/api";
  import type { File, Imported } from "../core/api-types";
  import { app, keep, type Mode } from "../core/state.svelte";
  import { T } from "../core/texts";
  import { refreshFreetype } from "../modes/freetype";
  import { refreshLearn } from "../modes/learn";
  import { refreshReinforce } from "../modes/reinforce";
  import { toast } from "../ui/messages.svelte";

  let { mode, close }: { mode: Mode | null; close: () => void } = $props();
  let dialog = $state<HTMLDialogElement>();
  let asking = $state<"learn" | "all" | null>(null);

  $effect(() => {
    dialog?.showModal();
  });

  const refresh = () => Promise.all([refreshLearn(), refreshReinforce(), refreshFreetype()]);

  async function exportFile() {
    const file = await get<File>("/api/export");
    const url = URL.createObjectURL(new Blob([JSON.stringify(file)], { type: "application/json" }));
    const a = Object.assign(document.createElement("a"), { href: url, download: `typetrainer-${new Date().toISOString().slice(0, 10)}.json` });
    a.click();
    URL.revokeObjectURL(url);
  }

  async function importFile(e: Event) {
    const f = (e.target as HTMLInputElement).files?.[0];
    if (!f) return;
    try {
      const out = await post<Imported>("/api/import", JSON.parse(await f.text()));
      await refresh();
      toast(T.settings.imported(out.rounds));
    } catch {
      toast(T.settings.importFail);
    }
  }

  async function confirm() {
    await post(asking === "learn" ? "/api/learn/reset" : "/api/reset");
    asking = null;
    await refresh();
    close();
  }

  function toggleKeyboard(m: Mode) {
    app.keyboard[m] = !app.keyboard[m];
    keep("keyboard", app.keyboard);
  }
</script>

<dialog
  bind:this={dialog}
  class="m-auto w-96 rounded-2xl bg-paper p-6 text-ink shadow-2xl backdrop:bg-ink/30"
  onclose={close}
  aria-label={T.settings.title}
>
  <div class="flex flex-col gap-5">
    <h2 class="text-xl font-semibold">{T.settings.title}</h2>
    {#if mode}
      <label class="flex items-center justify-between">
        <span>{T.keyboard.show} · {T.nav[mode]}</span>
        <input type="checkbox" class="size-5 accent-accent" checked={app.keyboard[mode]} onchange={() => toggleKeyboard(mode)} />
      </label>
    {/if}
    <div class="flex flex-col gap-2">
      <span class="text-sm text-sub">{T.settings.data}</span>
      <div class="flex gap-2">
        <button type="button" class="rounded-lg bg-track px-3 py-2" onclick={exportFile}>{T.settings.export}</button>
        <label class="cursor-pointer rounded-lg bg-track px-3 py-2"
          >{T.settings.import}<input type="file" accept="application/json" class="hidden" onchange={importFile} /></label
        >
      </div>
    </div>
    {#if asking}
      <div class="flex flex-col gap-3 rounded-xl bg-bad-soft p-4">
        <span>{asking === "learn" ? T.settings.confirmLearn : T.settings.confirmReset}</span>
        <div class="flex gap-2">
          <button type="button" class="rounded-lg bg-key px-3 py-2" onclick={() => (asking = null)}>{T.settings.cancel}</button>
          <button type="button" class="rounded-lg bg-bad px-3 py-2 text-white" onclick={confirm}
            >{asking === "learn" ? T.settings.restartLearn : T.settings.reset}</button
          >
        </div>
      </div>
    {:else}
      <div class="flex gap-2">
        <button type="button" class="rounded-lg bg-track px-3 py-2" onclick={() => (asking = "learn")}>{T.settings.restartLearn}</button>
        <button type="button" class="rounded-lg bg-track px-3 py-2 text-bad" onclick={() => (asking = "all")}>{T.settings.reset}</button>
      </div>
    {/if}
  </div>
</dialog>
