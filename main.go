package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"printbridge/printer"
)

const addr = "127.0.0.1:9100"

func main() {
	maybeInstall()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/printers", printersHandler)
	mux.HandleFunc("/config", configHandler)
	mux.HandleFunc("/print", printHandler)
	mux.HandleFunc("/uninstall", uninstallHandler)

	log.Fatal(http.ListenAndServe(addr, cors(mux)))
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GET /health
// Returns bridge status and the active printer (configured or auto-detected).
// "configured" tells the web app whether the user has explicitly chosen a printer.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	cfg := loadConfig()
	configured := cfg.Printer != ""

	name := cfg.Printer
	if name == "" {
		detected, err := printer.Detect()
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "ok",
				"printer":    nil,
				"ready":      false,
				"configured": false,
			})
			return
		}
		name = detected
	}

	// Verify the printer is currently available.
	ready := false
	if names, err := printer.List(); err == nil {
		for _, n := range names {
			if n == name {
				ready = true
				break
			}
		}
	}

	var printerField interface{} = name
	if !ready && !configured {
		printerField = nil
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"printer":    printerField,
		"ready":      ready,
		"configured": configured,
	})
}

// GET /printers
// Returns all locally installed printers.
func printersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	names, err := printer.List()
	if err != nil || names == nil {
		names = []string{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"printers": names,
	})
}

// GET /config  — returns current config
// PUT /config  — body: {"printer": "name"}, saves and returns {"ok": true}
func configHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(loadConfig())

	case http.MethodPut:
		var c Config
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "INVALID_BODY"})
			return
		}
		if err := saveConfig(c); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "SAVE_FAILED"})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// POST /print
// Sends raw ESC/POS bytes to the configured printer (falls back to first detected).
func printHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name, err := resolvedPrinter()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "PRINTER_NOT_FOUND",
		})
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil || len(data) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "INVALID_BODY",
		})
		return
	}

	if err := printer.Print(name, data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":     false,
			"error":  "PRINT_FAILED",
			"detail": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

// resolvedPrinter returns the configured printer or the first detected one.
func resolvedPrinter() (string, error) {
	if cfg := loadConfig(); cfg.Printer != "" {
		return cfg.Printer, nil
	}
	return printer.Detect()
}
