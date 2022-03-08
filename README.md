# Anki REST Helper

This is my personal Anki CLI helper. I use it to generate/enrich my Anki notes.

It interacts with Anki using REST API exposed via [AnkiConnect plugin](https://github.com/FooSoft/anki-connect). 

# Features

- automatic text-to-speech generation using Microsoft Azure transcription 
  for multiple note fields.
- note type generation with meta-templating of card templates.
  It's useful for verb conjugation learning.

# How to use it

1. Download the [latest release](https://github.com/lfyuomr-gylo/anki-rest-helper/releases) of the tool for your platform
   or build it from source code using `go build .` command.
2. Create your configuration file using [anki-helper.yaml](./anki-helper.yaml) as an example.
   For full list of supported configuration parameters, see [enhancerconf/yaml.go](./enhancerconf/yaml.go).
3. Run Anki App with AnkiConnect plugin enabled
4. Execute `path/to/anki-helper -config path/to/anki-helper.yaml` in your command line.

If you don't want to pass config file path to the tool at every execution, rename the file to `anki-helper.yaml`
and put it to one of the following locations:

- current work directory.
- your system's user default configuration directory, as defined by [UserConfigDir](https://pkg.go.dev/os#UserConfigDir).
- your user's home directory.

# How to build the binary

To build the tool, you need to install [Go](https://go.dev/) 1.17 or beyond.
Simply run `go build .` to get a binary for your platform.
If you want to get binaries for different platforms, check out [release.sh](./release.sh).
