package printing

import (
	"context"
	"fmt"
	"os"
)

// SerialPrinter writes raw ESC/POS bytes directly to a device path — a COM
// or LPT port on Windows (opened as "COM1", "\\.\COM1", "LPT1", etc., which
// Go's os.OpenFile supports natively on that platform), or a serial device
// path on Linux (useful for local development against a real serial
// printer or a loopback device). This covers brief §22's COM/LPT case for
// printers with no installed Windows driver; printers that do have one
// should use WINDOWS_SPOOL instead so multiple apps can share the queue.
type SerialPrinter struct {
	Path         string
	CharsPerLine int
}

func (p *SerialPrinter) open() (*os.File, error) {
	if p.Path == "" {
		return nil, fmt.Errorf("printing: no COM/LPT path configured")
	}
	f, err := os.OpenFile(p.Path, os.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("printing: open %s: %w", p.Path, err)
	}
	return f, nil
}

func (p *SerialPrinter) write(b []byte) error {
	f, err := p.open()
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(b)
	return err
}

func (p *SerialPrinter) PrintReceipt(ctx context.Context, r Receipt) error {
	return p.write(BuildESCPOS(r, p.CharsPerLine))
}

func (p *SerialPrinter) PrintTest(ctx context.Context) error {
	return p.write([]byte(escInit + "Printer test OK\n\n\n" + escCutPartial))
}

func (p *SerialPrinter) Cut(ctx context.Context) error {
	return p.write([]byte(escCutPartial))
}

func (p *SerialPrinter) OpenDrawer(ctx context.Context) error {
	return p.write(DrawerKickCommand())
}
