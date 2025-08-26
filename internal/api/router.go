package api

import (
	"embed"
	"io/fs"
	"net/http"
	"pa11y-go-wrapper/internal/analysis"
	"pa11y-go-wrapper/internal/discovery"

	"github.com/gin-gonic/gin"
)

// NewRouter creates a new Gin router.
func NewRouter(analysisService *analysis.Service, discoveryService *discovery.Service, frontendAssets embed.FS) *gin.Engine {
	r := gin.Default()
	h := NewHandlers(analysisService, discoveryService)

	api := r.Group("/api")
	{
		api.POST("/analyze", h.AnalyzeURL)
		api.POST("/queue", h.QueueURL)
		api.GET("/queue", h.GetQueue)
		api.GET("/queue/:id", h.GetQueueItem)
		api.GET("/completed/html", h.GetCompletedAnalysesHTML)
		api.GET("/completed/pdf", h.GetCompletedAnalysesPDF)
		api.POST("/discover", h.DiscoverSite)
	}

	// Serve the frontend
	staticFiles, _ := fs.Sub(frontendAssets, "frontend")
	r.StaticFS("/app", http.FS(staticFiles))
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/app")
	})

	return r
}
