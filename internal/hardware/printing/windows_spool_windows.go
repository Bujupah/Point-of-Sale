//go:build windows

package printing

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// windowsSpoolPrinter sends raw ESC/POS bytes through the Windows print
// spooler in RAW mode (OpenPrinter → StartDocPrinter "RAW" → WritePrinter →
// EndPagePrinter/EndDocPrinter/ClosePrinter), per brief §21. This is the
// transport almost all USB thermal printers use, since they install as a
// normal Windows printer queue regardless of physical connection.
type windowsSpoolPrinter struct {
	PrinterName  string
	CharsPerLine int
}

func newWindowsSpoolPrinter(target string, charsPerLine int) ReceiptPrinter {
	return &windowsSpoolPrinter{PrinterName: target, CharsPerLine: charsPerLine}
}

var (
	winspool             = windows.NewLazySystemDLL("winspool.drv")
	procOpenPrinterW     = winspool.NewProc("OpenPrinterW")
	procClosePrinter     = winspool.NewProc("ClosePrinter")
	procStartDocPrinterW = winspool.NewProc("StartDocPrinterW")
	procEndDocPrinter    = winspool.NewProc("EndDocPrinter")
	procStartPagePrinter = winspool.NewProc("StartPagePrinter")
	procEndPagePrinter   = winspool.NewProc("EndPagePrinter")
	procWritePrinter     = winspool.NewProc("WritePrinter")
)

type docInfo1 struct {
	DocName    *uint16
	OutputFile *uint16
	Datatype   *uint16
}

func (p *windowsSpoolPrinter) writeRaw(jobName string, data []byte) error {
	if p.PrinterName == "" {
		return fmt.Errorf("printing: no Windows printer name configured")
	}
	namePtr, err := syscall.UTF16PtrFromString(p.PrinterName)
	if err != nil {
		return err
	}
	var handle syscall.Handle
	r1, _, err := procOpenPrinterW.Call(uintptr(unsafe.Pointer(namePtr)), uintptr(unsafe.Pointer(&handle)), 0)
	if r1 == 0 {
		return fmt.Errorf("printing: OpenPrinter %s: %w", p.PrinterName, err)
	}
	defer procClosePrinter.Call(uintptr(handle))

	docNamePtr, _ := syscall.UTF16PtrFromString(jobName)
	dataTypePtr, _ := syscall.UTF16PtrFromString("RAW")
	info := docInfo1{DocName: docNamePtr, OutputFile: nil, Datatype: dataTypePtr}
	r1, _, err = procStartDocPrinterW.Call(uintptr(handle), 1, uintptr(unsafe.Pointer(&info)))
	if r1 == 0 {
		return fmt.Errorf("printing: StartDocPrinter: %w", err)
	}
	defer procEndDocPrinter.Call(uintptr(handle))

	r1, _, err = procStartPagePrinter.Call(uintptr(handle))
	if r1 == 0 {
		return fmt.Errorf("printing: StartPagePrinter: %w", err)
	}
	defer procEndPagePrinter.Call(uintptr(handle))

	var written uint32
	r1, _, err = procWritePrinter.Call(uintptr(handle), uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), uintptr(unsafe.Pointer(&written)))
	if r1 == 0 {
		return fmt.Errorf("printing: WritePrinter: %w", err)
	}
	if int(written) != len(data) {
		return fmt.Errorf("printing: WritePrinter wrote %d of %d bytes", written, len(data))
	}
	return nil
}

func (p *windowsSpoolPrinter) PrintReceipt(ctx context.Context, r Receipt) error {
	return p.writeRaw("POS Receipt "+r.ReceiptNumber, BuildESCPOS(r, p.CharsPerLine))
}

func (p *windowsSpoolPrinter) PrintTest(ctx context.Context) error {
	return p.writeRaw("POS Printer Test", []byte(escInit+"Printer test OK\n\n\n"+escCutPartial))
}

func (p *windowsSpoolPrinter) Cut(ctx context.Context) error {
	return p.writeRaw("POS Cut", []byte(escCutPartial))
}

func (p *windowsSpoolPrinter) OpenDrawer(ctx context.Context) error {
	return p.writeRaw("POS Drawer Kick", DrawerKickCommand())
}
