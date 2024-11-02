package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Mock os.Exit to prevent the test from exiting
var osExit = os.Exit

func TestParseFlags(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name     string
		args     []string
		expected Flags
	}{
		{
			name: "All flags provided",
			args: []string{
				"-f", "input.md",
				"-o", "output.mp3",
				"-v", "nova",
				"-m", "tts-1-hd",
				"-fmt", "mp3",
				"-s", "1.0",
				"-b",
				"-r", "10",
				"-c",
			},
			expected: Flags{
				InputFile:      "input.md",
				OutputFile:     "output.mp3",
				VoiceOption:    "nova",
				ModelOption:    "tts-1-hd",
				FormatOption:   "mp3",
				SpeedOption:    "1.0",
				ConfigureMode:  false,
				HelpFlag:       false,
				VersionFlag:    false,
				BufferTextFlag: true,
				RateLimit:      10,
				CombineFiles:   true,
			},
		},
		{
			name: "Default values",
			args: []string{
				"-f", "input.md",
				"-o", "output.mp3",
			},
			expected: Flags{
				InputFile:      "input.md",
				OutputFile:     "output.mp3",
				VoiceOption:    defaultVoice,
				ModelOption:    defaultModel,
				FormatOption:   defaultFormat,
				SpeedOption:    defaultSpeed,
				ConfigureMode:  false,
				HelpFlag:       false,
				VersionFlag:    false,
				BufferTextFlag: false,
				RateLimit:      0,
				CombineFiles:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			os.Args = append([]string{"cmd"}, tt.args...)
			flags := parseFlags()

			if flags != tt.expected {
				t.Errorf("Expected %+v, got %+v", tt.expected, flags)
			}
		})
	}
}

func FuzzParseFlags(f *testing.F) {
	f.Add("-f", "input.md", "-o", "output.mp3")
	f.Fuzz(func(t *testing.T, arg1, arg2, arg3 string) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", arg1, arg2, arg3}
		_ = parseFlags()
	})
}

func TestGetConfigPath(t *testing.T) {
	originalHomeDir := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHomeDir)

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	path := getConfigPath()
	expectedPath := filepath.Join(tempDir, CONFIG_DIR, CONFIG_FILE)

	if path != expectedPath {
		t.Errorf("Expected %s, got %s", expectedPath, path)
	}

	if _, err := os.Stat(filepath.Join(tempDir, CONFIG_DIR)); os.IsNotExist(err) {
		t.Errorf("Expected config directory to exist")
	}
}

func FuzzGetConfigPath(f *testing.F) {
	f.Add()
	f.Fuzz(func(t *testing.T) {
		_ = getConfigPath()
	})
}

func TestCheckFatalErrorExists(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	err := errors.New("test error")
	calledExit := false

	origExit := osExit
	osExit = func(code int) {
		calledExit = true
	}
	defer func() { osExit = origExit }()

	checkFatalErrorExists("Test message", err)

	if !calledExit {
		t.Errorf("Expected os.Exit to be called")
	}

	if !strings.Contains(buf.String(), "Test message: test error") {
		t.Errorf("Expected log message to contain 'Test message: test error'")
	}
}

func FuzzCheckFatalErrorExists(f *testing.F) {
	f.Add("Error message", errors.New("test error"))
	f.Fuzz(func(t *testing.T, msg string, err error) {
		checkFatalErrorExists(msg, err)
	})
}

func TestReadFileData(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "testinput*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := "This is a test content. It should be split into chunks."
	_, err = tempFile.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	flags.BufferTextFlag = false
	chunks := readFileData(tempFile.Name())
	expectedChunks := []string{content}

	if len(chunks) != len(expectedChunks) {
		t.Errorf("Expected %d chunks, got %d", len(expectedChunks), len(chunks))
	}

	for i, chunk := range chunks {
		if chunk != expectedChunks[i] {
			t.Errorf("Expected chunk %d to be %q, got %q", i, expectedChunks[i], chunk)
		}
	}

	flags.BufferTextFlag = true
	chunks = readFileData(tempFile.Name())
	startText := "Begin Text\n"
	endText := "\nEnd Text"
	expectedChunk := startText + content + endText

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != expectedChunk {
		t.Errorf("Expected chunk to be %q, got %q", expectedChunk, chunks[0])
	}
}

func FuzzReadFileData(f *testing.F) {
	f.Add("testinput.md")
	f.Fuzz(func(t *testing.T, inputFile string) {
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			return
		}
		_ = readFileData(inputFile)
	})
}

func TestIsCommandAvailable(t *testing.T) {
	if !isCommandAvailable("go") {
		t.Errorf("Expected 'go' command to be available")
	}
	if isCommandAvailable("nonexistentcommand") {
		t.Errorf("Expected 'nonexistentcommand' to not be available")
	}
}

func FuzzIsCommandAvailable(f *testing.F) {
	f.Add("go")
	f.Fuzz(func(t *testing.T, cmd string) {
		_ = isCommandAvailable(cmd)
	})
}

func TestCleanupFiles(t *testing.T) {
	tempFile1, err := ioutil.TempFile("", "testfile1")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile2, err := ioutil.TempFile("", "testfile2")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	files := []string{tempFile1.Name(), tempFile2.Name()}

	cleanupFiles(files)

	for _, file := range files {
		if _, err := os.Stat(file); !os.IsNotExist(err) {
			t.Errorf("Expected file %s to be deleted", file)
		}
	}
}

func FuzzCleanupFiles(f *testing.F) {
	f.Add([]string{"testfile1", "testfile2"})
	f.Fuzz(func(t *testing.T, files []string) {
		cleanupFiles(files)
	})
}

func TestPrintHelp(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	printHelp()

	if buf.Len() == 0 {
		t.Errorf("Expected help message to be printed")
	}
}

func FuzzPrintHelp(f *testing.F) {
	f.Add()
	f.Fuzz(func(t *testing.T) {
		printHelp()
	})
}

func TestPrintVersion(t *testing.T) {
	output := printVersion("tts", "v1.2.6")
	if !strings.Contains(output, "Version v1.2.6") {
		t.Errorf("Expected version information to contain 'Version v1.2.6'")
	}
}

func FuzzPrintVersion(f *testing.F) {
	f.Add("tts", "v1.2.6")
	f.Fuzz(func(t *testing.T, tool, version string) {
		_ = printVersion(tool, version)
	})
}

func TestMakeHttpRequest(t *testing.T) {
	server := httpTestServer()
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	tempFile, err := ioutil.TempFile("", "output*.mp3")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	makeHttpRequest(req, tempFile.Name())

	data, err := ioutil.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if string(data) != "test audio content" {
		t.Errorf("Expected 'test audio content', got %q", string(data))
	}
}

func httpTestServer() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "test audio content")
	})
	server := httptest.NewServer(handler)
	return server
}

func FuzzMakeHttpRequest(f *testing.F) {
	f.Add("http://example.com")
	f.Fuzz(func(t *testing.T, urlStr string) {
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			return
		}
		makeHttpRequest(req, "output.mp3")
	})
}

func TestTTS(t *testing.T) {
	server := httpTestServer()
	defer server.Close()

	ttsRequest := TTSRequest{
		Model:  "tts-1-hd",
		Voice:  "nova",
		Format: "mp3",
		Input:  "Hello, World!",
		Speed:  "1.0",
	}

	config := Config{
		OpenAIAPIKey: "testkey",
	}

	tempFile, err := ioutil.TempFile("", "output*.mp3")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Mock the API endpoint by replacing the URL in the request
	originalAPIURL := apiURL
	defer func() { apiURL = originalAPIURL }()
	apiURL = server.URL

	tts(ttsRequest, tempFile.Name(), config)

	data, err := ioutil.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if string(data) != "test audio content" {
		t.Errorf("Expected 'test audio content', got %q", string(data))
	}
}

func FuzzTTS(f *testing.F) {
	f.Add("Hello, World!")
	f.Fuzz(func(t *testing.T, input string) {
		ttsRequest := TTSRequest{
			Model:  "tts-1-hd",
			Voice:  "nova",
			Format: "mp3",
			Input:  input,
			Speed:  "1.0",
		}

		config := Config{
			OpenAIAPIKey: "testkey",
		}

		tts(ttsRequest, "output.mp3", config)
	})
}

func TestConfigConfigure(t *testing.T) {
	config := &Config{}

	tempDir := t.TempDir()
	config.configPath = filepath.Join(tempDir, CONFIG_FILE)

	// Mock user input
	var buf bytes.Buffer
	buf.WriteString("testkey\n")
	stdin := os.Stdin
	defer func() { os.Stdin = stdin }()
	os.Stdin = &buf

	config.configure()

	if config.OpenAIAPIKey != "testkey" {
		t.Errorf("Expected OpenAIAPIKey to be 'testkey', got '%s'", config.OpenAIAPIKey)
	}
}

func FuzzConfigConfigure(f *testing.F) {
	f.Add()
	f.Fuzz(func(t *testing.T) {
		config := &Config{}
		config.configure()
	})
}

func TestConfigReadConfig(t *testing.T) {
	config := &Config{}
	tempFile, err := ioutil.TempFile("", "config*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := "OPENAI_API_KEY=testkey"
	_, err = tempFile.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	config.configPath = tempFile.Name()
	config.readConfig()

	if config.OpenAIAPIKey != "testkey" {
		t.Errorf("Expected OpenAIAPIKey to be 'testkey', got %q", config.OpenAIAPIKey)
	}
}

func FuzzConfigReadConfig(f *testing.F) {
	f.Add()
	f.Fuzz(func(t *testing.T) {
		config := &Config{}
		config.readConfig()
	})
}
