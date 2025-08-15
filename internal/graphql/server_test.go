package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"nhknewseasybkend/internal/util"

	gql "github.com/graphql-go/graphql"
)

// mockLogger is a stub for util.LogToFile
var loggedErrors []string

func mockLog(msg string, logType util.LogType) {
	// You can implement a mock for Log here if needed
}

func TestGraphQLHandler_Success(t *testing.T) {
	// Minimal root query for testing
	rootQuery := gql.NewObject(gql.ObjectConfig{
		Name: "Query",
		Fields: gql.Fields{
			"hello": &gql.Field{
				Type: gql.String,
				Resolve: func(p gql.ResolveParams) (interface{}, error) {
					return "world", nil
				},
			},
		},
	})

	schema, err := gql.NewSchema(gql.SchemaConfig{Query: rootQuery})
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	handler := GraphQLHandler(&schema)
	query := `{ hello }`
	body, _ := json.Marshal(map[string]interface{}{"query": query})
	req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if data, ok := result["data"].(map[string]interface{}); !ok || data["hello"] != "world" {
		t.Errorf("Expected hello=world, got %v", result)
	}
}

func TestGraphQLHandler_BadRequest(t *testing.T) {
	rootQuery := gql.NewObject(gql.ObjectConfig{
		Name:   "Query",
		Fields: gql.Fields{},
	})
	schema, _ := gql.NewSchema(gql.SchemaConfig{Query: rootQuery})
	handler := GraphQLHandler(&schema)
	req := httptest.NewRequest("POST", "/graphql", bytes.NewReader([]byte("not-json")))
	w := httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestGraphQLHandler_ErrorResponse(t *testing.T) {
	rootQuery := gql.NewObject(gql.ObjectConfig{
		Name: "Query",
		Fields: gql.Fields{
			"fail": &gql.Field{
				Type: gql.String,
				Resolve: func(p gql.ResolveParams) (interface{}, error) {
					return nil, errors.New("forced error")
				},
			},
		},
	})
	schema, _ := gql.NewSchema(gql.SchemaConfig{Query: rootQuery})
	handler := GraphQLHandler(&schema)
	query := `{ fail }`
	body, _ := json.Marshal(map[string]interface{}{"query": query})
	req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Patch util.LogToFile for this test
	loggedErrors = nil
	// ...existing code...
	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if errs, ok := result["errors"].([]interface{}); !ok || len(errs) == 0 {
		t.Errorf("Expected error in response, got %v", result)
	}
}

func TestBuildSchema_Empty(t *testing.T) {
	_, err := BuildSchema()
	if err == nil {
		t.Errorf("Expected error when root query is nil, got nil")
	}
}
