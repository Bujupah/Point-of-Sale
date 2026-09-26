package api

import (
	"net/http"

	"pos/internal/refunds"
	"pos/internal/sales"
)

func (a *API) handleCheckout(w http.ResponseWriter, r *http.Request) {
	registerID, err := requireRegisterID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in sales.CheckoutInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	perms := permsFromContext(r.Context())
	sale, err := a.Sales.Checkout(r.Context(), registerID, user.ID, perms, in)
	if err != nil {
		writeError(w, err)
		return
	}

	// Printing and drawer-open happen only after the sale has committed
	// (brief §30/§80): a printer failure never rolls back or hides the
	// already-successful sale, it's just reported alongside it.
	printErr := a.Printing.PrintSale(r.Context(), sale)
	resp := map[string]any{"sale": sale}
	if printErr != nil {
		resp["print_error"] = printErr.Error()
	} else {
		_ = a.Printing.OpenDrawer(r.Context())
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (a *API) handleListSales(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := sales.ListFilter{
		ReceiptNumber: q.Get("receipt_number"),
		CustomerID:    queryInt64Ptr(r, "customer_id"),
		CashierID:     queryInt64Ptr(r, "cashier_id"),
		PaymentMethod: q.Get("payment_method"),
		Status:        q.Get("status"),
		DateFrom:      q.Get("date_from"),
		DateTo:        q.Get("date_to"),
		Limit:         queryInt(r, "limit", 50),
		Offset:        queryInt(r, "offset", 0),
	}
	list, err := a.Sales.ListSales(r.Context(), f)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) handleGetSale(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	sale, err := a.Sales.GetSale(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sale)
}

func (a *API) handleReceiptPreview(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	sale, err := a.Sales.GetSale(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	preview, err := a.Printing.PreviewSale(r.Context(), sale)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"preview": preview})
}

func (a *API) handlePrintReceipt(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	sale, err := a.Sales.GetSale(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := a.Printing.PrintSale(r.Context(), sale); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) handleRefund(w http.ResponseWriter, r *http.Request) {
	saleID, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	registerID, err := requireRegisterID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in refunds.Input
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	in.SaleID = saleID
	user := userFromContext(r.Context())
	perms := permsFromContext(r.Context())
	refund, err := a.Refunds.Create(r.Context(), registerID, user.ID, perms, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, refund)
}

func (a *API) handleHoldSale(w http.ResponseWriter, r *http.Request) {
	registerID, err := requireRegisterID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in sales.HoldInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	perms := permsFromContext(r.Context())
	held, err := a.Sales.Hold(r.Context(), registerID, user.ID, perms, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, held)
}

func (a *API) handleListHeld(w http.ResponseWriter, r *http.Request) {
	registerID, err := requireRegisterID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	list, err := a.Sales.ListHeld(r.Context(), registerID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) handleGetHeld(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	held, err := a.Sales.GetHeld(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, held)
}

// handleResumeHeld returns the held sale's detail for the frontend to load
// back into the active cart, then deletes the held row — resuming a sale
// removes it from the parked list.
func (a *API) handleResumeHeld(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	held, err := a.Sales.GetHeld(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	if err := a.Sales.DeleteHeld(r.Context(), id, user.ID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, held)
}

func (a *API) handleDeleteHeld(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	if err := a.Sales.DeleteHeld(r.Context(), id, user.ID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
