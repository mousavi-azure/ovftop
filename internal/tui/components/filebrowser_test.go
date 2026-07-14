package components

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mousavi-azure/ovftop/internal/tui/theme"
)

func setupTestTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.Mkdir(filepath.Join(root, "subdir"), 0o755))
	must(os.WriteFile(filepath.Join(root, "appliance.ova"), make([]byte, 100), 0o644))
	must(os.WriteFile(filepath.Join(root, "template.ovf"), make([]byte, 50), 0o644))
	must(os.WriteFile(filepath.Join(root, "readme.txt"), make([]byte, 10), 0o644))
	must(os.WriteFile(filepath.Join(root, ".hidden.ova"), make([]byte, 10), 0o644))
	return root
}

func TestFileBrowserDefaultFilterShowsOnlyOvaOvfAndDirs(t *testing.T) {
	root := setupTestTree(t)
	b := NewFileBrowser(theme.New(theme.Dark), root)

	if b.Err() != nil {
		t.Fatalf("unexpected error: %v", b.Err())
	}
	var names []string
	for _, e := range b.entries {
		names = append(names, e.Name)
	}
	want := map[string]bool{"subdir": true, "appliance.ova": true, "template.ovf": true}
	if len(names) != len(want) {
		t.Fatalf("expected %d entries, got %v", len(want), names)
	}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected entry %q in filtered listing", n)
		}
	}
}

func TestFileBrowserShowAllRevealsOtherFiles(t *testing.T) {
	root := setupTestTree(t)
	b := NewFileBrowser(theme.New(theme.Dark), root)
	b.ToggleShowAll()

	found := false
	for _, e := range b.entries {
		if e.Name == "readme.txt" {
			found = true
		}
	}
	if !found {
		t.Error("expected readme.txt to appear once ShowAll is toggled on")
	}
}

func TestFileBrowserHiddenToggle(t *testing.T) {
	root := setupTestTree(t)
	b := NewFileBrowser(theme.New(theme.Dark), root)
	b.ToggleShowAll()

	for _, e := range b.entries {
		if e.Name == ".hidden.ova" {
			t.Fatal("hidden file should not appear before ToggleHidden")
		}
	}
	b.ToggleHidden()
	found := false
	for _, e := range b.entries {
		if e.Name == ".hidden.ova" {
			found = true
		}
	}
	if !found {
		t.Error("expected .hidden.ova to appear after ToggleHidden")
	}
}

func TestFileBrowserNavigation(t *testing.T) {
	root := setupTestTree(t)
	b := NewFileBrowser(theme.New(theme.Dark), root)
	b.SetSize(60, 10)

	// Directories sort first: "subdir" should be entry 0.
	sel := b.Selected()
	if sel == nil || !sel.IsDir || sel.Name != "subdir" {
		t.Fatalf("expected first entry to be subdir, got %+v", sel)
	}

	b.Open()
	if filepath.Base(b.CurrentDir()) != "subdir" {
		t.Fatalf("expected to descend into subdir, got %s", b.CurrentDir())
	}
	if len(b.entries) != 0 {
		t.Fatalf("expected empty subdir, got %v", b.entries)
	}

	b.GoUp()
	if b.CurrentDir() != root {
		t.Fatalf("expected GoUp to return to %s, got %s", root, b.CurrentDir())
	}
	// GoUp should restore the cursor onto "subdir".
	sel = b.Selected()
	if sel == nil || sel.Name != "subdir" {
		t.Fatalf("expected cursor restored to subdir after GoUp, got %+v", sel)
	}
}

func TestFileBrowserSelectedPath(t *testing.T) {
	root := setupTestTree(t)
	b := NewFileBrowser(theme.New(theme.Dark), root)
	b.MoveDown() // subdir -> appliance.ova (alphabetically first file)

	sel := b.Selected()
	if sel == nil || sel.IsDir {
		t.Fatalf("expected a file selected, got %+v", sel)
	}
	want := filepath.Join(root, sel.Name)
	if got := b.SelectedPath(); got != want {
		t.Errorf("SelectedPath = %q, want %q", got, want)
	}
}

func TestFileBrowserPermissionError(t *testing.T) {
	b := NewFileBrowser(theme.New(theme.Dark), t.TempDir())
	// Force a bad directory directly (bypassing the constructor's
	// fall-back-to-home-dir behavior) to exercise the Reload error path.
	b.dir = filepath.Join(t.TempDir(), "does-not-exist")
	if err := b.Reload(); err == nil {
		t.Fatal("expected an error for a nonexistent directory")
	}
	if b.Err() == nil {
		t.Fatal("expected Err() to report the listing failure")
	}
}
