package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var defaultExtensions = []string{
	".mp4", ".mov", ".mkv", ".avi", ".webm", ".flv", ".wmv", ".m4v", ".mpeg", ".mpg", ".3gp",
}

// DefaultExtensions returns a copy of the default video file extensions.
func DefaultExtensions() []string {
	exts := make([]string, len(defaultExtensions))
	copy(exts, defaultExtensions)
	return exts
}

// VideoFile represents a video file candidate found during scanning.
type VideoFile struct {
	Path string
	Size int64
}

// Scanner scans directories for video files by extension.
type Scanner struct {
	extensions map[string]bool
}

// New creates a Scanner that recognizes files with any of the given extensions.
func New(extensions []string) *Scanner {
	extMap := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extMap[ext] = true
	}
	return &Scanner{extensions: extMap}
}

// Scan walks the given directory and returns video files found.
// It returns the list of video files, the total count of regular files scanned,
// and any non-fatal errors encountered during traversal.
func (s *Scanner) Scan(dir string, recursive bool) (files []VideoFile, totalFiles int, errs []error) {
	if recursive {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				errs = append(errs, err)
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			totalFiles++
			if !s.matchExt(path) {
				return nil
			}
			info, infoErr := d.Info()
			if infoErr != nil {
				errs = append(errs, infoErr)
				return nil
			}
			files = append(files, VideoFile{Path: path, Size: info.Size()})
			return nil
		})
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, 0, []error{err}
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !entry.Type().IsRegular() {
				continue
			}
			totalFiles++
			fullPath := filepath.Join(dir, entry.Name())
			if !s.matchExt(fullPath) {
				continue
			}
			info, infoErr := entry.Info()
			if infoErr != nil {
				errs = append(errs, infoErr)
				continue
			}
			files = append(files, VideoFile{Path: fullPath, Size: info.Size()})
		}
	}
	return
}

func (s *Scanner) matchExt(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return s.extensions[ext]
}
