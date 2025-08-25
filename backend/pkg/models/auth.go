package models

import (
	"time"
)

type User struct {
	ID       string    `json:"id"`
	Username string    `json:"username"`
	Password string    `json:"password"` // bcrypt hashed
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

type Session struct {
	ID      string    `json:"id"`
	UserID  string    `json:"user_id"`
	Token   string    `json:"token"`
	Created time.Time `json:"created"`
	Expires time.Time `json:"expires"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SetupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Token   string `json:"token,omitempty"`
}

type StatusResponse struct {
	IsSetup     bool `json:"is_setup"`
	AuthEnabled bool `json:"auth_enabled"`
}
