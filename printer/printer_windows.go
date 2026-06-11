package printer

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	winspool         = windows.NewLazySystemDLL("winspool.drv")
	procEnumPrinters = winspool.NewProc("EnumPrintersW")
	procOpenPrinter  = winspool.NewProc("OpenPrinterW")
	procClosePrinter = winspool.NewProc("ClosePrinter")
	procStartDoc     = winspool.NewProc("StartDocPrinterW")
	procEndDoc       = winspool.NewProc("EndDocPrinter")
	procStartPage    = winspool.NewProc("StartPagePrinter")
	procEndPage      = winspool.NewProc("EndPagePrinter")
	procWrite        = winspool.NewProc("WritePrinter")
)

// PRINTER_INFO_4 — minimal struct sufficient to enumerate printer names.
type printerInfo4 struct {
	PrinterName *uint16
	ServerName  *uint16
	Attributes  uint32
}

type docInfo1 struct {
	DocName    *uint16
	OutputFile *uint16
	Datatype   *uint16
}

const printerEnumLocal = 0x00000002

// List returns the names of all locally installed printers.
func List() ([]string, error) {
	var needed, returned uint32

	procEnumPrinters.Call(
		printerEnumLocal, 0, 4, 0, 0,
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)
	if needed == 0 {
		return nil, nil
	}

	buf := make([]byte, needed)
	r, _, err := procEnumPrinters.Call(
		printerEnumLocal, 0, 4,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(needed),
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)
	if r == 0 {
		return nil, fmt.Errorf("EnumPrinters: %w", err)
	}

	infoSize := unsafe.Sizeof(printerInfo4{})
	names := make([]string, 0, returned)
	for i := uintptr(0); i < uintptr(returned); i++ {
		info := (*printerInfo4)(unsafe.Pointer(&buf[i*infoSize]))
		names = append(names, windows.UTF16PtrToString(info.PrinterName))
	}
	return names, nil
}

// Detect returns the first locally installed printer.
func Detect() (string, error) {
	names, err := List()
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no printers found")
	}
	return names[0], nil
}

// Print sends raw ESC/POS bytes to the named printer.
func Print(name string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}

	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}

	var handle uintptr
	r, _, err := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(&handle)),
		0,
	)
	if r == 0 {
		return fmt.Errorf("OpenPrinter: %w", err)
	}
	defer procClosePrinter.Call(handle)

	docName, _ := windows.UTF16PtrFromString("ESC/POS")
	datatype, _ := windows.UTF16PtrFromString("RAW")
	doc := docInfo1{DocName: docName, Datatype: datatype}

	r, _, err = procStartDoc.Call(handle, 1, uintptr(unsafe.Pointer(&doc)))
	if r == 0 {
		return fmt.Errorf("StartDocPrinter: %w", err)
	}
	defer procEndDoc.Call(handle)

	r, _, err = procStartPage.Call(handle)
	if r == 0 {
		return fmt.Errorf("StartPagePrinter: %w", err)
	}
	defer procEndPage.Call(handle)

	var written uint32
	r, _, err = procWrite.Call(
		handle,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&written)),
	)
	if r == 0 {
		return fmt.Errorf("WritePrinter: %w", err)
	}

	return nil
}
