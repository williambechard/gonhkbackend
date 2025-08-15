package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterRoutes_RoutesHandler(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/article-links/getAll", nil)
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

func TestRegisterRoutes_ArticlesHandler(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/articles/getAll", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	body := w.Body.String()
	if body != "Articles endpoint reached!\n" {
		t.Errorf("Unexpected body: %q", body)
	}
}
