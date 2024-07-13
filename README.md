## Overview

This project is a continuation of my previous [Linux Tools for Windows](https://github.com/StevenDStanton/cli-tools-for-windows) initiative, aimed at bringing Linux CLI tools to Windows. The previous project was archived due to me switching from Windows to Linux for my daily driver OS. This project will include new tools I need moving forward.

## Tools Included

- tts: Converts text files to speech using the OpenAI API.

### tts

A simple CLI tool for converting text files to speech using the OpenAI API. The tool reads a text file (including Markdown), sends its content to the OpenAI API for text-to-speech conversion, and saves the generated audio file.

```bash
Usage: tts [OPTIONS]

Process text files with OpenAI's Text To Speech API.

Options:
  -f FILE       Input Markdown file
  -o FILE       Output audio file
  -v VOICE      Voice selection (default: nova)
                Options: alloy, echo, fable, onyx, nova, shimmer
  -m MODEL      Model selection (default: tts-1-hd)
                Options: tts-1, tts-1-hd
  -fmt FORMAT   Output format (default: mp3)
                Options: mp3, opus, aac, flac, wav, pcm
  -s SPEED      Set audio speed (default: 1.0)
                Range: 0.25 to 4.0
  -b            Place buffer words at start and end of text
  -r RATE       Rate limit for API calls per minute (default: unlimited)
  --configure   Enter configuration mode for API key setup
  --help        Display this help and exit
  --version     Output version information and exit

Example:
  tts -f input.md -o output.mp3
```

## Testing

### Status

Testing is not yet currently implemented.

Notes: Need to make sure each file has a test file matching it so it reports the correct value or need to fix how the flow is checking coverage

[![Coverage Status](https://coveralls.io/repos/github/StevenDStanton/cli-tools/badge.svg?branch=master)](https://coveralls.io/github/StevenDStanton/cli-tools?branch=master)

I have written all tests to use Fuzz. However, this is not set up in the pipeline due to how expensive those tests are to run.

[Fuzz Testing](https://go.dev/doc/security/fuzz/)

### Running Tests

#### Standard Tests

```bash
go test -v
```

#### Fuzz Tests

```bash
go test --fuzz=Fuzz -fuzztime=1m
```
