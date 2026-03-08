package sync

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	clientpkg "github.com/dhth/hours/internal/client"
	pers "github.com/dhth/hours/internal/persistence"
	syncpkg "github.com/dhth/hours/internal/sync"
	"github.com/dhth/hours/internal/types"
	"github.com/dhth/hours/tests/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestHoursServerBinarySupportsSeedBootstrapAcrossClients(t *testing.T) {
	fx, err := cli.NewServerFixture()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, fx.Cleanup()) })

	serverDBPath := filepath.Join(fx.TempDir(), "server.db")
	serverURL := startSyncServer(t, fx.BinPath(), serverDBPath)
	require.FileExists(t, serverDBPath)

	clientADB := newSyncClientDB(t, filepath.Join(fx.TempDir(), "client-a.db"))
	defer clientADB.Close()
	clientBDB := newSyncClientDB(t, filepath.Join(fx.TempDir(), "client-b.db"))
	defer clientBDB.Close()

	taskID, err := pers.InsertTask(clientADB, "cli bootstrap task")
	require.NoError(t, err)

	comment := "seed work"
	beginTS := time.Date(2026, time.March, 4, 9, 0, 0, 0, time.UTC)
	endTS := beginTS.Add(90 * time.Minute)
	_, err = pers.InsertManualTL(clientADB, taskID, beginTS, endTS, &comment)
	require.NoError(t, err)

	require.NoError(t, clientpkg.RunOnce(context.Background(), clientADB, serverURL))
	require.NoError(t, clientpkg.RunOnce(context.Background(), clientBDB, serverURL))
	assertSyncTotals(t, clientADB, 1, 5400)
	assertSyncTotals(t, clientBDB, 1, 5400)

	clientBTask := fetchOnlySyncTask(t, clientBDB)
	assert.Equal(t, "cli bootstrap task", clientBTask.Summary)

	secondBeginTS := endTS.Add(15 * time.Minute)
	secondEndTS := secondBeginTS.Add(30 * time.Minute)
	_, err = pers.InsertManualTL(clientBDB, clientBTask.LocalID, secondBeginTS, secondEndTS, nil)
	require.NoError(t, err)

	require.NoError(t, clientpkg.RunOnce(context.Background(), clientBDB, serverURL))
	require.NoError(t, clientpkg.RunOnce(context.Background(), clientADB, serverURL))
	assertSyncTotals(t, clientADB, 2, 7200)
	assertSyncTotals(t, clientBDB, 2, 7200)
}

func TestHoursServerBinaryProvidesHelp(t *testing.T) {
	fx, err := cli.NewServerFixture()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, fx.Cleanup()) })

	output, err := fx.RunCmd(cli.NewCmd([]string{"--help"}))
	require.NoError(t, err)
	assert.Contains(t, output, "success: true")
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "hours-server")
	assert.Contains(t, output, "Run the hours HTTP sync server")
}

func startSyncServer(t *testing.T, binPath string, serverDBPath string) string {
	t.Helper()
	listenAddr := reserveListenAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, binPath, "--dbpath", serverDBPath, "--listen", listenAddr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		cancel()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		waitErr := cmd.Wait()
		if waitErr != nil && !errors.Is(waitErr, context.Canceled) {
			var exitErr *exec.ExitError
			if !errors.As(waitErr, &exitErr) || exitErr.ExitCode() != -1 {
				t.Logf("sync server stdout:\n%s", stdout.String())
				t.Logf("sync server stderr:\n%s", stderr.String())
			}
		}
	})

	serverURL := "http://" + listenAddr
	require.Eventually(t, func() bool {
		resp, err := http.Get(serverURL + syncpkg.HealthPath)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 50*time.Millisecond, "sync server did not become healthy\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())

	return serverURL
}

func reserveListenAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	return listener.Addr().String()
}

func newSyncClientDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	require.NoError(t, pers.InitDB(db))
	require.NoError(t, pers.UpgradeDB(db, 1))
	return db
}

func fetchOnlySyncTask(t *testing.T, db *sql.DB) types.SyncTaskRecord {
	t.Helper()
	tasks, err := pers.FetchSyncTasks(db)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	return tasks[0]
}

func assertSyncTotals(t *testing.T, db *sql.DB, expectedLogCount int, expectedSecsSpent int) {
	t.Helper()
	task := fetchOnlySyncTask(t, db)
	assert.Equal(t, expectedSecsSpent, task.SecsSpent)
	logs, err := pers.FetchSyncTaskLogs(db)
	require.NoError(t, err)
	require.Len(t, logs, expectedLogCount)
}
