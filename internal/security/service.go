package security

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"pos/internal/audit"
	"pos/internal/domain"
	"pos/internal/storage"
)

const sessionTTL = 18 * time.Hour

type Service struct {
	db  *storage.DB
	log *audit.Logger
}

func NewService(db *storage.DB, log *audit.Logger) *Service {
	return &Service{db: db, log: log}
}

func (s *Service) userByUsername(ctx context.Context, username string) (*User, string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.name, u.username, u.pin_hash, u.role_id, r.name, u.active
		FROM users u JOIN roles r ON r.id = u.role_id
		WHERE u.username = ?`, username)
	var u User
	var pinHash string
	var active int
	if err := row.Scan(&u.ID, &u.Name, &u.Username, &pinHash, &u.RoleID, &u.RoleName, &active); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", domain.NewError("INVALID_CREDENTIALS", "Invalid username or PIN", 401)
		}
		return nil, "", err
	}
	u.Active = active == 1
	return &u, pinHash, nil
}

func (s *Service) UserByID(ctx context.Context, id int64) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.name, u.username, u.role_id, r.name, u.active
		FROM users u JOIN roles r ON r.id = u.role_id
		WHERE u.id = ?`, id)
	var u User
	var active int
	if err := row.Scan(&u.ID, &u.Name, &u.Username, &u.RoleID, &u.RoleName, &active); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	u.Active = active == 1
	return &u, nil
}

func (s *Service) permissionsForRole(ctx context.Context, roleID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.code FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = ?`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var perms []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		perms = append(perms, code)
	}
	return perms, rows.Err()
}

// Login authenticates a cashier by username+PIN and opens a new session
// bound to the given register.
func (s *Service) Login(ctx context.Context, username, pin string, registerID *int64) (*AuthResult, error) {
	user, pinHash, err := s.userByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if !user.Active {
		return nil, domain.NewError("USER_DISABLED", "This user account is disabled", 403)
	}
	if !VerifyPIN(pinHash, pin) {
		return nil, domain.NewError("INVALID_CREDENTIALS", "Invalid username or PIN", 401)
	}
	return s.createSession(ctx, user, registerID)
}

func (s *Service) createSession(ctx context.Context, user *User, registerID *int64) (*AuthResult, error) {
	token, err := NewToken()
	if err != nil {
		return nil, err
	}
	expires := time.Now().Add(sessionTTL)
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token, user_id, register_id, locked, expires_at) VALUES (?, ?, ?, 0, ?)`,
		token, user.ID, registerID, expires.UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, err
	}
	perms, err := s.permissionsForRole(ctx, user.RoleID)
	if err != nil {
		return nil, err
	}
	s.log.Write(ctx, audit.Event{Event: "LOGIN", UserID: &user.ID})
	return &AuthResult{Token: token, User: *user, Permissions: perms, RegisterID: registerID}, nil
}

// Authenticate resolves a bearer token into the session + user + permissions.
// Returns domain.ErrUnauthorized if the token is missing/expired.
func (s *Service) Authenticate(ctx context.Context, token string) (*Session, *User, []string, error) {
	if token == "" {
		return nil, nil, nil, domain.ErrUnauthorized
	}
	row := s.db.QueryRowContext(ctx, `SELECT user_id, register_id, locked, expires_at FROM sessions WHERE token = ?`, token)
	var sess Session
	var registerID sql.NullInt64
	var locked int
	var expiresAt string
	sess.Token = token
	if err := row.Scan(&sess.UserID, &registerID, &locked, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil, domain.ErrUnauthorized
		}
		return nil, nil, nil, err
	}
	if registerID.Valid {
		v := registerID.Int64
		sess.RegisterID = &v
	}
	sess.Locked = locked == 1
	exp, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err == nil {
		sess.ExpiresAt = exp
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, nil, nil, domain.ErrUnauthorized
	}
	user, err := s.UserByID(ctx, sess.UserID)
	if err != nil {
		return nil, nil, nil, err
	}
	perms, err := s.permissionsForRole(ctx, user.RoleID)
	if err != nil {
		return nil, nil, nil, err
	}
	return &sess, user, perms, nil
}

func (s *Service) Lock(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET locked = 1 WHERE token = ?`, token)
	return err
}

func (s *Service) Unlock(ctx context.Context, token, pin string) (*AuthResult, error) {
	_, user, perms, err := s.Authenticate(ctx, token)
	if err != nil {
		return nil, err
	}
	_, pinHash, err := s.userByUsername(ctx, user.Username)
	if err != nil {
		return nil, err
	}
	if !VerifyPIN(pinHash, pin) {
		return nil, domain.NewError("INVALID_CREDENTIALS", "Invalid PIN", 401)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE sessions SET locked = 0 WHERE token = ?`, token); err != nil {
		return nil, err
	}
	s.log.Write(ctx, audit.Event{Event: "UNLOCK", UserID: &user.ID})
	return &AuthResult{Token: token, User: *user, Permissions: perms}, nil
}

// SwitchCashier ends the current session and opens a new one for a different
// user, on the same register, without requiring an application restart.
func (s *Service) SwitchCashier(ctx context.Context, oldToken, username, pin string) (*AuthResult, error) {
	_, oldUser, _, err := s.Authenticate(ctx, oldToken)
	if err != nil {
		return nil, err
	}
	var registerID *int64
	row := s.db.QueryRowContext(ctx, `SELECT register_id FROM sessions WHERE token = ?`, oldToken)
	var reg sql.NullInt64
	if err := row.Scan(&reg); err == nil && reg.Valid {
		v := reg.Int64
		registerID = &v
	}

	user, pinHash, err := s.userByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if !user.Active {
		return nil, domain.NewError("USER_DISABLED", "This user account is disabled", 403)
	}
	if !VerifyPIN(pinHash, pin) {
		return nil, domain.NewError("INVALID_CREDENTIALS", "Invalid username or PIN", 401)
	}

	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, oldToken); err != nil {
		return nil, err
	}
	result, err := s.createSession(ctx, user, registerID)
	if err != nil {
		return nil, err
	}
	s.log.Write(ctx, audit.Event{Event: "CASHIER_SWITCH", UserID: &user.ID, ApproverID: &oldUser.ID})
	return result, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	_, user, _, _ := s.Authenticate(ctx, token)
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token); err != nil {
		return err
	}
	if user != nil {
		s.log.Write(ctx, audit.Event{Event: "LOGOUT", UserID: &user.ID})
	}
	return nil
}

// HasPermission reports whether the given permission set contains code.
func HasPermission(perms []string, code string) bool {
	for _, p := range perms {
		if p == code {
			return true
		}
	}
	return false
}
