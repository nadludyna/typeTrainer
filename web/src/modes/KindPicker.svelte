<script lang="ts">
  // The kind of round in Reinforce and Freetype: usual, weak spots, clean, sprint.
  import { app, keep, type Kind } from "../core/state.svelte";
  import { T } from "../core/texts";
  import Ring from "../ui/Ring.svelte";

  let { mode, spots }: { mode: "reinforce" | "freetype"; spots: number } = $props();

  const kinds: Kind[] = ["", "weak", "clean", "sprint"];

  function pick(k: Kind) {
    app.kind[mode] = k;
    keep("kind", app.kind);
  }
</script>

<Ring
  label={T.kind.label}
  value={app.kind[mode]}
  {pick}
  options={kinds.map((k) => ({
    value: k,
    label: T.kind[k],
    tip: k === "weak" && !spots ? T.kind.noWeak : T.kind.tip[k],
    disabled: k === "weak" && !spots && app.kind[mode] !== "weak",
  }))}
/>
