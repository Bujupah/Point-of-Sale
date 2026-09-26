package api

import (
	"net/http"

	"pos/internal/domain"
)

func (a *API) handleGetGiftCard(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	gc, err := a.GiftCards.ByCode(r.Context(), code)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gc)
}

func (a *API) handleIssueGiftCard(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Value int64 `json:"value"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if in.Value <= 0 {
		writeError(w, domain.NewError("INVALID_INPUT", "Gift card value must be positive", 400))
		return
	}
	gc, err := a.GiftCards.Issue(r.Context(), domain.Money(in.Value))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, gc)
}
