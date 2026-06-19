package app

import (
	"fmt"
	"io"
	"path/filepath"
)

func displayPath(baseDir, path string) string {
	baseAbs, err1 := filepath.Abs(baseDir)
	pathAbs, err2 := filepath.Abs(path)
	if err1 != nil || err2 != nil {
		return filepath.Clean(path)
	}
	rel, err := filepath.Rel(baseAbs, pathAbs)
	if err != nil || rel == "" {
		return filepath.Clean(path)
	}
	return rel
}

func printMatchedFiles(w io.Writer, baseDir string, files []ProbeResult) {
	if len(files) == 0 {
		return
	}
	numW := len("#")
	durW := len("Duration")
	sizeW := len("Size")
	for i, f := range files {
		n := len(fmt.Sprintf("%d", i+1))
		d := len(fmt.Sprintf("%.2fs", f.Duration))
		s := len(formatSize(f.Size))
		if n > numW {
			numW = n
		}
		if d > durW {
			durW = d
		}
		if s > sizeW {
			sizeW = s
		}
	}
	fmt.Fprintln(w, "Matched files:")
	fmt.Fprintf(w, "  %-*s  %-*s  %-*s  %s\n", numW, "#", durW, "Duration", sizeW, "Size", "Path")
	for i, f := range files {
		fmt.Fprintf(w, "  %-*d  %-*.2fs  %-*s  %s\n",
			numW, i+1,
			durW, f.Duration,
			sizeW, formatSize(f.Size),
			displayPath(baseDir, f.Path),
		)
	}
	fmt.Fprintln(w)
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.0f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}

func printReport(w io.Writer, cfg Config, s Summary) {
	mode := "dry-run"
	if !cfg.DryRun && s.Deleted > 0 {
		mode = "delete"
	}

	fmt.Fprintf(w, "Scan directory: %s\n", cfg.Dir)
	fmt.Fprintf(w, "Recursive: %v\n", cfg.Recursive)
	fmt.Fprintf(w, "Threshold: %.0fs\n", cfg.MaxDuration)
	fmt.Fprintf(w, "Mode: %s\n", mode)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Summary:")
	fmt.Fprintf(w, "  scanned files: %d\n", s.ScannedFiles)
	fmt.Fprintf(w, "  video candidates: %d\n", s.VideoCandidates)
	fmt.Fprintf(w, "  matched short videos: %d\n", s.Matched)
	fmt.Fprintf(w, "  deleted: %d\n", s.Deleted)
	fmt.Fprintf(w, "  failed probes: %d\n", s.FailedProbes)
	fmt.Fprintf(w, "  failed deletes: %d\n", s.FailedDeletes)
	fmt.Fprintf(w, "  scan errors: %d\n", s.ScanErrors)
}
