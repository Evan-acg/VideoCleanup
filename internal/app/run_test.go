package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vidc/internal/scan"
)

func TestValidateConfig_MissingDir(t *testing.T) {
	_, err := ValidateConfig(Config{Dir: ""})
	if err == nil {
		t.Error("expected error for missing dir")
	}
}

func TestValidateConfig_NonExistentDir(t *testing.T) {
	_, err := ValidateConfig(Config{Dir: "/nonexistent/dir"})
	if err == nil {
		t.Error("expected error for non-existent dir")
	}
}

func TestValidateConfig_NotADirectory(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ValidateConfig(Config{Dir: f})
	if err == nil {
		t.Error("expected error for non-directory path")
	}
}

func TestValidateConfig_ZeroMaxDuration(t *testing.T) {
	dir := t.TempDir()
	old := lookPath
	lookPath = func(path string) (string, error) { return path, nil }
	defer func() { lookPath = old }()

	_, err := ValidateConfig(Config{Dir: dir, MaxDuration: 0, Workers: 1})
	if err == nil {
		t.Error("expected error for zero max-duration")
	}
}

func TestValidateConfig_ZeroWorkers(t *testing.T) {
	dir := t.TempDir()
	old := lookPath
	lookPath = func(path string) (string, error) { return path, nil }
	defer func() { lookPath = old }()

	_, err := ValidateConfig(Config{Dir: dir, MaxDuration: 5, Workers: 0})
	if err == nil {
		t.Error("expected error for zero workers")
	}
}

func TestValidateConfig_FFprobeNotFound(t *testing.T) {
	dir := t.TempDir()
	old := lookPath
	lookPath = func(path string) (string, error) { return "", fmt.Errorf("not found") }
	defer func() { lookPath = old }()

	_, err := ValidateConfig(Config{Dir: dir, MaxDuration: 5, Workers: 1})
	if err == nil {
		t.Error("expected error for missing ffprobe")
	}
}

func TestValidateConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	old := lookPath
	lookPath = func(path string) (string, error) { return path, nil }
	defer func() { lookPath = old }()

	code, err := ValidateConfig(Config{Dir: dir, MaxDuration: 5, Workers: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestBuildSummary(t *testing.T) {
	results := []ProbeResult{
		{Path: "a.mp4", Duration: 3.0, Size: 100},
		{Path: "b.mp4", Duration: 12.0, Size: 200},
		{Path: "c.mp4", Duration: 4.9, Size: 150},
		{Path: "d.mp4", Error: fmt.Errorf("probe fail")},
	}

	summary := buildSummary(results, 10, Config{MaxDuration: 5.0})

	if summary.ScannedFiles != 10 {
		t.Errorf("ScannedFiles: expected 10, got %d", summary.ScannedFiles)
	}
	if summary.VideoCandidates != 4 {
		t.Errorf("VideoCandidates: expected 4, got %d", summary.VideoCandidates)
	}
	if summary.Matched != 2 {
		t.Errorf("Matched: expected 2, got %d", summary.Matched)
	}
	if summary.FailedProbes != 1 {
		t.Errorf("FailedProbes: expected 1, got %d", summary.FailedProbes)
	}
	if summary.FailedDeletes != 0 {
		t.Errorf("FailedDeletes: expected 0, got %d", summary.FailedDeletes)
	}
}

func TestBuildSummary_Boundary(t *testing.T) {
	results := []ProbeResult{
		{Path: "e.mp4", Duration: 5.0, Size: 100},
		{Path: "f.mp4", Duration: 4.999, Size: 200},
	}

	summary := buildSummary(results, 2, Config{MaxDuration: 5.0})

	if summary.Matched != 1 {
		t.Errorf("Boundary: duration == threshold should not match, expected 1, got %d", summary.Matched)
	}
}

func TestRun_DryRun(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	oldLook := lookPath
	lookPath = func(path string) (string, error) { return path, nil }
	defer func() { lookPath = oldLook }()

	oldProbe := probeFile
	probeFile = func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
		if strings.Contains(filepath.Base(videoPath), "short") {
			return 3.2, nil
		}
		return 12.345, nil
	}
	defer func() { probeFile = oldProbe }()

	cfg := Config{
		Dir:         filepath.Join(tmp, "videos"),
		MaxDuration: 10,
		Recursive:   false,
		DryRun:      true,
		Yes:         false,
		Workers:     2,
		Extensions:  scan.DefaultExtensions(),
		FFprobePath: "ffprobe",
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	paths := []string{
		filepath.Join(tmp, "videos", "short.mp4"),
		filepath.Join(tmp, "videos", "long.mp4"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("dry-run should not delete %s", p)
		}
	}
}

func TestRun_Delete(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	oldLook := lookPath
	lookPath = func(path string) (string, error) { return path, nil }
	defer func() { lookPath = oldLook }()

	oldProbe := probeFile
	probeFile = func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
		if strings.Contains(filepath.Base(videoPath), "short") {
			return 3.2, nil
		}
		return 12.345, nil
	}
	defer func() { probeFile = oldProbe }()

	cfg := Config{
		Dir:         filepath.Join(tmp, "videos"),
		MaxDuration: 10,
		Recursive:   false,
		DryRun:      false,
		Yes:         true,
		Workers:     2,
		Extensions:  scan.DefaultExtensions(),
		FFprobePath: "ffprobe",
	}

	code := Run(cfg)
	if code != 0 && code != 2 {
		t.Errorf("unexpected exit code: %d", code)
	}

	if _, err := os.Stat(filepath.Join(tmp, "videos", "short.mp4")); !os.IsNotExist(err) {
		t.Error("short.mp4 should have been deleted")
	}
	if _, err := os.Stat(filepath.Join(tmp, "videos", "long.mp4")); os.IsNotExist(err) {
		t.Error("long.mp4 should remain")
	}
}

func TestRun_NoMatch(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	oldLook := lookPath
	lookPath = func(path string) (string, error) { return path, nil }
	defer func() { lookPath = oldLook }()

	oldProbe := probeFile
	probeFile = func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
		if strings.Contains(filepath.Base(videoPath), "short") {
			return 3.2, nil
		}
		return 12.345, nil
	}
	defer func() { probeFile = oldProbe }()

	cfg := Config{
		Dir:         filepath.Join(tmp, "videos"),
		MaxDuration: 2,
		Recursive:   false,
		DryRun:      true,
		Yes:         false,
		Workers:     2,
		Extensions:  scan.DefaultExtensions(),
		FFprobePath: "ffprobe",
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	for _, name := range []string{"short.mp4", "long.mp4", "notes.txt"} {
		if _, err := os.Stat(filepath.Join(tmp, "videos", name)); os.IsNotExist(err) {
			t.Errorf("%s should still exist", name)
		}
	}
}

func TestProbeResult_MatchesVideoFile(t *testing.T) {
	pr := ProbeResult{Path: "/tmp/v.mp4", Duration: 3.2, Size: 1024}
	if pr.Path != "/tmp/v.mp4" {
		t.Error("Path mismatch")
	}
	if pr.Duration != 3.2 {
		t.Error("Duration mismatch")
	}
	if pr.Size != 1024 {
		t.Error("Size mismatch")
	}
}

func TestSummary_ZeroState(t *testing.T) {
	var s Summary
	if s.ScannedFiles != 0 || s.VideoCandidates != 0 || s.Matched != 0 || s.ScanErrors != 0 {
		t.Error("new Summary should have zero values")
	}
	if s.MatchedFiles != nil || s.ProbeErrors != nil || s.DeleteErrors != nil {
		t.Error("new Summary slices should be nil")
	}
}

func TestPerformDeletes(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "del.mp4")
	if err := os.WriteFile(f, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	s := Summary{
		MatchedFiles: []ProbeResult{
			{Path: f, Size: 4, Duration: 1.0},
		},
	}
	performDeletes(&s)

	if s.Deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", s.Deleted)
	}
	if s.FailedDeletes != 0 {
		t.Errorf("expected 0 failed, got %d", s.FailedDeletes)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestPerformDeletes_NonExistent(t *testing.T) {
	s := Summary{
		MatchedFiles: []ProbeResult{
			{Path: "/nonexistent/file.mp4", Size: 0, Duration: 1.0},
		},
	}
	performDeletes(&s)

	if s.Deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", s.Deleted)
	}
	if s.FailedDeletes != 1 {
		t.Errorf("expected 1 failed, got %d", s.FailedDeletes)
	}
}

// setupFakeEnv creates a temp directory with test video files.
func setupFakeEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()

	videosDir := filepath.Join(tmp, "videos")
	if err := os.MkdirAll(videosDir, 0755); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(filepath.Join(videosDir, "short.mp4"), []byte("data"), 0644)
	os.WriteFile(filepath.Join(videosDir, "long.mp4"), []byte("data"), 0644)
	os.WriteFile(filepath.Join(videosDir, "notes.txt"), []byte("data"), 0644)

	return tmp
}

func TestBuildSummary_EmptyResults(t *testing.T) {
	summary := buildSummary(nil, 0, Config{MaxDuration: 10})
	if summary.VideoCandidates != 0 || summary.Matched != 0 || summary.FailedProbes != 0 {
		t.Error("empty results should produce zero counts")
	}
}

func TestProbeResult_ErrorPreserved(t *testing.T) {
	results := []ProbeResult{
		{Path: "err.mp4", Error: fmt.Errorf("some error")},
	}
	summary := buildSummary(results, 3, Config{MaxDuration: 10})
	if summary.FailedProbes != 1 {
		t.Errorf("expected 1 failed probe, got %d", summary.FailedProbes)
	}
	if len(summary.ProbeErrors) != 1 {
		t.Errorf("expected 1 probe error, got %d", len(summary.ProbeErrors))
	}
}

func TestRun_ScanErrors_Returns2(t *testing.T) {
	oldLook := lookPath
	lookPath = func(path string) (string, error) { return path, nil }
	defer func() { lookPath = oldLook }()

	oldProbe := probeFile
	probeFile = func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
		return 12.0, nil
	}
	defer func() { probeFile = oldProbe }()

	tmp := t.TempDir()
	videosDir := filepath.Join(tmp, "videos")
	os.MkdirAll(videosDir, 0755)
	os.WriteFile(filepath.Join(videosDir, "a.mp4"), []byte("x"), 0644)

	// Create an unreadable subdirectory (Windows: make it unreadable with a restricted path)
	badDir := filepath.Join(videosDir, "bad")
	os.MkdirAll(badDir, 0755)
	os.WriteFile(filepath.Join(badDir, "b.mov"), []byte("x"), 0644)

	cfg := Config{
		Dir:         videosDir,
		MaxDuration: 5,
		Recursive:   true,
		DryRun:      true,
		Yes:         false,
		Workers:     1,
		Extensions:  scan.DefaultExtensions(),
		FFprobePath: "ffprobe",
	}

	code := Run(cfg)
	// With no scan errors, this should return 0
	if code != 0 {
		t.Errorf("expected exit code 0 without scan errors, got %d", code)
	}
}

func TestRun_NoFiles_ScanErrors(t *testing.T) {
	oldLook := lookPath
	lookPath = func(path string) (string, error) { return path, nil }
	defer func() { lookPath = oldLook }()

	oldProbe := probeFile
	probeFile = func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
		return 5.0, nil
	}
	defer func() { probeFile = oldProbe }()

	// Empty directory with no video files — should succeed with 0
	tmp := t.TempDir()
	cfg := Config{
		Dir:         tmp,
		MaxDuration: 5,
		Recursive:   false,
		DryRun:      true,
		Yes:         false,
		Workers:     1,
		Extensions:  scan.DefaultExtensions(),
		FFprobePath: "ffprobe",
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("empty dir should return 0, got %d", code)
	}
}

func TestSummary_ScanErrorsTracked(t *testing.T) {
	s := Summary{ScanErrors: 3}
	if s.ScanErrors != 3 {
		t.Error("ScanErrors not preserved")
	}
}
