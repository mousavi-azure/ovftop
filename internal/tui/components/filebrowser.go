package components

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mousavi-azure/ovftop/internal/tui/theme"
)

// FileEntry is one row in a FileBrowser listing.
type FileEntry struct {
	Name    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// FileBrowser is a single-pane, Midnight-Commander-style directory
// browser: breadcrumb path, sorted listing (directories first), size/date
// columns, an extension filter that highlights/limits to relevant files
// (OVA/OVF by default), and a hidden-files toggle.
type FileBrowser struct {
	styles theme.Styles

	dir     string
	entries []FileEntry
	cursor  int
	offset  int
	width   int
	height  int

	showHidden bool
	showAll    bool // when false, only directories + allowed extensions show
	extensions map[string]bool
	err        error
}

// defaultExtensions is the filter spec's Deploy Wizard cares about: OVA
// and OVF packages.
func defaultExtensions() map[string]bool {
	return map[string]bool{".ova": true, ".ovf": true}
}

// NewFileBrowser creates a browser rooted at startDir (falling back to the
// user's home directory if startDir can't be read).
func NewFileBrowser(styles theme.Styles, startDir string) *FileBrowser {
	b := &FileBrowser{styles: styles, dir: startDir, extensions: defaultExtensions()}
	if err := b.Reload(); err != nil {
		if home, herr := os.UserHomeDir(); herr == nil {
			b.dir = home
			_ = b.Reload()
		}
	}
	return b
}

// SetSize updates the viewport dimensions used for scrolling.
func (b *FileBrowser) SetSize(w, h int) {
	b.width, b.height = w, h
	b.clampOffset()
}

// CurrentDir returns the directory currently being displayed.
func (b *FileBrowser) CurrentDir() string { return b.dir }

// Err returns the last listing error, if any (e.g. permission denied).
func (b *FileBrowser) Err() error { return b.err }

// Reload re-reads the current directory's contents.
func (b *FileBrowser) Reload() error {
	dirEntries, err := os.ReadDir(b.dir)
	if err != nil {
		b.err = err
		b.entries = nil
		return err
	}
	b.err = nil

	entries := make([]FileEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		name := de.Name()
		if !b.showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		info, err := de.Info()
		if err != nil {
			continue
		}
		isDir := de.IsDir() || info.Mode()&os.ModeSymlink != 0 && isDirSymlink(b.dir, name)
		if !isDir && !b.showAll && !b.extensionAllowed(name) {
			continue
		}
		entries = append(entries, FileEntry{Name: name, IsDir: isDir, Size: info.Size(), ModTime: info.ModTime()})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	b.entries = entries
	if b.cursor >= len(b.entries) {
		b.cursor = len(b.entries) - 1
	}
	if b.cursor < 0 {
		b.cursor = 0
	}
	b.clampOffset()
	return nil
}

func isDirSymlink(dir, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))
	return err == nil && info.IsDir()
}

func (b *FileBrowser) extensionAllowed(name string) bool {
	if len(b.extensions) == 0 {
		return true
	}
	return b.extensions[strings.ToLower(filepath.Ext(name))]
}

// ToggleHidden shows/hides dotfiles.
func (b *FileBrowser) ToggleHidden() { b.showHidden = !b.showHidden; _ = b.Reload() }

// ToggleShowAll shows/hides files that don't match the extension filter.
func (b *FileBrowser) ToggleShowAll() { b.showAll = !b.showAll; _ = b.Reload() }

// MoveUp moves the cursor one row up.
func (b *FileBrowser) MoveUp() {
	if b.cursor > 0 {
		b.cursor--
	}
	b.clampOffset()
}

// MoveDown moves the cursor one row down.
func (b *FileBrowser) MoveDown() {
	if b.cursor < len(b.entries)-1 {
		b.cursor++
	}
	b.clampOffset()
}

// Selected returns the entry under the cursor, or nil if the directory is
// empty.
func (b *FileBrowser) Selected() *FileEntry {
	if b.cursor < 0 || b.cursor >= len(b.entries) {
		return nil
	}
	return &b.entries[b.cursor]
}

// GoUp navigates to the parent directory.
func (b *FileBrowser) GoUp() {
	parent := filepath.Dir(b.dir)
	if parent == b.dir {
		return
	}
	prevBase := filepath.Base(b.dir)
	b.dir = parent
	b.cursor = 0
	_ = b.Reload()
	for i, e := range b.entries {
		if e.Name == prevBase {
			b.cursor = i
			break
		}
	}
	b.clampOffset()
}

// Open descends into the entry under the cursor if it's a directory. It
// is a no-op for files — callers should check Selected().IsDir first and
// treat a file selection as "the user picked this file" instead.
func (b *FileBrowser) Open() {
	sel := b.Selected()
	if sel == nil || !sel.IsDir {
		return
	}
	b.dir = filepath.Join(b.dir, sel.Name)
	b.cursor = 0
	b.offset = 0
	_ = b.Reload()
}

// SelectedPath returns the full path of the entry under the cursor.
func (b *FileBrowser) SelectedPath() string {
	sel := b.Selected()
	if sel == nil {
		return ""
	}
	return filepath.Join(b.dir, sel.Name)
}

func (b *FileBrowser) clampOffset() {
	if b.height <= 0 {
		return
	}
	if b.cursor < b.offset {
		b.offset = b.cursor
	}
	if b.cursor >= b.offset+b.height {
		b.offset = b.cursor - b.height + 1
	}
	if b.offset < 0 {
		b.offset = 0
	}
}

func fileIcon(e FileEntry) string {
	if e.IsDir {
		return "📁"
	}
	switch strings.ToLower(filepath.Ext(e.Name)) {
	case ".ova":
		return "📦"
	case ".ovf":
		return "📄"
	case ".iso":
		return "💿"
	case ".vmdk":
		return "💾"
	case ".zip", ".tar", ".gz", ".tgz":
		return "🗜"
	default:
		return "📃"
	}
}

func formatSize(n int64) string {
	if n < 0 {
		return ""
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	f := float64(n)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d %s", n, units[i])
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

// View renders the breadcrumb and the currently visible window of entries.
func (b *FileBrowser) View() string {
	var lines []string
	lines = append(lines, b.styles.PanelTitle.Render("📂 "+b.dir))

	if b.err != nil {
		lines = append(lines, b.styles.Error.Render("⚠ "+b.err.Error()))
		return strings.Join(lines, "\n")
	}
	if len(b.entries) == 0 {
		lines = append(lines, b.styles.TreeMuted.Render("(empty directory)"))
		return strings.Join(lines, "\n")
	}

	end := b.offset + b.height
	if end > len(b.entries) || b.height <= 0 {
		end = len(b.entries)
	}

	for i := b.offset; i < end; i++ {
		e := b.entries[i]
		name := e.Name
		if e.IsDir {
			name += "/"
		}
		sizeCol := ""
		if !e.IsDir {
			sizeCol = formatSize(e.Size)
		}
		line := fmt.Sprintf("%s %-40s %10s", fileIcon(e), truncateName(name, 40), sizeCol)

		style := b.styles.TreeItem
		if e.IsDir {
			style = b.styles.TreeGroup
		}
		if i == b.cursor {
			style = b.styles.TreeSelected
		}
		if b.width > 0 {
			line = padOrTrim(line, b.width)
		}
		lines = append(lines, style.Render(line))
	}
	return strings.Join(lines, "\n")
}

func truncateName(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
