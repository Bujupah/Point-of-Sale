// Command pos is the POS backend: it opens SQLite, runs migrations, seeds
// default data on first run, and serves both the local REST API and the
// built frontend from a single process bound to 127.0.0.1 (brief §71,
// §89's startup flow).
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"pos/internal/api"
	"pos/internal/audit"
	"pos/internal/cash"
	"pos/internal/catalog"
	"pos/internal/config"
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

func main() {
	cfg, err := config.Load("./config.json")
	if err != nil {
		log.Fatalf("startup: load config: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.Logging.Path), 0o755); err != nil {
		log.Fatalf("startup: create log directory: %v", err)
	}
	logFile, err := os.OpenFile(cfg.Logging.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("startup: could not open log file %s, logging to stdout only: %v", cfg.Logging.Path, err)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
	}
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	db, err := storage.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("startup: open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(cfg.Migrations); err != nil {
		log.Fatalf("startup: run migrations: %v", err)
	}

	ctx := context.Background()

	auditLogger := audit.NewLogger(db)
	securitySvc := security.NewService(db, auditLogger)
	settingsSvc := settings.NewService(db)

	if err := seed(ctx, db, securitySvc, settingsSvc); err != nil {
		log.Fatalf("startup: seed database: %v", err)
	}

	catalogSvc := catalog.NewService(db)
	customersSvc := customers.NewService(db)
	loyaltySvc := loyalty.NewService(db)
	inventorySvc := inventory.NewService(db, auditLogger)
	salesSvc := sales.NewService(db, catalogSvc, settingsSvc, auditLogger)
	refundsSvc := refunds.NewService(db, auditLogger)
	shiftsSvc := shifts.NewService(db, auditLogger)
	cashSvc := cash.NewService(db, auditLogger)
	reportsSvc := reports.NewService(db)
	printingSvc := printing.NewService(db, settingsSvc)
	giftCardsSvc := giftcards.NewService(db)

	app := &api.API{
		DB: db, Security: securitySvc, Catalog: catalogSvc, Customers: customersSvc, Loyalty: loyaltySvc,
		Inventory: inventorySvc, Sales: salesSvc, Refunds: refundsSvc, Shifts: shiftsSvc, Cash: cashSvc,
		Reports: reportsSvc, Printing: printingSvc, GiftCards: giftCardsSvc, Settings: settingsSvc, Log: auditLogger,
	}

	if err := os.MkdirAll(cfg.Assets, 0o755); err != nil {
		log.Fatalf("startup: create assets directory: %v", err)
	}

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	// Locally stored product/UI images (brief §49): served straight off
	// disk, never hot-linked, so the catalog keeps working with no network
	// at all once a machine has been set up. Mounted at "/media/", not
	// "/assets/" — the built frontend's own JS/CSS bundle lives under
	// frontend/dist/assets/ (Vite's default), and "/assets/" would shadow it.
	mux.Handle("/media/", http.StripPrefix("/media/", http.FileServer(http.Dir(cfg.Assets))))
	mux.Handle("/", frontendHandler(cfg.Frontend))

	addr := cfg.Server.Host + ":" + strconv.Itoa(cfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("pos: listening on http://%s (loopback only)", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("startup: server failed: %v", err)
	}
}

// frontendHandler serves the built SPA with client-side routing fallback:
// unknown paths (not /api, not an existing static file) return index.html
// so React Router-style navigation works on refresh. If the frontend hasn't
// been built yet, it serves a small placeholder page instead of failing the
// whole process — the backend is independently useful during development.
func frontendHandler(dir string) http.Handler {
	indexPath := filepath.Join(dir, "index.html")
	fileServer := http.FileServer(http.Dir(dir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(indexPath); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(`<!doctype html><html><body style="font-family:sans-serif;padding:2rem">
				<h1>POS backend is running</h1>
				<p>Frontend build not found at ` + dir + `. Run the frontend build, or use the API directly.</p>
				<p><a href="/health">/health</a></p>
				</body></html>`))
			return
		}
		requestedPath := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(requestedPath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, indexPath)
	})
}
