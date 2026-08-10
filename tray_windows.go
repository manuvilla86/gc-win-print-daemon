//go:build windows

package main

import (
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"printbridge/printer"
)

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modShell32  = windows.NewLazySystemDLL("shell32.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modGdi32    = windows.NewLazySystemDLL("gdi32.dll")

	procRegisterClassExW   = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW    = modUser32.NewProc("CreateWindowExW")
	procDefWindowProcW     = modUser32.NewProc("DefWindowProcW")
	procGetMessageW        = modUser32.NewProc("GetMessageW")
	procTranslateMessage   = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW   = modUser32.NewProc("DispatchMessageW")
	procPostQuitMessage    = modUser32.NewProc("PostQuitMessage")
	procCreatePopupMenu    = modUser32.NewProc("CreatePopupMenu")
	procAppendMenuW        = modUser32.NewProc("AppendMenuW")
	procTrackPopupMenu     = modUser32.NewProc("TrackPopupMenu")
	procDestroyMenu        = modUser32.NewProc("DestroyMenu")
	procGetCursorPos       = modUser32.NewProc("GetCursorPos")
	procSetForeground      = modUser32.NewProc("SetForegroundWindow")
	procPostMessageW       = modUser32.NewProc("PostMessageW")
	procCreateIconIndirect = modUser32.NewProc("CreateIconIndirect")

	procShellNotifyIconW = modShell32.NewProc("Shell_NotifyIconW")
	procGetModuleHandleW = modKernel32.NewProc("GetModuleHandleW")

	procCreateCompatibleDC = modGdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = modGdi32.NewProc("DeleteDC")
	procCreateDIBSection   = modGdi32.NewProc("CreateDIBSection")
	procCreateBitmap       = modGdi32.NewProc("CreateBitmap")
)

const (
	wmTray    = uintptr(0x0401) // WM_USER+1
	wmCommand = uintptr(0x0111)
	wmRBUp    = uintptr(0x0205)

	nimAdd    = uintptr(0)
	nimModify = uintptr(1)
	nimDelete = uintptr(2)

	nifMessage = uint32(0x01)
	nifIcon    = uint32(0x02)
	nifTip     = uint32(0x04)

	cmdQuit = uintptr(1)
	mfStr   = uintptr(0)
)

type NOTIFYICONDATA struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type POINT struct{ X, Y int32 }

type bitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type bitmapInfo struct {
	BmiHeader bitmapInfoHeader
	BmiColors [1]uint32
}

type iconInfo struct {
	FIcon    uint32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

var (
	mainHwnd  uintptr
	iconGreen uintptr
	iconGray  uintptr
	// kept alive for the process lifetime so the icon bitmaps remain valid
	iconBitmaps []uintptr
)

// makeColorIcon creates a solid 16×16 HICON from RGB values using GDI.
// The bitmaps backing the icon are stored in iconBitmaps (never freed).
func makeColorIcon(r, g, b byte) uintptr {
	const sz = 16

	hdc, _, _ := procCreateCompatibleDC.Call(0)
	if hdc == 0 {
		return 0
	}

	bi := bitmapInfo{
		BmiHeader: bitmapInfoHeader{
			BiSize:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			BiWidth:    sz,
			BiHeight:   -sz, // top-down
			BiPlanes:   1,
			BiBitCount: 32,
		},
	}

	var ppvBits uintptr
	hBmpColor, _, _ := procCreateDIBSection.Call(
		hdc,
		uintptr(unsafe.Pointer(&bi)),
		0, // DIB_RGB_COLORS
		uintptr(unsafe.Pointer(&ppvBits)),
		0, 0,
	)
	procDeleteDC.Call(hdc)

	if hBmpColor == 0 || ppvBits == 0 {
		return 0
	}

	// Fill pixels: BGRA, fully opaque
	pix := (*[sz * sz * 4]byte)(unsafe.Pointer(ppvBits))
	for i := 0; i < sz*sz; i++ {
		pix[i*4+0] = b
		pix[i*4+1] = g
		pix[i*4+2] = r
		pix[i*4+3] = 255
	}

	// 1bpp AND mask — all zeros = draw from color bitmap
	maskStride := (sz + 15) / 16 * 2 // word-aligned row bytes
	mask := make([]byte, maskStride*sz)
	hBmpMask, _, _ := procCreateBitmap.Call(sz, sz, 1, 1, uintptr(unsafe.Pointer(&mask[0])))
	runtime.KeepAlive(mask)

	if hBmpMask == 0 {
		return 0
	}

	ii := iconInfo{FIcon: 1, HbmMask: hBmpMask, HbmColor: hBmpColor}
	hIcon, _, _ := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))

	// Keep bitmap handles alive for the process lifetime
	iconBitmaps = append(iconBitmaps, hBmpColor, hBmpMask)
	return hIcon
}

func trayWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmTray:
		if lParam == wmRBUp {
			showTrayMenu(hwnd)
		}
		return 0
	case wmCommand:
		if wParam&0xFFFF == cmdQuit {
			nid := NOTIFYICONDATA{
				CbSize: uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
				HWnd:   hwnd,
				UID:    1,
			}
			procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
			procPostQuitMessage.Call(0)
		}
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}

func showTrayMenu(hwnd uintptr) {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	label, _ := syscall.UTF16PtrFromString("Salir")
	procAppendMenuW.Call(hMenu, mfStr, cmdQuit, uintptr(unsafe.Pointer(label)))

	procSetForeground.Call(hwnd)
	procTrackPopupMenu.Call(hMenu, 0, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	procPostMessageW.Call(hwnd, 0, 0, 0)
}

func updateTray(icon uintptr, tip string) {
	if mainHwnd == 0 {
		return
	}
	nid := NOTIFYICONDATA{
		CbSize: uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		HWnd:   mainHwnd,
		UID:    1,
		UFlags: nifIcon | nifTip,
		HIcon:  icon,
	}
	t, _ := syscall.UTF16FromString(tip)
	copy(nid.SzTip[:], t)
	procShellNotifyIconW.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
}

func pollPrinterStatus() {
	for {
		name, err := printer.Detect()
		if err != nil || name == "" {
			updateTray(iconGray, "PrintBridge — Sin impresora")
		} else {
			updateTray(iconGreen, "PrintBridge — "+name)
		}
		time.Sleep(5 * time.Second)
	}
}

func runTray() {
	runtime.LockOSThread()

	hInst, _, _ := procGetModuleHandleW.Call(0)

	iconGreen = makeColorIcon(34, 197, 94)
	iconGray = makeColorIcon(156, 163, 175)

	clsName, _ := syscall.UTF16PtrFromString("PBTrayWnd")
	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		LpfnWndProc:   syscall.NewCallback(trayWndProc),
		HInstance:     hInst,
		LpszClassName: clsName,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	// HWND_MESSAGE = (HWND)-3
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(clsName)),
		0, 0, 0, 0, 0, 0,
		^uintptr(2),
		0, hInst, 0,
	)
	mainHwnd = hwnd

	startIcon := iconGreen
	nid := NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		HWnd:             hwnd,
		UID:              1,
		UFlags:           nifMessage | nifIcon | nifTip,
		UCallbackMessage: uint32(wmTray),
		HIcon:            startIcon,
	}
	tip, _ := syscall.UTF16FromString("PrintBridge — Iniciando...")
	copy(nid.SzTip[:], tip)
	procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))

	go pollPrinterStatus()

	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 || r == ^uintptr(0) {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	os.Exit(0)
}
