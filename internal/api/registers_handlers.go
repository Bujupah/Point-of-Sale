package api

import "net/http"

type registerInfo struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	LocationID   int64  `json:"location_id"`
	LocationName string `json:"location_name"`
}

// handleListRegisters is unauthenticated on purpose: the register picker
// appears on the login screen, before any session exists, mirroring the
// brief's "location/branch selector" in the top bar (§7) and letting a
// cashier pick which physical till they're signing into.
func (a *API) handleListRegisters(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.QueryContext(r.Context(), `
		SELECT r.id, r.name, l.id, l.name FROM registers r JOIN locations l ON l.id = r.location_id ORDER BY l.name, r.name`)
	if err != nil {
		writeError(w, err)
		return
	}
	defer rows.Close()
	out := []registerInfo{}
	for rows.Next() {
		var reg registerInfo
		if err := rows.Scan(&reg.ID, &reg.Name, &reg.LocationID, &reg.LocationName); err != nil {
			writeError(w, err)
			return
		}
		out = append(out, reg)
	}
	writeJSON(w, http.StatusOK, out)
}
