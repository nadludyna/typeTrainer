<script lang="ts">
  // One mode's screen: its bar, the text and, between rounds, a card (a new
  // letter, the finish, a result). Its source is built again whenever a
  // choice it reads (kind, level, list, length) changes, which starts a new round.
  import { app, type Mode } from "../core/state.svelte";
  import Checkin from "../modes/Checkin.svelte";
  import Finish from "../modes/Finish.svelte";
  import FreetypeBar from "../modes/FreetypeBar.svelte";
  import FreetypeResult from "../modes/FreetypeResult.svelte";
  import { freetypeSource } from "../modes/freetype";
  import LearnBar from "../modes/LearnBar.svelte";
  import { learnSource } from "../modes/learn";
  import NewLetter from "../modes/NewLetter.svelte";
  import ReinforceBar from "../modes/ReinforceBar.svelte";
  import { reinforceSource } from "../modes/reinforce";
  import type { After, Shown, Source } from "../modes/source";
  import Typing from "../typing/Typing.svelte";

  let { mode }: { mode: Mode } = $props();

  const make: Record<Mode, () => Source> = {
    learn: learnSource,
    reinforce: reinforceSource,
    freetype: freetypeSource,
  };
  const source = $derived.by(() => {
    // the choices each source reads, so a change makes a new one
    void [app.kind[mode], app.level, app.list, app.length];
    return make[mode]();
  });

  type Card = { letter: string; finish: boolean } | { result: Shown } | { finish: true; letter?: undefined };
  let card = $state<Card | null>(null);
  let release = $state(() => {});

  function after(a: After): Promise<void> {
    if (a.show === "next") return Promise.resolve();
    card = a.show === "letter" ? { letter: a.letter, finish: a.finish } : { result: a.result };
    return new Promise((resolve) => {
      release = () => {
        card = null;
        resolve();
      };
    });
  }

  const open = $derived(
    mode === "learn" && app.learn ? new Set(app.learn.letters.filter((l) => l.open).map((l) => l.char)) : null,
  );
</script>

{#if mode === "learn" && !app.checkedIn && app.learn && app.learn.history.count === 0}
  <Checkin />
{:else}
  {#if mode === "learn"}<LearnBar />{:else if mode === "reinforce"}<ReinforceBar />{:else}<FreetypeBar />{/if}

  {#if card && "result" in card}
    <FreetypeResult result={card.result} done={release} />
  {:else if card?.letter}
    <NewLetter
      letter={card.letter}
      done={() => {
        if (card && "finish" in card && card.finish) card = { finish: true };
        else release();
      }}
    />
  {:else if card}
    <Finish done={release} />
  {/if}

  <div class={card ? "hidden" : "contents"}>
    <Typing {source} {open} {after} />
  </div>
{/if}
