package client

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"

	pers "github.com/dhth/hours/internal/persistence"
	syncpkg "github.com/dhth/hours/internal/sync"
)

func RunOnce(ctx context.Context, db *sql.DB, serverURL string) error {
	tasks, err := pers.FetchSyncTasks(db)
	if err != nil {
		return err
	}

	taskLogs, err := pers.FetchSyncTaskLogs(db)
	if err != nil {
		return err
	}

	requestBody := syncpkg.Payload{Tasks: tasks, TaskLogs: taskLogs}
	var body bytes.Buffer
	if err := syncpkg.EncodePayload(&body, requestBody); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, syncpkg.URL(serverURL), &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.TrimSpace(string(payload))
		if message == "" {
			message = resp.Status
		}
		return fmt.Errorf("sync request failed: %s", message)
	}

	response, err := syncpkg.DecodePayload(resp.Body)
	if err != nil {
		return err
	}

	return pers.ApplySyncBundle(db, response.Tasks, response.TaskLogs)
}
