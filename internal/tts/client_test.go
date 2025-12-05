package tts

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/harrison/conductor/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := config.TTSConfig{
		Enabled: true,
		BaseURL: "http://localhost:5005",
		Timeout: 2 * time.Second,
	}

	client := NewClient(cfg)

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.httpClient == nil {
		t.Fatal("expected non-nil http client")
	}
	if client.httpClient.Timeout != 2*time.Second {
		t.Errorf("expected timeout 2s, got %v", client.httpClient.Timeout)
	}
	if client.config.BaseURL != "http://localhost:5005" {
		t.Errorf("expected BaseURL http://localhost:5005, got %s", client.config.BaseURL)
	}
}

func TestClient_CheckHealth(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{
			name:       "200 OK returns true",
			statusCode: http.StatusOK,
			want:       true,
		},
		{
			name:       "500 Internal Server Error returns false",
			statusCode: http.StatusInternalServerError,
			want:       false,
		},
		{
			name:       "503 Service Unavailable returns false",
			statusCode: http.StatusServiceUnavailable,
			want:       false,
		},
		{
			name:       "404 Not Found returns false",
			statusCode: http.StatusNotFound,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			cfg := config.TTSConfig{
				Enabled: true,
				BaseURL: server.URL,
				Timeout: 2 * time.Second,
			}
			client := NewClient(cfg)

			got := client.CheckHealth()
			if got != tt.want {
				t.Errorf("CheckHealth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_CheckHealth_ServerUnavailable(t *testing.T) {
	cfg := config.TTSConfig{
		Enabled: true,
		BaseURL: "http://localhost:59999", // Port that should be unavailable
		Timeout: 100 * time.Millisecond,
	}
	client := NewClient(cfg)

	got := client.CheckHealth()
	if got != false {
		t.Errorf("CheckHealth() = %v, want false for unavailable server", got)
	}
}

func TestClient_IsAvailable(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		statusCode int
		want       bool
	}{
		{
			name:       "enabled with healthy server returns true",
			enabled:    true,
			statusCode: http.StatusOK,
			want:       true,
		},
		{
			name:       "enabled with unhealthy server returns false",
			enabled:    true,
			statusCode: http.StatusInternalServerError,
			want:       false,
		},
		{
			name:       "disabled returns false immediately",
			enabled:    false,
			statusCode: http.StatusOK,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			cfg := config.TTSConfig{
				Enabled: tt.enabled,
				BaseURL: server.URL,
				Timeout: 2 * time.Second,
			}
			client := NewClient(cfg)

			got := client.IsAvailable()
			if got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_IsAvailable_CachesResult(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.TTSConfig{
		Enabled: true,
		BaseURL: server.URL,
		Timeout: 2 * time.Second,
	}
	client := NewClient(cfg)

	// Call IsAvailable multiple times
	result1 := client.IsAvailable()
	result2 := client.IsAvailable()
	result3 := client.IsAvailable()

	// All results should be the same
	if !result1 || !result2 || !result3 {
		t.Errorf("expected all IsAvailable() calls to return true")
	}

	// Server should only be called once due to sync.Once caching
	if callCount != 1 {
		t.Errorf("expected server to be called once, got %d calls", callCount)
	}
}

func TestClient_IsAvailable_DisabledSkipsHealthCheck(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.TTSConfig{
		Enabled: false, // Disabled
		BaseURL: server.URL,
		Timeout: 2 * time.Second,
	}
	client := NewClient(cfg)

	// Call IsAvailable - should return false without contacting server
	result := client.IsAvailable()

	if result {
		t.Error("expected IsAvailable() to return false when disabled")
	}

	// Server should NOT be called when TTS is disabled
	if callCount != 0 {
		t.Errorf("expected server not to be called when disabled, got %d calls", callCount)
	}
}

func TestClient_Config(t *testing.T) {
	cfg := config.TTSConfig{
		Enabled: true,
		BaseURL: "http://localhost:5005",
		Model:   "orpheus",
		Voice:   "tara",
		Timeout: 2 * time.Second,
	}
	client := NewClient(cfg)

	got := client.Config()
	if got != cfg {
		t.Errorf("Config() = %v, want %v", got, cfg)
	}
}

func TestClient_Speak_WhenNotAvailable(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.TTSConfig{
		Enabled: false, // Disabled, so not available
		BaseURL: server.URL,
		Model:   "orpheus",
		Voice:   "tara",
		Timeout: 2 * time.Second,
	}
	client := NewClient(cfg)

	// Speak should return immediately without making any requests
	client.Speak("Hello world")

	// Give goroutine time to run (if it would)
	time.Sleep(50 * time.Millisecond)

	if callCount != 0 {
		t.Errorf("expected no server calls when TTS disabled, got %d", callCount)
	}
}

func TestClient_Speak_SendsCorrectRequest(t *testing.T) {
	var receivedReq speechRequest
	var receivedPath string
	var receivedMethod string
	var receivedContentType string
	var wg sync.WaitGroup
	wg.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// Health check
			w.WriteHeader(http.StatusOK)
			return
		}
		// Speech request
		receivedPath = r.URL.Path
		receivedMethod = r.Method
		receivedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedReq)
		// Return fake audio data (minimal WAV header)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("RIFF----WAVEfmt "))
		wg.Done()
	}))
	defer server.Close()

	cfg := config.TTSConfig{
		Enabled: true,
		BaseURL: server.URL,
		Model:   "orpheus",
		Voice:   "tara",
		Timeout: 2 * time.Second,
	}
	client := NewClient(cfg)

	client.Speak("Hello world")

	// Wait for goroutine to complete
	wg.Wait()

	if receivedPath != "/v1/audio/speech" {
		t.Errorf("expected path /v1/audio/speech, got %s", receivedPath)
	}
	if receivedMethod != http.MethodPost {
		t.Errorf("expected method POST, got %s", receivedMethod)
	}
	if receivedContentType != "application/json" {
		t.Errorf("expected content-type application/json, got %s", receivedContentType)
	}
	if receivedReq.Model != "orpheus" {
		t.Errorf("expected model orpheus, got %s", receivedReq.Model)
	}
	if receivedReq.Input != "Hello world" {
		t.Errorf("expected input 'Hello world', got %s", receivedReq.Input)
	}
	if receivedReq.Voice != "tara" {
		t.Errorf("expected voice tara, got %s", receivedReq.Voice)
	}
	if receivedReq.ResponseFormat != "wav" {
		t.Errorf("expected response_format wav, got %s", receivedReq.ResponseFormat)
	}
	if receivedReq.Speed != 1.0 {
		t.Errorf("expected speed 1.0, got %f", receivedReq.Speed)
	}
}

func TestClient_Speak_NonBlocking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// Health check - respond quickly
			w.WriteHeader(http.StatusOK)
			return
		}
		// Speech request - delay 500ms
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.TTSConfig{
		Enabled: true,
		BaseURL: server.URL,
		Model:   "orpheus",
		Voice:   "tara",
		Timeout: 2 * time.Second,
	}
	client := NewClient(cfg)

	// Force health check first
	_ = client.IsAvailable()

	// Speak should return in <50ms even with 500ms server delay
	start := time.Now()
	client.Speak("Hello world")
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Errorf("Speak() took %v, expected <50ms (should be non-blocking)", elapsed)
	}
}

func TestClient_Speak_SilentlyHandlesErrors(t *testing.T) {
	// Server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := config.TTSConfig{
		Enabled: true,
		BaseURL: server.URL,
		Model:   "orpheus",
		Voice:   "tara",
		Timeout: 2 * time.Second,
	}
	client := NewClient(cfg)

	// This should not panic or cause any issues
	client.Speak("Hello world")

	// Give goroutine time to complete
	time.Sleep(50 * time.Millisecond)
}

func TestClient_Speak_HandlesUnavailableServer(t *testing.T) {
	// First create a server for health check, then close it
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cfg := config.TTSConfig{
		Enabled: true,
		BaseURL: healthServer.URL,
		Model:   "orpheus",
		Voice:   "tara",
		Timeout: 100 * time.Millisecond,
	}
	client := NewClient(cfg)

	// Force health check while server is up
	_ = client.IsAvailable()

	// Close server to simulate unavailability during Speak
	healthServer.Close()

	// This should not panic - errors are silently ignored
	client.Speak("Hello world")

	// Give goroutine time to attempt and fail
	time.Sleep(200 * time.Millisecond)
}
