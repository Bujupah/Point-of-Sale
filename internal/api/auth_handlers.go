package api

import (
	"net/http"

	"pos/internal/domain"
)

type loginRequest struct {
	Username   string `json:"username"`
	PIN        string `json:"pin"`
	RegisterID *int64 `json:"register_id,omitempty"`
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := a.Security.Login(r.Context(), req.Username, req.PIN, req.RegisterID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) handleLock(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromContext(r.Context())
	if err := a.Security.Lock(r.Context(), sess.Token); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"locked": true})
}

type unlockRequest struct {
	PIN string `json:"pin"`
}

func (a *API) handleUnlock(w http.ResponseWriter, r *http.Request) {
	token := tokenFromRequest(r)
	if token == "" {
		writeError(w, domain.ErrUnauthorized)
		return
	}
	var req unlockRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := a.Security.Unlock(r.Context(), token, req.PIN)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type switchRequest struct {
	Username string `json:"username"`
	PIN      string `json:"pin"`
}

func (a *API) handleSwitch(w http.ResponseWriter, r *http.Request) {
	token := tokenFromRequest(r)
	if token == "" {
		writeError(w, domain.ErrUnauthorized)
		return
	}
	var req switchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := a.Security.SwitchCashier(r.Context(), token, req.Username, req.PIN)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) handleLogout(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromContext(r.Context())
	if err := a.Security.Logout(r.Context(), sess.Token); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) handleMe(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	sess := sessionFromContext(r.Context())
	perms := permsFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"user": user, "permissions": perms, "locked": sess.Locked, "register_id": sess.RegisterID,
	})
}
