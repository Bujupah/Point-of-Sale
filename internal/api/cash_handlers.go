package api

import (
	"net/http"

	"pos/internal/domain"
)

func (a *API) handleRecordCashMovement(w http.ResponseWriter, r *http.Request) {
	registerID, err := requireRegisterID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Type   string `json:"type"`
		Amount int64  `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	perms := permsFromContext(r.Context())
	movement, err := a.Cash.Record(r.Context(), registerID, user.ID, perms, in.Type, in.Amount, in.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, movement)
}

func (a *API) handleListCashMovements(w http.ResponseWriter, r *http.Request) {
	shiftID := queryInt64Ptr(r, "shift_id")
	if shiftID == nil {
		writeError(w, domain.NewError("INVALID_INPUT", "shift_id query parameter is required", 400))
		return
	}
	list, err := a.Cash.List(r.Context(), *shiftID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
