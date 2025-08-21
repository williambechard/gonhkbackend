package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ...existing code...

func TestGraphQLHandler_Success(t *testing.T) {
	cfg := SupabaseConfig{Url: "http://example.com", Key: "key"}
	handler := GraphQLHandler(cfg)
	query := `{}`
	body, _ := json.Marshal(map[string]interface{}{"query": query})
	req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestGraphQLHandler_BadRequest(t *testing.T) {
	cfg := SupabaseConfig{Url: "http://example.com", Key: "key"}
	handler := GraphQLHandler(cfg)
	req := httptest.NewRequest("POST", "/graphql", bytes.NewReader([]byte("not-json")))
	w := httptest.NewRecorder()

	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestGraphQLHandler_ErrorResponse(t *testing.T) {
	cfg := SupabaseConfig{Url: "http://example.com", Key: "key"}
	handler := GraphQLHandler(cfg)
	query := `{ fail }`
	body, _ := json.Marshal(map[string]interface{}{"query": query})
	req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}
