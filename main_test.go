// main_test.go

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCalculateChunkSize(t *testing.T) {
	chunkSize := calculateChunkSize(false)
	expectedSize := API_MAX_CHARACTERS
	if chunkSize != expectedSize {
		t.Errorf("Expected chunk size %d, got %d", expectedSize, chunkSize)
	}
	chunkSizeWithBuffer := calculateChunkSize(true)
	startText := "Begin Text\n"
	endText := "\nEnd Text"
	startTextLen := utf8.RuneCountInString(startText)
	endTextLen := utf8.RuneCountInString(endText)
	expectedSizeWithBuffer := API_MAX_CHARACTERS - (startTextLen + endTextLen)
	if chunkSizeWithBuffer != expectedSizeWithBuffer {
		t.Errorf("Expected chunk size with buffer %d, got %d", expectedSizeWithBuffer, chunkSizeWithBuffer)
	}
}

func TestSplitIntoChunks(t *testing.T) {
	text := "This is a test. "
	chunkSize := 10
	chunks := splitIntoChunks(text, chunkSize)
	expectedChunks := []string{"This is a", " test. "}
	if !reflect.DeepEqual(chunks, expectedChunks) {
		t.Errorf("Expected chunks %v, got %v", expectedChunks, chunks)
	}

	text = "Short text"
	chunks = splitIntoChunks(text, chunkSize)
	expectedChunks = []string{"Short text"}
	if !reflect.DeepEqual(chunks, expectedChunks) {
		t.Errorf("Expected chunks %v, got %v", expectedChunks, chunks)
	}

	text = "Thisisaverylongwordthathasnospacesandshouldbesplitatmaximumchunksize."
	chunkSize = 20
	expectedChunks = []string{
		"Thisisaverylongwordt",
		"hathasnospacesandsho",
		"uldbesplitatmaximumc",
		"hunksize.",
	}
	chunks = splitIntoChunks(text, chunkSize)
	if !reflect.DeepEqual(chunks, expectedChunks) {
		t.Errorf("Expected chunks %v, got %v", expectedChunks, chunks)
	}
}

func TestAddBufferText(t *testing.T) {
	chunks := []string{"Chunk 1", "Chunk 2"}
	bufferedChunks := addBufferText(chunks)
	expectedChunks := []string{"Begin Text\nChunk 1\nEnd Text", "Begin Text\nChunk 2\nEnd Text"}
	if !reflect.DeepEqual(bufferedChunks, expectedChunks) {
		t.Errorf("Expected buffered chunks %v, got %v", expectedChunks, bufferedChunks)
	}
}

func TestReadFileData(t *testing.T) {
	text := "This is a test text to read and split into chunks."
	reader := strings.NewReader(text)
	bufferText := false
	chunks, err := readFileData(reader, bufferText)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Errorf("Expected chunk '%s', got '%s'", text, chunks[0])
	}
	bufferText = true
	reader = strings.NewReader(text)
	chunks, err = readFileData(reader, bufferText)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expectedText := "Begin Text\n" + text + "\nEnd Text"
	if chunks[0] != expectedText {
		t.Errorf("Expected chunk '%s', got '%s'", expectedText, chunks[0])
	}
}

func TestIsCommandAvailable(t *testing.T) {
	available := isCommandAvailable("go")
	if !available {
		t.Log("Command 'go' not found; test may be running in an environment without Go installed.")
	}
	available = isCommandAvailable("some_non_existent_command")
	if available {
		t.Errorf("Expected command 'some_non_existent_command' to be unavailable")
	}
}

type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestTTS(t *testing.T) {
	ttsRequest := TTSRequest{
		Model:  "test-model",
		Voice:  "test-voice",
		Format: "mp3",
		Input:  "Test input text",
		Speed:  "1.0",
	}
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method != "POST" {
				t.Errorf("Expected POST method, got %s", req.Method)
			}
			if req.URL.String() != API_URL {
				t.Errorf("Expected URL %s, got %s", API_URL, req.URL.String())
			}
			response := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("Mock audio data")),
			}
			return response, nil
		},
	}
	output := &bytes.Buffer{}
	config := Config{
		OpenAIAPIKey: "test-api-key",
	}
	err := tts(ttsRequest, output, mockClient, config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if output.String() != "Mock audio data" {
		t.Errorf("Expected output 'Mock audio data', got '%s'", output.String())
	}
}

func TestTTS_ErrorResponse(t *testing.T) {
	ttsRequest := TTSRequest{
		Model:  "test-model",
		Voice:  "test-voice",
		Format: "mp3",
		Input:  "Test input text",
		Speed:  "1.0",
	}
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			response := &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("Bad request")),
			}
			return response, nil
		},
	}
	output := &bytes.Buffer{}
	config := Config{
		OpenAIAPIKey: "test-api-key",
	}
	err := tts(ttsRequest, output, mockClient, config)
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		expectedError := "OpenAI API request failed with status code: 400, response body: Bad request"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	}
}

func TestTTS_RequestError(t *testing.T) {
	ttsRequest := TTSRequest{
		Model:  "test-model",
		Voice:  "test-voice",
		Format: "mp3",
		Input:  "Test input text",
		Speed:  "1.0",
	}
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}
	output := &bytes.Buffer{}
	config := Config{
		OpenAIAPIKey: "test-api-key",
	}
	err := tts(ttsRequest, output, mockClient, config)
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		expectedError := "unable to send request to OpenAI API: network error"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	}
}

func TestCleanupFiles(t *testing.T) {
	file1 := "testfile1.tmp"
	file2 := "testfile2.tmp"
	files := []string{file1, file2}
	for _, file := range files {
		f, err := os.Create(file)
		if err != nil {
			t.Fatalf("Failed to create temporary file %s: %v", file, err)
		}
		f.Close()
	}
	err := cleanupFiles(files)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	for _, file := range files {
		if _, err := os.Stat(file); !os.IsNotExist(err) {
			t.Errorf("Expected file %s to be deleted", file)
		}
	}
}

func TestCleanupFiles_Error(t *testing.T) {
	files := []string{"non_existent_file.tmp"}
	err := cleanupFiles(files)
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		if !strings.Contains(err.Error(), "error deleting file") {
			t.Errorf("Expected error message to contain 'error deleting file', got '%s'", err.Error())
		}
	}
}

func TestAppendToTextFile(t *testing.T) {
	textFileName := "test_append.txt"
	defer os.Remove(textFileName)
	err := appendToTextFile(textFileName, "output1.mp3")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	err = appendToTextFile(textFileName, "output2.mp3")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	data, err := os.ReadFile(textFileName)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", textFileName, err)
	}
	expectedContent := "file 'output1.mp3'\nfile 'output2.mp3'\n"
	if string(data) != expectedContent {
		t.Errorf("Expected file content:\n%s\nGot:\n%s", expectedContent, string(data))
	}
}

func TestProcessChunk(t *testing.T) {
	ttsRequest := TTSRequest{
		Model:  "test-model",
		Voice:  "test-voice",
		Format: "mp3",
		Input:  "Test input text",
		Speed:  "1.0",
	}
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			response := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("Mock audio data")),
			}
			return response, nil
		},
	}
	config := Config{
		OpenAIAPIKey: "test-api-key",
	}
	outputFileName := "test_output.mp3"
	defer os.Remove(outputFileName)
	err := processChunk(ttsRequest, outputFileName, mockClient, config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	data, err := os.ReadFile(outputFileName)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	if string(data) != "Mock audio data" {
		t.Errorf("Expected 'Mock audio data', got '%s'", string(data))
	}
}

func TestGetConfigPath(t *testing.T) {
	path, err := getConfigPath()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !strings.Contains(path, CONFIG_DIR) || !strings.HasSuffix(path, CONFIG_FILE) {
		t.Errorf("Expected path to contain '%s' and end with '%s', got '%s'", CONFIG_DIR, CONFIG_FILE, path)
	}
}

func TestCheckPrerequisites(t *testing.T) {
	flags := Flags{
		CombineFiles: false,
	}
	err := checkPrerequisites(flags)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	flags.CombineFiles = true
	originalIsCommandAvailable := isCommandAvailable
	defer func() { isCommandAvailable = originalIsCommandAvailable }()
	isCommandAvailable = func(name string) bool {
		return false
	}
	err = checkPrerequisites(flags)
	if err == nil {
		t.Errorf("Expected error due to missing ffmpeg, got nil")
	}
}

func TestReadInputFile(t *testing.T) {
	content := "This is test content for input file."
	inputFileName := "test_input.txt"
	err := os.WriteFile(inputFileName, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write input file: %v", err)
	}
	defer os.Remove(inputFileName)
	chunks, err := readInputFile(inputFileName, false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != content {
		t.Errorf("Expected chunk '%s', got '%s'", content, chunks[0])
	}
}

func TestCombineFiles(t *testing.T) {
	flags := Flags{
		OutputFile:   "combined_output.mp3",
		FormatOption: "mp3",
	}
	createdFiles := []string{"file1.mp3", "file2.mp3"}
	textFileName := fmt.Sprintf("%s.txt", strings.TrimSuffix(flags.OutputFile, filepath.Ext(flags.OutputFile)))
	err := os.WriteFile(textFileName, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}
	defer os.Remove(textFileName)
	err = combineFiles(flags, createdFiles)
	if err != nil {
		t.Logf("Expected error due to missing ffmpeg, got: %v", err)
	}
}

func TestPrintVersion(t *testing.T) {
	versionInfo := printVersion("tts", "v1.3.0")
	expected := `tts: Version v1.3.0

Copyright 2024 The Simple Dev

Author:         Steven Stanton
License:        MIT - No Warranty
Author Github:  https//github.com/StevenDStanton
Project Github: https://github.com/StevemStanton/cli-tools-for-windows

Part of my CLI Tools for Windows project.`
	if versionInfo != expected {
		t.Errorf("Expected version info:\n%s\nGot:\n%s", expected, versionInfo)
	}
}
