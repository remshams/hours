package server

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCommand_DefaultValues(t *testing.T) {
	cmd, err := NewRootCommand()
	require.NoError(t, err)

	assert.Equal(t, "hours-server", cmd.Use)
	assert.Equal(t, "Run the hours HTTP sync server", cmd.Short)

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
