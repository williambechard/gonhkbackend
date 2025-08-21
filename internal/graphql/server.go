package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"nhknewseasybkend/internal/util"
)

var (
	SupabaseUrl string
	SupabaseKey string
)

type SupabaseConfig struct {
	Url string
	Key string
}

func InitSupabase() SupabaseConfig {
	url := os.Getenv("SUPABASE_URL")
	key := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	maskedUrl := "(not set)"
	if url != "" {
		maskedUrl = "XXXXXXX"
	}
	maskedKey := "(not set)"
	if key != "" {
		maskedKey = "XXXXXXX"
	}
	fmt.Printf("[InitSupabase] SUPABASE_URL: %s, SUPABASE_SERVICE_ROLE_KEY: %s\n", maskedUrl, maskedKey)
	return SupabaseConfig{Url: url, Key: key}
}

// Minimal GraphQL proxy handler
func GraphQLHandler(cfg SupabaseConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all requests (must be first)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, apikey")

		if r.Method == http.MethodOptions {
			util.Log(fmt.Sprintf("[GraphQLHandler] OPTIONS request from %s. Headers: %v", r.RemoteAddr, r.Header), util.LogTypeLog)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var payload map[string]interface{}
		const maxBodySize = 1 << 20 // 1MB
		limitedReader := io.LimitReader(r.Body, maxBodySize)
		decoder := json.NewDecoder(limitedReader)
		if err := decoder.Decode(&payload); err != nil {
			util.Log(fmt.Sprintf("[GraphQLHandler] Error decoding JSON payload: %v", err), util.LogTypeError)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON payload"})
			return
		}
		var opType string = "query"
		var opName string
		var queryStr string
		if q, ok := payload["query"].(string); ok {
			queryStr = q
			trimmed := queryStr
			if len(trimmed) > 0 {
				if len(trimmed) >= 7 && (trimmed[:7] == "mutation" || trimmed[:7] == "Mutation") {
					opType = "mutation"
					trimmed = trimmed[8:]
				} else if len(trimmed) >= 5 && (trimmed[:5] == "query" || trimmed[:5] == "Query") {
					opType = "query"
					trimmed = trimmed[6:]
				}
				for i, c := range trimmed {
					if c == '(' || c == '{' || c == ' ' {
						opName = trimmed[:i]
						break
					}
				}
			}
		} else if payload["kind"] == "Document" {
			// AST format
			if defs, ok := payload["definitions"].([]interface{}); ok && len(defs) > 0 {
				if def, ok := defs[0].(map[string]interface{}); ok {
					if t, ok := def["operation"].(string); ok {
						opType = t
					}
					if nameObj, ok := def["name"].(map[string]interface{}); ok {
						if n, ok := nameObj["value"].(string); ok {
							opName = n
						}
					}
				}
			}
		}
		if opName != "" {
			util.Log(fmt.Sprintf("[GraphQLHandler] %s request from %s. Operation: %s %s", r.Method, r.RemoteAddr, opType, opName), util.LogTypeLog)
		} else {
			util.Log(fmt.Sprintf("[GraphQLHandler] %s request from %s. Operation: %s", r.Method, r.RemoteAddr, opType), util.LogTypeLog)
		}
		// Set CORS headers for all requests
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, apikey")

		if r.Method == http.MethodOptions {
			util.Log(fmt.Sprintf("[GraphQLHandler] OPTIONS request from %s. Headers: %v", r.RemoteAddr, r.Header), util.LogTypeLog)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// ...existing code...

		supabaseEndpoint := fmt.Sprintf("%s/graphql/v1", cfg.Url)
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			util.Log(fmt.Sprintf("[GraphQLHandler] Error marshaling payload: %v", err), util.LogTypeError)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to marshal payload"})
			return
		}

		req, err := http.NewRequest("POST", supabaseEndpoint, bytes.NewBuffer(bodyBytes))
		if err != nil {
			util.Log(fmt.Sprintf("[GraphQLHandler] Error creating request: %v", err), util.LogTypeError)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to create request"})
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", cfg.Key)
		req.Header.Set("Authorization", "Bearer "+cfg.Key)

		client := &http.Client{Timeout: 10 * 1e9} // 10 seconds
		resp, err := client.Do(req)
		if err != nil {
			util.Log(fmt.Sprintf("[GraphQLHandler] Error contacting Supabase: %v", err), util.LogTypeError)
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to contact Supabase"})
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}
