package app

import (
	"fmt"
	"io"

	"vidc/internal/selecting"
)

// SelectionResult holds the outcome of the file selection step.
type SelectionResult struct {
	Files     []ProbeResult
	Cancelled bool
}

// selectFiles determines which matched files should be deleted based on config and terminal state.
// Returns an error for invalid configuration (missing --select, invalid expression, missing --confirm-delete).
func selectFiles(cfg Config, interactive bool, matched []ProbeResult, stdin io.Reader, stdout, stderr io.Writer) (SelectionResult, error) {
	if interactive && cfg.SelectExpr == "" && !cfg.Yes {
		lines := make([]string, len(matched))
		for i, f := range matched {
			lines[i] = fmt.Sprintf("[%.2fs] %6s  %s", f.Duration, formatSize(f.Size), displayPath(cfg.Dir, f.Path))
		}
		indices, ok, err := selecting.PromptInteractive(lines, stdin, stderr)
		if err != nil {
			return SelectionResult{}, err
		}
		if !ok || len(indices) == 0 {
			fmt.Fprintln(stdout, "No files selected. Nothing to delete.")
			return SelectionResult{}, nil
		}
		var filtered []ProbeResult
		for _, idx := range indices {
			filtered = append(filtered, matched[idx-1])
		}
		if !confirmInteractive(stdin, stderr) {
			return SelectionResult{Files: filtered, Cancelled: true}, nil
		}
		return SelectionResult{Files: filtered}, nil
	}

	expr := cfg.SelectExpr
	if expr == "" && cfg.Yes {
		expr = "all"
		fmt.Fprintf(stderr, "Warning: --yes without --select defaults to 'all'. Use --select explicitly.\n")
	}
	if expr == "" {
		return SelectionResult{}, fmt.Errorf("non-interactive mode requires --select. Use --select all to select everything")
	}

	indices, err := selecting.Parse(expr, len(matched))
	if err != nil {
		return SelectionResult{}, fmt.Errorf("invalid --select expression: %w", err)
	}
	if len(indices) == 0 {
		fmt.Fprintln(stdout, "No files selected. Nothing to delete.")
		return SelectionResult{}, nil
	}
	var filtered []ProbeResult
	for _, idx := range indices {
		filtered = append(filtered, matched[idx-1])
	}

	if !cfg.ConfirmDelete && !cfg.Yes {
		return SelectionResult{}, fmt.Errorf("non-interactive mode requires --confirm-delete (or --yes) to proceed")
	}
	return SelectionResult{Files: filtered}, nil
}
