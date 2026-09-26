package api

import "net/http"

func (a *API) handlePrinterTest(w http.ResponseWriter, r *http.Request) {
	if err := a.Printing.TestPrint(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) handleDrawerOpen(w http.ResponseWriter, r *http.Request) {
	if err := a.Printing.OpenDrawer(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
