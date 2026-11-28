<script lang="ts">
  // The page: the header with the modes, and the screen of the one you are in.
  // ~ opens that mode's stats, alt+k shows or hides its keyboard, Esc the settings.
  import { onMount } from "svelte";
  import { get } from "./core/api";
  import type { Layout } from "./core/api-types";
  import { app, keep, type Mode, type Screen } from "./core/state.svelte";
  import { T } from "./core/texts";
  import { refreshFreetype } from "./modes/freetype";
  import { refreshLearn } from "./modes/learn";
  import { refreshReinforce } from "./modes/reinforce";
  import ModeScreen from "./screens/ModeScreen.svelte";
  import Settings from "./screens/Settings.svelte";
  import Stats from "./screens/Stats.svelte";
  import Welcome from "./screens/Welcome.svelte";
  import Toasts from "./ui/Toasts.svelte";
  import { toast } from "./ui/messages.svelte";

  const SCREENS: Screen[] = ["welcome", "learn", "reinforce", "freetype"];
  const mode = $derived<Mode | null>(app.screen === "welcome" ? null : app.screen);

  onMount(async () => {
    try {
      app.layout = await get<Layout>("/api/layout");
      await Promise.all([refreshLearn(), refreshReinforce(), refreshFreetype()]);
    } catch {
      toast(T.server);
    }
  });

  function onKey(e: KeyboardEvent) {
    if (app.settingsOpen) return;
    if (mode && e.altKey && e.code === "KeyK") {
      e.preventDefault();
      app.keyboard[mode] = !app.keyboard[mode];
      keep("keyboard", app.keyboard);
    } else if (mode && !app.statsOpen && e.code === "Backquote") {
      // the key left of 1, whatever the system layout prints on it (` or ё)
      e.preventDefault();
      app.statsOpen = true;
    } else if (e.key === "Escape" && !app.statsOpen) {
      e.preventDefault(); // else the same Escape closes the dialog it opens
      app.settingsOpen = true;
    }
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="flex min-h-screen flex-col">
  <header class="flex flex-wrap items-center justify-between gap-4 px-12 py-6">
    <span class="font-logo text-xl font-semibold tracking-tight">type<span class="text-accent">Trainer</span></span>
    <nav aria-label="Modes" class="flex gap-7">
      {#each SCREENS as s (s)}
        <button
          type="button"
          class="pb-1 {app.screen === s ? 'font-semibold text-ink shadow-[0_6px_0_-4px_var(--color-accent)]' : 'text-sub hover:text-ink'}"
          aria-current={app.screen === s ? "page" : undefined}
          title={s === "welcome" ? undefined : T.what[s]}
          onclick={() => app.go(s)}>{T.nav[s]}</button
        >
      {/each}
    </nav>
    <div class="flex gap-6">
      {#if mode}
        <button
          type="button"
          class={app.statsOpen ? "font-semibold" : "text-sub hover:text-ink"}
          aria-pressed={app.statsOpen}
          aria-keyshortcuts="`"
          onclick={() => (app.statsOpen = !app.statsOpen)}>{T.nav.stats}</button
        >
      {/if}
      <button type="button" class="text-sub hover:text-ink" aria-keyshortcuts="Escape" onclick={() => (app.settingsOpen = true)}
        >{T.nav.settings}</button
      >
    </div>
  </header>

  <main class="flex flex-1 flex-col items-center gap-7 px-12 pt-6 pb-10">
    {#if app.screen === "welcome"}
      <Welcome />
    {:else}
      {#key app.screen}
        <div
          class="flex w-full flex-col items-center gap-7 {app.statsOpen ? 'hidden' : ''} {mode && !app.keyboard[mode]
            ? 'flex-1 justify-center pb-[12vh]'
            : ''}"
        >
          <ModeScreen mode={app.screen} />
        </div>
      {/key}
      {#if app.statsOpen && mode}<Stats {mode} />{/if}
    {/if}
  </main>
</div>

{#if app.settingsOpen}
  <Settings {mode} close={() => (app.settingsOpen = false)} />
{/if}
<Toasts />
