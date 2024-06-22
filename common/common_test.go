package common

import (
	"fmt"
	"strings"
	"testing"
)

func FuzzPrintVersion(f *testing.F) {
	f.Add("ToolName", "v0.1.0")
	f.Add("AnotherTool", "v1.2.3")
	f.Add("Yet Another Tool", "v.2.0.1")

	f.Fuzz(func(t *testing.T, tool string, version string) {
		// Call the function with fuzzing inputs
		output := PrintVersion(tool, version)

		// Perform checks on the output
		expectedHeader := fmt.Sprintf("%s: Version %s", tool, version)
		if !strings.Contains(output, expectedHeader) {
			t.Errorf("Output does not contain expected header. Got: %s, Expected: %s", output, expectedHeader)
		}

		expectedAuthor := "Author:         Steven Stanton"
		if !strings.Contains(output, expectedAuthor) {
			t.Errorf("Output does not contain expected author line. Got: %s, Expected: %s", output, expectedAuthor)
		}

		expectedLicense := "License:        MIT - No Warranty"
		if !strings.Contains(output, expectedLicense) {
			t.Errorf("Output does not contain expected license line. Got: %s, Expected: %s", output, expectedLicense)
		}

		expectedGithub := "Project Github: https://github.com/StevemStanton/cli-tools-for-windows"
		if !strings.Contains(output, expectedGithub) {
			t.Errorf("Output does not contain expected GitHub URL. Got: %s, Expected: %s", output, expectedGithub)
		}
	})
}
