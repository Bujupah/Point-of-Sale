package api

import "net/http"

type taxRateInfo struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	RateBps   int64  `json:"rate_bps"`
	Inclusive bool   `json:"inclusive"`
}

func (a *API) handleListTaxRates(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id, name, rate_bps, inclusive FROM tax_rates ORDER BY rate_bps DESC`)
	if err != nil {
		writeError(w, err)
		return
	}
	defer rows.Close()
	out := []taxRateInfo{}
	for rows.Next() {
		var t taxRateInfo
		var inclusive int
		if err := rows.Scan(&t.ID, &t.Name, &t.RateBps, &inclusive); err != nil {
			writeError(w, err)
			return
		}
		t.Inclusive = inclusive == 1
		out = append(out, t)
	}
	writeJSON(w, http.StatusOK, out)
}
