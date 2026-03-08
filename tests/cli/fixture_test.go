package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepoRootDirReturnsModuleRoot(t *testing.T) {
	repoRoot, err := repoRootDir()
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(repoRoot, "go.mod"))
	require.DirExists(t, filepath.Join(repoRoot, "cmd", "hours"))
	require.DirExists(t, filepath.Join(repoRoot, "cmd", "hours-server"))
}
