package api

import (
	"net/http"

	"pos/internal/audit"
)

func (a *API) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	all, err := a.Settings.All(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, all)
}

func (a *API) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var in map[string]string
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if err := a.Settings.SetAll(r.Context(), in); err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	a.Log.Write(r.Context(), audit.Event{Event: audit.EventSettingsChanged, UserID: &user.ID})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
