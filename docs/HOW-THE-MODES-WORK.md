# How the modes work

How the trainer decides what you type next, as the code does it now. Every
formula here is exactly what the code does, and every number comes from
`internal/config/config.go`. Why each part is built this way: section 2.

Section 14 is Learn's accuracy rules: its accuracy, focus and next letter
are built and on (`AccWindow = 100`) and replace the level, mastery, focus
and unlock rules of sections 5-7 while letters are still closed; the
finger tour is built (in Welcome). Once every letter is open
in Learn, the rules of sections 5-7 apply as written (and in Reinforce).

Code: `internal/engine/model.go` (the math) and `internal/engine/generator.go`
(unlocking and word selection). Tests that check each claim:
`internal/engine/*_test.go`.

---

## 0. Three algorithms, one loop

The product has three modes, and each one trains one skill: Learn trains
accuracy, Reinforce trains speed, Freetype trains rhythm. Each mode runs the
same loop:

```
your presses  ->  a model of you  ->  a decision  ->  your next presses
(letter,          (numbers per        (which words
 right or wrong,   letter and per      come next, and
 milliseconds)     pair of letters)    when you move on)
```

This loop is what "adaptive" means here: the next text and the next step
are computed from your own measured presses. A fixed or random text does
not read them. All three modes share the input. Each mode has its own
model and its own output, and each mode changes its model once per round,
at `/results`.

### 0.1 Learn: accuracy

| | |
|---|---|
| Input | For every press: the letter, and whether it was right on the first try. |
| Model | The accuracy of each letter over its last `AccWindow = 100` presses, with `AccPrior = 3` presses imagined as misses: `accuracy = right / (presses + 3)`. An unseen letter is at 0 %. A letter falls below 85 % (`AccSlip`): it has slipped, and it stays slipped until it is back at 90 % (`AccMaster`). |
| Output 1: the next letter | It opens when the newest letter is at 90 % or more and no open letter is under 85 %. At most one letter per round. The order is fixed: the home row first, then by frequency. |
| Output 2: the focus | The older letter that slipped and is the least accurate. If no older letter slipped, the newest letter. Each portion of 6 words is built around the focus letter. |
| Output 3: the words | Real words made only of open letters. A word's weight is `exp(mean difficulty of its letters / 0.1)`, where `difficulty = 1 - accuracy` (+ 0.2 while a letter is under 90 %). Weak pairs make a word 3 times more likely, a recent word gets `x 0.3`, and a word comes at most 2 times per round. |
| Status | Built. Sections 6, 7 and 14. |

### 0.2 Reinforce: speed

| | |
|---|---|
| Input | Learn's input, plus the time of each clean press. |
| Model | For each letter: the error rate (EMA, `ErrAlpha = 0.05`) and the time of a press (EMA, `Alpha = 0.1`). Both are compared with the target speed: `TargetMs = 12000 / TargetWPM`. Together they give a difficulty and a level from 0 to 100 (section 5). The weak pairs are the pairs of letters with the highest cost over the last 20 rounds; a miss costs 1000 ms. |
| Output 1: the words | At the chosen word level (the 200 ... 25 000 most common words). Each portion has a focus letter, drawn with softmax over difficulty. Slow letters and weak pairs come more often. |
| Output 2: the speed goal | It goes up by one step of 10 wpm once every letter is mastered at the current goal. It starts at 30 and stops at 200. |
| Output 3: the word level | Moves by itself. Each round is judged at the speed goal of its own step (`freeplay.Judge`): it **passes** at the goal or faster with `PassAccuracy = 96 %` or more, **fails** under `FailAccuracy = 92 %` or under `FailSpeed = 90 %` of the goal, and **holds** in between. Three passes with no fail between them (`PassesUp = 3`) move one level up, two fails in a row (`FailsDown = 2`) one level down; a hold keeps the passes and breaks a row of fails (`freeplay.Standing`, over the last run of rounds at one level). The zone between keeps the level from jumping on one uneven round. A level picked by hand starts a run of its own: the browser follows a move only right after a round at the level it uses (`followLevel` in `web/typing.js`), with a short message; the stats say how close the next move is. |
| Status | Built: the words, the speed goal (section 6.1) and the word level. |

### 0.3 Freetype: rhythm

| | |
|---|---|
| Input | Learn's input, plus the time between keys and between words. |
| Model | Planned: for each pair of letters, how much its time varies, and the stumbles (a pause much longer than your own median). The number of the round is consistency (section 0.5). |
| Output | Planned: words with your stumble pairs come more often, and the level moves by consistency. |
| Status | Not built. Today Freetype draws words evenly from the N most common words, and consistency is only measured and shown. |

### 0.4 Advice, on top of the three

Planned: each mode's stats read its last rounds and say one thing, for
example "accuracy holds: try Reinforce" or "slow down, accuracy is
dropping". Advice never switches the mode by itself. It is not part of the
adaptive loop. It only helps the person choose a mode.

### 0.5 The numbers of a round

| Number | Formula | Where |
|---|---|---|
| WPM | `(correct characters / 5) / minutes` | `web/typing.js` `currentWpm` |
| Accuracy | `right presses / all presses * 100` | `web/typing.js` `currentAccuracy` |
| Consistency | `cv = σ / μ` of the keys per second, then `100 * (1 - tanh(cv + cv³/3 + cv⁵/5))`; none under 5 seconds | `internal/engine/consistency.go` |


### 0.6 The check-in: where a mode starts

The first time a mode is opened, one question or a 30-second sample sets
where it starts, so the loop does not begin from the same place for a
beginner and for someone who already types (`internal/api/checkin.go`,
`web/checkin.js`; the numbers in `Checkin`, `internal/config/config.go`).
Each answer changes exactly one thing; Skip keeps today's start. A mode
that already has rounds of its own skips it. The sample's presses are sent
nowhere: only its speed and accuracy reach `/checkin`.

| Mode | Input | Rule | Output |
|---|---|---|---|
| Learn | "Have you learned touch typing before?" no / a little / yes | no → the start (6 letters); a little or yes → `SomeLetters = 10` (the home row and e, i, r); only while Learn has no rounds, and never fewer letters than now | the letters open at the start; "yes" also offers Reinforce |
| Reinforce | 30 s on the 200 most common words: speed, accuracy | goal = the first speed goal step at least `GoalMargin = 5` wpm above the speed (47 → 60, 25 → 30, over 195 → 200); word level 1k from 97 %, 500 from 94 %, else 200 | the speed goal step and the word level |
| Freetype | 30 s on the 1 000 most common words: speed, consistency | list 5k from 70 wpm, 1k from 40, else 200; the consistency is kept in the browser for Freetype's own loop (0.3) | the word list |

---

## 1. The idea in plain words

1. For every letter the program remembers two numbers: how often you make a
   mistake on it (about the last 40 presses count) and how long it takes you
   (about the last 20). Recent presses weigh more than old ones.
2. The two numbers make one **difficulty** from 0 to 1, and the letter's
   **level** is the same number turned around: `level = 100 * (1 - difficulty)`.
   A new letter starts at level 30, so it gets practised.
3. A letter is **mastered** once it reaches level 80 after at least 40
   presses, checked at the end of each round. Mastered stays mastered: later
   slips only lower the level, so its words come more often. The next letter
   opens when every open letter is mastered. The home row comes first, then
   the most frequent letters.
4. Every 6 words the program picks one **focus** letter, more likely a weak
   one, and all 6 words contain it. Common words come first; a mistyped
   word comes back two lines later.
5. Only real words made of open letters. Learning has no settings: the
   model alone decides what to practise.
6. **Reinforce**, its own tab, open from the start with every letter: you
   pick how many common words to type (200 up to 25k), weak letters and letter pairs come
   more often, and the speed goal grows in steps up to 200 wpm (section 6.1).

Freetype (section 13) is hidden and not developed; it never feeds this model.

## 2. Design decisions

| Decision | Alternative | Why this one |
|---|---|---|
| Real words only | Generated letter combinations (pseudo-words) | Words are what people type in real life, they are easier to read and read faster, and they keep motivation up. Cost: early lessons have fewer possible texts. Handled by the unlock order and the `MinWords` rule (section 6). |
| Statistics per letter | Per key pair (bigram), per word | Simplest unit that can be explained and shown on the keyboard. Reinforce brings weak pairs more often (section 6.1). |
| Smoothed recent statistics (EMA) | Lifetime totals | Skill changes. A lifetime error rate remembers mistakes from weeks ago forever. |
| Probabilistic choice (softmax) | Always train the single worst letter | Always drilling one letter is boring and over-fits; probabilities give focus *and* variety. |
| Gradual unlocking | All 26 letters from the start | A beginner cannot learn 26 finger positions at once. Experienced typists go through the same steps, only much faster. |
| Statistics, not a neural network | ML model | No training data is needed, every decision can be explained with a formula, and it runs instantly on the server. |

### Why the home row first

Touch typing courses start from the home row, so the fingers learn where they
rest. Why a boost and not "all eight resting keys first": with real words a
rare home letter would starve. `j` has one word among the first nine letters
("jerk"), and a focus on it would repeat that one word. With the boost, the
frequent home letters (`a s l d f`) come first and a rare one (`j`) keeps its
place by frequency. 4 is the smallest boost that brings `f`, the left index
finger's home, into the first lesson.

### Why mastery has a margin

Without it, one mistake (ErrEMA jumps by `Alpha` = 0.1, over the 0.05 goal)
takes a mastered letter back for about 7 of its presses, and the next letter
needs every open letter to be clean at the same moment, so the wait grows with
every letter opened. A margin between "reach" and "keep" (hysteresis, as a
thermostat has) makes the state stable: a letter is reached once and lost only
for a clear reason.

### Why the forecast draws a line through the level

The forecast (section 6.2) fits a least-squares line through a letter's
level over its last 10 rounds. Under Learn's accuracy rules the level is
the letter's accuracy. One round is noisy: a single slip lowers the
accuracy by about one point, and it lifts `ErrEMA` by `ErrAlpha` = 0.05. A
line through ten rounds follows the trend, not the last slip. While the
line does not rise, the forecast gives no answer rather than a wrong one.

### Why a straight line for the speed

It is explainable in one sentence and needs no model of learning. Its weakness:
how fast a letter improves changes (it gets more presses once it becomes the
focus, and slows down near its limit), and a straight line sees that late.

### Why UnmasteredBonus

A letter a little too slow (450 ms, speed part 0.08) would otherwise get about
the same share as any mastered letter, and could block every new letter for
good.

### Why common words first

Without it, words are drawn evenly from the whole list: about half of the
possible words are from the rare end (ranks 1000-2000), already in the first
lesson. Common words are also short, so this rule makes the first lessons
shorter too.

### Flat while learning, ramp after

One difficulty at a time: while letters open only the words change their
letters, not their rarity; after the full alphabet only the rarity grows.

### Why at most two uses of a word

The repeat penalty alone is weak when a letter has few words: `z` and `q` have
10 words each in the whole list. With the cap a round on them has about 20
different words in 30. The price is a few percent more typing to the goal,
because the focus letter gets fewer words when its words run out.

### Why real sentences

Marks sprinkled on random words read as nonsense, so Freetype's sentences are
real ones.

## 3. What is measured

Typing never stops on a mistake (as on monkeytype, so the rhythm is kept):

- a wrong letter turns red and the cursor moves on;
- Backspace goes back inside the current word (Ctrl+Backspace: the whole
  word); at the start of a word it goes back into the previous word, at most
  one word back (from inside that word it does nothing more);
- space in the middle of a word jumps to the next word; the rest of the word is
  **skipped**;
- letters typed where a space belongs are **extra** letters, shown in red
  after the word.

With the setting **Stop on a mistake** (off by default) the cursor instead
waits on a wrong key until the right one comes: there are no skips and no
extra letters, and every wrong press there counts as one more `errors` of
that position. The sample is the same as without the setting: one per
position, its errors, and `ms` to the first press there.

For every **position of the text that was typed** the browser sends one sample:

```
{ char: "s", errors: 1, ms: 412 }
```

- `errors` – how many wrong keys were pressed at this position, including new
  attempts after a Backspace. Backspace changes only what the screen shows: a
  mistake corrected with Backspace **still counts**, because the first attempt
  was wrong. This also holds after going back one word: the samples of the
  first pass stay, the retyped letters only add what was missing (skipped
  letters) and new mistakes, and no letter is measured twice. If the
  previous word's portion was already sent to the server, its new presses
  are shown but not sent again. Extra letters count as errors of the space after the word.
- `ms` – time from the previous keypress to the **first** press at this
  position, measured with `keydown` and `performance.now()` (sub-millisecond
  resolution).
- `typed` – the first wrong key pressed at this position (only when
  `errors` > 0; missing in rounds from before 2026-10-21). The model never
  reads it: the stats screen counts which keys you mix up with it.
- **Skipped** letters were never typed, so they send no sample: a skip says
  nothing about how well the user types that letter. They still lower the
  accuracy (each skipped letter counts as one wrong press).

Some times are **ignored** for speed (but still count for accuracy):

1. the time of a position whose sample has a mistake (`errors > 0`), because
   the first attempt there was a wrong movement;
2. gaps longer than `MaxIntervalMs = 2000 ms`, which are pauses;
3. the first press of a round (no previous press) and the first press
   after a Backspace (that time is spent correcting): both get `ms = 0`
   (not measured);
4. the first letter of a word: the press right after a space, and the
   first press of a batch the browser sends (a batch is whole words). Its
   time includes reading the word, not only the finger, so letters that
   begin many words would look slow (`engine.Apply`). The raw times stay
   in the round's keys.

**Held keys are ignored.** When a key is held down, the operating system
repeats it every ~30 ms (`keydown` with `repeat = true`). The trainer ignores
these repeats (Backspace excepted, so holding it still deletes): in a word
like `all` a repeated `l` would otherwise be recorded as a correct press
10 times faster than any real finger movement.

There is deliberately **no lower limit** on a single interval. keybr.com drops
results under 40 ms, but it applies that to the *average* time of a letter in
a whole lesson. Single intervals of a fast typist are often that short when the
next key is pressed before the previous one is released (rollover, typical for
letters typed with different hands), so a per-press limit would throw away
exactly the fastest real presses and make fast typists look slower.

The press right after a wrong letter is measured normally: the cursor has
already moved on, so it is an ordinary finger movement from the previous key.

Exercise-level numbers, shown to the user:

```
WPM      = (characters that are right / 5) / minutes  (standard: 1 word = 5 characters)
raw      = (all keypresses / 5) / minutes             (wrong presses count too)
accuracy = correct keypresses / all keypresses * 100  (a skipped letter = one wrong press)
```

`raw` is shown on the result screen of Freetype; it is not sent to
the server and the model does not use it.

**Consistency** (`internal/engine/consistency.go`) tells how even the speed
was during a round. The browser counts the keypresses in every full second
from the first keypress (a second without keys is a 0; the last, unfinished
second is left out because it would always look slow). The server computes:

```
cv          = standard deviation / mean of the counts   (coefficient of variation)
consistency = 100 * (1 - tanh(cv + cv^3/3 + cv^5/5))     (0..100 %)
```

`cv` has no unit, so keypresses per second and raw WPM per second (12 times
more) give the same value. The `tanh` maps `cv` from `[0, ∞)` to `100 … 0 %`;
the terms inside it are the start of the series of `artanh(cv)`, so for a small
`cv` the result is close to `100 * (1 - cv)`, and it never goes below 0. The
formula is the one monkeytype uses (`kogasa` in `packages/util/src/numbers.ts`),
so the numbers can be compared with that site. A round shorter than
`ConsistencyMinSeconds = 5` seconds gets none (too few values to judge). It is
kept in the history and shown to the user; the model does **not** use it.

## 4. Per-letter statistics: exponential moving average

For each letter we keep two smoothed values:

- `ErrEMA` – the probability of making a mistake on this letter,
- `TimeEMA` – the typical time per keypress, from clean presses only.

Each new sample `x` updates the value:

```
value_new = value_old + a * (x - value_old)
a = max(Alpha, 1 / n)              n = number of samples so far
Alpha = 0.1 for time, ErrAlpha = 0.05 for mistakes
```

Mistakes use the smaller `ErrAlpha`: they are rare events, so the error rate
needs more presses to mean something, and one slip moves it by at most 5
points instead of 10.

For errors `x = 1` if the occurrence had a mistake, otherwise `0`. For time
`x = ms`.

**Why `max(Alpha, 1/n)`.** A plain EMA starts at 0 and needs many samples
before it means anything. With `a = 1/n`, the first samples give an exact
ordinary average (1st sample: weight 1, 2nd: 1/2, ...). Once `1/n` drops below
`Alpha` (after 10 samples), the update becomes a normal EMA.

**What `Alpha = 0.1` means.** The weight of a sample is multiplied by
`(1 - Alpha) = 0.9` with each newer sample, so it halves every
`ln 0.5 / ln 0.9 ≈ 6.6` occurrences of the letter. In practice the model
"remembers" about the last 20 occurrences. A larger `Alpha` reacts faster but is
noisier.

Lifetime counters (`Count`, `Mistakes`) are kept too, but only for display and
for the "enough evidence" rules below.

## 5. From statistics to difficulty

### 5.1 Raw difficulty

```
TargetMs  = 60000 / (TargetWPM * 5) = 400 ms           (TargetWPM = 30)

errorPart = clamp( ErrEMA / MaxErrorRate )             MaxErrorRate = 0.25
speedPart = clamp( (TimeEMA - TargetMs) / (SlowMs - TargetMs) )     SlowMs = 1000 ms

raw = ErrorWeight * errorPart + SpeedWeight * speedPart      0.6 and 0.4
```

`clamp` limits a value to the range 0..1.

- `errorPart` is 0 with no mistakes and 1 when a quarter or more of the
  occurrences have a mistake. Error rates of real typists are small numbers,
  so dividing by 0.25 spreads them over the whole 0..1 range.
- `speedPart` is 0 at the target speed or faster and grows linearly to 1 at
  one second per character. Being faster than the target is not rewarded
  further: the goal is to reach the target on every letter.
- Mistakes weigh a bit more (0.6) than speed (0.4). Touch typing courses
  usually put accuracy first, because a fast but wrong movement is being
  memorised.
- A letter with no clean, timed press yet gets `speedPart = 1`.

### 5.2 Shrinkage towards a prior

A single correct press would make a letter look perfect. So the raw value is
pulled towards an assumed **prior** while there is little evidence:

```
difficulty = (n * raw + k * prior) / (n + k)        prior = 0.7, k = 5, n = samples
```

This is a Bayesian average (also called additive smoothing). With `n = 0` the
difficulty equals the prior, so a **new letter starts as fairly hard** and gets
practised. After 5 samples data and prior weigh the same; after 50 the data
weighs 10 times more.

### 5.3 Worked example

Seven letters are open. The user is good at `e` and struggles with `s`:

| Letter | n | ErrEMA | TimeEMA | errorPart | speedPart | raw | difficulty | weight `exp(d/T)` |
|---|---|---|---|---|---|---|---|---|
| e | 60 | 0.02 | 300 ms | 0.08 | 0 | 0.048 | 0.098 | 2.66 |
| s | 30 | 0.12 | 520 ms | 0.48 | 0.20 | 0.368 | 0.415 | 63.4 |

If the other five letters look like `e`, then `s` is chosen as the focus with
probability `63.4 / (63.4 + 6 * 2.66) ≈ 80 %`, instead of `1/7 ≈ 14 %` for
uniform choice.

## 6. Unlocking letters

**Order.** The letters under the resting fingers come first, then every
other letter by how often it occurs in the word list of the language
(`Generator.SetHome`). The resting keys come from the layout data
(`layout.RestingKeys`): in the row with the home bumps, one key per finger,
`a s d f j k l ;` on QWERTY. Inside each group the more frequent letter
comes first. One exception: the most common other letter takes the last
place of the first lesson, before the rarest resting letters, because
without a vowel there are hardly any words (the home row alone makes 21).
The order is computed from the data, not written by hand (first lesson in
bold):

**s a l d f e** k j i r n t o c p g u m h b y v w x z q

So the fingers start where they rest, and every later letter is one step
from home, in the order real words need it. After the first lesson the
letters open one at a time.

**First lesson.** `StartLetters = 6` letters, more only if fewer than
`MinWords = 40` real words can be built from them: these six make 99.

**Level and mastery** (`engine.Level`, `MeetsGoal`, `Mastered`):

```
level          = 100 * (1 - difficulty)                     0..100, section 5
meets the goal = n >= MasteryMinSamples (40)  and  a timed press  and  level >= MasteryLevel (80)
```

The level is the difficulty of section 5 seen from the other side, so the
number the user sees is the number that picks the words. Mistakes weigh 0.6
and speed 0.4, so level 80 is reached by few mistakes at a speed near the
goal; it can not be reached by speed alone. Examples with many presses:
no mistakes at 400 ms (30 WPM) or faster is level 100; 4 % mistakes at the goal
speed is about 90; no mistakes at 700 ms (17 WPM) is about 80.

The rule is checked once per round, on the presses of the whole round
(`engine.Apply`), and once met the letter **stays mastered**
(`KeyStat.Held`): mastery is never taken back. A letter that gets worse later
only has a lower level, which makes it the focus more often (section 7).
Checking once per round keeps one lucky stretch inside a round from
deciding it.

**Next letter.** When a round ends (section 7): if every open
letter is mastered, the next letter in the order opens. At most one letter per
round. A new letter
starts at the prior difficulty 0.7 (level 30), which is higher than the
difficulty of the mastered letters, so it automatically becomes the focus.

**Experienced typists** start from the first letters too (there is no "open
all letters" choice): typing well, they master the open letters quickly and
unlock the rest in a short time. "Start Learn over" in the settings deletes
Learn's letters and rounds and starts again from the first lesson.

### 6.1 Reinforce: after every letter is open

**Reinforce** is its own tab, open from the start: a bar over the letter
row with the word level and the speed goal (`web/modebar.js`,
`api/reinforce.go`). It uses the same letter model as Learn: a Reinforce
round's presses count for each letter as a Learn round's would (with the
accuracy rules while Learn is not finished), so letters typed well here open
fast later in Learn; but only a Learn round opens a letter. Its portions use
every letter, with the focus and word weights of sections 4-7 (softmax over
difficulty); two things are its own.

**Word level.** The learner picks one of ten levels, the most common
200, 500, 1k, 2k, 3k, 5k, 7.5k, 10k, 15k or 25k words of the long list
(`free.txt`, `Freeplay.Levels`). The words come
only from that level (`Generator.WithWords`), the whole level at once:
no rank penalty inside it. The browser keeps the choice per language
(`tt.wordLevel.<lang>`) and sends it with every portion; a round keeps its
level (`sessions.level`).

| Level | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 |
|---|---|---|---|---|---|---|---|---|---|---|
| Words | 200 | 500 | 1 000 | 2 000 | 3 000 | 5 000 | 7 500 | 10 000 | 15 000 | 25 000 |

The steps roughly double because word frequencies fall steeply (Zipf's law).
The level moves only by hand for now. The rule planned for moving it by
itself (section 0.2) is already in `internal/freeplay` (`Judge`,
`Standing`): a round at 96 % accuracy or more passes, under 92 % fails,
between them counts for nothing; three passes with no fail between them
move one level up, two fails in a row one down.

**Weak pairs** (in Learn too, from its first rounds). A typist who can
already type is slow on a movement between two keys, not on a letter. From
the raw keys of the last `WeakRounds = 20` Learn rounds
(`internal/freeplay/weak.go`):

```
cost(pair) = (sum of clean timed ms + mistakes * 1000) / (clean timed presses + mistakes)
```

The `WeakPairs = 3` most costly pairs with at least `WeakMinSamples = 15`
presses and a cost over the typist's average are weak. They make their
words more likely: the weight of a word grows by `PairBoost` (2, so 3
times) for each weak pair in it (`Generator.WithPairs`).

**Speed goal steps** (`engine.LapParams`, `lap.go`):

```
goal of step n = min(TargetWPM + n * StepWPM, MaxWPM)
               = 30, 40, 50, ... 200 wpm   (step 0 .. 17; LastLap = 17)
```

- **When.** After the presses of a round are applied, if every letter is
  open and mastered at the step's goal (`LapDone`) and the step is not the
  last, the next step starts. The learner can also move a step up or down
  by hand (the − and + of the bar, `/learn/step`). The server keeps the
  step per user and language (`Store.Lap`, `progress.lap`) and every Learn
  round keeps the step it was typed in (`sessions.lap`).
- **What changes.** Only `TargetWPM` (`Params.ForLap`): the difficulty of a
  letter, its mastery and the word choice all follow it. `StartLap` turns
  `Held` off for every letter: each has to *reach* the new goal, since
  mastery is otherwise never taken back. The EMAs are kept, so a letter
  already good enough counts as mastered again at the end of its next round
  and the slow ones become the focus. No letter is closed again.
- **Why to 200.** Few typists pass 200 wpm even on common words; a higher
  goal would mean nothing to almost anyone.
- **The stats screen** judges every round at the goal of its own step
  (`history.LetterTimeline`), and starts a new step there as the trainer did.

### 6.2 The forecast: rounds to the next letter

The next letter's tooltip and the Learn stats say about how many
rounds the next letter is away ("≈ 6 rounds"; with every letter
open, on the stats screen, "Next step in ≈ 6 rounds"). It only reads the numbers; nothing in the
model or the word choice sees it (`engine.RoundsToMastery`,
`history.NextForecast`).

For each open letter that is not mastered, its numbers after each of its
last `Window = 10` rounds (presses `n`, `ErrEMA`, `TimeEMA`, and from them
the level; the rounds are the replay of the stats screen, the last point
the model as it is now). The two parts of the mastery rule each get an
estimate, and the slower decides:

```
per round = presses in the window / rounds in the window
presses:  (MasteryMinSamples - n) / per round
level:    (MasteryLevel - level) / (rise of the least-squares line through the level, per round)

rounds for the letter = ceil(max of the parts not met yet), at least 1
rounds to the next letter = max over the letters still waiting
```

The least-squares slope over the points `(i, y_i)` is
`sum((i - mean i)(y_i - mean y)) / sum((i - mean i)^2)`.

**No answer** rather than a wrong one: a letter with fewer than
`MinRounds = 3` rounds (a line through two points is noise), with no timed
press yet, not typed in the window, or whose level does not rise. One
letter without an answer means no answer for the next letter either, since
it waits for all of them; the panel then shows nothing (the letter line
already says what the gold letter needs). More than `MaxRounds = 50` is
shown as "50+".

**Why the slowest letter.** A letter opens only when every open letter is
mastered (section 6), so the next letter is as far away as the letter
furthest from its goal.

## 7. Choosing the words

The text is a **stream**: it is not built in advance but a few words at a
time while the user types, each time from the newest statistics. One piece
of the stream is a **portion** of `PortionWords = 6` words (about one line),
built by `Generator.Portion`. Only words made entirely of open letters are
allowed.

**Step 1, what to train: one focus letter for the whole portion.**

```
P(letter) = exp(d_letter / T) / Σ exp(d_i / T)        T = Temperature = 0.1
```

Here `d` is the difficulty plus `UnmasteredBonus = 0.2` for a letter that is
not mastered yet (only for choosing, never shown as the letter's difficulty):
a letter that holds back the next one is practised more. It gets about 7 times the
weight of an equal mastered letter. While nothing is mastered every letter
gets the bonus, and softmax does not change when every weight is shifted the
same: the first lessons are not affected.

This is the softmax (Boltzmann) distribution, used in the same way in
reinforcement learning for choosing between actions. The **temperature** `T` is
one knob for "how hard to push on weaknesses":

- difficulty 1 against difficulty 0 gives a ratio of `e^(1/0.1) = e^10 ≈ 22000`;
  a gap of 0.1 in difficulty already gives `e ≈ 2.7`;
- with `T → 0` it always picks the single worst letter;
- with a large `T` every letter is almost equally likely.

`exp` is never 0, so no letter disappears from practice. Because the focus is
drawn again for every portion, it changes every few words: the user works on
one letter at a time, but never on the same one for long.

**Step 2, how to train it: the portion's words contain the focus letter, while it has fresh ones.**

```
D(word)   = average difficulty of the word's letters (each occurrence counts)
F(word)   = RankPenalty^(max(0, rank(word) - MaxRank) / RankStep)
P(word)  ~ exp(D(word) / T) * F(word) * RepeatPenalty^(uses in the last RecentWords words)
P(word)  = 0 when it was used MaxRepeats times in the round (and for the word just before or after)
```

A word of the focus letter comes only if it is not among the last
`RecentWords` words. When the focus letter has no such word left (a new
letter with one or two words, like `j` early on), the rest of the portion
takes any open word, by the same weights: the weak letters still lead, and
no word is drummed.

Among words that contain the focus letter, the ones that also contain other weak
letters are preferred.

**Common words first.** `F(word)` works like monkeytype's "English 200",
"1k", "2k" word lists, but without a switch: while the user learns the
letters the text is made of the most common words, and rarer words come in
step by step once the whole keyboard is known. The word lists are sorted by
frequency (section 11), so a word's place in its list, its **rank**
(0 = the most common word), says how common it is.

```
letters still locked:  MaxRank = RankStart
every letter open:     MaxRank = RankStart + (words in the list - RankStart) * min(1, typed / RampLetters)
```

`typed` = letters typed in learning rounds that ended with every letter open
(`Store.LettersTyped`, the sum of `chars` of those rounds in the history).
With `RankStart = 200`, `RankStep = 200` and `RankPenalty = 0.5`, every lesson
before the full alphabet allows the 200 most common words freely, and every
200 places further down the list halve a word's chance (rank 1000: 1/16).
When the last letter opens, the limit grows in a straight line and reaches the
whole list (25 000) after `RampLetters = 10 000` letters, about 1700-1900
words or an hour at 30 WPM; then the penalty is gone. Choices behind this:

- **The ramp counts typed letters, not rounds.** Rounds have different
  lengths (10 words to 90 seconds, section 7, "The round"), so "N rounds"
  would mean different amounts of practice. Typed letters do not depend on
  the round length. Letter mastery no longer measures progress here: every
  letter is already mastered once the last one opens. Freetype does not
  count: it never touches learning data. The number is computed from the
  history, so it needs no new stored field and survives export/import; a
  reset starts it again.
- **A floor for rare letters.** Some letters have almost no words among the
  200 most common: `q`, `x`, `z` have none. Their words are all far down
  the list, and the penalty makes the most common of them much more likely
  than the others (two words 400 places apart: 4 times). So the focus letter
  uses its own limit:

  ```
  MaxRank(focus) = max(MaxRank, rank of the focus letter's FloorWords-th most common word in the list)
  ```

  With `FloorWords = 10`, the 10 most common words of such a letter come
  freely (all of its words if it has fewer). A portion has 6 words and the
  repeat penalty looks at the last 24, so 10 words fill a portion without a
  repeat. The words are counted in the whole list, not only among the open
  words: in the first lessons every letter has few open words, and counting
  only those would let rare words in exactly where common ones matter most. Letters with many common words are not affected: their 10th word is
  inside the top 200. Affected (rank of the 10th word): `j` 1217, `p` 273,
  `q` 1807, `x` 847, `z` 2170.
  The penalty only shapes which words of the focus letter are taken: the
  focus letter itself (step 1) does not depend on it, so a rare letter is
  never trained less because its words are rare.
- **A soft penalty, not a ban.** The weight is never 0, so rare words stay
  possible, and a letter whose words are all rare still gets practised. Early
  lessons have few words at all (in English 99 in the first lesson, only 4 of
  them among the 200 most common), so a hard limit would leave the user with
  a handful of words.
- **A floor for a lesson.** For the same reason the limit is never below the
  rank of the `MinWords`-th open word: the 40 most common words the open
  letters can make come freely (in the first lesson up to rank 3810), so
  a round of 30 words is not the same few words again.
- `RankPenalty` outside `(0, 1)` or `RankStep <= 0` turns the penalty off.
  A word list shorter than `RankStart` gets no penalty at all.

**No word too often.** `RepeatPenalty = 0.3` makes a word that was used in
the last `RecentWords = 24` words of the round unlikely (24 plus the 6 of the
portion being built: the whole round of 30), and the same word never appears
twice in a row (also across the border of two portions). On top of the
penalty, every slot first takes a word not used in that window: the
focus letter's words, then any open word (`D`, `F` with the general
`MaxRank`). Only when no fresh word is left does a word come again, at most
`MaxRepeats = 2` times. Only a dictionary too small for these rules relaxes
them (first the cap, then "not twice in a row").

**No one-letter words.** Words shorter than `MinWordLetters = 2` are left
out of learning when the generator is made (`NewGenerator`): `a` and
`i`. They are among the most common words of the list, so they came up often, and a lone letter is not a word to
practise. They are left out before anything is
counted, so they take no part in the unlock order, the first lesson or the
ranks. The word list keeps them (section 11): Freetype is not
affected.

**Mistyped words come back.** Words the user typed with a mistake (a wrong or
extra letter, or skipped letters; a mistake fixed with Backspace counts too)
are typed again: up to `RetryWords = 2` of them are put into the next portion
that is built, at random places, instead of new words, unless the word
already came `MaxRepeats` times in the round. They are not repeated
right away, because of the waiting portions (below), but about 2 portions
later. Repeating a missed item after a short gap is the idea of spaced
practice; here it is the simplest version, one repetition. A word the user
went back into (Backspace, section 3) and retyped without a new mistake is
already practised and does not come back, unless its portion was sent before
the fix.

**Focus shown to the user** is the focus letter of the portion being typed.

**When does the model react?** Once per round, when it ends: the round's
presses go to `/results`, the statistics are updated and maybe a letter
opens. Inside a round the letters' numbers stay as they were at its start:
when the space after a portion's last word is typed, the browser asks for
one new portion, built from those numbers and the words typed so far
(mistyped words come back, the repeat cap holds). It is added at the end
of the text. `AheadPortions = 2`
portions always wait after the one being typed, so the user can read ahead
(12 words, shown one more for each finished word) and **words already on the
screen never change**. The cost
is that what a round teaches the model shows from the next round on.

**The round.** The stream is cut into rounds of `RoundPortions = 5` portions
(30 words), so the user always sees how much is left (`3 / 30` over the text) and
gets a result at a clear point. The last letter of the last word ends it (no space after it is needed,
right or wrong). The WPM and accuracy of the round go to
the history, and the next round starts from scratch: new waiting portions, an
empty list of recent words. Inside a round the words are still chosen portion
by portion. The time of a round ends at its last keypress. Leaving the
window in the middle of a round (another window, a hidden tab), or 10
seconds without a key, pauses it: the time away is taken out of the round
(its start and its last key move forward by it; for the 10 seconds, from
the last key on), and the first key after the pause is not measured, like
the first key after a Backspace.

A Learn round is 30 words (`RoundPortions` portions); Freetype also offers 10 to 50 words or
15 to 90 seconds (`web/settings.js`, `LENGTHS`). Only the end of the round moves: the stream
keeps asking for portions until the round has enough words (a time round
until the time is up), and a words round ends on the last letter of its
last word even in the middle of a portion; the samples of that portion go
with `/results` as usual. Word choice and the model are the same for every
length. Each round keeps its length in the history, and the stats screen
never mixes lengths.

## 8. Parameters

All in `internal/config/config.go`.

| Name | Value | Meaning | Reason for the value |
|---|---|---|---|
| `Alpha` | 0.1 | EMA smoothing of the time | half-life ≈ 6.6 occurrences, memory ≈ 20 |
| `ErrAlpha` | 0.05 | EMA smoothing of the mistakes | memory ≈ 40 presses: one slip moves the error rate by at most 5 points (section 4) |
| `MaxIntervalMs` | 2000 | longer gaps are pauses | a 2 s gap is far slower than any real keystroke |
| `TargetWPM` | 30 | goal speed (400 ms / char) | reachable goal for a beginner; raise it for advanced users |
| `SlowMs` | 1000 | "maximally slow" | 1 s per letter = hunting for the key |
| `MaxErrorRate` | 0.25 | "maximally inaccurate" | spreads realistic error rates over 0..1 |
| `ErrorWeight` / `SpeedWeight` | 0.6 / 0.4 | mix of the two parts | accuracy first, but speed matters |
| `PriorDifficulty` | 0.7 | assumption about an unseen letter | new letters must get practised |
| `PriorStrength` | 5 | how many samples the prior is worth | a few presses are not yet evidence |
| `Temperature` | 0.1 | focus strength | chosen with the simulation (section 10): fastest learning at a small loss of variety |
| `MasteryMinSamples` | 40 | presses needed for mastery | a new letter gets about 30 presses a round as the focus: about two rounds |
| `MasteryLevel` | 80 | level needed for mastery (section 6) | few mistakes near the goal speed; with the simulation, a fast typist opens every letter in about 46 rounds instead of 348 |
| `UnmasteredBonus` | 0.2 | added to the difficulty of a letter not mastered yet, for the choice only | e^(0.2/0.1) ≈ 7 times the weight of an equal mastered letter: a blocking letter gets about half the portions among ten (section 7) |
| `StartLetters` / `MinWords` | 6 / 40 | first lesson; the open words that always come freely | letters open one by one from early on; a round of 30 words without the same few words |
| `RepeatPenalty` | 0.3 | less repetition of recent words | variety |
| `MaxRepeats` | 2 | a word comes at most this many times in a round | the penalty alone let a rare letter's 10 words come 5-6 times in 30 (section 7); 2 still lets a mistyped word come back |
| `MinWordLetters` | 2 | shorter words never appear in learning | `a`, `i`, `в` are filler, not practice (section 7) |
| `ExerciseWords` | 20 | words per exercise of the earlier strategy (`Next`), and the unit of the simulation | short loop, model updates often |
| `PortionWords` | 6 | words per portion of the stream, one focus letter each | about one line: the focus changes every line, often enough not to bore (the model itself learns once per round) |
| `AheadPortions` | 2 | portions waiting after the one being typed | about two lines to read ahead; fewer would leave the window half empty |
| `RecentWords` | 24 | the repeat penalty and the cap look at this many last words | with the portion being built (6), a whole round of 30 |
| `RetryWords` | 2 | most mistyped words put back into one portion | at most a third of a portion is repetition |
| `RoundPortions` | 5 | portions per round (30 words), one result in the history | a clear end; about 1-2 minutes for a beginner, 20 seconds for a fast typist |
| `RankStart` | 200 | words up to this frequency rank come without a penalty until every letter is open | the size of monkeytype's smallest list ("English 200"): the core words of a language |
| `RankStep` | 200 | distance in ranks that halves a word's chance | one "200 words" block per halving: rank 1000 while learning letters is 16 times rarer |
| `FloorWords` | 10 | the focus letter's most common words that always come without the frequency penalty | a portion (6 words) without a repeat, with room for the next one; only letters with few common words (`q`, `x`, `z`) are affected |
| `RampLetters` | 10 000 | letters typed after the full alphabet until every word comes freely | about 1700-1900 words (5.2-5.9 letters per word), ~60 rounds of 30 words, an hour at 30 WPM: the list grows by ~200 words every 6 minutes of typing |
| `ConsistencyMinSeconds` | 5 | shortest round that gets a consistency (section 3) | a standard deviation of 2-3 numbers says little; 5 s is a few words even for a beginner |
| `RankPenalty` | 0.5 | weight multiplier per `RankStep` over the limit | halving is easy to explain; with `T = 0.1` one step equals about 0.07 of difficulty, so weak letters still decide first |
| `Laps.StepWPM` / `Laps.MaxWPM` | 10 / 200 | in Reinforce the goal goes up 10 WPM a step, up to 200 (section 6.1) | ten wpm is a step a typist feels; about 200 wpm is the top of real typing |
| `AccWindow` / `AccPrior` | 100 / 3 | a letter's accuracy reads its last 100 presses plus 3 imaginary misses (section 14.2); 0 turns the rules off | 27 clean presses reach 90 %; a few lucky presses are not 100 % |
| `AccMaster` / `AccSlip` | 0.90 / 0.85 | mastered; an older letter under the second takes the focus until it is back at the first (section 14.3) | the owner's choice; two thresholds keep a letter near 90 % from flipping the focus |
| `PairBoost` | 2 | a word's weight grows by this for each weak pair in it, in Learn and Reinforce (section 6.1) | 3 times as likely: the pair comes up often, the letters' focus still leads |
| `Forecast.Window` / `MinRounds` / `MaxRounds` | 10 / 3 / 50 | the forecast reads a letter's last 10 rounds, needs 3, says "50+" beyond 50 (section 6.2) | 10 rounds of a few presses are about what the EMA remembers (20 presses); with the simulation (section 10), 3 rounds answered more often than 5 at about the same error |

The values were chosen by reasoning and the simulation (section 10), not
fitted to real data. Every round keeps its raw keys (`docs/HOW-THE-CODE-WORKS.md`,
"Raw round records"), so other values can be tried on real typing later.

## 9. Known simplifications

- **Time belongs to a transition, not a letter.** The measured time is from the
  previous key to this one, so a slow `s` after `t` may really be a slow `t→s`
  movement. It is assigned to `s`.
- **Per user, not per person.** The model is kept per account, or per browser
  for a guest (HOW-THE-CODE-WORKS.md, "Accounts"). Someone who types as a guest on
  two computers is two learners to the model; the model itself does not know
  about users at all.
- **Word lists from subtitles.** The 25 000 most frequent words,
  counted in film and TV subtitles (section 11). That is spoken, everyday
  language: good for typing practice, but some names or slang may remain after
  filtering. Rare letters (`q`, `j`, `x`, `z`) have few words.
- **One stubborn letter blocks progress.** If one letter never reaches mastery,
  nothing new opens. The focus pushes such a letter much
  harder (`UnmasteredBonus`, section 7), but a person who does not get
  faster on it still waits; a cap (for example "open anyway after N
  portions") may be needed. Real data will tell.
- **The order is frequency with a home row boost,** not a full course plan
  (home row, then the top row, then the bottom row). A full plan would need
  words for every step; real words decide where a rare letter goes.

## 10. Simulation

Code: `internal/sim` (tests in `internal/sim/sim_test.go`), run with
`go run ./tools/sim` from the repository root. All numbers are
in `internal/config/config.go` (`Sim`). `go test ./...` guards the main
result: `tools/sim/main_test.go` runs three typists and fails
if random words need less than 3 times the practice of the stream.

**The simulated typist.** Every letter has a hidden error probability, drawn
uniformly from 0.03..0.30, and a hidden time per press, drawn from
350..900 ms. Each press of a letter improves it:

```
value = floor + (value - floor) * (1 - LearnRate)      LearnRate = 0.02
```

with floors of 1 % errors and 200 ms. A press is wrong with the hidden
probability and takes the hidden time ±25 % (uniform). The goal is reached when
every letter's **hidden** values meet the mastery goal (≤ 5 % errors,
≤ 400 ms). The hidden values are used only to judge the result; the strategies
see only what a real browser would send.

**Five strategies,** all with the same statistics (`engine.Apply`), the same
unlocking rule (`NextUnlocked`), and for each run the same typist and the same
random numbers:

1. **random words:** any open word, uniformly;
2. **best of 20 (monkeytype):** a copy of monkeytype's "weakspot" mode. Each
   letter has an EMA of (press time + 5000 ms for a mistake) over about 50
   presses; for every word slot 20 random open words are drawn and the one with
   the highest mean letter score is taken;
3. **adaptive (softmax):** this project's `Generator.Next`, 20 words at once,
   statistics updated after each exercise (the app before the stream);
4. **stream:** this project's `Generator.Portion` (section 7): rounds of 5
   portions of 6 words, 2 waiting, mistyped words come back; statistics
   updated after each portion by default, or once per round as the app
   does with `-apply round`;
5. **stream, no frequency penalty:** the same, with `RankPenalty` off, to show
   what the "common words first" rule of section 7 changes.

Strategies 1-3 work in exercises of 20 words. The stream has no exercises, so
everything is counted in words and reported in "exercises" of 20 words: 163
exercises means 3260 words typed. The variety numbers of the stream are
measured on blocks of 20 words in a row.

**Variety.** A faster strategy is useless if it is boring, so the simulation
also measures how repetitive the exercises are (means over exercises):

- `dist`: share of different words in one exercise (100 % = no word twice);
- `rep`: share of words that were also in the previous exercise;
- `top`: share of words that contain the exercise's most common letter
  (how hard one letter is pushed);
- `len`: mean letters per word, and `rank`: mean place of the words in the
  frequency list, 0 = the most common word (printed by the tool, not in the
  table below; quoted in the bullets).

They are measured twice: on the **first 30 exercises** of the beginner above
(few open letters), and on an **experienced typist** with every letter
already open (every letter starts at ≤ 5 % errors and 150..350 ms) over 100
exercises.

**The numbers:** `go run ./tools/sim` on the configured model.
Median exercises to the goal (20 typists), and the stream's variety:

| Language | Random words | Best of 20 | Adaptive, T = 0.1 | Stream (the app) | Stream, no frequency penalty | Stream variety, first 30 | Stream variety, expert |
|---|---|---|---|---|---|---|---|
| en | 1088 | 179 | 124 | 134 | 120 | 96 / 32 / 81 | 99 / 3 / 68 |

The stream needs 8 times fewer exercises than random words and 1.3
times fewer than best of 20. The forecast is within
max(2, 25 %) in 54 % of cases (mean error 6.1 rounds): `j` and `k` come early
and have few words, so their waits are long and hard to foresee.

**Unlock pace.** The same tool also types Learn exactly as the app does
(rounds of 30 words, the model learns once per round) for three kinds of
typist and counts the rounds until every letter is open (median of 20):

| Typist | Mistakes | Time per press | Learns | Rounds to 13 letters | Rounds to all 26 |
|---|---|---|---|---|---|
| beginner | 3-30 % | 350-900 ms | yes | 53 | 90 |
| middle | 4-8 % | 250-450 ms | yes | 34 | 64 |
| fast | 2-4 % | 120-250 ms | no, already fast | 33 | 65 |

Before the level and "mastered stays mastered" (section 6) these were 104,
69 and 348 rounds: a fast typist waited for a round in which every open
letter was under 5 % mistakes at the same moment. The home row first
(section 6) costs more rounds than the order by frequency: `j` and `k`
have few words, so they collect their presses slowly. A small first lesson
(6 letters, then one at a time) costs about 10 more rounds than the 9 of a
home-row-and-`e` start; it is chosen for the step-by-step feel. The simulated typist has no fingers, so what the home row gives
(fingers in place from the first lesson) is not in these numbers.

Limits: the typist is invented. Letters do not influence each other, there is
no forgetting, and learning is the same exponential curve for every letter. The
simulation compares strategies under the same assumptions; it does not predict
how fast a real person learns. The variety numbers are simple proxies; how
boring an exercise feels can only be measured with real users.

## 11. Layout and word lists

**Layout** (`data/layouts/`): English QWERTY. Finger zones are by key position.

**The system layout is not replaced.** If the typed letter is not in the
alphabet at all (for example Cyrillic letters from a Russian system layout),
the app does not count a mistake but shows a hint to
switch the layout. Caps Lock is detected the same way. Wrong-layout presses
would otherwise ruin the statistics for reasons that have nothing to do with
typing skill.

**Word list.** Learn takes the whole long list, 25 000 words
(`data/langs/<id>/free.txt`, section 13), one-letter words
left out (`MinWordLetters`, section 7). Word frequencies come from
FrequencyWords (Hermit Dave), counted in the OpenSubtitles 2018 corpus
(CC BY-SA 4.0). Subtitle frequencies are a good model of everyday language
(Brysbaert & New, 2009). A large list keeps the early lessons, which can
only use a few letters, from repeating the same words: short words that are
not common are what those lessons need most.

## 13. Freetype

Hidden from the menu for now and not developed. Words drawn evenly from the n most
common of the long list (`FreetypeLists`: 200 to 25 000), or a kind of text
with its own history: long words (6+ letters), real sentences, or your own
pasted text (kept in the browser only). Nothing adapts.

**Sentences** (`data/langs/<id>/sentences.txt`, `tools/sentences/build.py`):
real sentences from Tatoeba, 4 to 18 words, only the language's letters,
capitals and `. , ? ! ; : ' -`, every word among the 25 000 most common.

**The long list** (`data/langs/en/free.txt`, `tools/wordlists/free.py`):
subtitle frequencies (section 11) filtered by the Hunspell en_US dictionary
in lower case (names drop out) and a list of offensive words; the first
25 000 of about 30 800 that pass.

## 14. Learn: the accuracy rules

Each stage adds one measure: Learn trains accuracy
only, Reinforce adds speed, Freetype adds consistency
(`docs/PRODUCT.md`, "Pillars"). Learn below replaces the
level, mastery, focus and unlock rules of sections 5-7.

### 14.1 Welcome: the finger tour

The **finger tour** is a step of the Welcome screen (`web/intro.js`), which
opens by itself on the first visit and from its tab any time later. Its
steps: what touch typing is; the home row and space (press all eight keys
and space); the finger tour; start. The fingers rest on F and J (the bumps). The tour walks the keyboard
**finger by finger**, from the left little finger to the right one: the
little finger's column `q a z`, the ring finger's `w s x`, and so on; an
index finger has two columns (`r f v t g b`); only the letter block
(`q`…`p`, `a`…`;`, `z`…`/`) is in the tour. The finger has its colour
on the keyboard picture and its column lights up; the next key is filled
with the finger's colour.

- Each key of the finger is one sub-step: press it once. A wrong key
  makes the right one pulse; nothing is counted against you. After the
  finger's last key the next finger comes by itself.
- Progress on two levels: a strip of eight fingers and "Finger 3 of 8 ·
  left middle" over the keyboard, and the finger's own keys ticked off
  under it (`E ✓ D C`).
- "Start Learn" (or Enter) is there on every step; whoever knows the
  keyboard leaves at once. ← → and the step chips move between steps.
- The Welcome tab opens it again at any time, from its first step.
- Its presses are never sent, so they never enter the letter
  statistics: the site sees the key, not the finger, so the tour can show the right finger but not check it.

It takes under a minute.

### 14.2 Accuracy, the one number of a letter

```
accuracy = correct / (presses + 3)        over the letter's last 100 presses
```

A press is one sample of section 3; it is correct when the position was
typed right with no error. The 3 imaginary misses keep a few lucky
presses from looking like 100 % (Laplace's rule of succession). There is
no fixed number of presses; how fast a letter is mastered depends on the
typing:

| Errors | Presses to 90 % |
|---|---|
| 0 | 27 |
| 1 | 37 |
| 2 | 47 |
| 5 | 77 |

With 100 presses in the window a letter stays at 90 % while it has at most
7 errors in them; the best a letter can show is 100/103 = 97 %. Speed does
not count in Learn. The letter row shows each letter's accuracy in %; it
is all the progress Learn shows. The difficulty that weighs the words is
`1 - accuracy`.

### 14.3 Focus and the next letter

- The **newest letter** is the focus until its accuracy reaches 90 %.
- An older letter that falls **under 85 %** takes the focus and keeps it
  until it is back at **90 %**. With several, the lowest goes first.
- In the first lesson the focus is the lowest of the six under 90 %.
- **The next letter opens** at the end of a round when the newest letter
  is at 90 % or more and no open letter is under 85 %. At most one letter
  per round.

The two thresholds are hysteresis, as in a thermostat: a letter that
wobbles around 90 % neither flips the focus back and forth nor holds back
the next letter.

### 14.4 Words

Section 7 stays: the stream, portions, the whole 25 000-word list from
open letters, no repeat while fresh words exist, the lesson floor, weak
pairs. What changes: the focus letter comes from 14.3 instead of the
softmax draw, and the weights use the difficulty of 14.2.

### 14.5 What you adapt to, what adapts to you

The trainer adapts the focus, the words, when the next letter opens, the
weak pairs and the speed goal. You adapt to the path, the fingers on F
and J with a column each, the thresholds of 90 % and 85 % and real words. You choose to reopen the tour, to turn
the on-screen keyboard off and the step of the speed goal
(`docs/PRODUCT.md`).

### 14.6 The simulation

The rules are in the engine behind a switch (`AccWindow`, 0 = off; the
app has 100); `go run ./tools/sim -apply round -acc 100` turns them on in
the simulation (without `-acc` it runs the rules of sections 5-7). Once
every letter is open the focus goes back to the softmax draw of section 7:
"the newest letter" would stay on the last one for good (a first run gave
96 % of the words with `q`).

| | built rules | accuracy rules |
|---|---|---|
| rounds to open every letter: beginner / middle / fast | 90 / 64 / 65 | 107 / 61 / 50 |
| exercises until the goal, stream (median) | 134 | 172 |
| beginner, first 30 exercises: dist / rep | 96 % / 33 % | 97 % / 31 % |

A typist who makes few mistakes opens the letters sooner, one who makes
many takes longer: accuracy now decides alone. The goal of the simulation
includes speed, which Learn no longer trains, so it is reached later;
Reinforce is meant to train it.

**In the app.** A letter keeps its last presses (`KeyStat.Hits`) and its
slip (`KeyStat.Slipped`). A letter typed before the rules starts from as
many presses as it has (up to 100), missed at its error rate. The letter
row shows the accuracy as the level and its tip says "N% accurate". The
forecast "rounds to the next letter" (section 6.2) follows the accuracy of
the letters that hold the next one back (`engine.Blocking`): the newest
under 90 % and any older one under 85 %.
