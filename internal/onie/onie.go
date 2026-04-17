// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package onie

import (
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// OnieImage maps a vendor string (matching the ONIE-MACHINE request header) to
// the filenames of the ONIE updater and SONiC installer images for that machine.
type OnieImage struct {
	Vendor        string `json:"vendor"`
	OnieUpdater   string `json:"onieUpdater"`
	OnieInstaller string `json:"onieInstaller"`
}

// Config holds the machine-to-image mappings loaded from onie.json.
type Config struct {
	OnieImages []OnieImage `json:"onieImages"`
}

// Register a handler which serves ONIE and SONiC installer images over HTTP.
// The correct image is selected based on the ONIE-OPERATION and ONIE-MACHINE
// request headers sent by ONIE clients on every download request.
func Register(mux *http.ServeMux, onieImagesDir string, cfg Config) {
	logger := slog.With(
		"component", "onie",
		"onieImagesDir", onieImagesDir,
	)

	// Log early if the directory looks wrong (common root cause for 404s).
	if st, err := os.Stat(onieImagesDir); err != nil {
		logger.Warn("images directory stat failed", "err", err)
	} else if !st.IsDir() {
		logger.Warn("images path is not a directory", "mode", st.Mode().String())
	} else {
		logger.Info("images directory configured", "mode", st.Mode().String())
	}

	images := make(map[string]OnieImage, len(cfg.OnieImages))
	for _, img := range cfg.OnieImages {
		images[img.Vendor] = img
	}

	h := &handler{onieImagesDir: onieImagesDir, images: images, logger: logger}
	mux.Handle("GET /onie", h)
}

type handler struct {
	onieImagesDir string
	images        map[string]OnieImage
	logger        *slog.Logger
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusRecorder) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusRecorder) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	clientIP, _, _ := clientIdentity(r)

	operation := r.Header.Get("ONIE-OPERATION")
	machine := r.Header.Get("ONIE-MACHINE")

	img, ok := h.images[machine]
	if !ok {
		h.logger.Warn("unknown ONIE-MACHINE, rejecting", "machine", machine, "clientIP", clientIP)
		http.NotFound(w, r)
		return
	}

	switch operation {
	case "onie-update":
		r.URL.Path = "/" + img.OnieUpdater
	case "os-install":
		r.URL.Path = "/" + img.OnieInstaller
	default:
		h.logger.Warn("unknown ONIE-OPERATION, rejecting", "operation", operation, "machine", machine, "clientIP", clientIP)
		http.Error(w, "unknown ONIE-OPERATION: "+operation, http.StatusBadRequest)
		return
	}

	// FileServer uses r.URL.Path as its lookup key. Log both escaped + decoded to
	// make it easier to debug strange client-side encoding issues.
	cleanURLPath := path.Clean("/" + r.URL.Path)

	// Best-effort mapping from URL path to filesystem path for debugging.
	rel := strings.TrimPrefix(cleanURLPath, "/")
	fsPath := filepath.Join(h.onieImagesDir, filepath.FromSlash(rel))

	reqLogger := h.logger.With(
		"cleanURLPath", cleanURLPath,
		"fsPath", fsPath,
		"clientIP", clientIP,
		"operation", operation,
		"machine", machine,
	)

	rec := &statusRecorder{ResponseWriter: w}
	installerFS := &onieFS{
		baseDir: h.onieImagesDir,
		inner:   os.DirFS(h.onieImagesDir),
		logger:  reqLogger,
	}

	// Create per-request handler so filesystem logs include request context.
	http.FileServer(http.FS(installerFS)).ServeHTTP(rec, r)

	status := rec.status
	if status == 0 {
		status = http.StatusOK
	}

	fields := []any{
		"status", status,
		"duration", time.Since(start).String(),
	}

	if status >= 400 {
		reqLogger.Warn("served request (failed)", fields...)
		return
	}

	// Treat successful file reads as "downloads" when we can prove it's a file.
	if status == http.StatusOK && r.Method == http.MethodGet {
		if st, err := os.Stat(fsPath); err == nil && st.Mode().IsRegular() {
			fields = append(fields, "fileSize", st.Size())
			reqLogger.Info("served download", fields...)
			return
		}
	}

	reqLogger.Info("served request", fields...)
}

type onieFS struct {
	baseDir string
	inner   fs.FS
	logger  *slog.Logger
}

func (f *onieFS) Open(name string) (fs.File, error) {
	file, err := f.inner.Open(name)
	if err != nil {
		// FileServer will turn most Open errors into a 404/403. Log the OS-level
		// reason so we can distinguish "missing file" from "permission denied",
		// "not a directory", or "bad mount".
		fullPath := filepath.Join(f.baseDir, filepath.FromSlash(strings.TrimPrefix(name, "/")))
		f.logger.Warn("failed to open installer path", "name", name, "fullPath", fullPath, "err", err)
		return nil, err
	}
	return file, nil
}

func clientIdentity(r *http.Request) (clientIP string, xForwardedFor string, forwarded string) {
	xForwardedFor = r.Header.Get("X-Forwarded-For")
	forwarded = r.Header.Get("Forwarded")

	// Prefer X-Forwarded-For if present; otherwise use RemoteAddr. Keep the raw
	// header values in logs for correlation/debugging.
	if xForwardedFor != "" {
		parts := strings.Split(xForwardedFor, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip, xForwardedFor, forwarded
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host, xForwardedFor, forwarded
	}
	return r.RemoteAddr, xForwardedFor, forwarded
}
