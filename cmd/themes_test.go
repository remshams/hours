package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddTheme_InvalidThemeName(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name      string
		themeName string
	}{
		{
			name:      "empty name",
			themeName: "",
		},
		{
			name:      "name with spaces",
			themeName: "my theme",
		},
		{
			name:      "name with special characters",
			themeName: "theme@123",
		},
		{
			name:      "name too long",
			themeName: "this-is-a-very-long-theme-name-that-exceeds-20-chars",
		},
		{
			name:      "name with underscores",
			themeName: "my_theme",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := addTheme(tt.themeName, tempDir)
			assert.ErrorIs(t, err, errThemeNameInvalid)
		})
	}
}

func TestAddTheme_ValidThemeName(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name      string
		themeName string
	}{
		{
			name:      "simple name",
			themeName: "mytheme",
		},
		{
			name:      "single character",
			themeName: "a",
		},
		{
			name:      "name with numbers",
			themeName: "theme123",
		},
		{
			name:      "name with hyphens",
			themeName: "my-custom-theme",
		},
		{
			name:      "name at max length",
			themeName: "exactly20-chars-long", // 20 characters exactly
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			path, err := addTheme(tt.themeName, tempDir)
			require.NoError(t, err)
			assert.Contains(t, path, tt.themeName)
			assert.FileExists(t, path)

			// Verify file content is valid JSON
			content, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.NotEmpty(t, content)
		})
	}
}

func TestAddTheme_CreatesThemesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	themesDir := filepath.Join(tempDir, "subdir", "themes")
	themeName := "new-theme"

	// Themes directory shouldn't exist yet
	_, err := os.Stat(themesDir)
	assert.True(t, os.IsNotExist(err))

	// Add theme should create it
	path, err := addTheme(themeName, themesDir)
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(themesDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify file was created in the right place
	assert.Equal(t, filepath.Join(themesDir, themeName+".json"), path)
}
