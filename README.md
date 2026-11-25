# typeTrainer

![Extensive WIP, subject to change](https://img.shields.io/badge/status-extensive%20WIP%2C%20subject%20to%20change-orange)

An adaptive touch-typing trainer. It measures every key you press and picks
the next words from your own weak letters.

**Demo:** <https://nadludyna.github.io/typeTrainer/>

## What is in it

- **Learn:** letters open one at a time, starting from the home row. A new
  letter opens when the newest one is at 90 % accuracy and no open letter
  is under 85 %. Only real English words, built around your weakest
  letter.
- **Reinforce:** once every letter is open, you pick how many common words
  to type (200 up to 25 000) and a speed goal from 30 to 200 wpm. Slow
  letters and weak letter pairs come more often.
- **Settings:** text, caret, keyboard, round length, data export, import
  and reset.

How the trainer decides what you type:
[docs/HOW-THE-MODES-WORK.md](docs/HOW-THE-MODES-WORK.md).

## Built with

- Go (standard library) for the server
- PostgreSQL for storage, optional (without it, progress lives in memory)
- Plain HTML, CSS and JavaScript, with no build step
- A WebAssembly build of the same Go code for GitHub Pages, the demo above

## Run it

You need [Go](https://go.dev/dl/) 1.25 or newer.

```
go run .
```

Open <http://localhost:3000>. To use PostgreSQL, set `DATABASE_URL`.
Tests and the rest: [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

## Credits

- Word frequencies: [FrequencyWords](https://github.com/hermitdave/FrequencyWords)
  by Hermit Dave (OpenSubtitles 2018), CC BY-SA 4.0.
- Sentences: [Tatoeba](https://tatoeba.org), CC BY 2.0 FR.
- Word list filters: the en_US Hunspell dictionary from
  [SCOWL](http://wordlist.aspell.net/) and
  [LDNOOBW](https://github.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words), CC BY 4.0.
- Fonts: [Geist Mono](https://github.com/vercel/geist-font) and
  [Sora](https://github.com/sora-xor/sora-font), SIL OFL 1.1.

## License

[GPL-3.0](LICENSE)
