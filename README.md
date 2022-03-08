# Anki REST Helper

This is my personal Anki CLI helper. I use it to generate/enrich my Anki notes.

It interacts with Anki using REST API exposed via [AnkiConnect plugin](https://github.com/FooSoft/anki-connect). 

# Features

- automatic text-to-speech generation using Microsoft Azure transcription 
  for multiple note fields.
- note type generation with meta-templating of card templates.
  It's useful for verb conjugation learning.

# How to use it

1. Run Anki App with AnkiConnect plugin enabled
2. Create your configuration file, use [anki-enhancer.yaml](./anki-enhancer.yaml) as an example.
3. Name your config file ``anki-enhancer.yaml`` and put it either in current directory or 
   at your user's config or home directory. 
4. Execute `go run .`

If you prefer to store the configuration file somewhere else, pass the file's path via `-config` flag.
