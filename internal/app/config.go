package app

import (
	"fmt"
	"os"
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

// ValidateConfig checks the configuration and returns (exitCode, error).
// exitCode is 1 for all validation errors.
func ValidateConfig(cfg Config, lookPath func(string) (string, error)) (int, error) {
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
	if err := checkFFprobe(lookPath, cfg.FFprobePath); err != nil {
		return 1, err
	}
	return 0, nil
}

func checkFFprobe(lookPath func(string) (string, error), path string) error {
	_, err := lookPath(path)
	if err != nil {
		return fmt.Errorf("ffprobe not found at %q: %w", path, err)
	}
	return nil
}
