# TTS CLI Tool

## Overview

This project provides a streamlined CLI tool to convert text files, including Markdown, into speech using the OpenAI API. Initially part of a broader CLI tools project, it has now been refined to focus on text-to-speech functionality.

## Setup

Upon first use, you’ll be prompted to enter your OpenAI API key. Alternatively, you can enter configuration mode to set up or modify the API key by running:

```bash
tts --configure
```

## Features

- TTS Conversion: Reads a text or Markdown file, converts it to speech using OpenAI's API, and saves it as an audio file.
- Customizable Voice and Model: Choose from different voice options and TTS models to match your preferred audio style.
- Flexible Output: Supports multiple audio formats, including MP3, WAV, FLAC, and more.
- Adjustable Speed: Control audio playback speed, from slow-paced narration to faster speech.
- File Combination: Optionally combine multiple text files into a single audio file.

## To Do

- [ ] tts add optional flag for break point between audio files in text.
- [ ] improve error messages
- [ ] Clean up created files on early exit

### tts

A simple CLI tool for converting text files to speech using the OpenAI API. The tool reads a text file (including Markdown), sends its content to the OpenAI API for text-to-speech conversion, and saves the generated audio file.

```bash
Usage: tts [OPTIONS]

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
  -c            Combine multiple text files into a single audio file
  --configure   Enter configuration mode for API key setup
  --help        Display help and exit
  --version     Output version information and exit

Example:
  tts -f input.md -o output.mp3
```

## Testing

### Status

[![Coverage Status](https://coveralls.io/repos/github/StevenDStanton/tts/badge.svg?branch=master)](https://coveralls.io/github/StevenDStanton/tts?branch=master)

### Running Tests

#### Standard Tests

```bash
go test -v
```

#### Fuzz Tests

```bash
go test --fuzz=Fuzz -fuzztime=1m
```
