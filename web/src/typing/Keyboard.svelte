<script lang="ts">
  // The keyboard under the text: every key carries its finger's colour on
  // its lower edge, and the next key is filled with it, so the finger to use
  // is on the key itself. A key pressed sinks for a moment.
  import type { Key, Layout } from "../core/api-types";
  import { T } from "../core/texts";

  let { layout, next, open = null }: { layout: Layout; next: string; open?: Set<string> | null } = $props();

  const UNIT = 48; // px: one key
  const GAP = 8;

  let down = $state<string | null>(null);

  function color(k: Key) {
    return layout.fingers[k.finger]?.color ?? "#9aa3ad";
  }

  function width(k: Key) {
    return (k.width ?? 1) * UNIT + ((k.width ?? 1) - 1) * GAP;
  }

  function label(k: Key) {
    return k.char === " " ? "" : k.char.toUpperCase();
  }

  // A letter not open yet in Learn has no colour yet: the finger is learned with it.
  const lit = (k: Key) => !open || k.char === " " || open.has(k.char);
</script>

<svelte:window onkeydown={(e) => (down = e.key.toLowerCase())} onkeyup={() => (down = null)} />

<section
  aria-label="Keyboard"
  class="rounded-[20px] bg-track px-5 pt-4.5 pb-5.5 shadow-[inset_0_1px_0_var(--color-white),0_10px_24px_rgb(21_25_31/0.12)]"
>
  <div class="flex flex-col gap-2">
    {#each layout.rows as row, r (r)}
      <div class="flex gap-2" style:padding-left="{(layout.offsets[r] ?? 0) * UNIT}px">
        {#each row as k (k.char)}
          {@const isNext = k.char === next}
          <div
            class="flex items-center justify-center rounded-[10px] font-mono text-base font-medium transition-[translate,box-shadow] duration-75 {isNext
              ? 'text-ink'
              : 'bg-key text-ink'}"
            style:width="{width(k)}px"
            style:height="{UNIT}px"
            style:background={isNext ? color(k) : undefined}
            style:translate={down === k.char ? "0 3px" : "0 0"}
            style:box-shadow={down === k.char
              ? "0 1px 0 var(--color-key-edge)"
              : isNext
                ? `0 4px 0 var(--color-key-edge), 0 0 0 3px var(--color-ink)`
                : `0 4px 0 var(--color-key-edge)${lit(k) ? `, inset 0 -5px 0 ${color(k)}` : ""}`}
            aria-label={isNext ? `${k.char === " " ? "space" : k.char}: ${T.fingers[k.finger] ?? ""}` : undefined}
          >
            {label(k)}
          </div>
        {/each}
      </div>
    {/each}
  </div>
</section>
