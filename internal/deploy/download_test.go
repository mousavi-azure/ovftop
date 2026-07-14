package deploy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadSavesFileAndReportsProgress(t *testing.T) {
	body := strings.Repeat("x", 1024*64)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "65536")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	destDir := t.TempDir()
	var lastProgress DownloadProgress
	path, err := Download(context.Background(), srv.URL+"/appliance.ova", destDir, func(p DownloadProgress) {
		lastProgress = p
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if filepath.Base(path) != "appliance.ova" {
		t.Errorf("expected filename appliance.ova, got %s", filepath.Base(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}
	if string(data) != body {
		t.Error("downloaded content mismatch")
	}

	if lastProgress.BytesRead != int64(len(body)) {
		t.Errorf("expected final progress BytesRead=%d, got %d", len(body), lastProgress.BytesRead)
	}
	if lastProgress.TotalBytes != int64(len(body)) {
		t.Errorf("expected TotalBytes=%d, got %d", len(body), lastProgress.TotalBytes)
	}

	if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
		t.Error("expected .part temp file to be cleaned up after rename")
	}
}

func TestDownloadDeduplicatesFilename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()

	destDir := t.TempDir()
	first, err := Download(context.Background(), srv.URL+"/x.ovf", destDir, nil)
	if err != nil {
		t.Fatalf("first Download: %v", err)
	}
	second, err := Download(context.Background(), srv.URL+"/x.ovf", destDir, nil)
	if err != nil {
		t.Fatalf("second Download: %v", err)
	}
	if first == second {
		t.Errorf("expected distinct paths for repeated downloads, both were %s", first)
	}
}

func TestDownloadHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := Download(context.Background(), srv.URL+"/missing.ova", t.TempDir(), nil)
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
}
