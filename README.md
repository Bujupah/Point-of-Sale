# Point of Sale

A standalone, offline-first Point of Sale desktop application: Go backend, SQLite storage, React/TypeScript UI.

## Status

**Core application implemented and working end-to-end** on the modern-Windows/desktop tier described in the architecture doc: auth & permissions, catalog (categories, products, barcodes, variants, modifiers), cart & checkout, split/multi-tender payments, held sales, refunds, shifts & cash management with Z-reports, customers & loyalty, inventory, reports, receipt printing (ESC/POS + simulated + Windows RAW spool), and a full React UI (Sell, Orders, Customers, Cash Register, Products, Reports, Settings) with i18n (Spanish/English/Arabic, full RTL) and keyboard shortcuts.

Read first:

1. [`docs/00-xp-compatibility-report.md`](docs/00-xp-compatibility-report.md) — the Windows XP SP3 compatibility decisions and open risks.
2. [`docs/01-architecture-and-repo-structure.md`](docs/01-architecture-and-repo-structure.md) — the system architecture this implementation follows.

**What's not done yet:** this build uses the modern Go/toolchain path (the "Windows 7+/10/11 tier"). The XP-specific build (Go 1.10.8, CEF-XP shell, go:embed-free asset loading) described in the compatibility report has not been produced or tested on real/virtual XP hardware — that remains the Phase 0 vertical-slice validation called for in the architecture doc before an XP artifact ships. The Go backend itself was written without post-1.10 language features where practical (no generics in the domain code) to keep that door open, but this has not been verified by actually building with the old toolchain.

## Running it

Requires Go 1.23+, Node 20+, and a C compiler (CGO is required for `mattn/go-sqlite3`).

```sh
# Backend (serves the API on 127.0.0.1:17831, and the built frontend if present)
go run ./cmd/pos

# Frontend, for development (proxies /api to the Go server above)
cd frontend
npm install
npm run dev

# Frontend, for production (Go then serves frontend/dist directly)
cd frontend
npm run build
```

On first run the backend seeds:
- Roles: **Admin**, **Manager**, **Cashier**, with the permission sets in `cmd/pos/seed.go`.
- Two demo users: **admin / 1234** (Admin) and **cashier / 1111** (Cashier) — change these PINs immediately in a real deployment.
- One location, one register, two tax rates, and a small demo menu (sample data only, never load-bearing for business logic).

## Non-negotiable stack

- **Backend:** Go, modular monolith (`internal/<domain>` packages), local REST API bound to `127.0.0.1` only.
- **Database:** SQLite (`mattn/go-sqlite3`), WAL mode, integer-minor-units money (`internal/domain.Money`), migrations in `migrations/*.sql`.
- **Frontend:** React + TypeScript + Vite, no heavy UI framework, i18n with full RTL support, keyboard-first workflow.
- **Architecture:** offline-first; SQLite is the single source of truth; React never owns business-critical data; hardware (printer/drawer) sits behind the `internal/hardware/printing.ReceiptPrinter` interface so a sale is never blocked or invalidated by a printer failure.

See the architecture doc for the full rationale and the reasons behind every version pin.
