package analysis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RunPa11y executes the pa11y command and returns the result.
func RunPa11y(url string, runner string) ([]Issue, error) {
	if runner == "" {
		runner = "htmlcs"
	}

	// Determine the pa11y command to run, allowing override via PA11Y_COMMAND (e.g., "npx pa11y").
	pa11yCmd := os.Getenv("PA11Y_COMMAND")
	if strings.TrimSpace(pa11yCmd) == "" {
		pa11yCmd = "pa11y"
	}
	parts := strings.Fields(pa11yCmd)
	var execName string
	var baseArgs []string
	if len(parts) > 0 {
		execName = parts[0]
		if len(parts) > 1 {
			baseArgs = parts[1:]
		}
	} else {
		execName = "pa11y"
	}

	args := append([]string{}, baseArgs...)
	args = append(args, "--reporter", "json", "--runner", runner, url)

	cmd := exec.Command(execName, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// pa11y exits with code 2 if there are accessibility issues.
		// We still want to see the JSON report in that case.
		// We only exit if there's a different error.
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() != 2 {
				return nil, fmt.Errorf("error running pa11y: %v\nOutput: %s", err, output)
			}
		} else {
			return nil, fmt.Errorf("error running pa11y: %v\nOutput: %s", err, output)
		}
	}

	var result []Issue
	err = json.Unmarshal(output, &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling pa11y output: %v\nOutput was: %s", err, output)
	}

	return result, nil
}

// Worker processes analysis tasks from the queue.
type Worker struct {
	service *Service
}

// NewWorker creates a new worker.
func NewWorker(service *Service) *Worker {
	return &Worker{
		service: service,
	}
}

// Start begins the worker's processing loop.
func (w *Worker) Start() {
	go func() {
		for {
			analysisID := w.service.GetNextFromQueue()
			analysis, ok := w.service.GetByID(analysisID)
			if !ok {
				fmt.Fprintf(os.Stderr, "Error: analysis with ID %s not found\n", analysisID)
				continue
			}

			w.service.UpdateStatus(analysis.ID, StatusProcessing)

			// Check URL reachability before running pa11y
			if size, err := checkURLReachable(analysis.URL); err != nil {
				fmt.Fprintf(os.Stderr, "URL not reachable %s: %v\n", analysis.URL, err)
				w.service.UpdateResult(analysis.ID, StatusFailed, nil, fmt.Sprintf("URL not reachable: %v", err))
				continue
			} else {
				w.service.UpdateSize(analysis.ID, size)
			}

			// Use the specified runner if provided; RunPa11y defaults to htmlcs when empty
			result, err := RunPa11y(analysis.URL, analysis.Runner)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running pa11y for %s: %v\n", analysis.URL, err)
				w.service.UpdateResult(analysis.ID, StatusFailed, nil, err.Error())
				continue
			}

			w.service.UpdateResult(analysis.ID, StatusCompleted, result, "")
		}
	}()
}

// checkURLReachable performs a direct GET request to verify reachability and returns the response size in bytes.
// It validates the URL scheme (http/https), performs the request with a timeout,
// and returns a descriptive error if the URL is not reachable or returns 4xx/5xx.
func checkURLReachable(rawURL string) (int64, error) {
	// Validate URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0, fmt.Errorf("invalid URL: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return 0, fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
	if u.Host == "" {
		return 0, fmt.Errorf("invalid URL: missing host")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "pa11y-go-wrapper/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()
	// read and count body to allow connection reuse and get actual size
	n, _ := io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return n, fmt.Errorf("received HTTP status %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return n, nil
}
