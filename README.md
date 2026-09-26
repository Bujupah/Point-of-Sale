# Point of Sale

A standalone, offline-first Point of Sale desktop application: Go backend, SQLite storage, React/TypeScript UI, targeting both Windows XP SP3 and modern Windows.

## Status

**Phase 0 — compatibility & architecture.** No application code exists yet. Before writing the vertical slice described in the architecture doc, read:

1. [`docs/00-xp-compatibility-report.md`](docs/00-xp-compatibility-report.md) — the technology decisions required to support Windows XP SP3, and the open risks that must be proven on real/virtual XP hardware first.
2. [`docs/01-architecture-and-repo-structure.md`](docs/01-architecture-and-repo-structure.md) — the resulting system architecture and proposed repository layout.

## Non-negotiable stack

- **Backend:** Go (dual toolchain: Go 1.10.8 for the XP build, current Go for the modern build)
- **Database:** SQLite (`mattn/go-sqlite3`)
- **Frontend:** React + TypeScript, compiled to two bundles (legacy CEF-XP tier and modern WebView2 tier) from one source tree
- **Architecture:** local desktop app, offline-first, local REST API bound to `127.0.0.1`, modular monolith

See the architecture doc for the full rationale and the reasons behind every version pin.
