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
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/StevenDStanton/cli-tools/common"
)

type TTSRequest struct {
	Model  string `json:"model"`
	Input  string `json:"input"`
	Voice  string `json:"voice"`
	Format string `json:"response_format"`
	Speed  string `json:"speed"`
}

const (
	CONFIG_FILE   = "tts.config"
	CONFIG_DIR    = ".cli-tools"
	defaultVoice  = "nova"
	defaultModel  = "tts-1-hd"
	defaultFormat = "mp3"
	defaultSpeed  = "1.0"
	version       = "v1.1.3"
	tool          = "tts"
)

var (
	configFilePath string
	OPENAI_API_KEY string
)

var (
	inputFile      = flag.String("f", "", "Input Markdown file")
	outputFile     = flag.String("o", "", "Output audio file")
	voiceOption    = flag.String("v", defaultVoice, "Voice Selection")
	modelOption    = flag.String("m", defaultModel, "Model Selection")
	formatOption   = flag.String("fmt", defaultFormat, "Select output format")
	speedOption    = flag.String("s", defaultSpeed, "Set audio speed")
	configureMode  = flag.Bool("configure", false, "Enter Configuration Mode")
	helpFlag       = flag.Bool("help", false, "Displays Help Menu")
	versionFlag    = flag.Bool("version", false, "Displays version information")
	bufferTextFlag = flag.Bool("b", false, "Places buffer words at start and end of text to help with abrupt starts and ends")
	rateLimit      = flag.Int("r", 0, "Rate limit for API calls per minute")
)

func init() {
	configure()
	flag.Parse()

	switch {
	case *helpFlag:
		printHelp()
		os.Exit(0)
	case *configureMode:
		writeNewConfig()
		os.Exit(0)
	case *versionFlag:
		versionInformation := common.PrintVersion(tool, version)
		fmt.Println(versionInformation)
		os.Exit(0)
	default:
		if *inputFile == "" || *outputFile == "" {
			fmt.Println("Usage: tts -f filename.md -o filename.mp3")
			os.Exit(0)
		}
	}
}

func main() {
	chunks := readFileData(*inputFile)
	if len(chunks) > 1 {
		fmt.Printf("This will create %d files. Are you sure you wish to continue? (y/n): ", len(chunks))
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Operation cancelled.")
			os.Exit(0)
		}
	}

	rateLimiter := time.Tick(time.Minute / time.Duration(*rateLimit))

	for i, chunk := range chunks {
		var outputFileName string
		if len(chunks) == 1 {
			outputFileName = *outputFile
		} else {
			outputFileName = fmt.Sprintf("%s_%d.mp3", strings.TrimSuffix(*outputFile, filepath.Ext(*outputFile)), i+1)
		}
		ttsRequest := TTSRequest{
			Model:  *modelOption,
			Voice:  *voiceOption,
			Format: *formatOption,
			Input:  chunk,
			Speed:  *speedOption,
		}

		if *rateLimit > 0 {
			<-rateLimiter
		}

		tts(ttsRequest, outputFileName)
	}
}

//This can be improved in the future to have a single config setup
//for all cli-tools-for-windows. However, to avoid over engineering the solution for
//now this single setup works. I will reveiew and refactor if it becomes an issue.
//For now each file gets a config for its usage

func configure() {
	home, err := os.UserHomeDir()
	checkFatalErrorExists("Unable to read user home directory", err)

	configDir := filepath.Join(home, CONFIG_DIR)

	err = os.MkdirAll(configDir, 0755)
	checkFatalErrorExists("Unable to create config directory", err)

	configFilePath = filepath.Join(configDir, CONFIG_FILE)

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		writeNewConfig()
		return
	}
	checkFatalErrorExists("Unknown issue accessing config", err)

	readConfig()

}

func checkFatalErrorExists(message string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

func writeNewConfig() {
	fmt.Print("Please enter your OpenAI API Key: ")
	fmt.Scanln(&OPENAI_API_KEY)
	fileData := "OPENAI_API_KEY=" + OPENAI_API_KEY
	err := os.WriteFile(configFilePath, []byte(fileData), 0600)
	checkFatalErrorExists("", err)
	if err != nil {
		log.Fatalf("Unable to save config: %v\n", err)
	}
}

func readConfig() {
	file, err := os.Open(configFilePath)
	checkFatalErrorExists("unable to read config fil", err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key == "OPENAI_API_KEY" {
				OPENAI_API_KEY = value
			}

		}
	}

	if err := scanner.Err(); err != nil {
		checkFatalErrorExists("unable to read config file", err)
	}

	if OPENAI_API_KEY == "" {
		writeNewConfig()
	}
}

func readFileData(inputFile string) []string {
	inputContent, err := os.ReadFile(inputFile)
	checkFatalErrorExists("Error: reading input file", err)
	startText := "Begin Text\n"
	endText := "\nEnd Text"

	chunkSize := 4096
	if *bufferTextFlag {

		startTextLen := utf8.RuneCountInString(startText)
		endTextLen := utf8.RuneCountInString(endText)

		chunkSize = chunkSize - startTextLen - endTextLen
	}

	var chunks []string
	inputString := string(inputContent)

	for len(inputString) > 0 {
		if utf8.RuneCountInString(inputString) <= chunkSize {
			if *bufferTextFlag {
				chunks = append(chunks, startText+inputString+endText)
				break
			}
			chunks = append(chunks, inputString)
			break
		}
		splitIndex := chunkSize
		for ; splitIndex > 0 && !unicode.IsSpace(rune(inputString[splitIndex])); splitIndex-- {
		}
		if *bufferTextFlag {
			chunks = append(chunks, startText+inputString[:splitIndex]+endText)
		} else {
			chunks = append(chunks, inputString[:splitIndex])
		}
		inputString = strings.TrimSpace(inputString[splitIndex:])
	}

	return chunks
}

func tts(ttsRequest TTSRequest, outputFile string) {
	requestBody, err := json.Marshal(ttsRequest)
	checkFatalErrorExists("Error: Unable to create request payload", err)

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/speech", bytes.NewBuffer(requestBody))
	checkFatalErrorExists("Error: Unable to create HTTP request", err)

	req.Header.Set("Authorization", "Bearer "+OPENAI_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	makeHttpRequest(req, outputFile)

}

func makeHttpRequest(req *http.Request, outputFile string) {
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request to OpenAI API: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		log.Printf("OpenAI API request failed with status code: %d, response body: %s", resp.StatusCode, responseBody)
		return
	}

	outputFileData, err := os.Create(outputFile)
	if err != nil {
		log.Printf("Error creating output file: %v", err)
		return
	}
	defer outputFileData.Close()

	_, err = io.Copy(outputFileData, resp.Body)
	if err != nil {
		log.Printf("Error saving audio file: %v", err)
		return
	}

	fmt.Printf("Audio file saved successfully: %s\n", outputFile)
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
	fmt.Println(help)
}
