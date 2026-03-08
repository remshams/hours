package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncConfigValidate(t *testing.T) {
	testCases := []struct {
		name       string
		config     SyncConfig
		errSnippet string
	}{
		{
			name: "valid disabled config",
			config: SyncConfig{
				Enabled:  false,
				Interval: "15m",
			},
		},
		{
			name: "enabled requires server url",
			config: SyncConfig{
				Enabled:  true,
				Interval: "15m",
			},
			errSnippet: "sync server URL is required",
		},
		{
			name: "invalid scheme rejected",
			config: SyncConfig{
				Enabled:   true,
				ServerURL: "ftp://sync.example.com",
				Interval:  "15m",
			},
			errSnippet: "must use http or https",
		},
		{
			name: "interval must be at least a minute",
			config: SyncConfig{
				Enabled:   true,
				ServerURL: "https://sync.example.com",
				Interval:  "30s",
			},
			errSnippet: "must be at least 1m0s",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.errSnippet == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errSnippet)
		})
	}
}
