<script lang="ts">
  // Welcome: each colour is one finger, then a tour that walks the keyboard
  // finger by finger, left pinky to right pinky: press each lit key once.
  // Its presses are never sent. Enter starts Learn at any time.
  import type { Key } from "../core/api-types";
  import { app, keep } from "../core/state.svelte";
  import { T } from "../core/texts";
  import Keyboard from "../typing/Keyboard.svelte";

  const FINGERS = ["lp", "lr", "lm", "li", "ri", "rm", "rr", "rp"];
  const BLOCK = 10; // the letter block: the first ten keys of the three letter rows

  // The tour: for each finger its keys of the letter block, column by column, top to bottom.
  const tour = $derived.by(() => {
    const layout = app.layout;
    if (!layout) return [];
    return FINGERS.map((f) => {
      const keys: { k: Key; col: number; row: number }[] = [];
      layout.rows.slice(0, 3).forEach((row, r) => {
        row.slice(0, BLOCK).forEach((k, c) => {
          if (k.finger === f) keys.push({ k, col: c, row: r });
        });
      });
      keys.sort((a, b) => a.col - b.col || a.row - b.row);
      return { finger: f, keys: keys.map((x) => x.k.char) };
    });
  });

  let finger = $state(0);
  let key = $state(0);
  const current = $derived(tour[finger]);
  const next = $derived(current?.keys[key] ?? "");
  const done = $derived(tour.length > 0 && finger >= tour.length);

  function start() {
    app.welcomed = true;
    keep("welcomed", true);
    app.go("learn");
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      start();
      return;
    }
    if (done || e.key.toLowerCase() !== next) return;
    if (key + 1 < (current?.keys.length ?? 0)) key++;
    else {
      finger++;
      key = 0;
    }
  }
</script>

<svelte:window onkeydown={onKey} />

<section class="flex w-full max-w-230 flex-col items-center gap-6">
  <h1 class="text-center text-3xl font-semibold">{T.welcome.title}</h1>
  <p class="text-center text-sub">{T.welcome.colours}</p>

  <div class="flex items-center gap-2" role="img" aria-label={T.welcome.tour}>
    {#each tour as f, i (f.finger)}
      <span
        class="h-2 w-10 rounded-full"
        style:background={i <= finger ? app.layout?.fingers[f.finger]?.color : "var(--color-track)"}
      ></span>
    {/each}
  </div>
  <span class="font-mono text-lg">
    {#if done}{T.welcome.done}{:else if current}{T.welcome.tourOf(finger + 1, tour.length, T.fingers[current.finger] ?? "")} ·
      <b>{T.welcome.press(next)}</b>{/if}
  </span>

  {#if app.layout}<Keyboard layout={app.layout} {next} />{/if}

  <button type="button" class="rounded-xl bg-ink px-6 py-3 text-lg text-paper" onclick={start}
    >{T.welcome.start} <kbd class="ml-2 rounded bg-paper/15 px-1.5 text-sm">{T.welcome.enter}</kbd></button
  >
</section>
