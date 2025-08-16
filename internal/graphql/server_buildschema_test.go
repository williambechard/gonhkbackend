package server

import "testing"

func TestBuildSchema(t *testing.T) {
	schema, err := BuildSchema()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Errorf("Expected schema, got nil")
	}
}
