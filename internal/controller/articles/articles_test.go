package articles

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_GetAllArticles_Handler(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/articles/getAll", nil)
	GetAllArticles_Handler(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	body := w.Body.String()
	if body != "Articles endpoint reached!\n" {
		t.Errorf("Unexpected body: %q", body)
	}
}
