package server

import (
	"bytes"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	pers "github.com/dhth/hours/internal/persistence"
	syncpkg "github.com/dhth/hours/internal/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestNewHandler_EncodeFailureDoesNotWritePartialResponse(t *testing.T) {
	db := newServerTestDB(t)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	handler := handlerWithEncoder(db, func(w io.Writer, _ syncpkg.Payload) error {
		_, _ = io.WriteString(w, `{"partial":true}`)
		return errors.New("encode failure")
	})

	req := httptest.NewRequest(http.MethodPost, syncpkg.SyncEndpointPath, bytes.NewBufferString(`{"tasks":[],"taskLogs":[]}`))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.NotContains(t, resp.Body.String(), `{"partial":true}`)
	assert.Contains(t, resp.Body.String(), "encode failure")
	assert.Contains(t, resp.Header().Get("Content-Type"), "text/plain")
}

func newServerTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	require.NoError(t, pers.InitDB(db))
	require.NoError(t, pers.UpgradeDB(db, 1))

	return db
}
