package api

import (
	"net/http"
	"pa11y-go-wrapper/internal/analysis"

	"github.com/gin-gonic/gin"
)

// Handlers holds the dependencies for the API handlers.
type Handlers struct {
	service *analysis.Service
}

// NewHandlers creates new handlers.
func NewHandlers(service *analysis.Service) *Handlers {
	return &Handlers{service: service}
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

	a := h.service.Create(req.URL, req.Runner)
	c.JSON(http.StatusAccepted, a)
}

// QueueURLRequest represents the request body for the /queue endpoint.
type QueueURLRequest struct {
	URL string `json:"url" binding:"required"`
}

// QueueURL adds a URL to the analysis queue.
func (h *Handlers) QueueURL(c *gin.Context) {
	var req QueueURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analysis := h.service.Create(req.URL, "")
	c.JSON(http.StatusAccepted, analysis)
}

// GetQueue returns all analysis tasks.
func (h *Handlers) GetQueue(c *gin.Context) {
	analyses := h.service.GetAll()
	c.JSON(http.StatusOK, analyses)
}

// GetQueueItem returns a specific analysis task.
func (h *Handlers) GetQueueItem(c *gin.Context) {
	id := c.Param("id")
	analysis, ok := h.service.GetByID(id)
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
		a, ok := h.service.GetByID(id)
		if !ok {
			c.String(http.StatusNotFound, "analysis not found")
			return
		}
		analyses = []*analysis.Analysis{a}
	} else {
		analyses = h.service.GetCompleted()
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
		a, ok := h.service.GetByID(id)
		if !ok {
			c.String(http.StatusNotFound, "analysis not found")
			return
		}
		analyses = []*analysis.Analysis{a}
	} else {
		analyses = h.service.GetCompleted()
	}

	pdf, err := GeneratePDF(analyses)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to generate PDF")
		return
	}

	c.Data(http.StatusOK, "application/pdf", pdf)
}
