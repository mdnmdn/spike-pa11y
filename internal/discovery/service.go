package discovery

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/beevik/etree"
)

// Service provides operations for discovering URLs from a sitemap.
type Service struct {
	llmService *LLMService
}

// NewService creates a new discovery service.
func NewService() (*Service, error) {
	llmService, err := NewLLMService()
	if err != nil {
		return nil, err
	}
	return &Service{llmService: llmService}, nil
}

// Result represents a discovered URL with its status.
type Result struct {
	URL      string `json:"url"`
	Status   string `json:"status"`
	Category string `json:"category"`
}

// Discover fetches and parses a sitemap to discover URLs, then uses an LLM to refine the list.
func (s *Service) Discover(siteURL string, siteCategory string) ([]Result, error) {
	// 1. Get initial list of URLs from sitemap
	initialURLs, err := s.getURLsFromSitemap(siteURL)
	if err != nil {
		return nil, err
	}

	// 2. Narrow down to 15 URLs using LLM
	narrowedURLs, err := s.llmService.NarrowDownURLs(initialURLs, siteCategory)
	if err != nil {
		return nil, err
	}

	// 3. Extract head section for each of the 15 URLs
	heads, err := s.extractHeads(narrowedURLs)
	if err != nil {
		return nil, err
	}

	// 4. Select and categorize 10 URLs using LLM
	finalResults, err := s.llmService.SelectAndCategorizeURLs(narrowedURLs, heads, siteCategory)
	if err != nil {
		return nil, err
	}
	// 5. check the status for each URL
	for i := range finalResults {
		finalResults[i].Status = s.checkURLStatus(finalResults[i].URL)
	}

	return finalResults, nil
}

func (s *Service) getURLsFromSitemap(siteURL string) ([]string, error) {
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

	var urls []string
	urlset := doc.SelectElement("urlset")
	if urlset == nil {
		return nil, fmt.Errorf("invalid sitemap format: <urlset> not found")
	}

	for _, urlElement := range urlset.SelectElements("url") {
		loc := urlElement.SelectElement("loc")
		if loc != nil {
			urls = append(urls, loc.Text())
		}
	}

	return urls, nil
}

func (s *Service) extractHeads(urls []string) (map[string]string, error) {
	heads := make(map[string]string)
	for _, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			// It's better to log this error and continue
			fmt.Printf("failed to get URL %s: %v\n", url, err)
			heads[url] = ""
			continue
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("failed to read body for URL %s: %v\n", url, err)
			heads[url] = ""
			continue
		}

		bodyString := string(bodyBytes)
		headStart := strings.Index(bodyString, "<head>")
		headEnd := strings.Index(bodyString, "</head>")

		if headStart != -1 && headEnd != -1 {
			heads[url] = bodyString[headStart+len("<head>") : headEnd]
		} else {
			heads[url] = ""
		}
		time.Sleep(100 * time.Millisecond) // Delay between calls
	}
	return heads, nil
}

func (s *Service) checkURLStatus(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("Error: %s", err.Error())
	}
	defer resp.Body.Close()
	return resp.Status
}
