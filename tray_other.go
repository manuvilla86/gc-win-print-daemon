//go:build !windows

package main

func runTray() {
	select {} // block forever on non-Windows
}
