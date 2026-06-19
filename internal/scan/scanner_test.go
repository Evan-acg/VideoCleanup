package scan

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDefaultExtensions(t *testing.T) {
	exts := DefaultExtensions()
	if len(exts) == 0 {
		t.Fatal("DefaultExtensions returned empty slice")
	}
	if !slices.Contains(exts, ".mp4") {
		t.Error("expected .mp4 in defaults")
	}
}

func TestNew_NormalizesExtensions(t *testing.T) {
	s := New([]string{"mp4", ".MOV"})
	if !s.extensions[".mp4"] {
		t.Error("expected .mp4 to be recognized")
	}
	if !s.extensions[".mov"] {
		t.Error("expected .mov to be recognized (lowercased)")
	}
}

func TestScan_NonRecursive(t *testing.T) {
	dir := t.TempDir()
	mustCreate(t, filepath.Join(dir, "a.mp4"))
	mustCreate(t, filepath.Join(dir, "b.MP4"))
	mustCreate(t, filepath.Join(dir, "c.mov"))
	mustCreate(t, filepath.Join(dir, "notes.txt"))
	mustCreate(t, filepath.Join(dir, "readme.md"))
	mustCreate(t, filepath.Join(dir, "video.mkv"))

	s := New(DefaultExtensions())
	files, total, errs := s.Scan(dir, false)

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if total != 6 {
		t.Errorf("expected 6 total files, got %d", total)
	}
	if len(files) != 4 {
		t.Errorf("expected 4 video files, got %d", len(files))
	}
}

func TestScan_Recursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	mustMkdir(t, sub)

	mustCreate(t, filepath.Join(dir, "a.mp4"))
	mustCreate(t, filepath.Join(sub, "b.mov"))
	mustCreate(t, filepath.Join(sub, "notes.txt"))

	s := New(DefaultExtensions())
	files, total, errs := s.Scan(dir, true)

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if total != 3 {
		t.Errorf("expected 3 total files, got %d", total)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 video files, got %d", len(files))
	}
}

func TestScan_NoVideos(t *testing.T) {
	dir := t.TempDir()
	mustCreate(t, filepath.Join(dir, "a.txt"))
	mustCreate(t, filepath.Join(dir, "b.md"))

	s := New(DefaultExtensions())
	files, total, errs := s.Scan(dir, false)

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if total != 2 {
		t.Errorf("expected 2 total files, got %d", total)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 video files, got %d", len(files))
	}
}

func TestScan_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	s := New(DefaultExtensions())
	files, total, errs := s.Scan(dir, false)

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if total != 0 {
		t.Errorf("expected 0 files, got %d", total)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 video files, got %d", len(files))
	}
}

func TestScan_IgnoresDirectories(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "videos"))
	// Create a subdirectory matching a video extension - should be ignored
	mustMkdir(t, filepath.Join(dir, "movie.mp4"))
	mustCreate(t, filepath.Join(dir, "real.mp4"))

	s := New(DefaultExtensions())
	files, total, errs := s.Scan(dir, false)

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if total != 1 {
		t.Errorf("expected 1 regular file, got %d", total)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 video file, got %d", len(files))
	}
}

func TestScan_NonExistentDir(t *testing.T) {
	s := New(DefaultExtensions())
	_, _, errs := s.Scan("/nonexistent/path", false)
	if len(errs) == 0 {
		t.Error("expected error for non-existent directory")
	}
}

func mustCreate(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("failed to create dir %s: %v", path, err)
	}
}
