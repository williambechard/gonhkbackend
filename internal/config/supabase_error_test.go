package config

import (
	"os"
	"testing"
)

func TestInitSupabase_WithEnv(t *testing.T) {
	os.Setenv("SUPABASE_URL", "http://example.com")
	os.Setenv("SUPABASE_SERVICE_ROLE_KEY", "key")
	InitSupabase()
	// No error expected, just coverage for env set branch
}
