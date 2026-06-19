package app

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vidc/internal/cleanup"
	"vidc/internal/scan"
)

func fakeLookPath(path string) (string, error) { return path, nil }

func fakeLookPathNotFound(path string) (string, error) { return "", fmt.Errorf("not found") }

func TestValidateConfig_MissingDir(t *testing.T) {
	_, err := ValidateConfig(Config{Dir: ""}, fakeLookPath)
	if err == nil {
		t.Error("expected error for missing dir")
	}
}

func TestValidateConfig_NonExistentDir(t *testing.T) {
	_, err := ValidateConfig(Config{Dir: "/nonexistent/dir"}, fakeLookPath)
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
	_, err := ValidateConfig(Config{Dir: f}, fakeLookPath)
	if err == nil {
		t.Error("expected error for non-directory path")
	}
}

func TestValidateConfig_ZeroMaxDuration(t *testing.T) {
	dir := t.TempDir()
	_, err := ValidateConfig(Config{Dir: dir, MaxDuration: 0, Workers: 1}, fakeLookPath)
	if err == nil {
		t.Error("expected error for zero max-duration")
	}
}

func TestValidateConfig_ZeroWorkers(t *testing.T) {
	dir := t.TempDir()
	_, err := ValidateConfig(Config{Dir: dir, MaxDuration: 5, Workers: 0}, fakeLookPath)
	if err == nil {
		t.Error("expected error for zero workers")
	}
}

func TestValidateConfig_FFprobeNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ValidateConfig(Config{Dir: dir, MaxDuration: 5, Workers: 1}, fakeLookPathNotFound)
	if err == nil {
		t.Error("expected error for missing ffprobe")
	}
}

func TestValidateConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	code, err := ValidateConfig(Config{Dir: dir, MaxDuration: 5, Workers: 2}, fakeLookPath)
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

func fakeProbeShort(videoPath string) float64 {
	if strings.Contains(filepath.Base(videoPath), "short") {
		return 3.2
	}
	return 12.345
}

func newRunnerForTest(t *testing.T, cfg Config, probeFn func(string) float64) *Runner {
	t.Helper()
	return NewRunner(Deps{
		LookPath:   fakeLookPath,
		IsTerminal: func() bool { return false },
		ProbeFile: func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
			return probeFn(videoPath), nil
		},
		RemoveFile: cleanup.Remove,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Stdin:      &bytes.Buffer{},
	})
}

func TestRun_DryRun(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

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
	runner := newRunnerForTest(t, cfg, fakeProbeShort)
	code := runner.Run(cfg)
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
	runner := newRunnerForTest(t, cfg, fakeProbeShort)
	code := runner.Run(cfg)
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
	runner := newRunnerForTest(t, cfg, fakeProbeShort)
	code := runner.Run(cfg)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	for _, name := range []string{"short.mp4", "long.mp4", "notes.txt"} {
		if _, err := os.Stat(filepath.Join(tmp, "videos", name)); os.IsNotExist(err) {
			t.Errorf("%s should still exist", name)
		}
	}
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
	tmp := t.TempDir()
	videosDir := filepath.Join(tmp, "videos")
	os.MkdirAll(videosDir, 0755)
	os.WriteFile(filepath.Join(videosDir, "a.mp4"), []byte("x"), 0644)

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
	runner := NewRunner(Deps{
		LookPath:   fakeLookPath,
		IsTerminal: func() bool { return false },
		ProbeFile: func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
			return 12.0, nil
		},
		RemoveFile: cleanup.Remove,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Stdin:      &bytes.Buffer{},
	})
	code := runner.Run(cfg)
	if code != 0 {
		t.Errorf("expected exit code 0 without scan errors, got %d", code)
	}
}

func TestRun_NoFiles_ScanErrors(t *testing.T) {
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
	runner := NewRunner(Deps{
		LookPath:   fakeLookPath,
		IsTerminal: func() bool { return false },
		ProbeFile: func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
			return 5.0, nil
		},
		RemoveFile: cleanup.Remove,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Stdin:      &bytes.Buffer{},
	})
	code := runner.Run(cfg)
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

func TestRun_SelectAll_NonInteractive(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	cfg := Config{
		Dir:           filepath.Join(tmp, "videos"),
		MaxDuration:   10,
		Recursive:     false,
		DryRun:        false,
		Yes:           false,
		Workers:       2,
		Extensions:    scan.DefaultExtensions(),
		FFprobePath:   "ffprobe",
		SelectExpr:    "all",
		ConfirmDelete: true,
	}
	runner := newRunnerForTest(t, cfg, fakeProbeShort)
	code := runner.Run(cfg)
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

func TestRun_SelectByNumber_NonInteractive(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	cfg := Config{
		Dir:           filepath.Join(tmp, "videos"),
		MaxDuration:   10,
		Recursive:     false,
		DryRun:        false,
		Yes:           false,
		Workers:       2,
		Extensions:    scan.DefaultExtensions(),
		FFprobePath:   "ffprobe",
		SelectExpr:    "1",
		ConfirmDelete: true,
	}
	runner := NewRunner(Deps{
		LookPath:   fakeLookPath,
		IsTerminal: func() bool { return false },
		ProbeFile: func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
			return 3.2, nil
		},
		RemoveFile: cleanup.Remove,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Stdin:      &bytes.Buffer{},
	})
	code := runner.Run(cfg)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_NonInteractive_MissingConfirm(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	cfg := Config{
		Dir:           filepath.Join(tmp, "videos"),
		MaxDuration:   10,
		Recursive:     false,
		DryRun:        false,
		Yes:           false,
		Workers:       2,
		Extensions:    scan.DefaultExtensions(),
		FFprobePath:   "ffprobe",
		SelectExpr:    "all",
		ConfirmDelete: false,
	}
	runner := NewRunner(Deps{
		LookPath:   fakeLookPath,
		IsTerminal: func() bool { return false },
		ProbeFile: func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
			return 3.2, nil
		},
		RemoveFile: cleanup.Remove,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Stdin:      &bytes.Buffer{},
	})
	code := runner.Run(cfg)
	if code != 1 {
		t.Errorf("expected exit code 1 for missing confirm, got %d", code)
	}

	if _, err := os.Stat(filepath.Join(tmp, "videos", "short.mp4")); os.IsNotExist(err) {
		t.Error("short.mp4 should not be deleted without confirm")
	}
}

func TestRun_NonInteractive_MissingSelect(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	cfg := Config{
		Dir:           filepath.Join(tmp, "videos"),
		MaxDuration:   10,
		Recursive:     false,
		DryRun:        false,
		Yes:           false,
		Workers:       2,
		Extensions:    scan.DefaultExtensions(),
		FFprobePath:   "ffprobe",
		SelectExpr:    "",
		ConfirmDelete: true,
	}
	runner := NewRunner(Deps{
		LookPath:   fakeLookPath,
		IsTerminal: func() bool { return false },
		ProbeFile: func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
			return 3.2, nil
		},
		RemoveFile: cleanup.Remove,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Stdin:      &bytes.Buffer{},
	})
	code := runner.Run(cfg)
	if code != 1 {
		t.Errorf("expected exit code 1 for missing select, got %d", code)
	}
}

func TestRun_YesBackwardCompat(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

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
	runner := newRunnerForTest(t, cfg, fakeProbeShort)
	code := runner.Run(cfg)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	if _, err := os.Stat(filepath.Join(tmp, "videos", "short.mp4")); !os.IsNotExist(err) {
		t.Error("short.mp4 should have been deleted")
	}
	if _, err := os.Stat(filepath.Join(tmp, "videos", "long.mp4")); os.IsNotExist(err) {
		t.Error("long.mp4 should remain")
	}
}

func TestRun_DryRun_NeverDeletes(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	cfg := Config{
		Dir:           filepath.Join(tmp, "videos"),
		MaxDuration:   10,
		Recursive:     false,
		DryRun:        true,
		Yes:           false,
		Workers:       2,
		Extensions:    scan.DefaultExtensions(),
		FFprobePath:   "ffprobe",
		SelectExpr:    "all",
		ConfirmDelete: true,
	}
	runner := NewRunner(Deps{
		LookPath:   fakeLookPath,
		IsTerminal: func() bool { return false },
		ProbeFile: func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
			return 3.2, nil
		},
		RemoveFile: cleanup.Remove,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Stdin:      &bytes.Buffer{},
	})
	code := runner.Run(cfg)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	if _, err := os.Stat(filepath.Join(tmp, "videos", "short.mp4")); os.IsNotExist(err) {
		t.Error("dry-run should not delete files")
	}
}

func TestRun_InvalidSelectExpr(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	cfg := Config{
		Dir:           filepath.Join(tmp, "videos"),
		MaxDuration:   10,
		Recursive:     false,
		DryRun:        false,
		Yes:           false,
		Workers:       2,
		Extensions:    scan.DefaultExtensions(),
		FFprobePath:   "ffprobe",
		SelectExpr:    "invalid",
		ConfirmDelete: true,
	}
	runner := NewRunner(Deps{
		LookPath:   fakeLookPath,
		IsTerminal: func() bool { return false },
		ProbeFile: func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
			return 3.2, nil
		},
		RemoveFile: cleanup.Remove,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Stdin:      &bytes.Buffer{},
	})
	code := runner.Run(cfg)
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid expression, got %d", code)
	}
}

func TestPerformSelectedDeletes(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.mp4")
	f2 := filepath.Join(dir, "b.mp4")
	os.WriteFile(f1, []byte("data"), 0644)
	os.WriteFile(f2, []byte("data"), 0644)

	s := &Summary{}
	files := []ProbeResult{
		{Path: f1, Size: 4, Duration: 1.0},
		{Path: f2, Size: 4, Duration: 2.0},
	}
	performSelectedDeletes(s, files[:1], cleanup.Remove, nil)

	if s.Deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", s.Deleted)
	}
	if _, err := os.Stat(f1); !os.IsNotExist(err) {
		t.Error("f1 should have been deleted")
	}
	if _, err := os.Stat(f2); os.IsNotExist(err) {
		t.Error("f2 should still exist")
	}
}

func TestPerformSelectedDeletes_WithProgress(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "x.mp4")
	os.WriteFile(f, []byte("data"), 0644)

	s := &Summary{}
	progressCalls := 0
	files := []ProbeResult{{Path: f, Size: 4, Duration: 1.0}}
	performSelectedDeletes(s, files, cleanup.Remove, func(deleted int) {
		progressCalls++
	})

	if progressCalls != 1 {
		t.Errorf("expected 1 progress call, got %d", progressCalls)
	}
	if s.Deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", s.Deleted)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1 KB"},
		{1536, "2 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
	}
	for _, tc := range tests {
		got := formatSize(tc.size)
		if got != tc.expected {
			t.Errorf("formatSize(%d): expected %q, got %q", tc.size, tc.expected, got)
		}
	}
}

func TestRun_NoMatch_DeleteMode(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	cfg := Config{
		Dir:         filepath.Join(tmp, "videos"),
		MaxDuration: 5,
		Recursive:   false,
		DryRun:      false,
		Yes:         false,
		Workers:     1,
		Extensions:  scan.DefaultExtensions(),
		FFprobePath: "ffprobe",
	}
	runner := NewRunner(Deps{
		LookPath:   fakeLookPath,
		IsTerminal: func() bool { return false },
		ProbeFile: func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
			return 12.0, nil
		},
		RemoveFile: cleanup.Remove,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Stdin:      &bytes.Buffer{},
	})
	code := runner.Run(cfg)
	if code != 0 {
		t.Errorf("expected exit code 0 when no files match, got %d", code)
	}
}

func TestRun_SelectExprTakesPriority(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	cfg := Config{
		Dir:           filepath.Join(tmp, "videos"),
		MaxDuration:   10,
		Recursive:     false,
		DryRun:        false,
		Yes:           false,
		Workers:       2,
		Extensions:    scan.DefaultExtensions(),
		FFprobePath:   "ffprobe",
		SelectExpr:    "1",
		ConfirmDelete: true,
	}
	runner := newRunnerForTest(t, cfg, fakeProbeShort)
	code := runner.Run(cfg)
	if code != 0 {
		t.Errorf("expected exit code 0 when --select is provided in non-interactive mode, got %d", code)
	}

	if _, err := os.Stat(filepath.Join(tmp, "videos", "short.mp4")); !os.IsNotExist(err) {
		t.Error("selected file should have been deleted")
	}
}

func TestPerformSelectedDeletes_ProgressOnFailure(t *testing.T) {
	s := &Summary{}

	progressCalls := 0
	files := []ProbeResult{
		{Path: "/nonexistent/a.mp4", Size: 4, Duration: 1.0},
		{Path: "/nonexistent/b.mp4", Size: 4, Duration: 2.0},
	}
	performSelectedDeletes(s, files, cleanup.Remove, func(processed int) {
		progressCalls++
	})

	if progressCalls != 2 {
		t.Errorf("expected 2 progress calls (one per file), got %d", progressCalls)
	}
	if s.FailedDeletes != 2 {
		t.Errorf("expected 2 failed deletes, got %d", s.FailedDeletes)
	}
	if s.Deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", s.Deleted)
	}
}

func TestRun_SelectExprOverridesTTY(t *testing.T) {
	tmp := setupFakeEnv(t)
	defer os.RemoveAll(tmp)

	cfg := Config{
		Dir:           filepath.Join(tmp, "videos"),
		MaxDuration:   10,
		Recursive:     false,
		DryRun:        false,
		Yes:           false,
		Workers:       2,
		Extensions:    scan.DefaultExtensions(),
		FFprobePath:   "ffprobe",
		SelectExpr:    "2",
		ConfirmDelete: true,
	}
	runner := NewRunner(Deps{
		LookPath:   fakeLookPath,
		IsTerminal: func() bool { return true },
		ProbeFile: func(ffprobePath, videoPath string, timeout time.Duration) (float64, error) {
			return 3.2, nil
		},
		RemoveFile: cleanup.Remove,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Stdin:      &bytes.Buffer{},
	})
	code := runner.Run(cfg)
	if code != 0 {
		t.Errorf("expected exit code 0 when --select overrides simulated TTY, got %d", code)
	}

	if _, err := os.Stat(filepath.Join(tmp, "videos", "short.mp4")); !os.IsNotExist(err) {
		t.Error("file #2 should have been deleted (--select took non-interactive path)")
	}
}

func TestDisplayPath_RelativeToScanDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "short.mp4"), []byte("x"), 0644)

	filePath := filepath.Join(sub, "short.mp4")
	got := displayPath(dir, filePath)

	expected := filepath.Join("sub", "short.mp4")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestDisplayPath_RootDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.mp4"), []byte("x"), 0644)

	filePath := filepath.Join(dir, "a.mp4")
	got := displayPath(dir, filePath)

	if got != "a.mp4" {
		t.Errorf("expected a.mp4, got %q", got)
	}
}

func TestDisplayPath_OutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	otherDir := t.TempDir()
	os.WriteFile(filepath.Join(otherDir, "b.mp4"), []byte("x"), 0644)

	filePath := filepath.Join(otherDir, "b.mp4")
	got := displayPath(baseDir, filePath)

	if !strings.Contains(got, "..") && got != filepath.Clean(filePath) {
		t.Errorf("unexpected result for path outside base: %q", got)
	}
}

func TestDisplayPath_NonExistentDir(t *testing.T) {
	got := displayPath("/nonexistent/base", "/nonexistent/base/sub/a.mp4")
	if got == "" {
		t.Error("expected non-empty result for non-existent directory")
	}
}

func TestPrintMatchedFiles_UsesTableAndRelativePaths(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "clips")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "a.mp4"), []byte("d"), 0644)

	files := []ProbeResult{
		{Path: filepath.Join(sub, "a.mp4"), Duration: 3.2, Size: 1572864},
	}

	var buf bytes.Buffer
	printMatchedFiles(&buf, dir, files)
	out := buf.String()

	if !strings.Contains(out, "Matched files:") {
		t.Error("expected 'Matched files:' header")
	}
	if !strings.Contains(out, "Duration") || !strings.Contains(out, "Size") || !strings.Contains(out, "Path") {
		t.Errorf("expected table headers Duration, Size, Path in output: %s", out)
	}
	if !strings.Contains(out, "1.5 MB") {
		t.Errorf("expected formatted size 1.5 MB in output: %s", out)
	}
	if strings.Contains(out, dir) {
		t.Errorf("output should not contain full base directory path: %s", out)
	}
}

func TestPrintMatchedFiles_EmptyList(t *testing.T) {
	var buf bytes.Buffer
	printMatchedFiles(&buf, "", nil)
	out := buf.String()

	if out != "" {
		t.Errorf("expected empty output, got: %s", out)
	}
}
