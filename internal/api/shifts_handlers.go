package api

import "net/http"

func (a *API) handleOpenShift(w http.ResponseWriter, r *http.Request) {
	registerID, err := requireRegisterID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		OpeningFloat int64 `json:"opening_float"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	perms := permsFromContext(r.Context())
	shift, err := a.Shifts.Open(r.Context(), registerID, user.ID, in.OpeningFloat, perms)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, shift)
}

func (a *API) handleCurrentShift(w http.ResponseWriter, r *http.Request) {
	registerID, err := requireRegisterID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	shift, err := a.Shifts.CurrentForRegister(r.Context(), registerID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, shift)
}

func (a *API) handleGetShift(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	shift, err := a.Shifts.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, shift)
}

func (a *API) handleShiftReport(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	report, err := a.Shifts.Report(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (a *API) handleShiftReportCSV(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	report, err := a.Shifts.Report(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=z-report.csv")
	w.Write(report.CSV())
}

func (a *API) handleCloseShift(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		CountedCash int64  `json:"counted_cash"`
		Notes       string `json:"notes"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	perms := permsFromContext(r.Context())
	shift, err := a.Shifts.Close(r.Context(), id, user.ID, in.CountedCash, in.Notes, perms)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, shift)
}
