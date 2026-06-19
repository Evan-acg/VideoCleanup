package app

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"sync/atomic"
	"time"

	"vidc/internal/cleanup"
	"vidc/internal/probe"
	"vidc/internal/progress"
	"vidc/internal/scan"
)

// Deps holds injectable external dependencies for a Runner.
type Deps struct {
	LookPath   func(string) (string, error)
	IsTerminal func() bool
	ProbeFile  func(ffprobePath, videoPath string, timeout time.Duration) (float64, error)
	RemoveFile func(path string) error
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
}

// Runner orchestrates a cleanup run with injected dependencies.
type Runner struct {
	deps Deps
}

// NewRunner creates a Runner with the given dependencies.
func NewRunner(deps Deps) *Runner {
	return &Runner{deps: deps}
}

// DefaultDeps returns dependencies backed by the real OS/filesystem/ffprobe.
func DefaultDeps() Deps {
	return Deps{
		LookPath:   exec.LookPath,
		IsTerminal: func() bool { return progress.IsTerminal(os.Stdin) && progress.IsTerminal(os.Stderr) },
		ProbeFile:  probe.Probe,
		RemoveFile: cleanup.Remove,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Stdin:      os.Stdin,
	}
}

// Run executes the cleanup with the given configuration.
// Returns an exit code: 0=success, 1=config error, 2=partial failure.
func (r *Runner) Run(cfg Config) int {
	exitCode, err := ValidateConfig(cfg, r.deps.LookPath)
	if err != nil {
		fmt.Fprintf(r.deps.Stderr, "Error: %v\n", err)
		return exitCode
	}

	interactive := r.deps.IsTerminal()

	scanSpinner := progress.NewSpinner(r.deps.Stderr, interactive && !cfg.NoProgress)
	scanSpinner.Update(0, 0)

	s := scan.New(cfg.Extensions)
	files, totalFiles, scanErrs := s.ScanWithProgress(cfg.Dir, cfg.Recursive, func(tf, vc int) {
		scanSpinner.Update(tf, vc)
	})
	scanSpinner.Stop()

	for _, e := range scanErrs {
		fmt.Fprintf(r.deps.Stderr, "Error: scan: %v\n", e)
	}

	if len(files) == 0 {
		if len(scanErrs) > 0 {
			return 2
		}
		fmt.Fprintln(r.deps.Stdout, "No video files found.")
		return 0
	}

	probeBar := progress.NewBar(r.deps.Stderr, interactive && !cfg.NoProgress)
	probeBar.Start("Probing", len(files))
	var probed atomic.Int64
	results := probeAll(files, cfg.FFprobePath, cfg.Workers, r.deps.ProbeFile, func() {
		probeBar.Advance(int(probed.Add(1)))
	})
	probeBar.Finish()

	summary := buildSummary(results, totalFiles, cfg)
	summary.ScanErrors = len(scanErrs)

	sort.Slice(summary.MatchedFiles, func(i, j int) bool {
		return summary.MatchedFiles[i].Path < summary.MatchedFiles[j].Path
	})

	showTable := cfg.DryRun || !interactive || cfg.SelectExpr != "" || cfg.Yes
	if len(summary.MatchedFiles) > 0 && showTable {
		printMatchedFiles(r.deps.Stdout, cfg.Dir, summary.MatchedFiles)
	}

	if len(summary.MatchedFiles) == 0 {
		fmt.Fprintln(r.deps.Stdout, "No files matched the duration threshold.")
		printReport(r.deps.Stdout, cfg, summary)
		if summary.ScanErrors > 0 || summary.FailedProbes > 0 {
			return 2
		}
		return 0
	}

	if cfg.DryRun {
		printReport(r.deps.Stdout, cfg, summary)
		if summary.ScanErrors > 0 || summary.FailedProbes > 0 || summary.FailedDeletes > 0 {
			return 2
		}
		return 0
	}

	selResult, err := selectFiles(cfg, interactive, summary.MatchedFiles, r.deps.Stdin, r.deps.Stdout, r.deps.Stderr)
	if err != nil {
		fmt.Fprintf(r.deps.Stderr, "Error: %v\n", err)
		return 1
	}
	if selResult.Cancelled {
		fmt.Fprintln(r.deps.Stdout, "Deletion cancelled.")
		return 0
	}
	if len(selResult.Files) == 0 {
		return 0
	}

	delBar := progress.NewBar(r.deps.Stderr, interactive && !cfg.NoProgress)
	delBar.Start("Deleting", len(selResult.Files))
	performSelectedDeletes(&summary, selResult.Files, r.deps.RemoveFile, func(deleted int) {
		delBar.Advance(deleted)
	})
	delBar.Finish()

	printReport(r.deps.Stdout, cfg, summary)

	if summary.ScanErrors > 0 || summary.FailedProbes > 0 || summary.FailedDeletes > 0 {
		return 2
	}
	return 0
}

func confirmInteractive(stdin io.Reader, stderr io.Writer) bool {
	fmt.Fprintln(stderr)
	fmt.Fprintf(stderr, "Type 'delete' to confirm permanent removal: ")
	var input string
	fmt.Fscanf(stdin, "%s", &input)
	return input == "delete"
}
