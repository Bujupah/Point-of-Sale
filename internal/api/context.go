package api

import (
	"context"
	"net/http"
	"strings"

	"pos/internal/domain"
	"pos/internal/security"
)

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxPerms
	ctxSession
)

func tokenFromRequest(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.Header.Get("X-Session-Token")
}

// withAuth resolves the bearer token into a user + permission set and stores
// them on the request context. Routes that allow anonymous access (login,
// health) do not wrap with this; every other route does.
func (a *API) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := tokenFromRequest(r)
		sess, user, perms, err := a.Security.Authenticate(r.Context(), token)
		if err != nil {
			writeError(w, domain.ErrUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUser, user)
		ctx = context.WithValue(ctx, ctxPerms, perms)
		ctx = context.WithValue(ctx, ctxSession, sess)
		next(w, r.WithContext(ctx))
	}
}

// requirePermission wraps withAuth and additionally rejects requests whose
// session lacks the given permission code. This is the actual enforcement
// point — the frontend hiding a button is never sufficient on its own.
func (a *API) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	return a.withAuth(func(w http.ResponseWriter, r *http.Request) {
		perms, _ := r.Context().Value(ctxPerms).([]string)
		if !security.HasPermission(perms, code) {
			writeError(w, domain.ErrForbidden)
			return
		}
		next(w, r)
	})
}

func userFromContext(ctx context.Context) *security.User {
	u, _ := ctx.Value(ctxUser).(*security.User)
	return u
}

func sessionFromContext(ctx context.Context) *security.Session {
	s, _ := ctx.Value(ctxSession).(*security.Session)
	return s
}

func permsFromContext(ctx context.Context) []string {
	p, _ := ctx.Value(ctxPerms).([]string)
	return p
}

func requireRegisterID(r *http.Request) (int64, error) {
	sess := sessionFromContext(r.Context())
	if sess == nil || sess.RegisterID == nil {
		return 0, domain.NewError("NO_REGISTER_SELECTED", "This session is not bound to a register", 400)
	}
	return *sess.RegisterID, nil
}
