package api

import (
	"net/http"

	"pos/internal/customers"
)

func (a *API) handleSearchCustomers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := queryInt(r, "limit", 50)
	list, err := a.Customers.Search(r.Context(), q, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) handleCreateCustomer(w http.ResponseWriter, r *http.Request) {
	var in customers.CustomerInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	c, err := a.Customers.Create(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (a *API) handleGetCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	c, err := a.Customers.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *API) handleUpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var in customers.CustomerInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	c, err := a.Customers.Update(r.Context(), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *API) handleCustomerHistory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	history, err := a.Customers.History(r.Context(), id, 50)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, history)
}

func (a *API) handleAddCustomerNote(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Note string `json:"note"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	if err := a.Customers.AddNote(r.Context(), id, in.Note, user.ID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) handleLoyaltyHistory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	history, err := a.Loyalty.History(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, history)
}

func (a *API) handleLoyaltyRedeem(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Points int64  `json:"points"`
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	if err := a.Loyalty.Redeem(r.Context(), id, in.Points, in.Reason, user.ID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) handleLoyaltyAdjust(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Points int64  `json:"points"`
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	if err := a.Loyalty.Adjust(r.Context(), id, in.Points, in.Reason, user.ID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
