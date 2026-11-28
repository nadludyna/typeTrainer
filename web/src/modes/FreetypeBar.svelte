<script lang="ts">
  // Freetype's bar: the word list, the round's length (words or seconds) and
  // the kind of round.
  import { app, keep } from "../core/state.svelte";
  import { T } from "../core/texts";
  import Ring from "../ui/Ring.svelte";
  import { refreshFreetype, setList } from "./freetype";
  import KindPicker from "./KindPicker.svelte";

  const WORDS = [10, 15, 20, 30, 50, 100];
  const SECONDS = [15, 30, 60];

  const s = $derived(app.freetype);
  const kind = $derived(app.length.startsWith("time") ? "time" : "words");

  function setLength(id: string) {
    app.length = id;
    keep("length", id);
  }

  async function pickList(n: number) {
    setList(n);
    await refreshFreetype();
  }
</script>

{#if s}
  <div class="flex w-full max-w-230 flex-wrap items-center justify-between gap-3">
    <Ring
      label={T.list}
      value={app.list}
      pick={pickList}
      options={s.lists.map((n) => ({ value: n, label: T.words(n), tip: T.level.tip(n) }))}
    />
    <div class="flex items-center gap-3">
      <KindPicker mode="freetype" spots={s.weakSpots} />
      {#if app.kind.freetype !== "sprint"}
        <Ring
          label="Length"
          value={kind}
          pick={(k) => setLength(k === "time" ? "time-30" : "words-30")}
          options={[
            { value: "words", label: T.lenKind.words },
            { value: "time", label: T.lenKind.time },
          ]}
        />
        <Ring
          label="Length"
          value={app.length}
          pick={setLength}
          options={(kind === "time" ? SECONDS : WORDS).map((n) => {
            const id = `${kind}-${n}`;
            return { value: id, label: `${n}`, tip: T.len(id) };
          })}
        />
      {/if}
    </div>
  </div>
{/if}
