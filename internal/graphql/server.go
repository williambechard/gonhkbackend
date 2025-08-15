package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"nhknewseasybkend/internal/config"
	"nhknewseasybkend/internal/graphql/queries"
	"nhknewseasybkend/internal/util"
	"os"

	"github.com/graphql-go/graphql"
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
	util.Log(fmt.Sprintf("[InitSupabase] SUPABASE_URL: %s, SUPABASE_SERVICE_ROLE_KEY: %s", maskedUrl, maskedKey), util.LogTypeLog)
}

var rootQuery = graphql.NewObject(graphql.ObjectConfig{
	Name: "Query",
	Fields: graphql.Fields{
		"hello": &graphql.Field{
			Type: graphql.String,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return "world", nil
			},
		},
		"links": queries.LinksField,
		// Add more queries here as you modularize
	},
})

// GraphQLHandler handles GraphQL HTTP requests
func GraphQLHandler(schema *graphql.Schema) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params struct {
			Query         string                 `json:"query"`
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}

		// Log the request method and payload
		var payload map[string]interface{}
		decoder := json.NewDecoder(r.Body)
		_ = decoder.Decode(&payload)
		logMsg := fmt.Sprintf("[GraphQLHandler] %s request. Payload: %v", r.Method, payload)
		util.Log(logMsg, util.LogTypeLog)

		// Decode again into params (if needed, buffer body first)
		// For now, just decode into params from payload if possible
		// But normally, decode into params from the original body
		// Since body is already read, we can't decode again unless we buffer it
		// So, for correct behavior, decode into params from payload if possible
		// If not, you may need to refactor to buffer body before decoding

		// For now, just check if payload has "query" etc
		params.Query = ""
		if q, ok := payload["query"].(string); ok {
			params.Query = q
		}
		params.OperationName = ""
		if op, ok := payload["operationName"].(string); ok {
			params.OperationName = op
		}
		if vars, ok := payload["variables"].(map[string]interface{}); ok {
			params.Variables = vars
		} else {
			params.Variables = make(map[string]interface{})
		}

		if params.Query == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}

		result := graphql.Do(graphql.Params{
			Schema:         *schema,
			RequestString:  params.Query,
			VariableValues: params.Variables,
			OperationName:  params.OperationName,
			Context:        r.Context(),
		})

		// Log array length for 'links' query
		if data, ok := result.Data.(map[string]interface{}); ok {
			if linksArr, ok := data["links"].([]interface{}); ok {
				util.Log(fmt.Sprintf("GraphQL query returned %d entries", len(linksArr)), util.LogTypeLog)
			}
		}

		if len(result.Errors) > 0 {
			util.Log("GraphQL error: "+result.Errors[0].Message, util.LogTypeError)
			w.WriteHeader(http.StatusBadRequest)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

// BuildSchema parses the GraphQL schema string and returns a graphql.Schema
func BuildSchema() (*graphql.Schema, error) {
	config.InitSupabase() // Initialize Supabase credentials once when building schema
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: rootQuery,
		// Mutation: nil, // Add mutation if needed
	})
	return &schema, err
}
