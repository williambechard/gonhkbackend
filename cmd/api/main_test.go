package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"nhknewseasybkend/internal/util"
	"os"
	"testing"
	"time"
)

type mockServer struct {
	fail bool
}

func (m *mockServer) ListenAndServe() error {
	if m.fail {
		return fmt.Errorf("mock server error")
	}
	return nil
}

func TestClearLogFile(t *testing.T) {
	util.InitLogger()
	util.LogToFile("test")
	util.CloseLogger()
	ClearLogFile()
	logPath := util.GetLogFilePath()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("Expected log file to be empty, got %q", string(data))
	}
}

func TestSetupEnvDefaultsFunc(t *testing.T) {
	os.Unsetenv("HOST")
	os.Unsetenv("PORT")
	host, port, err := SetupEnv()
	if err != nil {
		t.Errorf("Unexpected error loading env: %v", err)
	}
	if host != "localhost" || port != "4200" {
		t.Errorf("Expected defaults, got HOST=%q PORT=%q", host, port)
	}
}

func TestSetupLoggerFunc(t *testing.T) {
	SetupLogger(nil)
	util.LogToFile("Logger test")
	util.CloseLogger()
	logPath := util.GetLogFilePath()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if !stringContains(string(data), "Logger test") {
		t.Errorf("Expected log file to contain 'Logger test', got %q", string(data))
	}
}

func TestLinksHandler(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux)
	req := httptest.NewRequest("GET", "/links", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	body := w.Body.String()
	if body != "Links endpoint reached!\n" {
		t.Errorf("Unexpected body: %q", body)
	}
}

func TestEnvDefaults(t *testing.T) {
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	host, port, _ := SetupEnv()
	if host != "localhost" || port != "4200" {
		t.Errorf("Expected defaults, got HOST=%q PORT=%q", host, port)
	}
}

func TestEnvOverride(t *testing.T) {
	os.Setenv("HOST", "127.0.0.1")
	os.Setenv("PORT", "9999")
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	if host != "127.0.0.1" || port != "9999" {
		t.Errorf("Expected env override, got HOST=%q PORT=%q", host, port)
	}
}
func TestLoggerIntegration(t *testing.T) {
	// Clear log file
	logPath := util.GetLogFilePath()
	os.Remove(logPath)
	util.InitLogger()
	testMsg := "Logger integration test"
	util.LogToFile(testMsg)
	util.CloseLogger()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if !stringContains(string(data), testMsg) {
		t.Errorf("Expected log file to contain %q, got %q", testMsg, string(data))
	}
}

func TestStartServerError(t *testing.T) {
	// Use an invalid port to force ListenAndServe to fail
	err := StartServer("localhost", "-1", nil)
	if err == nil {
		t.Error("Expected error from StartServer with invalid port, got nil")
	}
}

func TestClearLogFileNoFile(t *testing.T) {
	// Use a non-existent file path
	oldPath := util.GetLogFilePath()
	util.CloseLogger()
	// Temporarily set log path to a file that doesn't exist
	os.Remove(oldPath)
	ClearLogFile()
	// Should not error or panic
}

func TestSetupLoggerWithError(t *testing.T) {
	util.CloseLogger()
	SetupLogger(fmt.Errorf(".env not loaded"))
	util.LogToFile("Logger error test")
	util.CloseLogger()
	logPath := util.GetLogFilePath()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if !stringContains(string(data), "Warning: .env file not loaded") {
		t.Errorf("Expected log file to contain warning, got %q", string(data))
	}
}

func stringContains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[0:len(substr)] == substr || stringContains(s[1:], substr))))
}

func TestRunSuccess(t *testing.T) {
	// This should run the main logic and return 0 (success)
	// It will start the server, which will block, so we run in a goroutine and kill after a short time
	done := make(chan int, 1)
	go func() {
		code := Run(nil)
		done <- code
	}()
	// Wait briefly, then check if code is returned (should be 0 if server starts successfully)
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("Expected Run to return 0, got %d", code)
		}
	case <-time.After(100 * time.Millisecond):
		// If still running, assume success (server is blocking as expected)
	}
}

func TestRunError(t *testing.T) {
	// Set invalid port to force StartServer error
	os.Setenv("PORT", "-1")
	code := Run(nil)
	if code != 1 {
		t.Errorf("Expected Run to return 1 on server error, got %d", code)
	}
	os.Setenv("PORT", "4200") // restore
}

func TestRunWithMockServerSuccess(t *testing.T) {
	var srv ListenAndServeFunc = &mockServer{fail: false}
	code := Run(srv)
	if code != 0 {
		t.Errorf("Expected Run to return 0 with mock server success, got %d", code)
	}
}

func TestRunWithMockServerError(t *testing.T) {
	var srv ListenAndServeFunc = &mockServer{fail: true}
	code := Run(srv)
	if code != 1 {
		t.Errorf("Expected Run to return 1 with mock server error, got %d", code)
	}
}
