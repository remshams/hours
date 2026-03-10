package server

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCommand_DefaultValues(t *testing.T) {
	cmd, err := NewRootCommand()
	require.NoError(t, err)

	assert.Equal(t, "hours-server", cmd.Use)
	assert.Contains(t, cmd.Short, "HTTP sync server")
	assert.Contains(t, cmd.Long, "does not start the TUI client")

	dbPath, err := cmd.Flags().GetString("dbpath")
	require.NoError(t, err)
	assert.NotEmpty(t, dbPath)
	assert.Equal(t, ".db", filepath.Ext(dbPath))

	listenAddr, err := cmd.Flags().GetString("listen")
	require.NoError(t, err)
	assert.Equal(t, defaultListenAddr, listenAddr)
}

func TestNewRootCommand_InvalidDBExtension(t *testing.T) {
	cmd, err := NewRootCommand()
	require.NoError(t, err)
	require.NoError(t, cmd.Flags().Set("dbpath", filepath.Join(t.TempDir(), "server.txt")))

	err = cmd.RunE(cmd, nil)
	assert.ErrorIs(t, err, ErrDBFileExtIncorrect)
}

func TestNewRootCommandWithHomeDirLookup_AllowsHelpAndFlagParsingWithoutHomeDir(t *testing.T) {
	cmd, err := newRootCommandWithHomeDirLookup(func() (string, error) {
		return "", errors.New("home directory unavailable")
	})
	require.NoError(t, err)

	dbPath, err := cmd.Flags().GetString("dbpath")
	require.NoError(t, err)
	assert.Equal(t, defaultDBName, dbPath)

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--dbpath", filepath.Join(t.TempDir(), "server.db"), "--help"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, output.String(), "hours-server")
	assert.Contains(t, output.String(), "--dbpath")
}
