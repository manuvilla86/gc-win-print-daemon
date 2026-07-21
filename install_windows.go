package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

const (
	regKey   = `Software\Microsoft\Windows\CurrentVersion\Run`
	regValue = "PrintBridge"
)

// maybeInstall registers the current exe path in Windows startup.
func maybeInstall() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, _ = filepath.EvalSymlinks(exe)

	k, _, err := registry.CreateKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()

	existing, _, _ := k.GetStringValue(regValue)
	if filepath.Clean(existing) != filepath.Clean(exe) {
		k.SetStringValue(regValue, exe)
	}
}

// DELETE /uninstall
// Uses the NSIS uninstaller if present, otherwise cleans up manually.
func uninstallHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})

	go func() {
		time.Sleep(500 * time.Millisecond)

		exe, _ := os.Executable()
		uninst := filepath.Join(filepath.Dir(exe), "Uninstall.exe")

		if _, err := os.Stat(uninst); err == nil {
			// Run NSIS silent uninstaller — handles registry, files, Add/Remove Programs.
			cmd := exec.Command(uninst, "/S")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			cmd.Start()
		} else {
			// Fallback for direct exe run (no installer).
			if k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.SET_VALUE); err == nil {
				k.DeleteValue(regValue)
				k.Close()
			}
			os.Remove(configPath())
		}

		os.Exit(0)
	}()
}
