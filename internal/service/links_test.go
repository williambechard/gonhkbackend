package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLinksHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/links", nil)
	w := httptest.NewRecorder()
	LinksHandler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	body := w.Body.String()
	if body != "Links endpoint reached!\n" {
		t.Errorf("Unexpected body: %q", body)
	}
}
