package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/dhth/hours/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	defer func() {
		os.Stderr = oldStderr
		_ = w.Close()
		_ = r.Close()
	}()

	fn()

	// Close the write end to signal EOF to the reader
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	return buf.String()
}

func TestHandleError_CouldntGenerateData(t *testing.T) {
	err := errCouldntGenerateData

	output := captureStderr(t, func() {
		handleError(err)
	})

	assert.Contains(t, output, "This isn't supposed to happen")
	assert.Contains(t, output, "let @dhth know about this error")
}

func TestHandleError_BuiltInThemeDoesntExist(t *testing.T) {
	err := theme.ErrBuiltInThemeDoesntExist

	output := captureStderr(t, func() {
		handleError(err)
	})

	assert.Contains(t, output, "custom:")
	assert.Contains(t, output, "hours themes list")
}

func TestHandleError_CustomThemeDoesntExist(t *testing.T) {
	err := theme.ErrCustomThemeDoesntExist

	output := captureStderr(t, func() {
		handleError(err)
	})

	assert.Contains(t, output, "hours themes list")
}

func TestHandleError_ThemeFileHasInvalidSchema(t *testing.T) {
	err := theme.ErrThemeFileHasInvalidSchema

	output := captureStderr(t, func() {
		handleError(err)
	})

	assert.Contains(t, output, "A valid theme file looks like this")
	assert.Contains(t, output, "activeTask")
	assert.Contains(t, output, "inactiveTask")
}

func TestHandleError_ThemeColorsAreInvalid(t *testing.T) {
	err := theme.ErrThemeColorsAreInvalid

	output := captureStderr(t, func() {
		handleError(err)
	})

	assert.Contains(t, output, "ANSI 16")
	assert.Contains(t, output, "ANSI 256")
	assert.Contains(t, output, "HEX")
	assert.Contains(t, output, "16,777,216")
}

func TestHandleError_UnhandledError(t *testing.T) {
	// An error that doesn't match any handled type should produce no output
	err := errors.New("some random error")

	output := captureStderr(t, func() {
		handleError(err)
	})

	assert.Empty(t, output)
}
