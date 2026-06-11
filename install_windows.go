package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

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
