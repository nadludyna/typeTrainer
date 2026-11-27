# Product

What typeTrainer is and the decisions that shape it. Ask the maintainer before
changing any of them.

## What it is

typeTrainer is a touch-typing trainer. It tracks every letter's mistakes and
speed and builds the next words around your weak letters.

## Pillars

1. **Three modes, three skills.** Learn the letters, reinforce them on words, then type freely. Each mode trains one thing: accuracy, then speed, then rhythm.
2. **It adapts to you.** The next words bring your weak letters and transitions.
3. **Real words and real sentences** are what you practise on.
4. **Fast and calm.** Nothing jumps while you type.
5. **Clear without words.** Bars and dots show what is going on.

## Principles

- **Beginners first.** Learn is the main screen.
- **The server with PostgreSQL is the main version.** GitHub Pages is a demo
  snapshot and need not have every feature.
- **The on-screen keyboard with the hands** is part of the product: it shows
  which finger presses which key; it can be turned off.
- **Phones:** the whole site adapts to the screen: the same screens as on a
  computer with fewer elements, and you can type. A round typed on a phone
  is shown in the stats, but the trainer does not learn from it: thumbs on
  glass make other mistakes than fingers on keys, and touch typing is
  trained on a keyboard. So on a phone alone no letter opens in Learn and
  the word level does not move; the stats say so there. Phone typing may
  grow its own training later.
- **Three modes, each its own tab, always open.** Each trains one skill
  with its own adaptive loop: what it measures, what it learns about you,
  what it changes (`docs/HOW-THE-MODES-WORK.md`, section 0). Welcome is the
  fourth tab, and each mode asks a question or takes a 30-second sample the
  first time it opens.

  | Mode | What you practise | What it trains | How it adapts |
  |---|---|---|---|
  | **Learn** | letters, from the home row, one at a time | accuracy | a letter opens when the newest is typed right 90 % of the time and none is under 85 %; the words bring your weak letters |
  | **Reinforce** | real words of a level, the 200 to the 25 000 most common, every letter | speed, with accuracy as the gate | the words bring your slow letters and letter pairs; the word level moves by itself (three rounds at your goal speed up, two far under it down); the speed goal grows in steps of 10 wpm, 30 to 200 |
  | **Freetype** | a list of common words, long words, sentences or your own text, by words or by seconds | rhythm: how even your keys are | the letter pairs where your rhythm stumbles come more often; the word list moves by itself with your consistency (three rounds at 80 % up, two under 70 % down) |

  Learn's last letter ends on a finish screen that offers Reinforce;
  Reinforce is open before that too, with every letter. Each mode has its
  own stats on `~`. Every mode, list and round length keeps its own
  history.
- **What adapts and what does not.** The trainer cannot bend to everyone,
  so the line is drawn here:
  - *The trainer adapts:* which letter is in focus, which words come, when
    the next letter opens, the weak letter pairs, the speed goal.
  - *You adapt to the method:* fingers rest on F and J (the bumps on the
    keys) and each finger owns its column, because from there every finger
    reaches its keys equally; the thresholds of 90 % and 85 %; real words.
  - *You choose:* the mode, going back to the finger tour, turning the
    on-screen keyboard off, Reinforce's word level and speed goal by hand,
    Freetype's list and round length.
- **Settings: quick ones at hand, the rest in the window.** What belongs to
  the stage you are in can be changed right on the typing screen, on the
  fly. The settings window holds what concerns the whole site. Which
  setting goes where is still to be decided.
- **Motion.** Short and quiet, opacity and a few pixels, at most 0.3 s:
  windows, screens, tooltips, the result card, a new round's words, the pause
  cover, the on-screen keyboard and messages fade in and out; a newly opened
  letter pops once; the stats bars grow once when the screen opens; the tab
  marker slides, hover colours ease (`web/base.css` "motion", `web/fx.js`).
  Nothing new moves while you type: the caret and the line scroll are the
  only motion then. With "reduce motion" in the system, none of it runs.
- **English** is the one language for now.
- **The letter model is Learn's core.** Do not change the formulas in
  `internal/engine/model.go` or their values in `config.go` on your own: a
  silent change moves every user's progress. Explain the change, show it
  in the simulation (`go run ./tools/sim`) and ask.
- **Try things freely:** build an idea, try it by feel, keep it or drop it.

## Next to other trainers

As they were on 2026-11-25. Each of the parts exists somewhere; the
difference is the combination: real words, a loop that adapts in every mode
and can be explained in a sentence, and a calm screen.

| Trainer | What you type | What adapts, and by what | Strong | Missing |
|---|---|---|---|---|
| keybr | made-up words from the letters' sounds (real words as an option) | letters open by speed; each word holds one target letter | the learning algorithm and its charts, open source | made-up words; one of its six modes adapts |
| Monkeytype | random common words, quotes | nothing | the feel, the settings, its community | no teaching, no stats per key |
| Typing Improver | real words | the next words by the key-to-key transitions slower than your own median | thorough statistics, a control group | a long calibration before it adapts, a fixed course, many numbers |
| typing.com | a fixed course, and stories written by a language model around the weak keys | a short practice after each block | school use, connected text | the course itself does not adapt; generated text is unpredictable |
| TypeGG | quotes rated by difficulty | the quotes' difficulty to the typist's level | speed corrected for difficulty | made for competing, not learning |
| Entertrained | books | nothing | the wish to finish the book | no adaptation, no stats per key |
| Klavarog | word lists by speed level | the level is picked by hand | the finger shown for each key, Russian and Ukrainian | no stats, nothing adapts by itself |
| **typeTrainer** | real words of the most common 200 to 25 000 | Learn: letters open by accuracy; Reinforce: the slow letters and pairs, the word level and the speed goal; Freetype: the stumble pairs and the word list, by rhythm | every mode adapts, by rules written as formulas and checked in a simulation; stats per mode | English only; letters and pairs, not whole transitions; no study with people |

## Use cases

The person is the only actor: a guest by default, an account if they sign
up. Each case is walked by a browser check (`tools/browser`) or a test
through HTTP; the step-by-step path of a round is in
`docs/HOW-THE-CODE-WORKS.md`, "Request lifecycle".

| # | Case | Steps | Result |
|---|---|---|---|
| U1 | First visit | Welcome opens: what touch typing is, the home row, the finger tour; Enter starts Learn | Learn's first round with 6 letters |
| U2 | A mode's first opening | Learn asks one question (no, a little, yes); Reinforce and Freetype take a 30-second sample | Learn starts with 6 or 10 letters; Reinforce gets its first goal and word level, Freetype its list |
| U3 | A Learn round | type 30 words; the server picks each portion from the letter model; at the end the round is saved and learned from | a letter opens when its rule is met; the last letter ends on the finish screen |
| U4 | A Reinforce round | pick or keep a word level and a speed goal; type the words | the slow letters and pairs come more often; the level moves after three passes or two fails; the goal steps up once every letter holds it |
| U5 | A Freetype round | pick a list, long words, sentences or your own text, a length in words or seconds; type | the result with speed, accuracy and consistency |
| U6 | Look at a mode's stats | `~` or the Stats button; `~`, Esc or Tab back | the mode's own measure, its chart, its panel, its days |
| U7 | Keep progress across devices | sign up (the guest's progress moves into the account), sign in elsewhere | the same letters, rounds and stats |
| U8 | Save or move progress | Settings, Data: Export to a file, Import from one | the whole language's progress written or replaced, all or nothing |
| U9 | Start over | Settings, Data: Start Learn over, or Reset progress | Learn's letters and rounds, or everything, gone after a confirmation |

## Requirements

**Functional**

| # | The site… | Where | Checked by |
|---|---|---|---|
| F1 | opens letters one at a time from the home row, by accuracy: the newest at 90 %, none under 85 %, over its last 100 presses | `internal/engine` (`NextUnlocked`, `Accuracy`) | `TestNextUnlocked_AccuracyRules`, `TestAccuracy_*` |
| F2 | picks real words, weighted to the weak letters (Learn) and the slow letters and pairs (Reinforce) | `engine.Generator.Portion`, `WithPairs` | `TestNext_WeakLetterGetsFocus`, `TestWithPairs_WeakPairsComeMoreOften` |
| F3 | measures every press: right or wrong and its time, a word's first letter not timed | `web/typing.js`, `engine.Apply` | `TestApply_WordStartsAreNotTimed` |
| F4 | moves Reinforce's word level by itself and its speed goal by steps | `levels.Judge`, `Standing`; `engine.LapDone` | `TestJudge`, `TestStanding`, `TestApp_ReinforceLevelMovesByItself` |
| F5 | gives Freetype a list, long words, sentences or the person's text, by words or seconds, with a consistency per round | `internal/freetype`, `engine.Consistency` | `TestApp_Freetype`, `TestConsistency*` |
| F6 | starts each mode from a check-in | `internal/api/checkin.go`, `web/checkin.js` | `TestApp_Checkin*`, the check-in browser checks |
| F7 | shows each mode's stats on one key | `web/stats.js`, `/stats?part=`, `/activity?part=` | `TestApp_StatsPerMode`, the stats browser checks |
| F8 | shows the next key and its finger on a keyboard that can be hidden | `web/keyboard.js`, `data/layouts/` | the keyboard browser checks |
| F9 | keeps accounts; a guest's progress moves into a new account | `internal/auth`, `internal/api/auth.go` | `TestApp_Accounts`, `TestApp_SignUpKeepsGuestProgress` |
| F10 | exports and imports a language's whole progress | `internal/api/data.go` | `TestApp_ExportImportRoundTrip`, `TestApp_ImportRejectsBadDataAndKeepsOldData` |

**Non-functional**

| # | Requirement | How it is met | Checked by |
|---|---|---|---|
| N1 | A key answers at once: the caret starts within 40 ms with the CPU slowed 6× | the keypress path touches only its own nodes (`docs/CODE-RULES.md`) | `node tools/browser/perf.mjs --against main` |
| N2 | A round sent twice, or a save that fails halfway, is counted once and whole | round ids, one transaction per save | `RoundSentTwice`, `TestApp_ResultsSentTwice`, `TestApp_ResultsAfterAFailedSave` |
| N3 | Passwords and sign-in tokens are kept only as hashes; sign-in attempts are limited | PBKDF2, SHA-256 of the token, a limiter | `TestPassword_*`, `TestToken_RandomAndHashed`, `TestApp_LoginAttemptLimit` |
| N4 | The same storage rules hold in memory and in PostgreSQL | one `Store` interface | `store_test.go` on both, PostgreSQL in CI |
| N5 | The same code runs as a server and in the browser (GitHub Pages) | WebAssembly build of the same Go code | the Pages build in `go run ./tools/check` and CI |
| N6 | Few dependencies: Go's standard library and one PostgreSQL driver; plain HTML, CSS and JavaScript | `docs/CODE-RULES.md`, "Stack" | `go.mod`, no build step |
| N7 | Every change is checked before it reaches `main` | `go run ./tools/check`; CI on every pull request, and every night | `.github/workflows/tests.yml` |
| N8 | A desktop with a physical keyboard is the main place; a phone shows the same screens with fewer elements, and its rounds are shown but not learned from | `web/responsive.css`, `storage.OnKeyboard` | the phone-width browser checks, `TestApp_PhoneRoundsTeachNothing` |

## Planned

A light theme and designed motion with the redesign, more languages,
typing with digits and symbols, the mascot as a quiet companion, other
layouts (QWERTZ, Dvorak and more); the order is the plan's (issue #305).
Not planned: leaderboards and public profiles (this is for practice, not a
contest), sounds.
