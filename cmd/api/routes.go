package main

import (
	"net/http"
	graphserver "nhknewseasybkend/internal/graphql"
	"nhknewseasybkend/internal/util"
)

// RegisterRoutes sets up all API routes for this service
func RegisterRoutes(mux *http.ServeMux) {

	// Setup GraphQL endpoint
	schema, err := graphserver.BuildSchema()

	if err == nil && schema != nil {
		util.Log("GraphQL schema built successfully", util.LogTypeLog)
		mux.HandleFunc("/graphql", graphserver.GraphQLHandler(schema))
		util.Log("Registered endpoint /graphql", util.LogTypeLog)
	} else {
		util.Log("Failed to build GraphQL schema: "+err.Error(), util.LogTypeError)
		panic("Failed to build GraphQL schema: " + err.Error())
	}

}
