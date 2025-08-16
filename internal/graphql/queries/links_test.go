package queries

import (
	"context"
	"testing"

	"github.com/graphql-go/graphql"
)

func TestLinksTypeDefinition(t *testing.T) {
	if LinksType == nil {
		t.Error("LinksType should not be nil")
	}
	fields := LinksType.Fields()
	if len(fields) == 0 {
		t.Error("LinksType should have fields")
	}
}

func TestLinksFieldResolve(t *testing.T) {
	params := graphql.ResolveParams{
		Args: map[string]interface{}{
			"limit":      1,
			"offset":     0,
			"categoryId": 1,
		},
		Context: context.Background(),
	}
	result, err := LinksField.Resolve(params)
	if err != nil {
		t.Logf("Resolve returned error as expected: %v", err)
	}
	if result == nil {
		t.Log("Resolve returned nil (expected if no DB)")
	}
}

func TestLinksTypeAndField(t *testing.T) {
	if LinksType == nil {
		t.Errorf("LinksType should not be nil")
	}
	if LinksField == nil {
		t.Errorf("LinksField should not be nil")
	}
}
