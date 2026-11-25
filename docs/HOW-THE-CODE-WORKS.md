# How the code works

## Overview

```
 Browser (web/)                         Go server
 ┌───────────────────────┐   JSON    ┌─────────────────────────────────────────┐
 │ ES modules: app.js,   │ ───────▶  │ api        HTTP handlers, /api/v1/*      │
 │  typing.js (keydown,  │ ◀───────  │   │                                      │
 │  timing), keyboard... │           │   ├──▶ engine   model + word generator   │
 └───────────────────────┘           │   └──▶ storage  Store interface          │
                                     │              ├─ MemoryStore (default)    │
                                     │              └─ PostgresStore (DB URL)   │
                                     │ config  all numbers   layout  key data   │
                                     └─────────────────────────────────────────┘
```

**Dependency rule:** `api` uses `engine`, `freeplay` and `storage`. `storage` imports
`engine` only for the `KeyStat` type. `engine` imports nothing from the project.
The engine therefore never knows about HTTP or the database and can be tested on
its own.

## Packages and files

- `setup.go`: Loads the embedded data files, creates the store, generator and API server (`newApp`), and serves `web/` through `internal/static`. Shared by both entry points.

- `main.go`: Entry point for the normal server: calls `newApp` and starts HTTP with the time limits of `config.Server` (a slow client cannot hold a connection); on Ctrl+C or SIGTERM it stops taking new requests and lets the running ones finish (up to `Server.Shutdown`), so a round being saved during a restart is not lost (#246).

- `main_wasm.go`: Entry point for the GitHub Pages build (see below).

- `pages/wasm-shim.js`: Pages only: routes `/api/v1/` requests from the frontend to the Go code in the browser, and keeps the progress in IndexedDB (the `/export` JSON, loaded back with `/import`).

- `.github/workflows/pages.yml`: On every push to `main`: test, build WebAssembly, publish to Pages.

- `.github/workflows/tests.yml`: On every pull request: gofmt, vet, tests, WebAssembly build.

- `build_server.go`, `build_wasm.go`: What differs between the normal server and GitHub Pages (picked by build tags): `accountsOn` (accounts only on the server) and `openStore` (PostgreSQL when `DATABASE_URL` is set, else memory; always memory on Pages).

- `internal/config`: Every tunable number in one place.

- `internal/auth`: Password hashing (PBKDF2-SHA256), random login tokens, the rules for account names and passwords. Knows nothing about HTTP or storage.

- `internal/engine/model.go`: Per-letter statistics, EMA update, difficulty, mastery, softmax weight.

- `internal/engine/generator.go`: Unlock order, first lesson size, unlocking rule, word selection; `WithWords` and `WithPairs`, Reinforce's word levels and the weak pairs of Learn and Reinforce.

- `internal/engine/consistency.go`: Consistency of a round's speed (a measurement shown to the user; the model does not use it).

- `internal/freeplay`: The words outside Learn's letter model. Freetype: words drawn evenly from the n most common (`Picker.Freetype`) and real sentences (`sentences.go`). The typist's weak letter pairs from the raw keys (`weak.go`, `WeakPairs`; Reinforce brings them more often). The pass / hold / fail rule of a level (`Judge`, `Standing`), for Reinforce's word levels (docs/HOW-THE-MODES-WORK.md, section 0.2). Imports nothing from the project; never touches the letter model.

- `internal/history`: Readings of a typist's finished rounds for the stats screen: the accuracy streaks (`AccuracyStreaks`), every letter after every Learn round (`LetterTimeline`: the letter model replayed with `engine.Apply` over the rounds' raw keys), the keys mixed up (`Confusions`, from `Sample.Typed`). Pure functions; `internal/api` loads the rounds.

- `internal/storage/store.go`: The `Store` interface.

- `internal/storage/memory.go`: In-memory `Store` (lost on restart).

- `internal/storage/postgres.go`: PostgreSQL `Store` through the standard `database/sql`; the pgx driver is only registered. Applies `migrations/*.sql` at start. Not in the WebAssembly build.

- `internal/storage/migrations/`: The database tables, one numbered SQL file per change (`001_init.sql` ... `005_letter_held.sql` (a letter's mastery is remembered), `006_round_level.sql` (the level of a round; now Reinforce's word level), `007_freetype_modes.sql` (a history per Freetype list), `008_climb_ten_levels.sql` (a mode that is gone), `009_freetype_variants.sql` (a history per Freetype kind of text), `010_learn_laps.sql` (Learn's lap per user and per round), `011_round_id.sql` (a round sent twice is kept once), `012_epoch.sql` (an `epoch` counter per user and language, bumped by every reset and import and stamped on every saved session, so records from before a wipe can be told apart from records after it), `013_aborted.sql` (an `aborted` flag: a round dropped by Tab is kept raw but shown nowhere and never learned), `014_round_seconds.sql` (a round's measured seconds, for the daily goal), `015_letter_hits.sql` (a letter's last presses and its slip under the lower threshold, for Learn's accuracy rules), `016_learn_and_freetype_only.sql` (Climb's rounds, the study group and the old typing test's flag go)).

- `internal/storage/store_test.go`: The tests every `Store` must pass; run on memory always and on PostgreSQL when `TT_TEST_DATABASE_URL` is set (`postgres_test.go`).

- `internal/api/handlers.go`: The `Server`, the route list (`Register`), `?lang=`, JSON helpers, locking around read-modify-write.

- `internal/api/learning.go`: Learning routes: `/portion`, `/results`, `/stats`, `/activity`, `/progress`.
- `internal/api/reinforce.go`: Reinforce: word levels, a learner's weak pairs (Learn uses them too), `/learn/step` (the speed goal by hand).

- `internal/api/data.go`: `/reset`, `/learn/reset` (Start Learn over), `/export`, `/import` (Learn and Freetype data together).

- `internal/api/validate.go`: Request checks, so nothing impossible or foreign reaches the statistics.

- `internal/api/rounds.go`: Helpers of the rounds outside Learn's letters: how many portions a round starts with, the weak pairs of a set of rounds (`pairsOf`) and a mode's history for its stats (`history`).

- `internal/api/freetype.go`: The `/api/v1/freetype/*` routes: Freetype's portions, results and stats per word list.

- `internal/api/auth.go`: The `/api/v1/auth/*` routes (sign up, log in, log out, who am I) and `userOf`: which user a request belongs to.

- `internal/static`: Serves the embedded `web/` files: an ETag (a hash of the file) with `Cache-Control: no-cache`, so a browser that has the file gets a short 304 instead of it again; text files gzipped once, when first asked for, if the browser accepts gzip. Anything else (a folder, a missing file) goes to the standard file server.

- `internal/layout`: Parses and validates the keyboard layout JSON; `RestingKeys` (the home row keys under the fingers, first in the learning order).

- `internal/language`: Parses `languages.json`; `Allows(word)` checks a word against the alphabet.

- `data/languages.json`: The language: id, name, label, layout, alphabet, OS layout name.

- `data/layouts/*.json`: Keys, rows, finger per key, AltGr letters, finger colors.

- `data/langs/en/free.txt`: The word list, most frequent first (25 000 words, `tools/wordlists/free.py`): used by every mode (Reinforce's word levels are its first n).

- `data/langs/en/sentences.txt`: Freetype's real sentences, one per line (from Tatoeba, CC BY 2.0 FR, `tools/sentences/build.py`).

- `internal/sim`: Simulation: a made-up typist and three word-choice strategies (docs/HOW-THE-MODES-WORK.md, section 10). Not used by the app.

- `tools/sim/`: `go run ./tools/sim`: runs the simulation on the real word lists and prints the table. Its test checks that Learn's stream beats random words (docs/HOW-THE-MODES-WORK.md, section 10).

- `tools/wordlists/`: Python script + block lists that build the word lists (not part of the app).

- `tools/browser/`: Browser checks (Playwright: a round in each mode, the caret, result and stats) and `perf.mjs`, the typing latency script. Node, not part of the app; how to run: DEVELOPMENT.md.

- `web/`: Static frontend: `index.html`, the stylesheets (`base.css` first, then `layout`, `typing`, `stats`, `screens`, `responsive`) and the JavaScript modules (see "Frontend modules" below). 

- `webmodules_test.go`: Checks the import rule of the frontend modules.

The `web/` and `data/` folders are embedded into the binary (`//go:embed`), so
the program runs the same from any folder.

## Frontend modules

The frontend is plain JavaScript split into native ES modules (`import` /
`export`). Browsers load them directly: no npm, no bundler, no build step.
`index.html` loads only `app.js`; each module loads the modules it needs.

**Bottom layer**

- `state.js`: Shared state `S`, page elements `el`, `store` (localStorage that may be blocked), small helpers.

- `i18n.js`: Every interface text, `t()`, `applyI18n()`.

- `fx.js`: The message queue in the floating toasts under the header (`#toasts`, fixed, so nothing on the page moves for them).

- `feel.js`: `FEEL`: the caret glide, the line scroll, the hand above the keyboard and the mistake marks, also given to the CSS as `--feel-*` (`applyFeel()` at load).

- `api.js`: `getJSON` / `postJSON` for `/api/v1/`, with `?lang=` added.

**Middle layer**

- `minipop.js`: The Ring switch groups (`ring`, `ringBtn`; the settings window and the stats period switch use the same `.ring` look), and that small window itself (`openPop`): a few settings of one screen next to the control that opens it (the keyboard's gear, the daily goal on Learn's ring and in the stats activity legend).

- `tools.js`: Tooltips, and the data buttons of the settings window: export / import, reset and Start Learn over (one confirm window for both).

- `keyboard.js`: On-screen keyboard from the layout data, the next key, the small hand that stands on its top edge above the next key's column (moved by `translate` at once, no glide, no layout read) and shows the finger to use: the hand is a picture, one PNG per finger (or fingers) to light, drawn once from the SVG model in `index.html` for the theme and the layout, so a key only swaps the `src` of one `<img>` per hand (#148), shown by default only in Learn while letters are still locked (`autoKb`, called on a mode change and whenever the letters are drawn); alt+k shows or hides it, and that choice is kept per mode (`toggleKb`, `tt.kb.<mode>`: `on` or `off`, then it wins over the default; the same choice is Show keyboard in the settings; when it is hidden the body gets `kb-hidden` and the keyboard takes no room), a gear in the bottom right corner of the panel, beside the space bar, with the keyboard's own settings (next key, hand; `openSettingsPop`), the automatic size that fits the window (the hand's room stays when the hand is off; when even the smallest keys don't fit, the hand and its room go first, `kb-nohand`, and if that is not enough the keyboard hides and one line says to zoom out, `kb-nofit`, `#kb-away`), kept in CSS px when the browser zoom changes, so it zooms with the page.

- `stats.js`: The stats screen of Learn and Freetype (Learn adds its letters below, `learnstats.js`): the switch of the mode and Freetype's kind of text and list on top, the personal bests, time typed today, in 7 days or all time with the activity calendar from `/activity` (its legend opens the daily goal picker, `.act-goal`), tiles with trends, the accuracy streaks and the personal goal, the WPM chart with the 10-round average and the goal line (`goalFor`; under 10 rounds the bars keep their width and the latest stands at the right edge): a number of your own, 20–300, on a stepper "− Goal … wpm +" that never moves (`stepGoal`: ± 5 wpm by the buttons, the mouse wheel or the arrow keys, snapping to steps of 5; below 20 it is off, + from off starts at 40; a click on the number itself switches it off at once and back to the last number, `tt.goal.last`; the number changes at once and is saved when the steps stop, `GOAL_SAVE_MS`), in Learn always Learn's target from `/stats`, which you can't change there; and `refreshStats()` (the letters of learning, `S.lastStats`).


- `typing.js`: The text stream (portions added while you type), where the caret goes (`placeCaret`), `handleChar()` / `backspace()`, live WPM / accuracy (plain text set at the end of each word, #254), the "typing now" header mode, sending results.

- `caret.js`: The caret's own movements: the pace of its glide and the scroll of the text to the next line. At rest it stands still.


- `warn.js`: The Caps Lock / wrong system layout warning in the toasts under the header (it hides a message while it shows): shown on keys that do not fit, checked before a round with the Keyboard API, hidden on a right key or a language change.

- `letters.js`: The letters bar above the text: learning's progress (open letters filled, mastered ones tinted green, each open one with a bar by how close it is to mastered, locked ones only an outline, the next one a dashed frame, a tooltip of what each still needs, `letterProgress`, `letterNeeds`, `rule`), locked keys dimmed and mastered letters fading from their keys on the keyboard.

- `customtext.js`: Freetype's own text: the editor window, the cleaning of a pasted text (`cleanText`), the portions cut from it in order (`customPortions`, as `/freetype/portion` would give them) and where the next round starts (`customRoundDone`). The text stays in the browser.

- `learnstats.js`: The Learn part of the stats screen, under the tiles and the chart: every letter round by round (a square per round in the letters row's colour), one letter's speed against the goal (a click on a letter picks it), the keys you mix up; from `/learn/history`.

- `learnpanel.js`: Learn's part of the line over the text: the last round's speed and accuracy at rest (`#lp-cols`; green when it is a new best; the comparison with the average of the 10 rounds before it is the tooltip), which the round's live numbers replace while typing, and the daily goal as minutes and seconds (`#goal-count`, "7:42 of 15:00", "15:00 done" once reached; the time today from `/activity` plus the round so far; a click opens the picker, `pickDaily`, `settings.js`). No labels, no news text: what a round changed shows on the letter row (`letters.js`), the forecast and the mastery rule are the next letter's tooltip, the lap after Learn complete is in the count's tooltip (`typing.js` `renderProgress`). `tickGoal`, from typing.js's half-second clock, updates the number; reaching the goal only turns it green, the round goes on. `roundMilestone()` marks a new best; `compareStats()` works out what a round changed.



- `intro.js`: The intro, "touch typing basics" (what it is, home row, space, eyes up); opens by itself on the first visit only; ← → / A D to go back and on.

- `languages.js`: `loadLanguage()`.

- `profile.js`: The account in the top right corner ("log in" for a guest where accounts work): the account (log in / sign up / log out; the page reloads after each), and a picture kept only in the browser.

- `result.js`: The result screen in the text window: `showResult(r)` shows the result of Freetype; Learn has none (its rounds follow one another; `learnpanel.js` shows the last one). Speed and accuracy carry their unit instead of a label; beside them, each word's speed on a line chart (a dot's tooltip gives its speed); under them, the words coloured by speed (five bands, slow to fast; red is for mistakes only), then one line of facts (consistency, time, correct characters, and wrong, extra and missed only when not zero). The words of the round with their own speed come from `typedWords()` in `typing.js`, the characters and the time from `roundFacts()`. A small ↻ button (and Enter / Tab, shown in the hint line) starts the next round. A new best shows "New best" under the speed.

- `peek.js`: A quick look at the stats of the current mode, list and round length, opened by holding `~` or resting the pointer on the Stats tab (`HOVER_MS`), closed by letting go, by any other key or a click. The panel stands fixed over what is under the text (`body.peek-on` hides the keyboard or `#stand`), so the text does not move. Only before the first key of a round, never on the result, and like the Stats tab only once Learn has become Reinforce (`statsShown`). It is drawn when it opens, from what the page already holds (`S.lastStats`, `S.freetype`); closed, it does no work.

- `modes.js`: The modes (learn / freetype), the navigation tabs that show them (`renderNav`: the shown modes, then stats; see "Modes"), `refreshFreeplay()` (`S.freetype`: the chosen list's history and the lists offered), `setFreetypeList()`, the mode and the list remembered per language.


- `settings-list.js`: Every setting as one entry of `SETTINGS`, in `CATEGORIES` (text, caret, typing, keyboard, goals, data), plus `LENGTHS` and `DAILY`. The fields of an entry: `docs/CODE-RULES.md`, "Settings".

- `settings.js`: Everything built from that list: `loadSettings` (each entry from its `localStorage` key, or its default), `setSetting(id, value)` for any setting from any screen, then `onSetting` listeners (`input.js`) and the `<body data-*>` values for the CSS (`data-ws`, `data-live`, `data-caret`, `data-glide`, `data-hand`); `control(id)` draws a setting's switch the same way everywhere. The window (Esc or the gear): a search that has the focus (it looks in the names, categories, values, tooltips, the modes and an entry's `words`; matches in a name are marked; the first Esc clears it, the next closes), the categories as one row of buttons (the ones with nothing found are dimmed), then the rows: ↑ ↓ pick, ← → change, Enter presses a button; a mode's own setting has its mode as a tag; the round length and Show keyboard are those of the current mode (the length isn't listed in Learn, which has one); a small preview beside the window types itself in the chosen caret look while the Caret or Caret movement row is hovered or picked; export / import / reset are wired by `tools.js`. `openSettingsPop` shows a few settings in the small window of one screen (the keyboard's gear; `pickDaily` for Learn's ring and the stats activity legend). Also the round lengths and each mode's own (`lengthFor`, `setLength`, `tt.length.<mode>`).

- `modebar.js`: The round bar above the text, in Freetype (Learn has one length, 30 words, and shows its letters there instead): what you type on the left, how long on the right (words or time, then the length); a click starts a new round (`onBarPick`, handled in `input.js`). In Learn the bar appears once every letter is open: Reinforce, its word levels and the speed goal step (− and +), on its own line over the letters (`body.reinforce`). Freetype's left side is the kind of text and the word list. Under the text, when the keyboard is off, `#stand` shows Freetype's best and last 20 rounds for this list and length (`freetypeStand`); it hides while typing, nothing moves. With the keyboard on (alt+k) it stays as a thin strip between the text and the keys.

**Top layer**

- `input.js`: Keydown and phone input (with the one-space sentinel that makes Backspace work on phone keyboards), the navigation (learn · freetype · stats), window resize: sends each action to the module that handles it. Hides the mouse pointer from the first typed key until the mouse moves (`no-pointer`).

- `app.js`: Entry point: `init()`.

**Import rule:** imports only go down. `state.js`, `i18n.js` and `feel.js` import
nothing, `fx.js` nothing, `api.js` only `state.js`. Middle modules import
the bottom layer and each other in one direction only. Nothing imports the top
layer (`app.js` loads `input.js`). There are no import cycles, so every module
can be read and explained starting from the modules it uses.
`webmodules_test.go` checks this on every `go test ./...`.

A module can change a field of an object it imports, but it cannot assign to an
imported variable. That is why the state that several modules share lives in
one object, `S` (`S.pos`, `S.lang`, ...), while state used by only one module
stays a plain variable inside it.

`file-check.js` is a small plain script, not a module: browsers do not load
ES modules from a `file://` address, so when `index.html` is opened by
double-click it is the only script that runs and it says how to open the page.
Its one sentence is the only interface text outside `i18n.js`.

### Screen layout

Two fixed places, the same in every view (`index.html`, `layout.css`):

- **header**: the logo and the account (`profile.js`);
- **navigation** (`nav.rail`, wired in `input.js`): the three modes, stats,
  then the settings. In the header of a wide window, a bar at the
  bottom below 900 px (so a 1920 px screen at 200% browser zoom keeps the wide layout).

On the typing screen every mode has the same frame, so the text stands at one
place: the band (`--band`: Learn's letters or the round bar), the line with
the numbers (speed and accuracy on the left, the daily goal in Learn and the
round's count on the right, `3 / 30` or the seconds left), the text, and below
it `#stand` and the keyboard when it is shown; a quick look at the stats
(`peek.js`) covers those two for as long as it is held. Warnings and messages float as toasts under
the header. The result opens where the text stands.

Colours are CSS variables on `:root`; there is one theme, dark.

## Request lifecycle: from a keypress to the next words

The text is a stream of short portions (`Generator.Portion`, see
`docs/HOW-THE-MODES-WORK.md`, section 7), not one exercise built in advance. A round
is `RoundPortions` portions (30 words).

1. Start of a round (`loadExercise()` in `typing.js`) → `POST /api/v1/portion {start: true}`
   returns the portion to type plus `AheadPortions` waiting after it, and
   `round` (portions per round). The browser gives the round an id once, at
   this moment, and sends it with every `/portion` and `/results` request of
   the round, together with `portionIndex` (how many portions of the round
   were finished) and `ack` (how many were received). The server keeps each
   portion answer in memory by (round, portionIndex): a request sent twice
   by a flaky network gets the same portions back and its samples are applied
   once. The cache is process memory, nothing else: a server restart forgets
   it, and a retry after a restart is treated as a first request. Answers the
   browser acked are dropped at once; abandoned rounds are dropped after
   `PortionCacheTTL` or when the cache passes `PortionCacheRounds` entries
   (both in `internal/config/config.go`).
2. `keydown` (`input.js`) → `handleChar()` in `typing.js` compares the key with the expected
   character and moves on either way (wrong = red). Each typed position keeps
   one sample `{char, errors, ms}`; `backspace()` changes only the screen, never
   the samples. Rules: `docs/HOW-THE-MODES-WORK.md`, section 3.
3. The space after the last word of a portion → `POST /api/v1/portion` with the
   portion's samples, the words of the stream so far (`recent`) and the words
   typed with a mistake (`retry`). The handler (under a mutex) reads the stats
   from the `Store` and answers with ONE new portion built from the current
   statistics; the samples are not applied here. The whole round's presses are
   applied to the model once, at its end, by `/results` — so a round dropped
   with Tab (marked aborted) never touches the model. The browser adds the
   new portion at the end of the text. Once the round has all its portions the
   browser sends `last: true` and gets none. Backspace can go back one word,
   also into a portion that was sent; those new presses are only shown, not
   sent again.

   **Rendering: one word at a time in Learn.** Portions arrive whole, but the browser
   shows only the words up to `SHOWN_AHEAD = 12` (two portions) after the
   current one; every finished word reveals exactly one more
   (`typing.js` `reveal()`, CSS `.later`). Freetype shows
   everything at once: a words round asks for all its words at the start
   (`words`), a time round for `TIME_AHEAD_WORDS = 36` and then one portion
   per finished portion, so the three visible lines are always full. A hidden word is
   `visibility: hidden`, so it already has its place in the line: nothing
   visible moves when it appears. Two portions are always waiting, so the
   next word to show is always already there.
4. The last letter of the last word of the round (no space needed after it;
   right or wrong) →
   `POST /api/v1/results` with the samples not sent yet, every keypress of
   the round (`keys`), WPM, accuracy, the number of typed characters and the
   keypresses per second; the handler learns from the samples the same way
   (never from `keys`, so nothing is counted twice), computes the consistency
   (`engine.Consistency`) and stores the round: its numbers plus its raw
   keys (see "Raw round records" below). The letters, the open letters, the
   lap and the round are written in one step (`Store.SaveLearned`, issue
   #177): a failed save keeps none of them. Then the next round starts (step 1).
5. After each portion and each round the browser calls `GET /api/v1/stats`
   (letter chips, heatmap, history).

## API

All responses are JSON. Every route except `/languages` and `/auth/*` takes `?lang=en`
(the default). An unknown id is a 400 error.

| Method | Path | Request | Response |
|---|---|---|---|
| GET | `/api/v1/languages` | | `[{id, name, label, layout, alphabet, systemLayout}]` |
| POST | `/api/v1/auth/register` | `{name, password}` | `{name}` and a login cookie; this browser's guest progress moves to the new account; 400 if a rule is broken, 409 if the name is taken, 429 after too many attempts (Retry-After), 404 where accounts are off |
| POST | `/api/v1/auth/login` | `{name, password}` | `{name}` and a login cookie; 401 for a wrong name or password (one answer for both), 429 after too many attempts |
| POST | `/api/v1/auth/logout` | | `{ok: true}`; ends this login and deletes the cookie |
| GET | `/api/v1/auth/me` | | `{accounts, name}`; `name` is null for a guest, `accounts` false where they cannot work (GitHub Pages) |
| GET | `/api/v1/layout` | | layout JSON |
| POST | `/api/v1/portion` | `{samples: [{char, errors, ms, typed?}], recent: [words], retry: [words], start: bool, last: bool, round, portionIndex, ack, level}` | `{portions: [{words, focus, weakest, unlocked, locked}], newLetter, newLap, round}`; `level` (1..10) picks Reinforce's word level once every letter is open; one portion, `AheadPortions + 1` with `start`, none with `last`. The samples are not applied here; a repeated `(round, portionIndex)` gets the cached answer back |
| POST | `/api/v1/abort` | `{round, keys: [the round's raw presses], length, seconds}` | `{ok: true}`; Learn only: a round dropped by Tab — kept raw, shown nowhere, never learned; only its `seconds` count toward the daily goal |
| POST | `/api/v1/results` | `{samples: [{char, errors, ms}], keys: [the same shape, the whole round], chars, wpm, accuracy, perSecond: [keypresses in each full second], length, seconds}` | `{unlocked, newLetter, newLap, consistency}` (`consistency` 0–100, `null` for a round under `ConsistencyMinSeconds`); the round is over: one history entry |
| POST | `/api/v1/progress` | `{unlockAll: true\|false}` | `{unlocked}`; for development only, the interface no longer calls it |
| POST | `/api/v1/reset` | | `{ok: true}`; deletes all progress (the reset button) |
| POST | `/api/v1/learn/step` | `{step}` | `{step, targetWpm}`: Reinforce's speed goal step by hand, 0 to the last; 409 while letters are locked |
| POST | `/api/v1/learn/reset` | | `{ok: true}`; Start Learn over: Learn's letters, rounds and test go, the other modes stay |
| GET | `/api/v1/export` | | `{unlocked, lap, keys: {char: {count, mistakes, errEma, timeEma, timedCount, held}}, sessions: [all, OLDEST first, each with its raw keys and its `epoch`], freetype: {"1000": {sessions}, "1000-long": ..., "sentences": ..., "custom": ...}}`; everything stored for the language (`freetype` is missing in older files; an older file's `freeplay`, `climbLevels` and `group` are ignored on import) |
| POST | `/api/v1/import` | the same JSON as `/export` | `{unlocked, sessions}`; REPLACES the data, or changes nothing if the data is invalid |
| GET | `/api/v1/stats?length=` | | `{keys: [{char, count, errorRate, avgMs, difficulty, level, mastered, unlocked}], sessions: [last 50 of that length, newest first, without raw keys], order: [the unlock order, for the letters a round opened], streaks (as in `/freeplay/stats`), sessionCount (of that length), learnRounds (Learn rounds of every length), lengths: [every length in the history], bests: {length: the fastest round of it}, targetWpm (the goal of the current lap), lap (0 = the first), lastLap, forecast: {known, rounds, more, letter, few} (about how many rounds to the next letter, `history.NextForecast`, HOW-THE-MODES-WORK.md 6.2), masteryMin, masteryLevel (the mastery rule: level masteryLevel after masteryMin presses, for the texts), wordLevels (Reinforce's levels, in words)}`; no `length` = every length |
| GET | `/api/v1/learn/history` | | `{order: [the letters in unlock order], rounds: [{wpm, accuracy, unlocked, lap, letters: [{count, errorRate, avgMs, mastered} or null while locked, in order]}], confusions: [{want, typed, n}]}`: every Learn round oldest first, every length, with the letter model replayed over the rounds' raw keys (`history.LetterTimeline`); the pairs mixed up at least `ConfusionMin` times, at most `ConfusionShown` (`history.Confusions`) |
| GET | `/api/v1/activity?tz=` | | `{days: [{date, rounds, seconds, chars}]}`: every day with a finished round, every mode together, oldest first; `tz` = the browser's minutes from UTC (east positive), so days end at the user's midnight. `seconds` = the rounds' `seconds` as the browser measured them, from the first key to the last with pauses left out, Tab-dropped Learn rounds included (`Session.TypedSeconds`; kept in 0..`MaxTypedSeconds`; a round saved before it was sent counts right letters / 5 / WPM, `storage.RoundSeconds`). The same clock as the live daily goal, so the number never jumps back. The stats screen builds the total time typing, the days in a row and the activity calendar from it |
| POST | `/api/v1/freetype/portion` | `{recent: [words], start: bool, words, list, variant}` (variant: "" or "long" with a list; "sentences" with none) | `{portions: [{words}]}`; with `start`, enough portions for `words` (0..`MaxStartWords`), otherwise one; Freetype: words drawn evenly from the `list` most common ones (`freeplay.Picker.Freetype`); a list not offered is a 400 |
| POST | `/api/v1/freetype/results` | `{chars, wpm, accuracy, perSecond, keys, length, list, variant, seconds}` (your own text: list 0, variant "custom") | `{consistency}`; one entry in the history of that list (`storage.FreetypeMode`) |
| GET | `/api/v1/freetype/stats?list=&variant=&length=` | | `{lists (offered: those of `FreetypeLists` the long list has words for), list (no `list` = the first), sessions, sessionCount, lengths, bests, streaks}` of that list, as in `/freeplay/stats` without the level |

Validation on `/portion` and `/results`: at most 5000 samples (zero is fine:
nothing new to learn), each with exactly one character that is a letter of the
language or a space, `errors >= 0`, `ms >= 0`; `keys` follows the same rules. `/results` also needs
`chars >= 1` (something was typed; 0 means "the number of samples"), WPM 0–400,
accuracy 0–100, and `perSecond` with at most 5000 entries, each between 0 and 5000. `recent` and `retry` hold at most 5000 words; a word that
is not in the list or has a locked letter is simply ignored. Body limit 1 MB.
The page sends the end of a round (`/results`, `/freeplay/results`,
`/freetype/results`) again twice when the server was not reached or
answered 5xx, never after a 4xx (`web/api.js` `saveRound`, issue #149).
Each try carries the same `round` id (up to 64 letters, digits and `-`),
so a round whose first answer was lost is stored once and, in Learn, its
samples are learned once (`Store.HasRound`, migration 011). A try that
failed with a 5xx stored nothing (`Store.SaveLearned`), so the next try
learns it once. A request without `round` is always stored.

**Round length** (`length` on `/results`, `/freeplay/results` and
`/freetype/results`, and
`?length=` on the stats routes): `words-N` or `time-N`, the length chosen in
the round bar of its mode (`web/modebar.js`, `web/settings.js`). Missing = `words-30`, the only length
before the settings existed (`storage.DefaultLength`); every stored round
has one. N is at most `MaxRoundWords` words or `MaxRoundSeconds` seconds
(`config.go`); anything else is a 400. The stats routes show one length at
a time, so a record or an average never mixes a 10-word round with a
60-second one. The letter model does not care about the length: it learns
from every keypress.

Validation on `/import`: `unlocked` between 0 and the number of letters; every key is
one letter of the language or a space; per letter `0 <= mistakes <= count`,
`0 <= timedCount <= count`, `0 <= errEma <= 1`, `timeEma >= 0`; `lap` (and each Learn session's `lap`) between 0 and the last lap, 0 in Freetype; per session WPM
0–400, accuracy 0–100, `chars >= 0`, consistency 0–100 or missing (older files have none). Limits (`MaxImportBytes`,
`MaxImportSessions`) are in `internal/config/config.go`. The browser saves one file
`{app, version, exportedAt, languages: {en: <export>}}`
(see `web/tools.js`, "export / import"). Import and export use only the existing
`Store` methods, so the PostgreSQL store needs nothing extra.

### Raw round records

Each history entry (`storage.Session`) keeps the round's numbers (WPM,
accuracy, characters, consistency, letters open after it) and its raw
record `keys`: every typed position of the round in order, with its
mistakes and its time, exactly as the browser measured it (a skipped letter
has no entry). Freetype rounds keep their keys too; they are never
learned from. The letter model can be rebuilt from the keys by running
`engine.Apply` over them round by round, so other parameter values can be
tried on real data. A round with its keys is
5-7 KB of JSON; `MaxImportBytes` allows about 10 000 rounds in one import.
History from before this has no keys (the field is missing). `device`
says what the round was typed on (`keyboard` or `touch`; missing in older
history); it is recorded only, the model does not use it (HOW-THE-MODES-WORK.md,
section 9). `/stats` and
`/freeplay/stats` and `/freetype/stats` drop the keys (`Session.Summary`), `/export` has them.

Every `Store` method takes the user id and the language id; the language is
always `en` now.
`userOf` (`internal/api/auth.go`) decides the user of every request: the
owner of a valid login cookie, or else this browser's guest (see "Guests"
below).

## Accounts

Standard library only. A user signs up or logs in with a name and a
password; the answer sets a login cookie, and from then on every request of
that browser works with that user's data.

- **Names**: 3..20 characters of `a-z`, `0-9`, `_`, `-`, stored in lower
  case (`auth.NormalizeName`, `auth.ValidName`). Latin only, so look-alike
  letters from other alphabets cannot make two accounts that look the same.
- **Passwords**: 8..128 characters (NIST SP 800-63B). Only a hash is stored:
  `pbkdf2-sha256$<iterations>$<salt>$<key>` (`auth.HashPassword`), 600 000
  rounds of PBKDF2-SHA256 (OWASP, 2023) with a random 16-byte salt. One check
  takes about 0.3 s: nothing for a login, very slow for guessing millions of
  passwords from a stolen database. The comparison is constant-time.
- **Login cookie** `tt_login`: a random 256-bit token (`auth.NewToken`).
  The store keeps only its SHA-256 (`AddLogin`, `LoginUser`), so the stored
  data cannot be used to log in. The cookie is `HttpOnly` (page scripts
  cannot read it), `SameSite=Lax` (other sites cannot send requests with it),
  `Secure` over HTTPS, and lasts 30 days (`Auth.LoginDays`). Logging out
  deletes that login on the server.
- **Wrong name or wrong password** get the same 401 answer, and an unknown
  name is still checked against a dummy hash, so neither the answer nor its
  time tells which names exist.
- **Signing up keeps the guest's progress.** `CreateUser` moves the
  guest's rows (letters, rounds, Freetype, every language) to the
  new account in the same transaction, so the guest is left empty. Logging
  in to an existing account moves nothing: the account keeps its own data
  and the guest's stays with the guest cookie (two letter models are never
  mixed). The shared fallback guest of a browser without cookies is never
  moved.
- **Guests**: every browser without a login is its own guest. `WithGuest`
  (`internal/api/auth.go`) wraps the whole site: a request without the
  cookie `tt_guest` gets a new random token in it (same kind as a login
  token, `HttpOnly`, `SameSite=Lax`, a year: `Auth.GuestDays`). The page
  itself (`index.html`) is the first request, so the cookie is there before
  the scripts' API calls. The guest's user id is `g` + the SHA-256 of the
  token (`storage.GuestPrefix`), so the database never holds the cookie that
  opens a guest's data. A browser that refuses cookies falls back to the one
  shared `storage.LocalUser` ("local"). Clearing cookies starts a new guest:
  the old guest's data stays in the store, unreachable (not cleaned up yet).
- **Limit on attempts** (`auth.Limiter`): per IP address, at most
  `Auth.Attempts` (10) failed logins plus sign-ups in `Auth.AttemptWindow`
  (15) minutes; then `429 Too Many Requests` with `Retry-After`, even for the
  right password, until the window ends. Every sign-up counts (each costs a
  slow hash); a successful login clears the count. The counts live in memory,
  so a restart forgets them, and several server processes would each count
  on their own. No background goroutine: old windows are dropped while
  counting.
- **Behind a proxy**: on a real server a reverse proxy (nginx, Caddy) ends
  HTTPS and passes requests to this server over plain HTTP. Start it with
  the environment variable `BEHIND_PROXY=1` there (`config.BehindProxy`):
  cookies are then `Secure` when the proxy's `X-Forwarded-Proto` is `https`,
  and the limit counts the last address in `X-Forwarded-For` (the one the
  proxy added; earlier ones come from the client). Without the variable both
  headers are ignored, because without a proxy anyone could write them.
- All numbers are in `internal/config/config.go` (`Auth`).

On GitHub Pages accounts are off (`build_wasm.go`): requests never leave
the page there, so there are no cookies, and the only guest is `local`.

## Modes

Two modes in the code (`web/modes.js`); only Learn is shown. Freetype is
hidden: no tab, no setting, no stats switch, and a saved hidden mode opens
Learn (`SHOWN_MODES` in `web/state.js`). The browser checks turn it back on
with `localStorage` `tt.allModes = "1"`. They share no data: Freetype has its
own routes and histories (the `AddFreeplaySession` family of the store, with
a mode: `FreetypeMode(n, variant)` for each list and kind of text).

- **Learn**, for beginners, is the adaptive trainer described in
  docs/HOW-THE-MODES-WORK.md: the `/portion` and `/results` routes, the letter model,
  unlocking from the home row. It has no stats screen: its progress is the
  letters row (a colour and a bar per letter, tooltips with what each still
  needs, the mastered letters fading from the keyboard) and the line over
  the text (`web/learnpanel.js`); its rounds follow one another with no
  result screen. Once every letter is open it becomes Reinforce (word
  levels and the speed goal, docs/HOW-THE-MODES-WORK.md section 6.1).
- **Freetype**, for just typing (`internal/freeplay`, docs/HOW-THE-MODES-WORK.md
  section 13): words drawn evenly from the n most common of the long list
  (`free.txt`), n picked in the row above the text (200 to 25 000;
  remembered in `tt.list.<lang>`), plain or of a kind
  (long words, real sentences from `sentences.txt`, or your own text: `tt.variant.<lang>`;
  your text in `tt.custom.<lang>`, never sent, `web/customtext.js`).
  Nothing adapts; every list and kind has its own history and bests.
  `/freetype/results` drops any samples.

The chosen mode is interface state: it lives in the browser (`localStorage`,
`tt.mode.<lang>`), not on the server. Reset clears Freetype's histories;
export / import carry them.

## Concurrency

The random generators are not safe for parallel use, so `Portion` (Learn) and `Freetype` run under
`genMu`. "Read stats → update → save" must not interleave between two
requests, so it runs under `writeMu`. That is enough while one server
process serves everyone, with either store. Several server processes on one
database would need a transaction around it instead (`SELECT ... FOR UPDATE`).

## PostgreSQL

`go run .` uses the memory store. With the environment variable
`DATABASE_URL` set (for example
`postgres://user:password@localhost:5432/typetrainer`), `openStore` in
`build_server.go` opens `storage.PostgresStore` instead, and the log says
`storage: PostgreSQL`. The address is not in the code, so no password ends up
in git. How to install PostgreSQL on Windows: DEVELOPMENT.md.

- **One third-party module.** The code uses only the standard `database/sql`
  package; `github.com/jackc/pgx/v5/stdlib` is imported only to register the
  driver that speaks PostgreSQL's protocol (the standard library has none).
  Another driver would be a one-line change. pgx needs Go 1.25, so `go.mod`
  says 1.25 (1.24 is no longer supported by the Go team either). The
  WebAssembly build does not include it (`postgres.go` has a build tag).
- **Tables** (`internal/storage/migrations/001_init.sql`): `users`, `logins`
  (token hashes), `progress` (unlocked letters, Learn's lap (migration 010)),
  `key_stats`
  (the letter model) and `sessions` (the history of every mode, `mode`
  column: `learning`, `freetype-<n>` for each
  Freetype list, `freetype-<n>-long` for its long words,
  `freetype-sentences` and `freetype-custom` (migration 009); the raw keys as `JSONB`;
  `round_id`, unique per user and language, so a round sent twice is kept
  once (migration 011); `epoch`, the wipe counter of migration 012;
  `aborted`, the Tab flag of migration 013: such a round is kept raw but
  shown nowhere and never learned), and
  `epochs` (that counter itself). Every data table is keyed by user and
  language. `user_id` is text and not a foreign key, because guests
  (`g...`, `local`) have data but no row in `users`; an account's id is
  `u` + `users.id`.
- **Migrations**: at start `migrate` applies every file in `migrations/`
  that the table `schema_migrations` has not recorded yet, in number order,
  each in a transaction. A new version of the app updates an old database by
  itself; a change to the tables is a new file (`002_...sql`), never an edit
  of an old one.
- **Transactions**: `SaveKeyStats` (delete + insert the letters),
  `SaveLearned` (the letters, the progress and the Learn round) and `Reset`
  (three tables) are all-or-nothing.
- **Tests**: the same tests run on both stores (`store_test.go`). The
  PostgreSQL run needs `TT_TEST_DATABASE_URL` (it empties the tables, so use a
  separate database); the pull request checks start a PostgreSQL 16 for it.
  The app tests (`setup_test.go`) always use memory.

## Two ways to run: server and GitHub Pages

The same code runs in two places. Go build tags pick the entry point:

| | Normal server | GitHub Pages |
|---|---|---|
| Entry point | `main.go` (`//go:build !(js && wasm)`) | `main_wasm.go` (`//go:build js && wasm`) |
| Build | `go run .` | `GOOS=js GOARCH=wasm go build -o app.wasm .` |
| Where Go runs | on your computer | in the visitor's browser (WebAssembly) |
| How the frontend reaches the API | HTTP to `localhost` | `pages/wasm-shim.js` replaces `fetch` for `/api/v1/` and calls `goApi(method, path, body)` |
| Statistics | until the server stops (in memory) or for good (PostgreSQL) | in the page's memory, copied to the browser's IndexedDB after every round and loaded back on the next visit (`pages/wasm-shim.js`) |
| Accounts | yes | no (no cookies inside the page; `/auth/me` says `accounts: false`) |

`main_wasm.go` does not listen on a port. It builds the same handler with
`newApp()` and gives JavaScript one function, `goApi`, which runs a request
through that handler (`httptest.NewRequest` / `NewRecorder` serve as an
in-memory request and response). Engine, store, validation and JSON are
therefore identical in both versions; only the transport differs.

The workflow builds the site by copying `web/`, adding `app.wasm`,
`wasm_exec.js` (the Go runtime for browsers, shipped with Go) and the shim,
and inserting two `<script>` tags before `app.js` in `index.html`. The
server version never loads them. Every file is linked with `?v=<commit>`
(`app.wasm` too, through the shim, and every `import "./x.js"` inside the
modules), because GitHub Pages lets browsers cache files for 10 minutes:
without it, a visitor right after a deploy could get the new page with an old
module or an old `app.wasm`.

## Where to add something

Where something new is likely to be added:

- a new **API route**: `internal/api/handlers.go` (routes)
- a new **Store method**: `internal/storage/store.go` (both stores and a test)
- a new **table or column**: `internal/storage/migrations/` (a new numbered file)
- a new **tunable number**: `internal/config/config.go`
- a new **interface text**: `web/i18n.js` (`docs/TEXT-RULES.md`)
- a new **setting**: one entry in `web/settings-list.js` and its texts (`docs/CODE-RULES.md`, "Settings")
- a new **navigation item**: `web/index.html` (`nav.rail`) and `web/input.js`
- a new **frontend module**: its layer in "Frontend modules" above and
  `webmodules_test.go`

