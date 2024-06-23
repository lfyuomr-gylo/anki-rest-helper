# Anki REST Helper

This is my personal Anki CLI helper. I use it to generate/enrich my [Anki](https://apps.ankiweb.net/) notes.

It interacts with Anki using REST API exposed via [AnkiConnect plugin](https://github.com/FooSoft/anki-connect).

# Features

- [automatic text-to-speech generation](#configure-text-to-speech) using Microsoft Azure transcription.
- [custom script-based note processing](#configure-note-processing) --- modify arbitrary fields of notes using a
  custom-defined script
- [note type generation](#configure-note-type-definitions) with meta-templating of card templates. It's useful for verb
  conjugation learning.
- [static media files upload](#configure-static-media-files-upload) to Anki
- [automatic cards organization](#configure-cards-organization): put your cards in the appropriate decks by defining
  organization rules.

# How to use it

1. Install [AnkiConnect plugin](https://github.com/FooSoft/anki-connect).

2. Download the [latest release](https://github.com/lfyuomr-gylo/anki-rest-helper/releases) of the tool for your
   platform or build it from source code using `go build .` command.

3. Create your configuration file following [documentation below](#configuration-format).

4. Run [Anki](https://apps.ankiweb.net/) with [AnkiConnect plugin](https://github.com/FooSoft/anki-connect) enabled

5. Execute `path/to/anki-helper -config path/to/anki-helper.yaml` in your command line.

If you don't want to pass config file path to the tool at every execution, rename the file to `anki-helper.yaml`
and put it to one of the following locations:

- current work directory.
- your system's user default configuration directory, as defined
  by [UserConfigDir](https://pkg.go.dev/os#UserConfigDir).
- your user's home directory.

# Configuration format

To see real and up-to-date example of a working configuration,
check out [anki-helper.yaml](./anki-helper.yaml).

For full list of supported configuration fields, see [ankihelperconf/yaml.go](./ankihelperconf/yaml.go).

## Configure text-to-speech

**Prerequisite:** in order to use text-to-speech (TTS), you need an API key to access Microsoft Azure text-to-speech
service.
As of 2023-01-04, you can create a free Azure Account and create a TTS resource using freebie quota.
This quota is more than enough for a personal use, so you can use TTS free of charge.
To generate the API key, follow
the [official documentation](https://learn.microsoft.com/en-us/azure/cognitive-services/speech-service/overview#get-started).

Once you've created a TTS resource, and chosen
the [voice to use](https://learn.microsoft.com/en-us/azure/cognitive-services/speech-service/language-support?tabs=stt-tts),
put this information into a configuration file:

```yaml
azure:
  # Relative path to a file that contains Azure API key.
  apiKeyFile: azure-key.txt
  # Endpoint URL you can find in your Azure Console. 
  endpointUrl: https://germanywestcentral.tts.speech.microsoft.com/cognitiveservices/v1
  # Voice you want to use for TTS
  voice: es-ES-AlvaroNeural
  # Requests throttling. With freebie quota, TTS requests are heavily throttled on the Azure side,
  # so parameters below can mitigate this Azure-side throttling.
  # Feel free to remove them.
  minPauseBetweenRequests: 2100ms
  retryOnTooManyRequests: true
```

Now that you configured Microsoft Azure TTS, configure what text in what Anki notes you want to convert to speech
and where you want to store that speech:

```yaml
actions:
  tts:
    - noteFilter: 'Word:_* WordVoiceover:' # (1)
      textField: Word # (2)
      textPreprocessing: # (3)
        - regexp: '\s+'
          replacement: ' '
      audioField: WordVoiceover # (4)
```

In the example above we told the tool to

1. Find all notes that have a non-empty field `Word` and an empty field `WordVoiceover`.
   You can learn search syntax in the [official documentation](https://docs.ankiweb.net/searching.html).

2. For each note, extract `Word` field.

3. Replace all substrings of the field that match `\s+` regexp with a single space character.
   Check out regexp syntax [here](https://github.com/google/re2/wiki/Syntax)

4. Convert this processed text to speech and store tha audio in `WordVoiceover` field of the note.

Note: `noteFilter` in the example is the default filter, so it may be omitted (the tool will automatically asume it).

## Configure note processing

You can write a custom script that processes an Anki note, and run that script against all notes matching a filter:

```yaml
actions:
  noteProcessing:
    - noteFilter: Gender:_* -(Gender:femenino OR Gender:masculino) # (1)
      exec:
        command: ./normalize_gender.py # (2)
        stdin: "$$ .Note.Fields | to_json $$" # (3)
        args:
          - "$$.Note.Fields.Gender$$" # (4)
      # Fields below configure command execution throttling and timeouts.
      # These fields are optional and may be omitted.
      minPauseBetweenExecutions: 1200ms
      timeout: 5s
```

The configuration above tels the tool to

1. Find all notes that have a non-empty `Gender` field that is not `masculino` or `femenino`.

2. For each such note, execute the `./normalize_gender.py` program.
   Relative paths starting at `./` or `../` are resolved against configuration file directory.

3. All note fields are passed to the program via stdin as a JSON object like `{"field": "value"}`.

4. `Gender` field of the note is passed as a first command-line argument to the program.

5. Apply note modification commands that the program printed to its stdout. Sample program stdout:
   ```json
   [
       {"set_field": {"Gender": "f"}}, 
       {"add_tag": "gender_normalized"}
   ]
   ```

   Supported modification commands (full list is defined in [Modification](noteprocessing/scriptapi.go) struct):

    - `{"set_field": {"field": "value"}}`
    - `{"set_field_if_empty": {"field": "value"}}`
    - `{"add_tag": "tag"}`

Stdin and args may be plain text or [go templates](https://pkg.go.dev/text/template) with `$$` used as a delimiter.

## Configure note type definitions

To be documented... See a working example in [anki-helper.yaml](./anki-helper.yaml).

## Configure static media files upload

To be documented... See a working example in [anki-helper.yaml](./anki-helper.yaml).

## Configure cards organization

To be documented... See a working example in [anki-helper.yaml](./anki-helper.yaml).

# How to build the binary

To build the tool, you need to install [Go](https://go.dev/) 1.17 or beyond.
Simply run `go build .` to get a binary for your platform.
If you want to get binaries for different platforms, check out [release.sh](./release.sh).

If you don't want to install Go on your system, try building the tool using Docker:

```
docker run --rm -v `pwd`:/projects/anki-helper -w /projects/anki-helper golang:1.17  ./release.sh
chown -R `whoami` build/
chmod -R +x build/
```