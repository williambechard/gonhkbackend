package main

import (
	"net/http"
	graphserver "nhknewseasybkend/internal/graphql"
	"nhknewseasybkend/internal/util"
)

// RegisterRoutes sets up all API routes for this service
func RegisterRoutes(mux *http.ServeMux) {

	// Setup GraphQL proxy endpoint
	cfg := graphserver.InitSupabase()
	mux.HandleFunc("/graphql", graphserver.GraphQLHandler(cfg))
	util.Log("Registered endpoint /graphql (proxy mode)", util.LogTypeLog)

}
