package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry represents a single audit log entry
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	UserID    string    `json:"user_id,omitempty"`
	Username  string    `json:"username,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
}

// Service handles audit logging
type Service struct {
	mu       sync.RWMutex
	dataDir  string
	filename string
}

// NewService creates a new audit log service
func NewService(dataDir string) *Service {
	return &Service{
		dataDir:  dataDir,
		filename: filepath.Join(dataDir, "audit.log"),
	}
}

// Log writes an audit log entry
func (s *Service) Log(action, details, userID, username, ipAddress string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure data directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create entry
	entry := Entry{
		Timestamp: time.Now(),
		Action:    action,
		Details:   details,
		UserID:    userID,
		Username:  username,
		IPAddress: ipAddress,
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	// Open file for appending
	file, err := os.OpenFile(s.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}
	defer file.Close()

	// Write entry as JSONL (JSON Line)
	_, err = file.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write to audit log file: %w", err)
	}

	return nil
}

// GetRecentEntries retrieves the most recent audit log entries
func (s *Service) GetRecentEntries(limit int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Open file for reading
	file, err := os.Open(s.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil // Return empty slice if file doesn't exist
		}
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}
	defer file.Close()

	// Read file in reverse to get most recent entries first
	scanner := bufio.NewScanner(file)
	entries := []Entry{}

	// For simplicity, we'll read all entries and then limit
	// In a production system, you might want to implement a more efficient approach
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed entries
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading audit log file: %w", err)
	}

	// Reverse the slice to get most recent first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	// Limit results
	if len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}
