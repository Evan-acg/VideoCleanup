package app

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync"

	"vidc/internal/cleanup"
	"vidc/internal/probe"
	"vidc/internal/scan"
)

// Config holds all configuration for a cleanup run.
type Config struct {
	Dir         string
	MaxDuration float64
	Recursive   bool
	DryRun      bool
	Yes         bool
	Workers     int
	Verbose     bool
	FFprobePath string
	Extensions  []string
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

	s := scan.New(cfg.Extensions)
	files, totalFiles, scanErrs := s.Scan(cfg.Dir, cfg.Recursive)
	for _, e := range scanErrs {
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "scan warning: %v\n", e)
		}
	}

	if len(files) == 0 {
		fmt.Println("No video files found.")
		return 0
	}

	results := probeAll(files, cfg.FFprobePath, cfg.Workers)

	summary := buildSummary(results, totalFiles, cfg)

	if !cfg.DryRun && cfg.Yes {
		performDeletes(&summary)
	}

	printReport(cfg, summary)

	if summary.FailedProbes > 0 || summary.FailedDeletes > 0 {
		return 2
	}

	return 0
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

func checkFFprobe(path string) error {
	_, err := lookPath(path)
	if err != nil {
		return fmt.Errorf("ffprobe not found at %q: %w", path, err)
	}
	return nil
}

func probeAll(files []scan.VideoFile, ffprobePath string, workers int) []ProbeResult {
	jobs := make(chan scan.VideoFile, len(files))
	results := make(chan ProbeResult, len(files))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				dur, err := probe.Probe(ffprobePath, f.Path, probe.DefaultTimeout)
				results <- ProbeResult{
					Path:     f.Path,
					Size:     f.Size,
					Duration: dur,
					Error:    err,
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
	if !cfg.DryRun && cfg.Yes {
		mode = "delete"
	}

	fmt.Printf("Scan directory: %s\n", cfg.Dir)
	fmt.Printf("Recursive: %v\n", cfg.Recursive)
	fmt.Printf("Threshold: %.0fs\n", cfg.MaxDuration)
	fmt.Printf("Mode: %s\n", mode)
	fmt.Println()

	if len(s.MatchedFiles) > 0 {
		sort.Slice(s.MatchedFiles, func(i, j int) bool {
			return s.MatchedFiles[i].Path < s.MatchedFiles[j].Path
		})

		fmt.Println("Matched files:")
		for _, f := range s.MatchedFiles {
			fmt.Printf("  [%.2fs] %s\n", f.Duration, f.Path)
		}
		fmt.Println()
	}

	fmt.Println("Summary:")
	fmt.Printf("  scanned files: %d\n", s.ScannedFiles)
	fmt.Printf("  video candidates: %d\n", s.VideoCandidates)
	fmt.Printf("  matched short videos: %d\n", s.Matched)
	fmt.Printf("  deleted: %d\n", s.Deleted)
	fmt.Printf("  failed probes: %d\n", s.FailedProbes)
	fmt.Printf("  failed deletes: %d\n", s.FailedDeletes)
}

func performDeletes(s *Summary) {
	for _, r := range s.MatchedFiles {
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
	}
}
