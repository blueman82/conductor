// Package tts provides a client for text-to-speech functionality.
package tts

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/harrison/conductor/internal/config"
)

// speechRequest is the JSON body for TTS synthesis requests.
type speechRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
	Voice string `json:"voice"`
}

// Client provides TTS functionality with lazy health checking.
type Client struct {
	config     config.TTSConfig
	httpClient *http.Client
	available  bool
	once       sync.Once
}

// NewClient creates a new TTS client with the given configuration.
// The HTTP client timeout is set from the config.
func NewClient(cfg config.TTSConfig) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// CheckHealth performs a health check against the TTS server.
// Returns true if the server responds with HTTP 200 OK.
func (c *Client) CheckHealth() bool {
	resp, err := c.httpClient.Get(c.config.BaseURL + "/")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// IsAvailable returns whether the TTS service is available.
// If TTS is disabled in config, returns false immediately.
// Otherwise, performs a lazy health check using sync.Once (only checks once, caches result).
func (c *Client) IsAvailable() bool {
	// Short-circuit if TTS is disabled
	if !c.config.Enabled {
		return false
	}

	// Perform health check once and cache result
	c.once.Do(func() {
		c.available = c.CheckHealth()
	})

	return c.available
}

// Config returns the client's configuration.
func (c *Client) Config() config.TTSConfig {
	return c.config
}

// Speak sends text to the TTS service for synthesis in a fire-and-forget manner.
// If the service is not available, it returns immediately without doing anything.
// The actual HTTP request is made in a goroutine, so this method never blocks.
// All errors are silently ignored.
func (c *Client) Speak(text string) {
	if !c.IsAvailable() {
		return
	}

	go func() {
		reqBody := speechRequest{
			Model: c.config.Model,
			Input: text,
			Voice: c.config.Voice,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			return
		}

		req, err := http.NewRequest(http.MethodPost, c.config.BaseURL+"/v1/audio/speech", bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}
