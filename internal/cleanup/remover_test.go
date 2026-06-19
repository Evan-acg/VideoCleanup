package cleanup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemove_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp4")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	if err := Remove(path); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should not exist after removal")
	}
}

func TestRemove_NonExistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nofile.mp4")
	err := Remove(path)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestRemove_Directory(t *testing.T) {
	dir := t.TempDir()
	err := Remove(dir)
	if err == nil {
		t.Error("expected error when trying to remove a directory")
	}
}
