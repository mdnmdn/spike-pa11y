package api

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"pa11y-go-wrapper/internal/analysis"
	"pa11y-go-wrapper/internal/discovery"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Embed local test frontend assets so fs.Sub works in router
//
//go:embed frontend/*
var frontendAssets embed.FS

func TestCompletedHTML(t *testing.T) {
	// Create a new analysis service and add a completed analysis
	service := analysis.NewService(10)
	a := service.Create("http://example.com", "")
	service.UpdateResult(a.ID, analysis.StatusCompleted, nil, "")

	// Create a new router
	discoveryService := discovery.NewService()
	router := NewRouter(service, discoveryService, frontendAssets)

	// Create a new request to the /completed/html endpoint
	req, _ := http.NewRequest("GET", "/api/completed/html", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check that the response is correct
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "<h1>Accessibility Analyses</h1>")
}

func TestCompletedPDF(t *testing.T) {
	// Create a new analysis service and add a completed analysis
	service := analysis.NewService(10)
	a := service.Create("http://example.com", "")
	service.UpdateResult(a.ID, analysis.StatusCompleted, nil, "")

	// Create a new router
	discoveryService := discovery.NewService()
	router := NewRouter(service, discoveryService, frontendAssets)

	// Create a new request to the /completed/pdf endpoint
	req, _ := http.NewRequest("GET", "/api/completed/pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check that the response is correct
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
}
