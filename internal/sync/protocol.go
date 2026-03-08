package sync

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/dhth/hours/internal/types"
)

const (
	DefaultInterval  = "15m"
	MinInterval      = time.Minute
	HealthPath       = "/healthz"
	SyncEndpointPath = "/v1/sync"
)

type Payload struct {
	Tasks    []types.SyncTaskRecord    `json:"tasks"`
	TaskLogs []types.SyncTaskLogRecord `json:"taskLogs"`
}

type Config struct {
	Enabled   bool   `json:"enabled"`
	ServerURL string `json:"serverURL,omitempty"`
	Interval  string `json:"interval"`
}

type configValidationError struct {
	issues []string
}

func (e configValidationError) Error() string {
	return strings.Join(e.issues, "; ")
}

func EncodePayload(w io.Writer, payload Payload) error {
	return json.NewEncoder(w).Encode(payload)
}

func DecodePayload(r io.Reader) (Payload, error) {
	var payload Payload
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return Payload{}, err
	}

	return payload, nil
}

func URL(serverURL string) string {
	return strings.TrimRight(strings.TrimSpace(serverURL), "/") + SyncEndpointPath
}

func DefaultConfig() Config {
	return Config{
		Enabled:  false,
		Interval: DefaultInterval,
	}
}

func (c Config) Sanitized() Config {
	c.ServerURL = strings.TrimSpace(c.ServerURL)
	c.Interval = strings.TrimSpace(c.Interval)
	return c
}

func (c Config) Validate() error {
	c = c.Sanitized()

	var issues []string
	if _, err := ParseInterval(c.Interval); err != nil {
		issues = append(issues, err.Error())
	}

	if c.ServerURL != "" {
		if err := ValidateServerURL(c.ServerURL); err != nil {
			issues = append(issues, err.Error())
		}
	}

	if c.Enabled && c.ServerURL == "" {
		issues = append(issues, "sync server URL is required when sync is enabled")
	}

	if len(issues) == 0 {
		return nil
	}

	return configValidationError{issues: issues}
}

func ValidateServerURL(serverURL string) error {
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("sync server URL must be a valid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("sync server URL must use http or https")
	}

	if parsed.Host == "" {
		return errors.New("sync server URL must include a host")
	}

	return nil
}

func ParseEnabled(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "on", "true", "yes", "1":
		return true, nil
	case "off", "false", "no", "0":
		return false, nil
	default:
		return false, errors.New("sync enabled must be one of: on/off, true/false, yes/no, 1/0")
	}
}

func FormatEnabled(enabled bool) string {
	if enabled {
		return "on"
	}

	return "off"
}

func ParseInterval(raw string) (time.Duration, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, errors.New("sync interval is required (for example: 15m or 1h)")
	}

	d, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("sync interval must be a valid duration like 15m or 1h: %w", err)
	}

	if d < MinInterval {
		return 0, fmt.Errorf("sync interval must be at least %s", MinInterval)
	}

	return d, nil
}
