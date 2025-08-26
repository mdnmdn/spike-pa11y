package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter creates a new Gin router.
func NewRouter(h *Handlers, frontendAssets embed.FS) *gin.Engine {
	r := gin.Default()

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
