package config

import (
	"log"
	"os"
)

var (
	SupabaseUrl string
	SupabaseKey string
)

func InitSupabase() {
	SupabaseUrl = os.Getenv("SUPABASE_URL")
	SupabaseKey = os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	maskedUrl := "(not set)"
	if SupabaseUrl != "" {
		maskedUrl = "XXXXXXX"
	}
	maskedKey := "(not set)"
	if SupabaseKey != "" {
		maskedKey = "XXXXXXX"
	}
	log.Printf("[InitSupabase] SUPABASE_URL: %s, SUPABASE_SERVICE_ROLE_KEY: %s", maskedUrl, maskedKey)
}
