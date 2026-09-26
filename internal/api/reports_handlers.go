package api

import (
	"net/http"

	"pos/internal/reports"
)

func dateRangeFromQuery(r *http.Request) reports.DateRange {
	q := r.URL.Query()
	return reports.DateRange{From: q.Get("date_from"), To: q.Get("date_to")}
}

func (a *API) handleReportsSummary(w http.ResponseWriter, r *http.Request) {
	sum, err := a.Reports.Summary(r.Context(), dateRangeFromQuery(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (a *API) handleReportsSummaryCSV(w http.ResponseWriter, r *http.Request) {
	dr := dateRangeFromQuery(r)
	sum, err := a.Reports.Summary(r.Context(), dr)
	if err != nil {
		writeError(w, err)
		return
	}
	top, err := a.Reports.TopProducts(r.Context(), dr, 50)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=report-summary.csv")
	w.Write(reports.SummaryCSV(sum, top))
}

func (a *API) handleReportsTopProducts(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 20)
	top, err := a.Reports.TopProducts(r.Context(), dateRangeFromQuery(r), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, top)
}

func (a *API) handleReportsPaymentMix(w http.ResponseWriter, r *http.Request) {
	mix, err := a.Reports.PaymentMix(r.Context(), dateRangeFromQuery(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mix)
}

func (a *API) handleReportsCashiers(w http.ResponseWriter, r *http.Request) {
	metrics, err := a.Reports.CashierMetrics(r.Context(), dateRangeFromQuery(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}
