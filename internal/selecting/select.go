package selecting

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse parses a selection expression and returns the 1-based indices selected.
// n is the total number of available items.
func Parse(expr string, n int) ([]int, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty selection")
	}

	switch strings.ToLower(expr) {
	case "all":
		return rangeSlice(1, n), nil
	case "none", "q":
		return nil, nil
	}

	tokens := strings.Split(expr, ",")
	exclude := false
	if strings.ToLower(strings.TrimSpace(tokens[0])) == "all" {
		exclude = true
		tokens = tokens[1:]
	}

	addSet := make(map[int]bool)
	remSet := make(map[int]bool)

	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if strings.HasPrefix(tok, "-") {
			num, err := strconv.Atoi(strings.TrimPrefix(tok, "-"))
			if err != nil {
				return nil, fmt.Errorf("invalid token %q", tok)
			}
			if num < 1 || num > n {
				return nil, fmt.Errorf("index %d out of range [1..%d]", num, n)
			}
			remSet[num] = true
			continue
		}
		if strings.Contains(tok, "-") {
			parts := strings.SplitN(tok, "-", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid range %q", tok)
			}
			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q", tok)
			}
			end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q", tok)
			}
			if start < 1 || end > n || start > end {
				return nil, fmt.Errorf("range %d-%d out of range [1..%d]", start, end, n)
			}
			for i := start; i <= end; i++ {
				addSet[i] = true
			}
			continue
		}
		num, err := strconv.Atoi(tok)
		if err != nil {
			return nil, fmt.Errorf("invalid token %q", tok)
		}
		if num < 1 || num > n {
			return nil, fmt.Errorf("index %d out of range [1..%d]", num, n)
		}
		addSet[num] = true
	}

	if exclude {
		base := make(map[int]bool)
		for i := 1; i <= n; i++ {
			base[i] = true
		}
		for k := range remSet {
			delete(base, k)
		}
		return sortedKeys(base), nil
	}

	for k := range remSet {
		delete(addSet, k)
	}
	if len(addSet) == 0 {
		return nil, nil
	}
	return sortedKeys(addSet), nil
}

func rangeSlice(start, end int) []int {
	if start > end {
		return nil
	}
	out := make([]int, end-start+1)
	for i := range out {
		out[i] = start + i
	}
	return out
}

func sortedKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[i] > out[j] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
