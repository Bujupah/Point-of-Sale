package security

// Permission codes. The backend enforces these on every route that needs
// them; the frontend hiding a button is a UX nicety, never the real gate.
const (
	PermSaleCreate  = "sale.create"
	PermSaleHold    = "sale.hold"
	PermSaleCancel  = "sale.cancel"
	PermRefund      = "refund.create"

	PermDiscountApply    = "discount.apply"
	PermDiscountOverride = "discount.override"
	PermPriceOverride    = "price.override"

	PermDrawerOpen = "drawer.open"

	PermCashIn  = "cash.in"
	PermCashOut = "cash.out"

	PermShiftOpen  = "shift.open"
	PermShiftClose = "shift.close"

	PermProductRead = "product.read"
	PermProductWrite = "product.write"

	PermInventoryAdjust = "inventory.adjust"

	PermCustomerWrite = "customer.write"

	PermReportRead = "report.read"

	PermSettingsWrite = "settings.write"
	PermUsersManage   = "users.manage"

	PermOpenItem = "sale.open_item"
)

// AllPermissions lists every known permission code, used to seed the
// permissions table and the built-in Admin role.
var AllPermissions = []string{
	PermSaleCreate, PermSaleHold, PermSaleCancel, PermRefund,
	PermDiscountApply, PermDiscountOverride, PermPriceOverride,
	PermDrawerOpen, PermCashIn, PermCashOut,
	PermShiftOpen, PermShiftClose,
	PermProductRead, PermProductWrite,
	PermInventoryAdjust, PermCustomerWrite,
	PermReportRead, PermSettingsWrite, PermUsersManage,
	PermOpenItem,
}

// CashierPermissions is the default permission set for the built-in
// "Cashier" role: everything needed to sell, nothing that changes catalog,
// settings, or requires manager approval.
var CashierPermissions = []string{
	PermSaleCreate, PermSaleHold, PermRefund,
	PermDiscountApply, PermDrawerOpen, PermCashIn, PermCashOut,
	PermShiftOpen, PermShiftClose, PermProductRead, PermCustomerWrite,
	PermReportRead,
}
