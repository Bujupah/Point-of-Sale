//go:build !windows

package printing

import (
	"context"
	"fmt"
)

// On non-Windows builds (this repo's own Linux dev/test environment
// included) there is no Windows spooler to talk to. Configuring
// WINDOWS_SPOOL here fails closed with a clear error from the printer test
// button rather than silently doing nothing.
type windowsSpoolPrinter struct {
	PrinterName string
}

func newWindowsSpoolPrinter(target string, charsPerLine int) ReceiptPrinter {
	return &windowsSpoolPrinter{PrinterName: target}
}

func (p *windowsSpoolPrinter) unsupported() error {
	return fmt.Errorf("printing: WINDOWS_SPOOL connection to %q is only available on Windows builds", p.PrinterName)
}

func (p *windowsSpoolPrinter) PrintReceipt(ctx context.Context, r Receipt) error { return p.unsupported() }
func (p *windowsSpoolPrinter) PrintTest(ctx context.Context) error              { return p.unsupported() }
func (p *windowsSpoolPrinter) Cut(ctx context.Context) error                    { return p.unsupported() }
func (p *windowsSpoolPrinter) OpenDrawer(ctx context.Context) error             { return p.unsupported() }
