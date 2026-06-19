package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

const DefaultTimeout = 30 * time.Second

type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// Probe runs ffprobe on the given video file and returns its duration in seconds.
func Probe(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("ffprobe timeout after %v: %w", timeout, err)
		}
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	return ParseDuration(output)
}

// ParseDuration extracts the duration from ffprobe's JSON output.
func ParseDuration(output []byte) (float64, error) {
	var result ffprobeOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	if result.Format.Duration == "" {
		return 0, fmt.Errorf("duration not found in ffprobe output")
	}

	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value %q: %w", result.Format.Duration, err)
	}

	return duration, nil
}
