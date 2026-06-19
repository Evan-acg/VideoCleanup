package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"vidc/internal/app"
	"vidc/internal/scan"
)

func main() {
	cfg := parseFlags()
	code := app.Run(cfg)
	os.Exit(code)
}

func parseFlags() app.Config {
	var dir string
	var maxDuration float64
	var recursive bool
	var dryRun bool
	var yes bool
	var workers int
	var extensions string
	var ffprobe string
	var verbose bool
	var selectExpr string
	var confirmDelete bool
	var noProgress bool

	flag.StringVar(&dir, "dir", "", "Directory to scan (required)")
	flag.StringVar(&dir, "d", "", "Directory to scan (short)")
	flag.Float64Var(&maxDuration, "max-duration", 0, "Delete videos shorter than this threshold in seconds (required)")
	flag.Float64Var(&maxDuration, "m", 0, "Delete threshold in seconds (short)")
	flag.BoolVar(&recursive, "recursive", false, "Scan subdirectories recursively")
	flag.BoolVar(&recursive, "r", false, "Scan subdirectories recursively (short)")
	flag.BoolVar(&dryRun, "dry-run", true, "Preview mode, no files are deleted (default true)")
	flag.BoolVar(&yes, "yes", false, "Confirm deletion (non-interactive)")
	flag.BoolVar(&yes, "y", false, "Confirm deletion non-interactive (short)")
	flag.IntVar(&workers, "workers", runtime.NumCPU(), "Number of concurrent ffprobe workers")
	flag.IntVar(&workers, "w", runtime.NumCPU(), "Number of concurrent workers (short)")
	flag.StringVar(&extensions, "extensions", strings.Join(scan.DefaultExtensions(), ","), "Comma-separated video file extensions")
	flag.StringVar(&extensions, "e", strings.Join(scan.DefaultExtensions(), ","), "Video extensions (short)")
	flag.StringVar(&ffprobe, "ffprobe", "ffprobe", "Path to ffprobe executable")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&verbose, "v", false, "Verbose output (short)")
	flag.StringVar(&selectExpr, "select", "", "Non-interactive selection expression: all, 1,2,5, 1-5, all,-2")
	flag.BoolVar(&confirmDelete, "confirm-delete", false, "Non-interactive deletion confirmation")
	flag.BoolVar(&noProgress, "no-progress", false, "Disable progress display")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: vidc -d <directory> -m <seconds> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Delete video files shorter than the specified duration.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  vidc -d \"D:\\videos\" -m 10                          Dry-run, preview short videos\n")
		fmt.Fprintf(os.Stderr, "  vidc -d \"D:\\videos\" -m 10 -r --dry-run=false       Interactive delete with selection\n")
		fmt.Fprintf(os.Stderr, "  vidc -d \"D:\\videos\" -m 10 -r --dry-run=false --select all --confirm-delete  Scripted delete all\n")
	}

	flag.Parse()

	exts := parseExtensions(extensions)

	return app.Config{
		Dir:           dir,
		MaxDuration:   maxDuration,
		Recursive:     recursive,
		DryRun:        dryRun,
		Yes:           yes,
		Workers:       workers,
		Extensions:    exts,
		FFprobePath:   ffprobe,
		Verbose:       verbose,
		SelectExpr:    selectExpr,
		ConfirmDelete: confirmDelete,
		NoProgress:    noProgress,
	}
}

func parseExtensions(raw string) []string {
	parts := strings.Split(raw, ",")
	var exts []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			exts = append(exts, p)
		}
	}
	if len(exts) == 0 {
		return scan.DefaultExtensions()
	}
	return exts
}
