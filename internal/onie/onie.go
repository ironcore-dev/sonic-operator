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

// Register a handler which serves the given directory over HTTP. See the ONIE
// documentation for which file names are tried:
// https://opencomputeproject.github.io/onie/design-spec/discovery.html
func Register(mux *http.ServeMux, installerDir string) {
	logger := slog.With(
		"component", "onie",
		"installerDir", installerDir,
	)

	// Log early if the directory looks wrong (common root cause for 404s).
	if st, err := os.Stat(installerDir); err != nil {
		logger.Warn("installer directory stat failed", "err", err)
	} else if !st.IsDir() {
		logger.Warn("installer path is not a directory", "mode", st.Mode().String())
	} else {
		logger.Info("installer directory configured", "mode", st.Mode().String())
	}

	h := &handler{installerDir: installerDir, logger: logger}
	mux.Handle("GET /", h)
}

type handler struct {
	installerDir string
	logger       *slog.Logger
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

	// FileServer uses r.URL.Path as its lookup key. Log both escaped + decoded to
	// make it easier to debug strange client-side encoding issues.
	cleanURLPath := path.Clean("/" + r.URL.Path)

	// Best-effort mapping from URL path to filesystem path for debugging.
	rel := strings.TrimPrefix(cleanURLPath, "/")
	fsPath := filepath.Join(h.installerDir, filepath.FromSlash(rel))

	clientIP, _, _ := clientIdentity(r)

	reqLogger := h.logger.With(
		"cleanURLPath", cleanURLPath,
		"fsPath", fsPath,
		"clientIP", clientIP,
	)

	rec := &statusRecorder{ResponseWriter: w}
	installerFS := &onieFS{
		baseDir: h.installerDir,
		inner:   os.DirFS(h.installerDir),
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
