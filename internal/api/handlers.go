package api

import (
	"net/http"
	"pa11y-go-wrapper/internal/analysis"
	"pa11y-go-wrapper/internal/discovery"

	"github.com/gin-gonic/gin"
)

// Handlers holds the dependencies for the API handlers.
type Handlers struct {
	analysisService *analysis.Service
	discoveryService *discovery.Service
}

// NewHandlers creates new handlers.
func NewHandlers(analysisService *analysis.Service, discoveryService *discovery.Service) *Handlers {
	return &Handlers{analysisService: analysisService, discoveryService: discoveryService}
}

// DiscoverSiteRequest represents the request body for the /discover endpoint.
type DiscoverSiteRequest struct {
	URL          string `json:"url" binding:"required"`
	SiteCategory string `json:"siteCategory"`
}

// DiscoverSite handles site discovery.
func (h *Handlers) DiscoverSite(c *gin.Context) {
	var req DiscoverSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.discoveryService.Discover(req.URL, req.SiteCategory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

// AnalyzeURLRequest represents the request body for the /analyze endpoint.
type AnalyzeURLRequest struct {
	URL    string `json:"url" binding:"required"`
	Runner string `json:"runner"`
}

// AnalyzeURL handles direct analysis of a URL.
func (h *Handlers) AnalyzeURL(c *gin.Context) {
	var req AnalyzeURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a := h.analysisService.Create(req.URL, req.Runner)
	c.JSON(http.StatusAccepted, a)
}

// QueueURLRequest represents the request body for the /queue endpoint.
type QueueURLRequest struct {
	URL    string `json:"url" binding:"required"`
	Runner string `json:"runner"`
}

// QueueURL adds a URL to the analysis queue.
func (h *Handlers) QueueURL(c *gin.Context) {
	var req QueueURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analysis := h.analysisService.Create(req.URL, req.Runner)
	c.JSON(http.StatusAccepted, analysis)
}

// GetQueue returns all analysis tasks.
func (h *Handlers) GetQueue(c *gin.Context) {
	analyses := h.analysisService.GetAll()
	c.JSON(http.StatusOK, analyses)
}

// GetQueueItem returns a specific analysis task.
func (h *Handlers) GetQueueItem(c *gin.Context) {
	id := c.Param("id")
	analysis, ok := h.analysisService.GetByID(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "analysis not found"})
		return
	}
	c.JSON(http.StatusOK, analysis)
}

// GetCompletedAnalysesHTML returns all completed analysis tasks as an HTML page.
func (h *Handlers) GetCompletedAnalysesHTML(c *gin.Context) {
	id := c.Query("id")
	var analyses []*analysis.Analysis
	if id != "" {
		a, ok := h.analysisService.GetByID(id)
		if !ok {
			c.String(http.StatusNotFound, "analysis not found")
			return
		}
		analyses = []*analysis.Analysis{a}
	} else {
		analyses = h.analysisService.GetCompleted()
	}

	html, err := GenerateHTML(analyses)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to generate HTML")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// GetCompletedAnalysesPDF returns all completed analysis tasks as a PDF file.
func (h *Handlers) GetCompletedAnalysesPDF(c *gin.Context) {
	id := c.Query("id")
	var analyses []*analysis.Analysis
	if id != "" {
		a, ok := h.analysisService.GetByID(id)
		if !ok {
			c.String(http.StatusNotFound, "analysis not found")
			return
		}
		analyses = []*analysis.Analysis{a}
	} else {
		analyses = h.analysisService.GetCompleted()
	}

	pdf, err := GeneratePDF(analyses)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to generate PDF")
		return
	}

	c.Data(http.StatusOK, "application/pdf", pdf)
}
