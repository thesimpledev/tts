package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type TTSRequest struct {
	Model  string `json:"model"`
	Input  string `json:"input"`
	Voice  string `json:"voice"`
	Format string `json:"response_format"`
	Speed  string `json:"speed"`
}

type Config struct {
	OpenAIAPIKey string
	rateLimiter  <-chan time.Time
	configPath   string
}

type Flags struct {
	InputFile      string
	OutputFile     string
	VoiceOption    string
	ModelOption    string
	FormatOption   string
	SpeedOption    string
	ConfigureMode  bool
	HelpFlag       bool
	VersionFlag    bool
	BufferTextFlag bool
	RateLimit      int
	CombineFiles   bool
}

const (
	CONFIG_FILE        = "tts.config"
	CONFIG_DIR         = ".cli-tools"
	defaultVoice       = "nova"
	defaultModel       = "tts-1-hd"
	defaultFormat      = "mp3"
	defaultSpeed       = "1.0"
	version            = "v1.2.0"
	tool               = "tts"
	API_MAX_CHARACTERS = 4096
)

var (
	multiFile = false
	flags     Flags
)

func main() {
	flags = parseFlags()
	var config Config
	config.configure()

	if flags.CombineFiles && !isCommandAvailable("ffmpeg") {
		log.Fatal("ffmpeg is not installed or not found in PATH")
	}

	switch {
	case flags.HelpFlag:
		printHelp()
		os.Exit(0)
	case flags.ConfigureMode:
		config.writeNewConfig()
		os.Exit(0)
	case flags.VersionFlag:
		versionInformation := printVersion(tool, version)
		log.Print(versionInformation)
		os.Exit(0)
	default:
		if flags.InputFile == "" || flags.OutputFile == "" {
			log.Printf("Usage: tts -f filename.md -o filename.mp3")
			os.Exit(0)
		}
	}

	chunks := readFileData(flags.InputFile)
	if len(chunks) > 1 {
		log.Printf("This will create %d files. Are you sure you wish to continue? (y/n): ", len(chunks))
		multiFile = true
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			log.Printf("Operation cancelled.")
			os.Exit(0)
		}
	}
	var createdFiles []string
	var textFileName string

	if flags.CombineFiles {
		textFileName = fmt.Sprintf("%s.txt", strings.TrimSuffix(flags.OutputFile, filepath.Ext(flags.OutputFile)))
		createdFiles = append(createdFiles, textFileName)
	}

	for i, chunk := range chunks {
		outputFileName := flags.OutputFile
		if multiFile {

			outputFileName = fmt.Sprintf("%s_%d.mp3", strings.TrimSuffix(flags.OutputFile, filepath.Ext(flags.OutputFile)), i+1)
			createdFiles = append(createdFiles, outputFileName)
			if flags.CombineFiles {
				file, err := os.OpenFile(textFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

				checkFatalErrorExists("Error opening text file", err)

				defer file.Close()

				_, err = file.WriteString("file '" + outputFileName + "'\n")

				checkFatalErrorExists("Error writing to text file", err)
			}
		}

		ttsRequest := TTSRequest{
			Model:  flags.ModelOption,
			Voice:  flags.VoiceOption,
			Format: flags.FormatOption,
			Input:  chunk,
			Speed:  flags.SpeedOption,
		}

		if flags.RateLimit > 0 {
			<-config.rateLimiter
		}

		tts(ttsRequest, outputFileName, config)
	}

	if multiFile && flags.CombineFiles {
		textFileName := fmt.Sprintf("%s.txt", strings.TrimSuffix(flags.OutputFile, filepath.Ext(flags.OutputFile)))

		cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", textFileName, "-c", "copy", flags.OutputFile)

		err := cmd.Run()

		checkFatalErrorExists("Error combining audio files", err)

		cleanupFiles(createdFiles)
	}
}

func parseFlags() Flags {
	flags := Flags{}

	flag.StringVar(&flags.InputFile, "f", "", "Input Markdown file")
	flag.StringVar(&flags.OutputFile, "o", "", "Output audio file")
	flag.StringVar(&flags.VoiceOption, "v", defaultVoice, "Voice Selection")
	flag.StringVar(&flags.ModelOption, "m", defaultModel, "Model Selection")
	flag.StringVar(&flags.FormatOption, "fmt", defaultFormat, "Select output format")
	flag.StringVar(&flags.SpeedOption, "s", defaultSpeed, "Set audio speed")
	flag.BoolVar(&flags.ConfigureMode, "configure", false, "Enter Configuration Mode")
	flag.BoolVar(&flags.HelpFlag, "help", false, "Displays Help Menu")
	flag.BoolVar(&flags.VersionFlag, "version", false, "Displays version information")
	flag.BoolVar(&flags.BufferTextFlag, "b", false, "Places buffer words at start and end of text to help with abrupt starts and ends")
	flag.IntVar(&flags.RateLimit, "r", 0, "Rate limit for API calls per minute")
	flag.BoolVar(&flags.CombineFiles, "c", false, "Combine multiple files into a single audio file")

	flag.Parse()
	return flags
}

//This can be improved in the future to have a single config setup
//for all cli-tools-for-windows. However, to avoid over engineering the solution for
//now this single setup works. I will reveiew and refactor if it becomes an issue.
//For now each file gets a config for its usage

func (c *Config) configure() {

	c.configPath = getConfigPath()

	if _, err := os.Stat(c.configPath); os.IsNotExist(err) {
		c.writeNewConfig()
	} else {
		c.readConfig()
	}

	if flags.RateLimit > 0 {
		c.rateLimiter = time.Tick(time.Minute / time.Duration(flags.RateLimit))
	}

}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	checkFatalErrorExists("Unable to read user home directory", err)

	configDir := filepath.Join(home, CONFIG_DIR)

	err = os.MkdirAll(configDir, 0755)
	checkFatalErrorExists("Unable to create config directory", err)

	configFilePath := filepath.Join(configDir, CONFIG_FILE)

	return configFilePath

}

func checkFatalErrorExists(message string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

func (c *Config) writeNewConfig() {
	log.Printf("Please enter your OpenAI API Key: ")
	fmt.Scanln(&c.OpenAIAPIKey)
	fileData := "OPENAI_API_KEY=" + c.OpenAIAPIKey
	err := os.WriteFile(c.configPath, []byte(fileData), 0600)
	checkFatalErrorExists("Unable to save config file", err)
}

func (c *Config) readConfig() {
	file, err := os.Open(c.configPath)
	checkFatalErrorExists("Unable to open config file", err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key == "OPENAI_API_KEY" {
				c.OpenAIAPIKey = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		checkFatalErrorExists("unable to read config file", err)
	}

	if c.OpenAIAPIKey == "" {
		c.writeNewConfig()
	}

}

func readFileData(inputFile string) []string {
	inputContent, err := os.ReadFile(inputFile)
	checkFatalErrorExists("Error: reading input file", err)
	startText := "Begin Text\n"
	endText := "\nEnd Text"

	chunkSize := API_MAX_CHARACTERS
	if flags.BufferTextFlag {

		startTextLen := utf8.RuneCountInString(startText)
		endTextLen := utf8.RuneCountInString(endText)

		chunkSize = chunkSize - startTextLen - endTextLen
	}

	var chunks []string
	inputRunes := []rune(string(inputContent))

	for len(inputRunes) > 0 {
		if len(inputRunes) <= chunkSize {
			chunk := string(inputRunes)
			if flags.BufferTextFlag {
				chunks = append(chunks, startText+chunk+endText)
				break
			}
			chunks = append(chunks, chunk)
			break
		}
		splitIndex := chunkSize
		for ; splitIndex > 0 && !unicode.IsSpace(inputRunes[splitIndex]); splitIndex-- {
		}
		chunk := string(inputRunes[:splitIndex])
		if flags.BufferTextFlag {
			chunks = append(chunks, startText+chunk+endText)
		} else {
			chunks = append(chunks, chunk)
		}
		inputRunes = inputRunes[splitIndex:]
	}

	return chunks
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func tts(ttsRequest TTSRequest, outputFile string, config Config) {
	requestBody, err := json.Marshal(ttsRequest)
	checkFatalErrorExists("Error: Unable to create request payload", err)

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/speech", bytes.NewBuffer(requestBody))
	checkFatalErrorExists("Error: Unable to create HTTP request", err)

	req.Header.Set("Authorization", "Bearer "+config.OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	makeHttpRequest(req, outputFile)

}

func cleanupFiles(files []string) {
	for _, file := range files {
		log.Printf("Deleting file: %s\n", file)
		err := os.Remove(file)
		checkFatalErrorExists("Error deleting file", err)
	}
}

func makeHttpRequest(req *http.Request, outputFile string) {
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	checkFatalErrorExists("Error: Unable to send request to OpenAI API", err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		log.Printf("OpenAI API request failed with status code: %d, response body: %s", resp.StatusCode, responseBody)
		return
	}

	outputFileData, err := os.Create(outputFile)

	checkFatalErrorExists("Error: Unable to create output file", err)
	defer outputFileData.Close()

	_, err = io.Copy(outputFileData, resp.Body)
	checkFatalErrorExists("Error: Unable to write to output file", err)

	log.Printf("Audio file saved successfully: %s\n", outputFile)
}

func printHelp() {
	help := `Usage: tts [OPTIONS]

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
`
	log.Print(help)
}

func printVersion(tool string, version string) string {
	return fmt.Sprintf(`%s: Version %s

Copyright 2024 The Simple Dev

Author:         Steven Stanton
License:        MIT - No Warranty
Author Github:  https//github.com/StevenDStanton
Project Github: https://github.com/StevemStanton/cli-tools-for-windows

Part of my CLI Tools for Windows project.`, tool, version)
}
