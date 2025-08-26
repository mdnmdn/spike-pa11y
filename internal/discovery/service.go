package discovery

import (
	"compress/gzip"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
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

	// 2. Sample URLs if there are more than 200
	initialURLs = s.sampleUrls(siteURL, initialURLs)

	// 3. Narrow down to 15 URLs using LLM
	narrowedURLs, err := s.llmService.NarrowDownURLs(initialURLs, siteCategory)
	if err != nil {
		return nil, err
	}

	// 4. Extract head section for each of the 15 URLs
	heads, err := s.extractHeads(narrowedURLs)
	if err != nil {
		return nil, err
	}

	// 5. Select and categorize 10 URLs using LLM
	finalResults, err := s.llmService.SelectAndCategorizeURLs(narrowedURLs, heads, siteCategory)
	if err != nil {
		return nil, err
	}
	// 6. check the status for each URL
	for i := range finalResults {
		finalResults[i].Status = s.checkURLStatus(finalResults[i].URL)
	}

	return finalResults, nil
}

// sampleUrls samples URLs if there are more than 200:
// - Takes first 20 shortest URLs (ordered by length)
// - Takes 50 random URLs from the remaining ones
// - Returns up to 70 URLs total
func (s *Service) sampleUrls(siteURL string, urls []string) []string {
	if len(urls) <= 200 {
		return urls
	}

	// Create a copy to avoid modifying the original slice
	urlsCopy := make([]string, len(urls))
	copy(urlsCopy, urls)

	// Sort by URL length (shortest first)
	sort.Slice(urlsCopy, func(i, j int) bool {
		return len(urlsCopy[i]) < len(urlsCopy[j])
	})

	// Take first 20 (shortest URLs)
	result := make([]string, 0, 70)
	if len(urlsCopy) >= 20 {
		result = append(result, urlsCopy[:20]...)
		urlsCopy = urlsCopy[20:] // Remove the first 20 from remaining
	} else {
		result = append(result, urlsCopy...)
		urlsCopy = []string{} // No remaining URLs
	}

	// Take 50 random URLs from the remaining ones
	if len(urlsCopy) > 0 {
		// Shuffle the remaining URLs
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(urlsCopy), func(i, j int) {
			urlsCopy[i], urlsCopy[j] = urlsCopy[j], urlsCopy[i]
		})

		// Take up to 180 random URLs
		sampleSize := 180
		if len(urlsCopy) < sampleSize {
			sampleSize = len(urlsCopy)
		}
		result = append(result, urlsCopy[:sampleSize]...)
	}

	return result
}

func (s *Service) getURLsFromSitemap(siteURL string) ([]string, error) {
	sitemapURL := fmt.Sprintf("%s/sitemap.xml", siteURL)
	return s.parseXMLSitemap(sitemapURL)
}

func (s *Service) parseXMLSitemap(sitemapURL string) ([]string, error) {
	resp, err := http.Get(sitemapURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sitemap: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sitemap not found or accessible, status code: %d", resp.StatusCode)
	}

	// Handle gzipped response body
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" || strings.HasSuffix(sitemapURL, ".gz") {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("failed to parse sitemap XML: %w", err)
	}

	var urls []string

	// Check if this is a sitemap index
	sitemapIndex := doc.SelectElement("sitemapindex")
	if sitemapIndex != nil {
		// This is a sitemap index, fetch URLs from all referenced sitemaps
		for _, sitemapElement := range sitemapIndex.SelectElements("sitemap") {
			loc := sitemapElement.SelectElement("loc")
			if loc != nil {
				subSitemapURLs, err := s.parseXMLSitemap(loc.Text())
				if err != nil {
					// Log error but continue with other sitemaps
					fmt.Printf("failed to parse sub-sitemap %s: %v\n", loc.Text(), err)
					continue
				}
				urls = append(urls, subSitemapURLs...)
			}
		}
		return urls, nil
	}

	// Check if this is a regular sitemap
	urlset := doc.SelectElement("urlset")
	if urlset != nil {
		for _, urlElement := range urlset.SelectElements("url") {
			loc := urlElement.SelectElement("loc")
			if loc != nil {
				urls = append(urls, loc.Text())
			}
		}
		return urls, nil
	}

	return nil, fmt.Errorf("invalid sitemap format: neither <sitemapindex> nor <urlset> found")
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
			headContent := bodyString[headStart+len("<head>") : headEnd]
			heads[url] = s.cleanupHTML(headContent)
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

// cleanupHTML removes <script> and <style> tags and their content from HTML
func (s *Service) cleanupHTML(html string) string {
	result := html

	// Remove all <script> tags and their content
	result = s.removeTagsAndContent(result, "script")

	// Remove all <style> tags and their content
	result = s.removeTagsAndContent(result, "style")

	return result
}

// removeTagsAndContent removes all instances of a specific tag and its content
func (s *Service) removeTagsAndContent(html, tagName string) string {
	result := html
	openTag := "<" + tagName
	closeTag := "</" + tagName + ">"

	for {
		// Find the opening tag (could have attributes)
		openStart := strings.Index(strings.ToLower(result), strings.ToLower(openTag))
		if openStart == -1 {
			break // No more opening tags found
		}

		// Find the end of the opening tag (look for '>')
		openEnd := strings.Index(result[openStart:], ">")
		if openEnd == -1 {
			break // Malformed tag, stop processing
		}
		openEnd += openStart + 1 // Convert to absolute position and include '>'

		// Find the corresponding closing tag
		closeStart := strings.Index(strings.ToLower(result[openEnd:]), strings.ToLower(closeTag))
		if closeStart == -1 {
			// No closing tag found, remove from opening tag to end of string
			result = result[:openStart]
			break
		}
		closeStart += openEnd // Convert to absolute position
		closeEnd := closeStart + len(closeTag)

		// Remove the entire tag and its content
		result = result[:openStart] + result[closeEnd:]
	}

	return result
}
