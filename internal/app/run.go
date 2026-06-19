package app

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"vidc/internal/cleanup"
	"vidc/internal/probe"
	"vidc/internal/progress"
	"vidc/internal/scan"
	"vidc/internal/selecting"
)

// Config holds all configuration for a cleanup run.
type Config struct {
	Dir           string
	MaxDuration   float64
	Recursive     bool
	DryRun        bool
	Yes           bool
	Workers       int
	Verbose       bool
	FFprobePath   string
	Extensions    []string
	SelectExpr    string
	ConfirmDelete bool
	NoProgress    bool
}

// ProbeResult holds the result of probing a single video file.
type ProbeResult struct {
	Path     string
	Size     int64
	Duration float64
	Error    error
}

// Summary holds the aggregated results of a cleanup run.
type Summary struct {
	ScannedFiles    int
	VideoCandidates int
	Matched         int
	Deleted         int
	FailedProbes    int
	FailedDeletes   int
	ScanErrors      int
	MatchedFiles    []ProbeResult
	ProbeErrors     []ProbeResult
	DeleteErrors    []ProbeResult
}

// Run executes the cleanup with the given configuration.
// Returns an exit code: 0=success, 1=config error, 2=partial failure.
func Run(cfg Config) int {
	exitCode, err := ValidateConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return exitCode
	}

	tty := progress.IsTerminal(os.Stderr)

	scanSpinner := progress.NewSpinner(os.Stderr, tty && !cfg.NoProgress)
	scanSpinner.Update(0, 0)

	s := scan.New(cfg.Extensions)
	files, totalFiles, scanErrs := s.ScanWithProgress(cfg.Dir, cfg.Recursive, func(tf, vc int) {
		scanSpinner.Update(tf, vc)
	})
	scanSpinner.Stop()

	for _, e := range scanErrs {
		fmt.Fprintf(os.Stderr, "Error: scan: %v\n", e)
	}

	if len(files) == 0 {
		if len(scanErrs) > 0 {
			return 2
		}
		fmt.Println("No video files found.")
		return 0
	}

	probeBar := progress.NewBar(os.Stderr, tty && !cfg.NoProgress)
	probeBar.Start("Probing", len(files))
	var probed atomic.Int64
	results := probeAll(files, cfg.FFprobePath, cfg.Workers, func() {
		probeBar.Advance(int(probed.Add(1)))
	})
	probeBar.Finish()

	summary := buildSummary(results, totalFiles, cfg)
	summary.ScanErrors = len(scanErrs)

	sort.Slice(summary.MatchedFiles, func(i, j int) bool {
		return summary.MatchedFiles[i].Path < summary.MatchedFiles[j].Path
	})

	if len(summary.MatchedFiles) > 0 {
		fmt.Println("Matched files:")
		for i, f := range summary.MatchedFiles {
			fmt.Printf("  %d. [%.2fs] %6s  %s\n", i+1, f.Duration, formatSize(f.Size), f.Path)
		}
		fmt.Println()
	}

	if cfg.DryRun {
		printReport(cfg, summary)
		if summary.ScanErrors > 0 || summary.FailedProbes > 0 || summary.FailedDeletes > 0 {
			return 2
		}
		return 0
	}

	selectedFiles := summary.MatchedFiles
	userCancelled := false

	if tty {
		lines := make([]string, len(summary.MatchedFiles))
		for i, f := range summary.MatchedFiles {
			lines[i] = fmt.Sprintf("[%.2fs] %6s  %s", f.Duration, formatSize(f.Size), f.Path)
		}
		indices, ok, err := selecting.PromptInteractive(lines, os.Stdin, os.Stderr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		if !ok || len(indices) == 0 {
			fmt.Println("No files selected. Nothing to delete.")
			return 0
		}
		var filtered []ProbeResult
		for _, idx := range indices {
			filtered = append(filtered, summary.MatchedFiles[idx-1])
		}
		selectedFiles = filtered

		if !confirmInteractive(os.Stdin, os.Stderr) {
			userCancelled = true
		}
	} else {
		expr := cfg.SelectExpr
		if expr == "" && cfg.Yes {
			expr = "all"
			fmt.Fprintf(os.Stderr, "Warning: --yes without --select defaults to 'all'. Use --select explicitly.\n")
		}
		if expr == "" {
			fmt.Fprintf(os.Stderr, "Error: non-interactive mode requires --select. Use --select all to select everything.\n")
			return 1
		}

		indices, err := selecting.Parse(expr, len(summary.MatchedFiles))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid --select expression: %v\n", err)
			return 1
		}
		if len(indices) == 0 {
			fmt.Println("No files selected. Nothing to delete.")
			return 0
		}
		var filtered []ProbeResult
		for _, idx := range indices {
			filtered = append(filtered, summary.MatchedFiles[idx-1])
		}
		selectedFiles = filtered

		if cfg.ConfirmDelete || cfg.Yes {
			userCancelled = false
		} else {
			fmt.Fprintf(os.Stderr, "Error: non-interactive mode requires --confirm-delete (or --yes) to proceed.\n")
			return 1
		}
	}

	if userCancelled {
		fmt.Println("Deletion cancelled.")
		return 0
	}

	if len(selectedFiles) > 0 {
		delBar := progress.NewBar(os.Stderr, tty && !cfg.NoProgress)
		delBar.Start("Deleting", len(selectedFiles))
		performSelectedDeletes(&summary, selectedFiles, func(deleted int) {
			delBar.Advance(deleted)
		})
		delBar.Finish()
	}

	printReport(cfg, summary)

	if summary.ScanErrors > 0 || summary.FailedProbes > 0 || summary.FailedDeletes > 0 {
		return 2
	}
	return 0
}

func confirmInteractive(stdin *os.File, stderr *os.File) bool {
	fmt.Fprintln(stderr)
	fmt.Fprintf(stderr, "Type 'delete' to confirm permanent removal: ")
	var input string
	fmt.Fscanf(stdin, "%s", &input)
	return strings.TrimSpace(input) == "delete"
}

func performSelectedDeletes(s *Summary, files []ProbeResult, onDelete func(int)) {
	for _, r := range files {
		if r.Error != nil {
			continue
		}
		err := cleanup.Remove(r.Path)
		if err != nil {
			s.FailedDeletes++
			s.DeleteErrors = append(s.DeleteErrors, ProbeResult{
				Path:  r.Path,
				Size:  r.Size,
				Error: fmt.Errorf("delete failed: %w", err),
			})
		} else {
			s.Deleted++
		}
		if onDelete != nil {
			onDelete(s.Deleted)
		}
	}
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

// ValidateConfig checks the configuration and returns (exitCode, error).
// exitCode is 1 for all validation errors.
func ValidateConfig(cfg Config) (int, error) {
	if cfg.Dir == "" {
		return 1, fmt.Errorf("--dir is required")
	}
	info, err := os.Stat(cfg.Dir)
	if err != nil {
		return 1, fmt.Errorf("directory %q: %w", cfg.Dir, err)
	}
	if !info.IsDir() {
		return 1, fmt.Errorf("%q is not a directory", cfg.Dir)
	}
	if cfg.MaxDuration <= 0 {
		return 1, fmt.Errorf("--max-duration must be greater than 0")
	}
	if cfg.Workers < 1 {
		return 1, fmt.Errorf("--workers must be at least 1")
	}
	if err := checkFFprobe(cfg.FFprobePath); err != nil {
		return 1, err
	}
	return 0, nil
}

// lookPath is a variable so tests can replace it with a mock.
var lookPath = exec.LookPath

// probeFile is a variable so tests can replace it with a mock.
var probeFile = probe.Probe

func checkFFprobe(path string) error {
	_, err := lookPath(path)
	if err != nil {
		return fmt.Errorf("ffprobe not found at %q: %w", path, err)
	}
	return nil
}

func probeAll(files []scan.VideoFile, ffprobePath string, workers int, onProbe func()) []ProbeResult {
	jobs := make(chan scan.VideoFile, len(files))
	results := make(chan ProbeResult, len(files))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				dur, err := probeFile(ffprobePath, f.Path, probe.DefaultTimeout)
				results <- ProbeResult{
					Path:     f.Path,
					Size:     f.Size,
					Duration: dur,
					Error:    err,
				}
				if onProbe != nil {
					onProbe()
				}
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var out []ProbeResult
	for r := range results {
		out = append(out, r)
	}
	return out
}

func buildSummary(results []ProbeResult, totalFiles int, cfg Config) Summary {
	s := Summary{
		ScannedFiles:    totalFiles,
		VideoCandidates: len(results),
	}
	for _, r := range results {
		if r.Error != nil {
			s.FailedProbes++
			s.ProbeErrors = append(s.ProbeErrors, r)
			continue
		}
		if r.Duration < cfg.MaxDuration {
			s.Matched++
			s.MatchedFiles = append(s.MatchedFiles, r)
		}
	}
	return s
}

func printReport(cfg Config, s Summary) {
	mode := "dry-run"
	if !cfg.DryRun && s.Deleted > 0 {
		mode = "delete"
	}

	fmt.Printf("Scan directory: %s\n", cfg.Dir)
	fmt.Printf("Recursive: %v\n", cfg.Recursive)
	fmt.Printf("Threshold: %.0fs\n", cfg.MaxDuration)
	fmt.Printf("Mode: %s\n", mode)
	fmt.Println()

	fmt.Println("Summary:")
	fmt.Printf("  scanned files: %d\n", s.ScannedFiles)
	fmt.Printf("  video candidates: %d\n", s.VideoCandidates)
	fmt.Printf("  matched short videos: %d\n", s.Matched)
	fmt.Printf("  deleted: %d\n", s.Deleted)
	fmt.Printf("  failed probes: %d\n", s.FailedProbes)
	fmt.Printf("  failed deletes: %d\n", s.FailedDeletes)
	fmt.Printf("  scan errors: %d\n", s.ScanErrors)
}

func performDeletes(s *Summary) {
	performSelectedDeletes(s, s.MatchedFiles, nil)
}
