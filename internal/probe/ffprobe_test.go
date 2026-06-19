package probe

import (
	"testing"
)

func TestParseDuration_Valid(t *testing.T) {
	in := []byte(`{"format": {"duration": "12.345000"}}`)
	d, err := ParseDuration(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 12.345 {
		t.Errorf("expected 12.345, got %f", d)
	}
}

func TestParseDuration_MissingDuration(t *testing.T) {
	in := []byte(`{"format": {}}`)
	_, err := ParseDuration(in)
	if err == nil {
		t.Error("expected error for missing duration")
	}
}

func TestParseDuration_EmptyDuration(t *testing.T) {
	in := []byte(`{"format": {"duration": ""}}`)
	_, err := ParseDuration(in)
	if err == nil {
		t.Error("expected error for empty duration")
	}
}

func TestParseDuration_InvalidJSON(t *testing.T) {
	in := []byte(`not json`)
	_, err := ParseDuration(in)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseDuration_NonNumeric(t *testing.T) {
	in := []byte(`{"format": {"duration": "abc"}}`)
	_, err := ParseDuration(in)
	if err == nil {
		t.Error("expected error for non-numeric duration")
	}
}

func TestParseDuration_Zero(t *testing.T) {
	in := []byte(`{"format": {"duration": "0.000000"}}`)
	d, err := ParseDuration(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 0 {
		t.Errorf("expected 0, got %f", d)
	}
}

func TestParseDuration_Integer(t *testing.T) {
	in := []byte(`{"format": {"duration": "42"}}`)
	d, err := ParseDuration(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 42 {
		t.Errorf("expected 42, got %f", d)
	}
}

func TestParseDuration_MissingFormat(t *testing.T) {
	in := []byte(`{}`)
	_, err := ParseDuration(in)
	if err == nil {
		t.Error("expected error for missing format")
	}
}
