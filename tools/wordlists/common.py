"""What free.py shares: the alphabets, the source counts and the block list."""
import os

ALPHABET = {"en": "abcdefghijklmnopqrstuvwxyz"}
ONE_LETTER = {"en": set("ai")}
VOWELS = set("aeiouy")
HERE = os.path.dirname(os.path.abspath(__file__))


def load(src, lang):
    counts, total = {}, 0
    with open(os.path.join(src, lang, f"{lang}_50k.txt"), encoding="utf-8") as f:
        for line in f:
            parts = line.split()
            if len(parts) != 2:
                continue
            word, n = parts[0].lower(), int(parts[1])
            counts[word] = counts.get(word, 0) + n
            total += n
    return {w: c / total * 1e6 for w, c in counts.items()}


def blocklist(lang):
    path = os.path.join(HERE, f"block_{lang}.txt")
    return set(open(path, encoding="utf-8").read().split()) if os.path.exists(path) else set()
