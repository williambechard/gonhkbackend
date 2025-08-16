package db

import (
	"net/http"
	linksdb "nhknewseasybkend/internal/db/links"
	"testing"
)

func TestGetCategoryById(t *testing.T) {
	cat, err := GetCategoryById(1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if cat == nil || cat.ID != 1 {
		t.Errorf("Expected category ID 1, got %v", cat)
	}
}

func TestGetLinks(t *testing.T) {
	// Use a mock client to avoid real network requests
	localClient := &http.Client{Transport: new(linksdb.MockRoundTripper)}
	_, err := linksdb.GetLinks(localClient, 1, 0, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	// Should be empty array, but test for no error
}
