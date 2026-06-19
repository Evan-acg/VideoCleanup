package progress

import (
	"bytes"
	"strings"
	"testing"
)

func TestSpinner_Active(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(&buf, true)
	s.Update(100, 5)
	output := buf.String()
	if !strings.Contains(output, "100 files") {
		t.Errorf("expected file count in output: %s", output)
	}
	if !strings.Contains(output, "5 video candidates") {
		t.Errorf("expected video count in output: %s", output)
	}
	if !strings.HasPrefix(output, "\r") {
		t.Errorf("expected \\r prefix: got %q", output)
	}
}

func TestSpinner_Inactive(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(&buf, false)
	s.Update(100, 5)
	s.Stop()
	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

func TestSpinner_Stop_ClearsLine(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(&buf, true)
	s.Update(10, 2)
	s.Stop()
	output := buf.String()
	if !strings.Contains(output, "\r") {
		t.Errorf("expected \\r in output: %q", output)
	}
}

func TestBar_Active(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, true)
	b.Start("Probing", 10)
	output := buf.String()
	if !strings.Contains(output, "Probing") {
		t.Errorf("expected label in output: %s", output)
	}
	if !strings.Contains(output, "[") {
		t.Errorf("expected bar in output: %s", output)
	}
	if !strings.Contains(output, "0/10") {
		t.Errorf("expected progress in output: %s", output)
	}
}

func TestBar_Advance(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, true)
	b.Start("Probing", 10)
	b.Advance(5)
	output := buf.String()
	if !strings.Contains(output, "5/10") {
		t.Errorf("expected updated progress, got: %s", output)
	}
}

func TestBar_Complete(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, true)
	b.Start("Probing", 10)
	b.Advance(10)
	output := buf.String()
	if !strings.Contains(output, "10/10") {
		t.Errorf("expected complete progress, got: %s", output)
	}
}

func TestBar_Inactive(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, false)
	b.Start("Probing", 10)
	b.Advance(5)
	b.Finish()
	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

func TestBar_Finish(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, true)
	b.Start("Probing", 10)
	b.Finish()
	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("expected newline at end, got %q", output)
	}
}

func TestBar_ZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, true)
	b.Start("Test", 0)
	if buf.Len() != 0 {
		t.Errorf("expected no output for zero total, got %q", buf.String())
	}
}

func TestBar_ClampMax(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, true)
	b.Start("Probing", 5)
	b.Advance(10)
	if !strings.Contains(buf.String(), "5/5") {
		t.Errorf("expected clamped to 5/5, got: %s", buf.String())
	}
}
