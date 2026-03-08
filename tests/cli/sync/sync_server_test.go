package sync

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
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

const syncServerStartupAttempts = 5

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
	var lastErr error
	for attempt := 1; attempt <= syncServerStartupAttempts; attempt++ {
		serverURL, retryable, err := startSyncServerAttempt(t, binPath, serverDBPath)
		if err == nil {
			return serverURL
		}

		lastErr = err
		if !retryable {
			require.NoError(t, err)
		}

		t.Logf("retrying sync server startup after listen race (%d/%d): %v", attempt, syncServerStartupAttempts, err)
	}

	require.NoError(t, lastErr)
	return ""
}

func reserveListenAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	return listener.Addr().String()
}

func startSyncServerAttempt(t *testing.T, binPath string, serverDBPath string) (string, bool, error) {
	t.Helper()

	listenAddr := reserveListenAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, binPath, "--dbpath", serverDBPath, "--listen", listenAddr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		cancel()
		return "", false, err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	serverURL := "http://" + listenAddr
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case waitErr := <-done:
			cancel()
			if isListenAddrInUse(waitErr, stderr.String()) {
				return "", true, fmt.Errorf("listen address %s was claimed before the child bound it", listenAddr)
			}
			return "", false, fmt.Errorf("sync server exited before becoming healthy: %w\nstdout:\n%s\nstderr:\n%s", waitErr, stdout.String(), stderr.String())
		case <-ticker.C:
			resp, err := http.Get(serverURL + syncpkg.HealthPath)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				t.Cleanup(func() {
					cancel()
					if cmd.Process != nil {
						_ = cmd.Process.Kill()
					}
					waitErr := <-done
					if waitErr != nil && !errors.Is(waitErr, context.Canceled) {
						var exitErr *exec.ExitError
						if !errors.As(waitErr, &exitErr) || exitErr.ExitCode() != -1 {
							t.Logf("sync server stdout:\n%s", stdout.String())
							t.Logf("sync server stderr:\n%s", stderr.String())
						}
					}
				})
				return serverURL, false, nil
			}
		case <-timeout.C:
			cancel()
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			waitErr := <-done
			if isListenAddrInUse(waitErr, stderr.String()) {
				return "", true, fmt.Errorf("listen address %s was claimed before the child bound it", listenAddr)
			}
			return "", false, fmt.Errorf("sync server did not become healthy\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
		}
	}
}

func isListenAddrInUse(waitErr error, stderr string) bool {
	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		return false
	}

	return strings.Contains(strings.ToLower(stderr), "address already in use")
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
