package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

const (
	exeName  = "printbridge.exe"
	regKey   = `Software\Microsoft\Windows\CurrentVersion\Run`
	regValue = "PrintBridge"
)

func installDir() string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		local = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	return filepath.Join(local, "Programs", "PrintBridge")
}

func maybeInstall() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, _ = filepath.EvalSymlinks(exe)
	target := filepath.Join(installDir(), exeName)

	if filepath.Clean(exe) == filepath.Clean(target) {
		return // already running from install location
	}

	if err := os.MkdirAll(installDir(), 0755); err != nil {
		return
	}

	if err := copyFile(exe, target); err != nil {
		return
	}

	if k, _, err := registry.CreateKey(registry.CURRENT_USER, regKey, registry.SET_VALUE); err == nil {
		k.SetStringValue(regValue, target)
		k.Close()
	}

	// Launch installed copy and exit so the running process is always the installed one.
	cmd := exec.Command(target)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if cmd.Start() == nil {
		os.Exit(0)
	}
}

// DELETE /uninstall
// Removes the startup registry entry and schedules deletion of the install dir, then exits.
func uninstallHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Remove startup registry entry.
	if k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.SET_VALUE); err == nil {
		k.DeleteValue(regValue)
		k.Close()
	}

	// Respond before exiting so the browser receives the response.
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})

	// Schedule self-deletion via a detached cmd and exit.
	dir := installDir()
	go func() {
		time.Sleep(500 * time.Millisecond)
		cmd := exec.Command("cmd", "/C", "ping -n 3 127.0.0.1 > nul && rmdir /S /Q \""+dir+"\"")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008}
		cmd.Start()
		os.Exit(0)
	}()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
