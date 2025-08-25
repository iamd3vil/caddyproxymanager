package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sarat/caddyproxymanager/pkg/models"
)

// Service manages health checks for proxies
type Service struct {
	mu       sync.RWMutex
	statuses map[string]*models.HealthStatus
	cancels  map[string]context.CancelFunc
	client   *http.Client
}

// NewService creates a new health check service
func NewService() *Service {
	return &Service{
		statuses: make(map[string]*models.HealthStatus),
		cancels:  make(map[string]context.CancelFunc),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// StartHealthCheck starts health checking for a proxy
func (s *Service) StartHealthCheck(proxy models.Proxy) error {
	if !proxy.HealthCheckEnabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop existing health check if running
	if cancel, exists := s.cancels[proxy.ID]; exists {
		cancel()
		delete(s.cancels, proxy.ID)
	}

	// Initialize status as pending
	s.statuses[proxy.ID] = &models.HealthStatus{
		Status:      "Pending",
		LastChecked: time.Now().Format(time.RFC3339),
		Message:     "Health check starting",
	}

	// Parse interval
	interval, err := time.ParseDuration(proxy.HealthCheckInterval)
	if err != nil {
		s.statuses[proxy.ID].Status = "Unhealthy"
		s.statuses[proxy.ID].Message = fmt.Sprintf("Invalid interval: %v", err)
		return err
	}

	// Start background goroutine
	ctx, cancel := context.WithCancel(context.Background())
	s.cancels[proxy.ID] = cancel

	go s.runHealthCheck(ctx, proxy, interval)

	return nil
}

// StopHealthCheck stops health checking for a proxy
func (s *Service) StopHealthCheck(proxyID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cancel, exists := s.cancels[proxyID]; exists {
		cancel()
		delete(s.cancels, proxyID)
		delete(s.statuses, proxyID)
	}
}

// GetHealthStatus returns the health status for a proxy
func (s *Service) GetHealthStatus(proxyID string) (*models.HealthStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, exists := s.statuses[proxyID]
	if !exists {
		return &models.HealthStatus{
			Status:      "Unknown",
			LastChecked: "",
			Message:     "Health check not enabled",
		}, false
	}

	// Return a copy to avoid race conditions
	return &models.HealthStatus{
		Status:      status.Status,
		LastChecked: status.LastChecked,
		Message:     status.Message,
	}, true
}

// GetAllHealthStatuses returns all health statuses
func (s *Service) GetAllHealthStatuses() map[string]*models.HealthStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*models.HealthStatus)
	for id, status := range s.statuses {
		result[id] = &models.HealthStatus{
			Status:      status.Status,
			LastChecked: status.LastChecked,
			Message:     status.Message,
		}
	}
	return result
}

// runHealthCheck performs periodic health checks
func (s *Service) runHealthCheck(ctx context.Context, proxy models.Proxy, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Perform initial check immediately
	s.performHealthCheck(proxy)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.performHealthCheck(proxy)
		}
	}
}

// performHealthCheck performs a single health check
func (s *Service) performHealthCheck(proxy models.Proxy) {
	healthURL := proxy.TargetURL + proxy.HealthCheckPath
	now := time.Now().Format(time.RFC3339)

	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		s.updateStatus(proxy.ID, "Unhealthy", now, fmt.Sprintf("Failed to create request: %v", err))
		return
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.updateStatus(proxy.ID, "Unhealthy", now, fmt.Sprintf("Request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == proxy.HealthCheckExpectedStatus {
		s.updateStatus(proxy.ID, "Healthy", now, "Health check passed")
	} else {
		s.updateStatus(proxy.ID, "Unhealthy", now, fmt.Sprintf("Expected status %d, got %d", proxy.HealthCheckExpectedStatus, resp.StatusCode))
	}
}

// updateStatus updates the health status for a proxy
func (s *Service) updateStatus(proxyID, status, lastChecked, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.statuses[proxyID]; exists {
		s.statuses[proxyID].Status = status
		s.statuses[proxyID].LastChecked = lastChecked
		s.statuses[proxyID].Message = message
	}
}
