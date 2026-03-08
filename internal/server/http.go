package server

import (
	"bytes"
	"database/sql"
	"errors"
	"net/http"
	"time"

	pers "github.com/dhth/hours/internal/persistence"
	syncpkg "github.com/dhth/hours/internal/sync"
)

var encodePayload = syncpkg.EncodePayload

func NewHandler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(syncpkg.HealthPath, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc(syncpkg.SyncEndpointPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		request, err := syncpkg.DecodePayload(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := pers.ApplySyncBundle(db, request.Tasks, request.TaskLogs); err != nil {
			statusCode := http.StatusInternalServerError
			if errors.Is(err, pers.ErrConflictingActiveSyncTaskLog) {
				statusCode = http.StatusConflict
			}
			http.Error(w, err.Error(), statusCode)
			return
		}

		response, err := buildSyncPayload(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var responseBody bytes.Buffer
		if err := encodePayload(&responseBody, response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(responseBody.Bytes())
	})

	return mux
}

func ListenAndServe(addr string, db *sql.DB) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           NewHandler(db),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return server.ListenAndServe()
}

func buildSyncPayload(db *sql.DB) (syncpkg.Payload, error) {
	tasks, err := pers.FetchSyncTasks(db)
	if err != nil {
		return syncpkg.Payload{}, err
	}

	taskLogs, err := pers.FetchSyncTaskLogs(db)
	if err != nil {
		return syncpkg.Payload{}, err
	}

	return syncpkg.Payload{Tasks: tasks, TaskLogs: taskLogs}, nil
}
