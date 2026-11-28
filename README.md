# typeTrainer

An adaptive touch-typing trainer on real words. Three modes, each with its own loop:

- **Learn** trains accuracy: letters open one at a time from the home row.
- **Reinforce** trains speed: common words, a speed goal that grows, a word level that moves by itself.
- **Freetype** trains rhythm: the letter pairs where your rhythm stumbles come more often.

## Run it

Needs [Go](https://go.dev/dl/) 1.25+.

```
go run .
```

Open <http://localhost:3000>. Set `DATABASE_URL` to keep progress in PostgreSQL.

## Credits

- Word frequencies: [FrequencyWords](https://github.com/hermitdave/FrequencyWords) by Hermit Dave (OpenSubtitles 2018), CC BY-SA 4.0.
- Word list filters: the en_US Hunspell dictionary from [SCOWL](http://wordlist.aspell.net/) and [LDNOOBW](https://github.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words), CC BY 4.0.
- Fonts: [Geist Mono](https://github.com/vercel/geist-font) and [Sora](https://github.com/sora-xor/sora-font), SIL OFL 1.1.

## License

[GPL-3.0](LICENSE)
