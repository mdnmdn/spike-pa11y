package analysis

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// AnalysisStatus represents the status of an analysis task.
type AnalysisStatus string

const (
	// StatusPending means the analysis is waiting to be processed.
	StatusPending AnalysisStatus = "pending"
	// StatusProcessing means the analysis is currently being processed.
	StatusProcessing AnalysisStatus = "processing"
	// StatusCompleted means the analysis has been completed.
	StatusCompleted AnalysisStatus = "completed"
	// StatusFailed means the analysis failed.
	StatusFailed AnalysisStatus = "failed"
)

// Issue represents a single accessibility issue.
type Issue struct {
	Code         string                 `json:"code"`
	Context      string                 `json:"context"`
	Message      string                 `json:"message"`
	Runner       string                 `json:"runner"`
	RunnerExtras map[string]interface{} `json:"runnerExtras"`
	Selector     string                 `json:"selector"`
	Type         string                 `json:"type"`
	TypeCode     int                    `json:"typeCode"`
}

// Analysis represents a single analysis task.
type Analysis struct {
	ID           string         `json:"id"`
	URL          string         `json:"url"`
	Runner       string         `json:"runner,omitempty"`
	Status       AnalysisStatus `json:"status"`
	Result       []Issue        `json:"result,omitempty"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
	SizeBytes    int64          `json:"sizeBytes,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	StartedAt    time.Time      `json:"startedAt,omitempty"`
	CompletedAt  time.Time      `json:"completedAt,omitempty"`
	DurationMs   int64          `json:"durationMs,omitempty"`
}

// Service provides operations for managing analysis tasks.
type Service struct {
	mu       sync.RWMutex
	analyses map[string]*Analysis
	queue    chan string
}

// NewService creates a new analysis service.
func NewService(queueSize int) *Service {
	return &Service{
		analyses: make(map[string]*Analysis),
		queue:    make(chan string, queueSize),
	}
}

// Create new analysis task and add it to the queue.
func (s *Service) Create(url string, runner string) *Analysis {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	analysis := &Analysis{
		ID:        id,
		URL:       url,
		Runner:    runner,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.analyses[analysis.ID] = analysis
	s.queue <- analysis.ID
	return analysis
}

// GetAll returns all analysis tasks.
func (s *Service) GetAll() []*Analysis {
	s.mu.RLock()
	defer s.mu.RUnlock()

	analyses := make([]*Analysis, 0, len(s.analyses))
	for _, analysis := range s.analyses {
		analyses = append(analyses, analysis)
	}
	return analyses
}

// GetCompleted returns all completed analysis tasks.
func (s *Service) GetCompleted() []*Analysis {
	s.mu.RLock()
	defer s.mu.RUnlock()

	analyses := make([]*Analysis, 0, len(s.analyses))
	for _, analysis := range s.analyses {
		if analysis.Status == StatusCompleted {
			analyses = append(analyses, analysis)
		}
	}
	return analyses
}

// GetByID returns an analysis task by its ID.
func (s *Service) GetByID(id string) (*Analysis, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	analysis, ok := s.analyses[id]
	return analysis, ok
}

// GetNextFromQueue gets the next analysis ID from the queue. This will block if the queue is empty.
func (s *Service) GetNextFromQueue() string {
	return <-s.queue
}

// UpdateStatus updates the status of an analysis task.
func (s *Service) UpdateStatus(id string, status AnalysisStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if analysis, ok := s.analyses[id]; ok {
		now := time.Now()
		// Transition-specific timestamps
		if status == StatusProcessing {
			if analysis.StartedAt.IsZero() {
				analysis.StartedAt = now
			}
		}
		if status == StatusCompleted || status == StatusFailed {
			if analysis.CompletedAt.IsZero() {
				analysis.CompletedAt = now
			}
			// Compute duration from StartedAt if available, otherwise from CreatedAt
			start := analysis.StartedAt
			if start.IsZero() {
				start = analysis.CreatedAt
			}
			dur := analysis.CompletedAt.Sub(start)
			if dur < 0 {
				dur = 0
			}
			analysis.DurationMs = dur.Milliseconds()
		}

		analysis.Status = status
		analysis.UpdatedAt = now
	}
}

// UpdateResult updates the result of an analysis task.
func (s *Service) UpdateResult(id string, status AnalysisStatus, result []Issue, errorMessage string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if analysis, ok := s.analyses[id]; ok {
		now := time.Now()
		analysis.Status = status
		analysis.Result = result
		analysis.ErrorMessage = errorMessage

		if status == StatusCompleted || status == StatusFailed {
			if analysis.CompletedAt.IsZero() {
				analysis.CompletedAt = now
			}
			start := analysis.StartedAt
			if start.IsZero() {
				start = analysis.CreatedAt
			}
			dur := analysis.CompletedAt.Sub(start)
			if dur < 0 {
				dur = 0
			}
			analysis.DurationMs = dur.Milliseconds()
		}

		analysis.UpdatedAt = now
	}
}

// UpdateSize updates the fetched size of the target URL in bytes.
func (s *Service) UpdateSize(id string, size int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if analysis, ok := s.analyses[id]; ok {
		analysis.SizeBytes = size
		analysis.UpdatedAt = time.Now()
	}
}
