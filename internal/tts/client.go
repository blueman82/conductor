// Package tts provides a client for text-to-speech functionality.
package tts

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/harrison/conductor/internal/config"
)

// speechRequest is the JSON body for TTS synthesis requests.
type speechRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
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
// The audio is fetched from the TTS server, saved to a temp file, and played using
// the system's audio player (afplay on macOS, aplay on Linux).
// This method never blocks - playback happens in a goroutine.
// All errors are silently ignored.
func (c *Client) Speak(text string) {
	if !c.IsAvailable() {
		return
	}

	go func() {
		reqBody := speechRequest{
			Model:          c.config.Model,
			Input:          text,
			Voice:          c.config.Voice,
			ResponseFormat: "wav",
			Speed:          1.0,
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
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return
		}

		// Read audio bytes from response
		audioData, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		// Play the audio
		playAudio(audioData)
	}()
}

// playAudio writes audio bytes to a temp file and plays it using the system audio player.
// On macOS, uses afplay. On Linux, uses aplay.
// The temp file is cleaned up after playback completes.
func playAudio(audioData []byte) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "conductor-tts-*.wav")
	if err != nil {
		return
	}
	tmpPath := tmpFile.Name()

	// Write audio data
	_, err = tmpFile.Write(audioData)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return
	}

	// Determine audio player based on OS
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("afplay", tmpPath)
	case "linux":
		cmd = exec.Command("aplay", "-q", tmpPath)
	default:
		// Unsupported platform, clean up and return
		os.Remove(tmpPath)
		return
	}

	// Play audio (blocking within this goroutine)
	cmd.Run()

	// Clean up temp file
	os.Remove(tmpPath)
}
