package discovery

import (
	"fmt"
	"net/http"
	"time"

	"github.com/beevik/etree"
)

// Service provides operations for discovering URLs from a sitemap.
type Service struct{}

// NewService creates a new discovery service.
func NewService() *Service {
	return &Service{}
}

// Result represents a discovered URL with its status.
type Result struct {
	URL      string `json:"url"`
	Status   string `json:"status"`
	Category string `json:"category"`
}

// Discover fetches and parses a sitemap to discover URLs.
func (s *Service) Discover(siteURL string) ([]Result, error) {
	sitemapURL := fmt.Sprintf("%s/sitemap.xml", siteURL)
	resp, err := http.Get(sitemapURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sitemap: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sitemap not found or accessible, status code: %d", resp.StatusCode)
	}

	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to parse sitemap XML: %w", err)
	}

	var results []Result
	urlset := doc.SelectElement("urlset")
	if urlset == nil {
		return nil, fmt.Errorf("invalid sitemap format: <urlset> not found")
	}

	for _, urlElement := range urlset.SelectElements("url") {
		loc := urlElement.SelectElement("loc")
		if loc != nil {
			pageURL := loc.Text()
			status := s.checkURLStatus(pageURL)
			results = append(results, Result{URL: pageURL, Status: status, Category: "sitemap"})
			time.Sleep(100 * time.Millisecond) // Delay between calls
		}
	}

	return results, nil
}

func (s *Service) checkURLStatus(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("Error: %s", err.Error())
	}
	defer resp.Body.Close()
	return resp.Status
}
