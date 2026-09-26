package security

import "time"

type Role struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	RoleID   int64  `json:"role_id"`
	RoleName string `json:"role_name"`
	Active   bool   `json:"active"`
}

type Session struct {
	Token      string    `json:"token"`
	UserID     int64     `json:"-"`
	RegisterID *int64    `json:"register_id,omitempty"`
	Locked     bool      `json:"locked"`
	ExpiresAt  time.Time `json:"-"`
}

// AuthResult is what /api/auth/login, /switch, /unlock, and /me return.
type AuthResult struct {
	Token       string   `json:"token"`
	User        User     `json:"user"`
	Permissions []string `json:"permissions"`
	Locked      bool     `json:"locked"`
	RegisterID  *int64   `json:"register_id,omitempty"`
}
