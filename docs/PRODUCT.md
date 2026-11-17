# Product

What typeTrainer is and the decisions that shape it. Ask the maintainer before
changing any of them.

## What it is

typeTrainer is a touch-typing trainer. It tracks every letter's mistakes and
speed and builds the next words around your weak letters. Real words only.

## Pillars

1. **One path.** First you learn the letters (Learn), then you polish (Reinforce).
2. **It adapts to you.** The next words bring your weak letters and transitions.
3. **Real words and real sentences only.**
4. **Fast and calm.** Nothing jumps while you type.
5. **Clear without words.** Bars and dots show what is going on.

## Principles

- **Beginners first.** Learn is the main screen.
- **The server with PostgreSQL is the main version.** GitHub Pages is a demo
  snapshot and need not have every feature.
- **The keyboard with the hands stays.** It shows which finger presses
  which key, the core of touch typing: make it cheap, hide it where it is
  not needed, never cut it.
- **Real words only.** No generated letter combinations.
- **A computer keyboard is needed to practise.** Phones and tablets show only
  the stats and the account: thumbs on glass would skew the letter model.
  The phone layout of the typing screen is not maintained.
- **One mode: Learn, then Reinforce.** *Learn* teaches from zero: letters
  open one by one from the home row, and the algorithm alone decides what to
  practise (no difficulty switches, no "open all letters"). When every letter
  is open, *Reinforce* begins: you choose what to type (a word level or real
  sentences), and inside that choice the algorithm brings your weak letters
  and transitions. *Climb* and *Freetype* are not part of this path: they get
  hidden, and Climb is removed once Reinforce has real sentences. Until then
  they never touch Learn's data, and every mode, list and round length keeps
  its own history: rounds of different kinds are never mixed in one chart or
  record. How each mode works: `docs/HOW-THE-MODES-WORK.md`.
- **Settings change how much and how it looks, never what is practised.**
  What a round is belongs to each mode and sits in the round bar above the
  text (`web/modebar.js`). The settings window (`web/settings.js`) holds
  every setting, with a search and categories; a mode's own ones are tagged
  with the mode.
- **Plain defaults.** No decorative animation, dimming, glow or blur. Motion
  is short and quiet: windows, screens and tooltips fade in (opacity and a few
  pixels, under 0.2 s, `web/base.css` "motion"), the tab marker slides, hover
  colours ease; nothing new moves while you type. What moves to show position:
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
  it on real people (`docs/HOW-THE-MODES-WORK.md`, section 10.1). Every round stores its
  group. It is never shown or offered in the interface.
- **Try things freely:** build an idea, try it by feel, keep it or drop it.

## Not now

Leaderboards, public profile pages, more languages, more themes (there is
one, dark, as CSS variables in `web/base.css`), frameworks. Later: the mascot, a light theme, special word sets, Freetype with
symbols and digits.
Accounts exist (sign up / log in / log out on the normal server, kept in
PostgreSQL when it is configured). Structure code so these can be added
later, but do not build them yet.
