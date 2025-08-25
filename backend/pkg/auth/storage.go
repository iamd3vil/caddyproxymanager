package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sarat/caddyproxymanager/pkg/models"
)

type Storage struct {
	mu       sync.RWMutex
	dataDir  string
	users    map[string]*models.User
	sessions map[string]*models.Session
}

func NewStorage(dataDir string) *Storage {
	return &Storage{
		dataDir:  dataDir,
		users:    make(map[string]*models.User),
		sessions: make(map[string]*models.Session),
	}
}

func (s *Storage) Initialize() error {
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := s.loadUsers(); err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}

	if err := s.loadSessions(); err != nil {
		return fmt.Errorf("failed to load sessions: %w", err)
	}

	return nil
}

func (s *Storage) IsSetup() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users) > 0
}

func (s *Storage) CreateUser(username, password string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user already exists
	for _, user := range s.users {
		if user.Username == username {
			return nil, fmt.Errorf("user already exists")
		}
	}

	id, err := GenerateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate user ID: %w", err)
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		ID:       id,
		Username: username,
		Password: hashedPassword,
		Created:  time.Now(),
		Updated:  time.Now(),
	}

	s.users[id] = user

	if err := s.saveUsers(); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return user, nil
}

func (s *Storage) GetUserByUsername(username string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Username == username {
			return user, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

func (s *Storage) GetUserByID(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if user, exists := s.users[id]; exists {
		return user, nil
	}

	return nil, fmt.Errorf("user not found")
}

func (s *Storage) CreateSession(userID string) (*models.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := GenerateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	token, err := GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	session := &models.Session{
		ID:      id,
		UserID:  userID,
		Token:   token,
		Created: time.Now(),
		Expires: time.Now().Add(GetSessionDuration()),
	}

	s.sessions[token] = session

	if err := s.saveSessions(); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

func (s *Storage) GetSession(token string) (*models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[token]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if IsSessionExpired(session.Expires) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

func (s *Storage) DeleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, token)
	return s.saveSessions()
}

func (s *Storage) CleanExpiredSessions() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for token, session := range s.sessions {
		if IsSessionExpired(session.Expires) {
			delete(s.sessions, token)
		}
	}

	return s.saveSessions()
}

func (s *Storage) loadUsers() error {
	filePath := filepath.Join(s.dataDir, "users.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // File doesn't exist, that's OK
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read users file: %w", err)
	}

	var users map[string]*models.User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("failed to unmarshal users: %w", err)
	}

	s.users = users
	return nil
}

func (s *Storage) saveUsers() error {
	filePath := filepath.Join(s.dataDir, "users.json")

	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write users file: %w", err)
	}

	return nil
}

func (s *Storage) loadSessions() error {
	filePath := filepath.Join(s.dataDir, "sessions.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // File doesn't exist, that's OK
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read sessions file: %w", err)
	}

	var sessions map[string]*models.Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return fmt.Errorf("failed to unmarshal sessions: %w", err)
	}

	s.sessions = sessions
	return nil
}

func (s *Storage) saveSessions() error {
	filePath := filepath.Join(s.dataDir, "sessions.json")

	data, err := json.MarshalIndent(s.sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sessions: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write sessions file: %w", err)
	}

	return nil
}
