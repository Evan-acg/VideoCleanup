package app

import (
	"fmt"

	"vidc/internal/cleanup"
)

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

func performSelectedDeletes(s *Summary, files []ProbeResult, remove func(string) error, onDelete func(int)) {
	processed := 0
	for _, r := range files {
		processed++
		if r.Error != nil {
			continue
		}
		err := remove(r.Path)
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
			onDelete(processed)
		}
	}
}

func performDeletes(s *Summary) {
	performSelectedDeletes(s, s.MatchedFiles, cleanup.Remove, nil)
}
