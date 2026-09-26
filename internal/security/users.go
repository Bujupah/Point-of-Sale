package security

import (
	"context"
	"strings"

	"pos/internal/domain"
)

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.name, u.username, u.role_id, r.name, u.active
		FROM users u JOIN roles r ON r.id = u.role_id ORDER BY u.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		var active int
		if err := rows.Scan(&u.ID, &u.Name, &u.Username, &u.RoleID, &u.RoleName, &active); err != nil {
			return nil, err
		}
		u.Active = active == 1
		out = append(out, u)
	}
	return out, rows.Err()
}

type CreateUserInput struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	PIN      string `json:"pin"`
	RoleID   int64  `json:"role_id"`
}

func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (*User, error) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Username) == "" {
		return nil, domain.NewError("INVALID_INPUT", "Name and username are required", 400)
	}
	if len(in.PIN) < 4 {
		return nil, domain.NewError("INVALID_INPUT", "PIN must be at least 4 digits", 400)
	}
	hash, err := HashPIN(in.PIN)
	if err != nil {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO users (name, username, pin_hash, role_id, active) VALUES (?, ?, ?, ?, 1)`,
		in.Name, in.Username, hash, in.RoleID)
	if err != nil {
		if isUniqueConstraint(err) {
			return nil, domain.NewError("USERNAME_EXISTS", "A user with this username already exists", 409)
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.UserByID(ctx, id)
}

type UpdateUserInput struct {
	Name   string `json:"name"`
	RoleID int64  `json:"role_id"`
	Active bool   `json:"active"`
	PIN    string `json:"pin,omitempty"`
}

func (s *Service) UpdateUser(ctx context.Context, id int64, in UpdateUserInput) (*User, error) {
	if in.PIN != "" {
		if len(in.PIN) < 4 {
			return nil, domain.NewError("INVALID_INPUT", "PIN must be at least 4 digits", 400)
		}
		hash, err := HashPIN(in.PIN)
		if err != nil {
			return nil, err
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE users SET name=?, role_id=?, active=?, pin_hash=? WHERE id=?`,
			in.Name, in.RoleID, boolToInt(in.Active), hash, id); err != nil {
			return nil, err
		}
	} else {
		if _, err := s.db.ExecContext(ctx, `UPDATE users SET name=?, role_id=?, active=? WHERE id=?`,
			in.Name, in.RoleID, boolToInt(in.Active), id); err != nil {
			return nil, err
		}
	}
	return s.UserByID(ctx, id)
}

func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Role{}
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		perms, err := s.permissionsForRole(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		r.Permissions = perms
		out = append(out, r)
	}
	return out, rows.Err()
}

func isUniqueConstraint(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
