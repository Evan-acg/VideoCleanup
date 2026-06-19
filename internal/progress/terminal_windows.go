package progress

import (
	"os"
	"syscall"
)

// IsTerminal reports whether f is attached to a terminal.
func IsTerminal(f *os.File) bool {
	ft, err := syscall.GetFileType(syscall.Handle(f.Fd()))
	if err != nil {
		return false
	}
	return ft == syscall.FILE_TYPE_CHAR
}
