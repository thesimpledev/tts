The continuation of my old [Linux Tools for Windows](https://github.com/StevenDStanton/cli-tools-for-windows) project

## Testing

Notes: Need to make sure each file has a test file matching it so it reports the correct value or need to fix how the flow is checking coverage

[![Coverage Status](https://coveralls.io/repos/github/StevenDStanton/cli-tools/badge.svg?branch=master)](https://coveralls.io/github/StevenDStanton/cli-tools?branch=master)

I have written all tests to use Fuzz. However, this is not set up in the pipeline due to how expensive those tests are to run.

[Fuzz Testing](https://go.dev/doc/security/fuzz/)

### Normal Test Running

```bash
go test -v
```

### Fuzz Testing

```bash
go test --fuzz=Fuzz -fuzztime=1m
```

## Tools Included

- tts: Converts Markdown files to speech using the OpenAI API.

### tts

A simple CLI tool for converting text files to speech using the OpenAI API. The tool reads a text file (including Markdown), sends its content to the OpenAI API for text-to-speech conversion, and saves the generated audio file.

```bash
tts -f filename.md -o filename.mp3
```

```bash
tts --help

Usage: tts [OPTION]

	--configure          enter configuration prompt for API key
	--help               displays help
	--version            displays version information

	To use the program both of the below flags are require
	-o output audio file
	-f input text file

	Optional flags
	-v voice defaults to nova.
		Voice options are: alloy, echo, fable, onyx, nova, and shimmer

	-m model defaults to tts-1-hd
		Model options are: tts-1 and tts-1-hd

	-b laces buffer words at start and end of text to help with abrupt
		starts and ends


	-fmt output format defaults to mp3
		Format options are: mp3, opus, aac, flac, wav, pcm

	-s speed defaults to 1
		Speed options 0.25 to 4.0

```
