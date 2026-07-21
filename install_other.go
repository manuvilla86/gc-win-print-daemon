//go:build !windows

package main

import (
	"encoding/json"
	"net/http"
)

func maybeInstall() {}

func uninstallHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "NOT_SUPPORTED"})
}
