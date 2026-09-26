package api

import (
	"net/http"

	"pos/internal/security"
)

func (a *API) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.Security.ListUsers(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (a *API) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var in security.CreateUserInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	u, err := a.Security.CreateUser(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (a *API) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var in security.UpdateUserInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	u, err := a.Security.UpdateUser(r.Context(), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (a *API) handleListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := a.Security.ListRoles(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, roles)
}
