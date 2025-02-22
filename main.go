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

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

const (
	config_file    = "tts.config"
	config_dir     = ".cli-tools"
	default_voice  = "nova"
	default_model  = "tts-1-hd"
	default_format = "mp3"
	default_speed  = "1.0"
	version        = "v1.3.2"
	tool           = "tts"
	api_max_chars  = 4096
	api_url        = "https://api.openai.com/v1/audio/speech"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	flags := parseFlags()
	var config Config

	if err := config.configure(flags.RateLimit); err != nil {
		return fmt.Errorf("unable to configure: %w", err)
	}

	exit, err := handleFlags(flags, &config)
	if err != nil {
		return fmt.Errorf("unable to handle flags: %w", err)
	}
	if exit {
		return nil
	}

	if err := checkPrerequisites(flags); err != nil {
		return err
	}

	chunks, err := readInputFile(flags.InputFile, flags.BufferTextFlag)
	if err != nil {
		return err
	}

	multiFile := len(chunks) > 1

	if multiFile {
		proceed, err := promptForConfirmation(len(chunks))
		if err != nil {
			return err
		}
		if !proceed {
			log.Printf("Operation cancelled.")
			return nil
		}
	}

	var createdFiles []string

	if err := processChunks(chunks, flags, config, &createdFiles); err != nil {
		return err
	}

	if multiFile && flags.CombineFiles {
		if err := combineFiles(flags, createdFiles); err != nil {
			return err
		}
	}

	return nil
}

func parseFlags() Flags {
	flags := Flags{}

	flag.StringVar(&flags.InputFile, "f", "", "Input Markdown file")
	flag.StringVar(&flags.OutputFile, "o", "", "Output audio file")
	flag.StringVar(&flags.VoiceOption, "v", default_voice, "Voice Selection")
	flag.StringVar(&flags.ModelOption, "m", default_model, "Model Selection")
	flag.StringVar(&flags.FormatOption, "fmt", default_format, "Select output format")
	flag.StringVar(&flags.SpeedOption, "s", default_speed, "Set audio speed")
	flag.BoolVar(&flags.ConfigureMode, "configure", false, "Enter Configuration Mode")
	flag.BoolVar(&flags.HelpFlag, "help", false, "Displays Help Menu")
	flag.BoolVar(&flags.VersionFlag, "version", false, "Displays version information")
	flag.BoolVar(&flags.BufferTextFlag, "b", false, "Places buffer words at start and end of text to help with abrupt starts and ends")
	flag.IntVar(&flags.RateLimit, "r", 0, "Rate limit for API calls per minute")
	flag.BoolVar(&flags.CombineFiles, "c", false, "Combine multiple files into a single audio file")

	flag.Parse()
	return flags
}

func handleFlags(flags Flags, config *Config) (bool, error) {
	switch {
	case flags.HelpFlag:
		log.Print(printHelp())
		return true, nil
	case flags.ConfigureMode:
		config.writeNewConfig()
		return true, nil
	case flags.VersionFlag:
		log.Print(printVersion(tool, version))
		return true, nil
	default:
		if flags.InputFile == "" || flags.OutputFile == "" {
			return false, fmt.Errorf("input and output files must be specified. Usage: tts -f filename.md -o filename.mp3")
		}
	}

	return false, nil
}

func (c *Config) configure(ratelimit int) error {

	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("unable to get config path: %w", err)
	}
	c.configPath = configPath

	if _, err := os.Stat(c.configPath); os.IsNotExist(err) {
		if err := c.writeNewConfig(); err != nil {
			return err
		}
	} else {
		if err := c.readConfig(); err != nil {
			return err
		}
	}

	if ratelimit > 0 {
		c.rateLimiter = time.Tick(time.Minute / time.Duration(ratelimit))
	}

	return nil
}

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to get user home directory: %w", err)
	}

	configDir := filepath.Join(home, config_dir)

	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return "", fmt.Errorf("unable to create config directory: %w", err)
	}

	configFilePath := filepath.Join(configDir, config_file)

	return configFilePath, nil

}

func (c *Config) writeNewConfig() error {
	apiKey, err := promptForAPIKey()
	if err != nil {
		return err
	}
	c.OpenAIAPIKey = apiKey
	fileData := "OPENAI_API_KEY=" + c.OpenAIAPIKey
	err = os.WriteFile(c.configPath, []byte(fileData), 0600)
	if err != nil {
		return fmt.Errorf("unable to write config file: %w", err)
	}
	return nil
}

func promptForAPIKey() (string, error) {
	fmt.Print("Please enter your OpenAI API Key: ")
	var apiKey string
	_, err := fmt.Scanln(&apiKey)
	if err != nil {
		return "", fmt.Errorf("unable to read API key: %w", err)
	}
	return apiKey, nil
}

func (c *Config) readConfig() error {
	file, err := os.Open(c.configPath)
	if err != nil {
		return fmt.Errorf("unable to open config file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		key, value, found := strings.Cut(line, "=")
		if found {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if key == "OPENAI_API_KEY" {
				c.OpenAIAPIKey = value
			}
		}

	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("unable to read config file: %w", err)
	}

	if c.OpenAIAPIKey == "" {
		c.writeNewConfig()
	}

	return nil

}

func readFileData(r io.Reader, bufferText bool) ([]string, error) {
	inputContent, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading input data: %w", err)
	}

	chunkSize := calculateChunkSize(bufferText)
	chunks := splitIntoChunks(string(inputContent), chunkSize)

	if bufferText {
		chunks = addBufferText(chunks)
	}

	return chunks, nil
}

func calculateChunkSize(bufferText bool) int {
	chunkSize := api_max_chars
	if bufferText {
		startText := "Begin Text\n"
		endText := "\nEnd Text"
		startTextLen := utf8.RuneCountInString(startText)
		endTextLen := utf8.RuneCountInString(endText)
		chunkSize -= (startTextLen + endTextLen)
	}
	return chunkSize
}

func splitIntoChunks(text string, chunkSize int) []string {
	var chunks []string
	inputRunes := []rune(text)

	for len(inputRunes) > 0 {
		if len(inputRunes) <= chunkSize {
			chunks = append(chunks, string(inputRunes))
			break
		}
		splitIndex := chunkSize
		for ; splitIndex > 0 && !unicode.IsSpace(inputRunes[splitIndex]); splitIndex-- {
		}
		if splitIndex == 0 {
			splitIndex = chunkSize // If no space found, force split
		}
		chunks = append(chunks, string(inputRunes[:splitIndex]))
		inputRunes = inputRunes[splitIndex:]
	}

	return chunks
}

func addBufferText(chunks []string) []string {
	startText := "Begin Text\n"
	endText := "\nEnd Text"
	for i, chunk := range chunks {
		chunks[i] = startText + chunk + endText
	}
	return chunks
}

func checkPrerequisites(flags Flags) error {
	if flags.CombineFiles && !isCommandAvailable("ffmpeg") {
		return fmt.Errorf("ffmpeg is required for combining files. Please install ffmpeg and try again")
	}
	return nil
}

func readInputFile(inputFileName string, bufferText bool) ([]string, error) {
	inputFile, err := os.Open(inputFileName)
	if err != nil {
		return nil, fmt.Errorf("unable to open input file: %w", err)
	}
	defer inputFile.Close()

	chunks, err := readFileData(inputFile, bufferText)
	if err != nil {
		return nil, fmt.Errorf("unable to read input file data: %w", err)
	}
	return chunks, nil
}

func processChunks(chunks []string, flags Flags, config Config, createdFiles *[]string) error {
	multiFile := len(chunks) > 1
	httpClient := &http.Client{Timeout: 90 * time.Second}
	var textFileName string

	if flags.CombineFiles && multiFile {
		textFileName = fmt.Sprintf("%s.txt", strings.TrimSuffix(flags.OutputFile, filepath.Ext(flags.OutputFile)))
		*createdFiles = append(*createdFiles, textFileName)
	}

	for i, chunk := range chunks {
		outputFileName := flags.OutputFile
		if multiFile {
			outputFileName = fmt.Sprintf("%s_%d.%s", strings.TrimSuffix(flags.OutputFile, filepath.Ext(flags.OutputFile)), i+1, flags.FormatOption)
			*createdFiles = append(*createdFiles, outputFileName)

			if flags.CombineFiles {
				if err := appendToTextFile(textFileName, outputFileName); err != nil {
					return err
				}
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

		if err := processChunk(ttsRequest, outputFileName, httpClient, config); err != nil {
			return err
		}
	}

	return nil
}

func processChunk(ttsRequest TTSRequest, outputFileName string, client HTTPClient, config Config) error {
	outputFileData, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("unable to create output file: %w", err)
	}
	defer outputFileData.Close()

	err = tts(ttsRequest, outputFileData, client, config)
	if err != nil {
		return fmt.Errorf("unable to process audio data: %w", err)
	}

	return nil
}

func appendToTextFile(textFileName, outputFileName string) error {
	file, err := os.OpenFile(textFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to open text file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("file '%s'\n", outputFileName))
	if err != nil {
		return fmt.Errorf("unable to write to text file: %w", err)
	}
	return nil
}

func combineFiles(flags Flags, createdFiles []string) error {
	textFileName := fmt.Sprintf("%s.txt", strings.TrimSuffix(flags.OutputFile, filepath.Ext(flags.OutputFile)))

	cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", textFileName, "-c", "copy", flags.OutputFile)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("unable to combine files: %w", err)
	}

	if err := cleanupFiles(createdFiles); err != nil {
		log.Printf("Cleanup completed with errors:\n%v", err)
	}
	return nil
}

func promptForConfirmation(numFiles int) (bool, error) {
	log.Printf("This will create %d files. Are you sure you wish to continue? (y/n): ", numFiles)
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(strings.TrimSpace(response)) != "y" {
		return false, nil
	}
	return true, nil
}

var isCommandAvailable = func(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func tts(ttsRequest TTSRequest, output io.Writer, client HTTPClient, config Config) error {
	requestBody, err := json.Marshal(ttsRequest)
	if err != nil {
		return fmt.Errorf("unable to create request payload: %w", err)
	}

	req, err := http.NewRequest("POST", api_url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("unable to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send request to OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI API request failed with status code: %d, response body: %s", resp.StatusCode, responseBody)
	}

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return fmt.Errorf("unable to write to output: %w", err)
	}

	log.Printf("Audio data processed successfully.\n")
	return nil

}

func cleanupFiles(files []string) error {
	var errs []string
	for _, file := range files {
		log.Printf("Deleting file: %s\n", file)
		err := os.Remove(file)
		if err != nil {
			errs = append(errs, fmt.Sprintf("error deleting file %s: %v", file, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

func printHelp() string {
	return `Usage: tts [OPTIONS]

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
}

func printVersion(tool, version string) string {
	return fmt.Sprintf(`%s: Version %s

Copyright 2024 The Simple Dev

Author:         Steven Stanton
License:        MIT - No Warranty
Author Github:  https//github.com/StevenDStanton
Project Github: https://github.com/StevenStanton/tts

Part of my CLI Tools for Windows project.`, tool, version)
}
