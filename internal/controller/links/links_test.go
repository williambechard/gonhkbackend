package links

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_GetAllLinks_Handler(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/article-links/getAll", nil)
	w := httptest.NewRecorder()
	GetAllLinks_Handler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	body := w.Body.String()
	if body != "Links endpoint reached!\n" {
		t.Errorf("Unexpected body: %q", body)
	}
}
