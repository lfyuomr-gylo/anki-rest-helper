# Anki REST Helper

This is my personal Anki helper. I use it to generate/enrich my Anki notes.

It interacts with Anki using REST API exposed via [AnkiConnect plugin](https://github.com/FooSoft/anki-connect). 

# Features

- automatic text-to-speech generation using Microsoft Azure transcription 
  for multiple note fields.
- note type generation with meta-templating of card templates.
  It's useful for verb conjugation learning.

# How to use it

1. Run Anki App with AnkiConnect plugin enabled
2. Adjust [anki-enhancer.yaml](./anki-enhancer.yaml) for your needs
3. Execute `go run .`

# Backwards Compatibility

Right now, no backwards compatibility is guaranteed.