package wasm_test

import (
	"os"
	"path"
	"strings"
	"testing"
)

func CaptureStdio(t *testing.T, main func()) (stdout, stderr string) {
	// Setup files to capture stdout and stderr
	tmp := t.TempDir()

	stdoutPath := path.Join(tmp, "stdout.txt")
	stdoutF, err := os.Create(stdoutPath)
	if err != nil {
		t.Fatal(err)
	}

	stderrPath := path.Join(tmp, "stderr.txt")
	stderrF, err := os.Create(stderrPath)
	if err != nil {
		t.Fatal(err)
	}

	// Save the old os.XXX and revert regardless of the outcome.
	oldStdout := os.Stdout
	os.Stdout = stdoutF
	oldStderr := os.Stderr
	os.Stderr = stderrF
	revertOS := func() {
		os.Stdout = oldStdout
		_ = stderrF.Close()
		os.Stderr = oldStderr
	}
	defer revertOS()

	// Run the main command.
	main()

	// Revert os.XXX so that test output is visible on failure.
	revertOS()

	// Capture any output and return it in a portable way (ex without windows newlines)
	stdoutB, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatal(err)
	}
	stdout = strings.ReplaceAll(string(stdoutB), "\r\n", "\n")

	stderrB, err := os.ReadFile(stderrPath)
	if err != nil {
		t.Fatal(err)
	}
	stderr = strings.ReplaceAll(string(stderrB), "\r\n", "\n")

	return
}
