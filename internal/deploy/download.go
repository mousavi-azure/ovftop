package deploy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DownloadProgress reports incremental progress of a Download call. The
// callback is invoked synchronously from the copy loop (throttled to avoid
// excessive calls on fast local transfers); callers that need this on a
// TUI tick should stash the latest value in a shared/atomic field rather
// than rendering directly from the callback.
type DownloadProgress struct {
	BytesRead  int64
	TotalBytes int64 // 0 if the server didn't report Content-Length.
	Elapsed    time.Duration
}

// Download streams url into destDir, returning the path to the downloaded
// file. The filename is derived from the URL's path and de-duplicated if
// a file with that name already exists in destDir. The file is written to
// a ".part" sibling and renamed into place only on success, so a failed
// or canceled download never leaves a corrupt final file behind.
func Download(ctx context.Context, rawURL, destDir string, onProgress func(DownloadProgress)) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("creating download directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: HTTP %s", resp.Status)
	}

	destPath := uniquePath(filepath.Join(destDir, filenameFromURL(rawURL)))
	tmpPath := destPath + ".part"

	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	start := time.Now()
	pr := &progressReader{r: resp.Body, total: resp.ContentLength, start: start, onProgress: onProgress}

	if _, err := io.Copy(f, pr); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if onProgress != nil {
		onProgress(DownloadProgress{BytesRead: pr.read, TotalBytes: pr.total, Elapsed: time.Since(start)})
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", err
	}
	return destPath, nil
}

type progressReader struct {
	r          io.Reader
	read       int64
	total      int64
	start      time.Time
	lastReport time.Time
	onProgress func(DownloadProgress)
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.read += int64(n)
	if p.onProgress != nil && time.Since(p.lastReport) >= 150*time.Millisecond {
		p.lastReport = time.Now()
		p.onProgress(DownloadProgress{BytesRead: p.read, TotalBytes: p.total, Elapsed: time.Since(p.start)})
	}
	return n, err
}

func filenameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	name := ""
	if err == nil {
		name = filepath.Base(u.Path)
	}
	if name == "" || name == "." || name == "/" {
		name = "download"
	}
	return name
}

// uniquePath appends " (N)" before the extension until it finds a path
// that doesn't already exist, so repeated downloads of the same filename
// don't clobber each other.
func uniquePath(path string) string {
	if _, err := os.Stat(path); err != nil {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
}
