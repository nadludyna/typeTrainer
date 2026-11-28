<script lang="ts" generics="T extends string | number">
  // A row of cells read as one switch: the chosen one is filled.
  interface Option {
    value: T;
    label: string;
    tip?: string;
    disabled?: boolean;
  }
  let { label, options, value, pick }: { label: string; options: Option[]; value: T; pick: (v: T) => void } = $props();
</script>

<div class="inline-flex overflow-hidden rounded-lg bg-track p-0.5" role="group" aria-label={label}>
  {#each options as o (o.value)}
    <button
      type="button"
      class="h-8 min-w-9 rounded-md px-2.5 font-mono text-sm transition-colors disabled:opacity-40 {o.value === value
        ? 'bg-key text-ink shadow-sm'
        : 'text-sub hover:text-ink'}"
      aria-pressed={o.value === value}
      title={o.tip}
      disabled={o.disabled}
      onclick={() => pick(o.value)}>{o.label}</button
    >
  {/each}
</div>
