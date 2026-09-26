package printing

import (
	"context"
	"log"
	"sync"
)

// SimulatedPrinter "prints" by logging and keeping the last receipt in
// memory. It is the default on any machine with no printer configured, and
// the only implementation used in this repository's own test/dev
// environment (no physical printer or Windows spooler available there).
type SimulatedPrinter struct {
	Name         string
	CharsPerLine int

	mu   sync.Mutex
	last string
}

func (p *SimulatedPrinter) PrintReceipt(ctx context.Context, r Receipt) error {
	preview := AsciiPreview(r, p.CharsPerLine)
	p.mu.Lock()
	p.last = preview
	p.mu.Unlock()
	log.Printf("printing: [SIMULATED %s] receipt %s\n%s", p.Name, r.ReceiptNumber, preview)
	return nil
}

func (p *SimulatedPrinter) PrintTest(ctx context.Context) error {
	log.Printf("printing: [SIMULATED %s] test page OK", p.Name)
	return nil
}

func (p *SimulatedPrinter) Cut(ctx context.Context) error {
	log.Printf("printing: [SIMULATED %s] cut", p.Name)
	return nil
}

func (p *SimulatedPrinter) OpenDrawer(ctx context.Context) error {
	log.Printf("printing: [SIMULATED %s] drawer kick", p.Name)
	return nil
}

// LastPreview returns the most recently "printed" receipt text, useful for
// tests and for a dev-mode UI panel that shows what would have printed.
func (p *SimulatedPrinter) LastPreview() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.last
}
