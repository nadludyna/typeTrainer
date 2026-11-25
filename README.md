<p align="center">
  <img src="web/favicon.svg" width="72" alt="">
</p>

<h1 align="center">typeTrainer</h1>

<p align="center">
  A typing trainer that finds your weak letters and builds the next words around them.<br><br>
  <a href="https://nadludyna.github.io/typeTrainer/"><b>Start typing</b></a>
  ·
  <a href="docs/HOW-THE-MODES-WORK.md">How it works</a>
</p>

<p align="center">
  <a href="https://github.com/nadludyna/typeTrainer/actions/workflows/pages.yml"><img src="https://github.com/nadludyna/typeTrainer/actions/workflows/pages.yml/badge.svg" alt="Deploy"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-GPL--3.0-blue" alt="License: GPL-3.0"></a>
</p>

## About

typeTrainer times every key you press, notices which letters slow you down
or trip you up, and picks the next words around them. Every word is a real
word you could meet in a book.

A computer keyboard is best for practice. A phone shows the same site and
you can type on it too.

The project is changing fast right now, so this page stays short. To see
what it does today, [open the site](https://nadludyna.github.io/typeTrainer/).

## Run it yourself

You need [Go](https://go.dev/dl/) 1.25 or newer.

```
go run .
```

Then open <http://localhost:3000>. Accounts and a PostgreSQL database are
optional; [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) has the details.

## Contributing

Issues welcome. Pull requests are not accepted, feel free to fork.
How to run the tests: [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md); how the
parts fit together: [docs/HOW-THE-CODE-WORKS.md](docs/HOW-THE-CODE-WORKS.md).

## Credits

- Word frequencies: [FrequencyWords](https://github.com/hermitdave/FrequencyWords)
  by Hermit Dave (OpenSubtitles 2018), **CC BY-SA 4.0**.
- Sentences: [Tatoeba](https://tatoeba.org), **CC BY 2.0 FR**.
- The long English list is checked against the en_US
  [Hunspell dictionary from SCOWL](http://wordlist.aspell.net/) (via
  [wooorm/dictionaries](https://github.com/wooorm/dictionaries)) and
  [LDNOOBW](https://github.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words)
  (**CC BY 4.0**) to leave out names and offensive words.
- Fonts: [Geist Mono](https://github.com/vercel/geist-font) and
  [Sora](https://github.com/sora-xor/sora-font), **SIL OFL 1.1**.

How the word lists were made: [docs/HOW-THE-MODES-WORK.md](docs/HOW-THE-MODES-WORK.md).

## License

[GPL-3.0](LICENSE).
