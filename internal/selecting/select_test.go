package selecting

import (
	"testing"
)

func TestParse_All(t *testing.T) {
	idx, err := Parse("all", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx) != 5 {
		t.Errorf("expected 5 indices, got %d", len(idx))
	}
	for i, v := range idx {
		if v != i+1 {
			t.Errorf("expected %d, got %d", i+1, v)
		}
	}
}

func TestParse_None(t *testing.T) {
	idx, err := Parse("none", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx) != 0 {
		t.Errorf("expected 0 indices, got %d", len(idx))
	}
}

func TestParse_Q(t *testing.T) {
	idx, err := Parse("q", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx) != 0 {
		t.Errorf("expected 0 indices, got %d", len(idx))
	}
}

func TestParse_Single(t *testing.T) {
	idx, err := Parse("3", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx) != 1 || idx[0] != 3 {
		t.Errorf("expected [3], got %v", idx)
	}
}

func TestParse_Multiple(t *testing.T) {
	idx, err := Parse("1,3,5", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx) != 3 {
		t.Errorf("expected 3 indices, got %d: %v", len(idx), idx)
	}
	if idx[0] != 1 || idx[1] != 3 || idx[2] != 5 {
		t.Errorf("expected [1,3,5], got %v", idx)
	}
}

func TestParse_Range(t *testing.T) {
	idx, err := Parse("3-7", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{3, 4, 5, 6, 7}
	if len(idx) != len(expected) {
		t.Fatalf("expected %d indices, got %d", len(expected), len(idx))
	}
	for i, v := range expected {
		if idx[i] != v {
			t.Errorf("position %d: expected %d, got %d", i, v, idx[i])
		}
	}
}

func TestParse_AllExcept(t *testing.T) {
	idx, err := Parse("all,-2,-4", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 3, 5}
	if len(idx) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, idx)
	}
	for i, v := range expected {
		if idx[i] != v {
			t.Errorf("position %d: expected %d, got %d", i, v, idx[i])
		}
	}
}

func TestParse_ExcludeOnly(t *testing.T) {
	idx, err := Parse("all,-1,-2,-3", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx) != 0 {
		t.Errorf("expected empty, got %v", idx)
	}
}

func TestParse_EmptyString(t *testing.T) {
	_, err := Parse("", 5)
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestParse_Whitespace(t *testing.T) {
	idx, err := Parse(" 1 , 3 , 5 ", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 3, 5}
	if len(idx) != len(expected) || idx[0] != 1 || idx[1] != 3 || idx[2] != 5 {
		t.Errorf("expected [1,3,5], got %v", idx)
	}
}

func TestParse_OutOfRange(t *testing.T) {
	_, err := Parse("6", 5)
	if err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestParse_ZeroIndex(t *testing.T) {
	_, err := Parse("0", 5)
	if err == nil {
		t.Error("expected error for index 0")
	}
}

func TestParse_RangeOutOfRange(t *testing.T) {
	_, err := Parse("3-11", 10)
	if err == nil {
		t.Error("expected error for out-of-range range")
	}
}

func TestParse_NegativeRange(t *testing.T) {
	_, err := Parse("3-1", 5)
	if err == nil {
		t.Error("expected error for reversed range")
	}
}

func TestParse_InvalidToken(t *testing.T) {
	_, err := Parse("abc", 5)
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestParse_ExcludeInvalid(t *testing.T) {
	_, err := Parse("all,-abc", 5)
	if err == nil {
		t.Error("expected error for invalid exclude token")
	}
}

func TestParse_MixedCommasAndRanges(t *testing.T) {
	idx, err := Parse("1,3-5,8", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 3, 4, 5, 8}
	if len(idx) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, idx)
	}
	for i, v := range expected {
		if idx[i] != v {
			t.Errorf("position %d: expected %d, got %d", i, v, idx[i])
		}
	}
}

func TestParse_Deduplicates(t *testing.T) {
	idx, err := Parse("1,1,2,2", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 2}
	if len(idx) != len(expected) {
		t.Errorf("expected %v, got %v", expected, idx)
	}
}

func TestParse_SortedOutput(t *testing.T) {
	idx, err := Parse("5,2,8,1", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i < len(idx); i++ {
		if idx[i] <= idx[i-1] {
			t.Errorf("output not sorted: %v", idx)
			break
		}
	}
}

func TestParse_AllWithNoItems(t *testing.T) {
	idx, err := Parse("all", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx) != 0 {
		t.Errorf("expected empty, got %v", idx)
	}
}
