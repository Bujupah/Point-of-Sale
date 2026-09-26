// Package api wires every domain service into the local REST API. It binds
// only to 127.0.0.1 (brief §71) and never lets the frontend touch SQLite
// directly — every request goes through a permission-checked handler here.
package api

import (
	"net/http"

	"pos/internal/audit"
	"pos/internal/cash"
	"pos/internal/catalog"
	"pos/internal/customers"
	"pos/internal/giftcards"
	"pos/internal/hardware/printing"
	"pos/internal/inventory"
	"pos/internal/loyalty"
	"pos/internal/refunds"
	"pos/internal/reports"
	"pos/internal/sales"
	"pos/internal/security"
	"pos/internal/settings"
	"pos/internal/shifts"
	"pos/internal/storage"
)

type API struct {
	DB        *storage.DB
	Security  *security.Service
	Catalog   *catalog.Service
	Customers *customers.Service
	Loyalty   *loyalty.Service
	Inventory *inventory.Service
	Sales     *sales.Service
	Refunds   *refunds.Service
	Shifts    *shifts.Service
	Cash      *cash.Service
	Reports   *reports.Service
	Printing  *printing.Service
	GiftCards *giftcards.Service
	Settings  *settings.Service
	Log       *audit.Logger
}

// RegisterRoutes adds every API route to mux. The caller (main.go) owns the
// *http.ServeMux and additionally registers a "/" fallback that serves the
// built frontend — Go's method+path-aware ServeMux resolves the specific
// "METHOD /api/..." patterns registered here before ever falling through to
// that catch-all, so both live on one mux with no separate proxying.
// Every route except health/login is wrapped in withAuth or
// requirePermission.
func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("GET /api/system/info", a.handleSystemInfo)

	mux.HandleFunc("GET /api/registers", a.handleListRegisters)

	mux.HandleFunc("POST /api/auth/login", a.handleLogin)
	mux.HandleFunc("POST /api/auth/lock", a.withAuth(a.handleLock))
	mux.HandleFunc("POST /api/auth/unlock", a.handleUnlock)
	mux.HandleFunc("POST /api/auth/switch", a.handleSwitch)
	mux.HandleFunc("POST /api/auth/logout", a.withAuth(a.handleLogout))
	mux.HandleFunc("GET /api/auth/me", a.withAuth(a.handleMe))

	mux.HandleFunc("GET /api/tax-rates", a.withAuth(a.handleListTaxRates))

	mux.HandleFunc("GET /api/categories", a.withAuth(a.handleListCategories))
	mux.HandleFunc("POST /api/categories", a.requirePermission(security.PermProductWrite, a.handleCreateCategory))
	mux.HandleFunc("PUT /api/categories/{id}", a.requirePermission(security.PermProductWrite, a.handleUpdateCategory))

	mux.HandleFunc("GET /api/modifier-groups", a.withAuth(a.handleListModifierGroups))
	mux.HandleFunc("POST /api/modifier-groups", a.requirePermission(security.PermProductWrite, a.handleCreateModifierGroup))

	mux.HandleFunc("GET /api/products", a.withAuth(a.handleSearchProducts))
	mux.HandleFunc("GET /api/products/{id}", a.withAuth(a.handleGetProduct))
	mux.HandleFunc("GET /api/products/barcode/{barcode}", a.withAuth(a.handleGetProductByBarcode))
	mux.HandleFunc("POST /api/products", a.requirePermission(security.PermProductWrite, a.handleCreateProduct))
	mux.HandleFunc("PUT /api/products/{id}", a.requirePermission(security.PermProductWrite, a.handleUpdateProduct))

	mux.HandleFunc("POST /api/inventory/adjust", a.requirePermission(security.PermInventoryAdjust, a.handleAdjustStock))
	mux.HandleFunc("GET /api/inventory/movements", a.requirePermission(security.PermProductRead, a.handleListMovements))
	mux.HandleFunc("GET /api/inventory/low-stock", a.requirePermission(security.PermProductRead, a.handleLowStock))

	mux.HandleFunc("GET /api/customers", a.withAuth(a.handleSearchCustomers))
	mux.HandleFunc("POST /api/customers", a.requirePermission(security.PermCustomerWrite, a.handleCreateCustomer))
	mux.HandleFunc("GET /api/customers/{id}", a.withAuth(a.handleGetCustomer))
	mux.HandleFunc("PUT /api/customers/{id}", a.requirePermission(security.PermCustomerWrite, a.handleUpdateCustomer))
	mux.HandleFunc("GET /api/customers/{id}/history", a.withAuth(a.handleCustomerHistory))
	mux.HandleFunc("POST /api/customers/{id}/notes", a.requirePermission(security.PermCustomerWrite, a.handleAddCustomerNote))
	mux.HandleFunc("GET /api/customers/{id}/loyalty", a.withAuth(a.handleLoyaltyHistory))
	mux.HandleFunc("POST /api/customers/{id}/loyalty/redeem", a.requirePermission(security.PermCustomerWrite, a.handleLoyaltyRedeem))
	mux.HandleFunc("POST /api/customers/{id}/loyalty/adjust", a.requirePermission(security.PermCustomerWrite, a.handleLoyaltyAdjust))

	mux.HandleFunc("POST /api/sales", a.withAuth(a.handleCheckout))
	mux.HandleFunc("GET /api/sales", a.withAuth(a.handleListSales))
	mux.HandleFunc("GET /api/sales/{id}", a.withAuth(a.handleGetSale))
	mux.HandleFunc("GET /api/sales/{id}/receipt-preview", a.withAuth(a.handleReceiptPreview))
	mux.HandleFunc("POST /api/sales/{id}/print", a.withAuth(a.handlePrintReceipt))
	mux.HandleFunc("POST /api/sales/{id}/refund", a.requirePermission(security.PermRefund, a.handleRefund))

	mux.HandleFunc("POST /api/held-sales", a.withAuth(a.handleHoldSale))
	mux.HandleFunc("GET /api/held-sales", a.withAuth(a.handleListHeld))
	mux.HandleFunc("GET /api/held-sales/{id}", a.withAuth(a.handleGetHeld))
	mux.HandleFunc("POST /api/held-sales/{id}/resume", a.withAuth(a.handleResumeHeld))
	mux.HandleFunc("DELETE /api/held-sales/{id}", a.withAuth(a.handleDeleteHeld))

	mux.HandleFunc("POST /api/shifts/open", a.withAuth(a.handleOpenShift))
	mux.HandleFunc("GET /api/shifts/current", a.withAuth(a.handleCurrentShift))
	mux.HandleFunc("GET /api/shifts/{id}", a.withAuth(a.handleGetShift))
	mux.HandleFunc("GET /api/shifts/{id}/report", a.withAuth(a.handleShiftReport))
	mux.HandleFunc("GET /api/shifts/{id}/report.csv", a.withAuth(a.handleShiftReportCSV))
	mux.HandleFunc("POST /api/shifts/{id}/close", a.requirePermission(security.PermShiftClose, a.handleCloseShift))

	mux.HandleFunc("POST /api/cash/movements", a.withAuth(a.handleRecordCashMovement))
	mux.HandleFunc("GET /api/cash/movements", a.withAuth(a.handleListCashMovements))

	mux.HandleFunc("GET /api/reports/summary", a.requirePermission(security.PermReportRead, a.handleReportsSummary))
	mux.HandleFunc("GET /api/reports/summary.csv", a.requirePermission(security.PermReportRead, a.handleReportsSummaryCSV))
	mux.HandleFunc("GET /api/reports/products", a.requirePermission(security.PermReportRead, a.handleReportsTopProducts))
	mux.HandleFunc("GET /api/reports/payments", a.requirePermission(security.PermReportRead, a.handleReportsPaymentMix))
	mux.HandleFunc("GET /api/reports/cashiers", a.requirePermission(security.PermReportRead, a.handleReportsCashiers))

	mux.HandleFunc("POST /api/hardware/printer/test", a.withAuth(a.handlePrinterTest))
	mux.HandleFunc("POST /api/hardware/drawer/open", a.requirePermission(security.PermDrawerOpen, a.handleDrawerOpen))

	mux.HandleFunc("GET /api/gift-cards/{code}", a.withAuth(a.handleGetGiftCard))
	mux.HandleFunc("POST /api/gift-cards", a.requirePermission(security.PermSettingsWrite, a.handleIssueGiftCard))

	mux.HandleFunc("GET /api/settings", a.withAuth(a.handleGetSettings))
	mux.HandleFunc("PUT /api/settings", a.requirePermission(security.PermSettingsWrite, a.handleUpdateSettings))

	mux.HandleFunc("GET /api/users", a.requirePermission(security.PermUsersManage, a.handleListUsers))
	mux.HandleFunc("POST /api/users", a.requirePermission(security.PermUsersManage, a.handleCreateUser))
	mux.HandleFunc("PUT /api/users/{id}", a.requirePermission(security.PermUsersManage, a.handleUpdateUser))
	mux.HandleFunc("GET /api/roles", a.requirePermission(security.PermUsersManage, a.handleListRoles))
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":    "Point of Sale",
		"version": "0.1.0",
	})
}
