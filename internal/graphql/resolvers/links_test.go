package resolvers

import (
	"context"
	"testing"
)

func TestGetLinks(t *testing.T) {
	_, err := GetLinks(context.Background(), nil, 1, 0)
	if err == nil {
		t.Log("GetLinks ran (expected error due to config)")
	}
}
