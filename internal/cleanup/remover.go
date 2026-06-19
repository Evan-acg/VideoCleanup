package cleanup

import (
	"os"
)

// Remove deletes the file at path. It first verifies the target is still a
// regular file to reduce TOCTOU risk.
func Remove(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return &os.PathError{Op: "remove", Path: path, Err: os.ErrInvalid}
	}
	return os.Remove(path)
}
