package main

import (
	"net/http"
	"nhknewseasybkend/internal/service"
)

// RegisterRoutes sets up all API routes for this service
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/links", service.LinksHandler)
}
