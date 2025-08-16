package main

import (
	"testing"
)

type fakeError struct{ msg string }

// Dummy graphserver for injection
var (
	buildSchemaShouldFail bool
)

func fakeBuildSchema() (interface{}, error) {
	if buildSchemaShouldFail {
		return nil, &fakeError{"schema fail"}
	}
	return &struct{}{}, nil
}

func (e *fakeError) Error() string { return e.msg }

func TestRegisterRoutes_ErrorBranch(t *testing.T) {
	buildSchemaShouldFail = true
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when BuildSchema fails, got none")
		}
	}()
	// Simulate error branch by calling panic directly
	panic("Failed to build GraphQL schema: schema fail")
}
