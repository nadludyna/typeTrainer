# Product

What typeTrainer is and the decisions that shape it. Ask the owner before
changing any of them.

## What it is

typeTrainer is a touch-typing trainer. It tracks every letter's mistakes and
speed and builds the next words around your weak letters. Real words only.

The product comes first. It is also an engineering thesis (2027), a side
quest: its material collects in `docs/thesis/`, one file per item of the
thesis plan (`01-...md` to `08-...md`, issue #146). A PR that yields
material for an item (a number, a measurement, a decision) adds a few lines
to that item's file. Product docs stay where they are; thesis files link to
them instead of copying. Thesis notes do not limit what we build: if the
code and a thesis note disagree, the code and `docs/` win. What the thesis
needs does limit what we cut: the adaptive algorithm, the keyboard with the
finger layout, input latency, PostgreSQL and the adaptive-vs-random check
are thesis items and stay.

## Pillars

1. **One path in three stages.** Learn the letters, reinforce them on words, then type freely. Each stage adds one thing to train: accuracy, then speed, then consistency.
2. **It adapts to you.** The next words bring your weak letters and transitions.
3. **Real words and real sentences only.**
4. **Fast and calm.** Nothing jumps while you type.
5. **Clear without words.** Bars and dots show what is going on.

## Principles

- **Beginners first.** Learn is the main screen.
- **The server with PostgreSQL is the main version.** GitHub Pages is a demo
  snapshot and need not have every feature.
- **The keyboard with the hands stays.** It is part of the thesis: make it
  cheap, hide it where it is not needed, never cut it.
- **Real words only.** No generated letter combinations.
- **A computer keyboard is needed to practise.** Phones and tablets show only
  the stats and the account: thumbs on glass would skew the letter model.
  The phone layout of the typing screen is not maintained.
- **One mode, three stages.** Every stage adapts to you; each adds one
  measure on top of the ones before. How the trainer picks what you type:
  `docs/HOW-THE-MODES-WORK.md`.

  | Stage | What you practise | What it trains | How it adapts | Opens when |
  |---|---|---|---|---|
  | **Learn** | letters, from the home row, one at a time | accuracy | words around your weak letters and the letter pairs you stumble on | from the start |
  | **Reinforce** | words in levels that add up: 200, then 200 and 500, then up to 1k, and so on to 25k | accuracy and speed | the same, inside the open levels; the speed goal grows by a step you choose, with no ceiling | every letter is open |
  | **Freetype** | free typing with your own settings | accuracy, speed and consistency | the next word is picked by the move from the end of one word into the start of the next, so the hands find a flow | the first Reinforce level is passed once |

  Consistency belongs to Freetype because it depends most on the moves
  between words. The letter row is all the progress Learn shows; the stats
  screen opens with Reinforce. Learn never switches difficulty and never
  "opens all letters": the algorithm alone decides what to practise.

  **Built now:** Learn, and Reinforce with one word level at a time and a
  speed goal from 30 to 200 in steps of 10. Learn's letter level still
  counts speed as well as accuracy. **To build:** Learn on accuracy only,
  levels that add up, a level that can be passed, a goal step in the
  settings and no ceiling, Freetype as the third stage with word-to-word
  flow. *Climb* is hidden and goes once the stages are built; until then it
  never touches Learn's data, and every mode, list and round length keeps
  its own history.
- **Settings change how much and how it looks, never what is practised.**
  What a round is belongs to each mode and sits in the round bar above the
  text (`web/modebar.js`). The settings window (`web/settings.js`) holds
  every setting, with a search and categories; a mode's own ones are tagged
  with the mode.
- **Plain defaults.** No decorative animation, dimming, glow or blur. Motion
  is short and quiet, opacity and a few pixels, at most 0.3 s, never a snap:
  windows, screens, tooltips, the result card, a new round's words, the pause
  cover, the on-screen keyboard and messages fade in and out; a newly opened
  letter pops once; the stats bars grow once when the screen opens; the tab
  marker slides, hover colours ease (`web/base.css` "motion", `web/fx.js`).
  Nothing new moves while you type. With "reduce motion" in the system, none of
  it runs. What moves to show position:
  the caret and the line scroll. The on-screen keyboard keeps its keycap look and its lit next key. Effects may come back one by one, each decided
  on its own; back so far: the falling keycaps in the empty sides, on by default, still while you type. A rule that does not change is spelled out once, then lives in
  a tooltip. Text only where nothing else can say it.
- **One language: English.** Other languages are removed, not hidden.
- **The letter model is Learn's core.** Do not change the formulas in
  `internal/engine/model.go` or their values in `config.go` on your own: a
  silent change moves every user's progress. Explain the change, show it
  in the simulation (`go run ./tools/sim`) and ask.
- **A hidden A/B switch.** `?group=random` makes Learn draw random open words
  instead of using the algorithm, so the algorithm can be compared against
  it on real people (`docs/thesis/08-testing-and-verification.md`). Every round stores its
  group. It is never shown or offered in the interface.
- **Try things freely:** build an idea, I test it, we keep it or drop it.

## Not now

Leaderboards, public profile pages, more languages, more themes (there is
one, dark, as CSS variables in `web/base.css`), frameworks. After the
thesis defence: the mascot, a light theme, special word sets, Freetype with
symbols and digits.
Accounts exist (sign up / log in / log out on the normal server, kept in
PostgreSQL when it is configured). Structure code so these can be added
later, but do not build them yet.
