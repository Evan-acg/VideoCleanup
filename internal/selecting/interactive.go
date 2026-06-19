package selecting

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// PromptInteractive displays the numbered file list and prompts the user for a selection.
// It loops until a valid selection or cancellation is entered.
// Returns the sorted 1-based indices of selected items, or nil if cancelled.
func PromptInteractive(displayLines []string, stdin io.Reader, stderr io.Writer) ([]int, bool, error) {
	scanner := bufio.NewScanner(stdin)

	for i, line := range displayLines {
		fmt.Fprintf(stderr, "  %d. %s\n", i+1, line)
	}
	fmt.Fprintln(stderr)
	fmt.Fprintln(stderr, "Choose files to delete:")
	fmt.Fprintln(stderr, "  all          select all matched files")
	fmt.Fprintln(stderr, "  none / q     cancel")
	fmt.Fprintln(stderr, "  1,2,3        select by numbers")
	fmt.Fprintln(stderr, "  1-5          select a range")
	fmt.Fprintln(stderr, "  all,-2,-4    select all except numbers 2 and 4")

	for {
		fmt.Fprint(stderr, "Selection: ")
		if !scanner.Scan() {
			return nil, false, nil
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			return nil, false, nil
		}
		if strings.ToLower(input) == "q" {
			return nil, false, nil
		}

		n := len(displayLines)
		indices, err := Parse(input, n)
		if err != nil {
			fmt.Fprintf(stderr, "Invalid selection: %v\n", err)
			continue
		}
		if len(indices) == 0 {
			return nil, false, nil
		}
		return indices, true, nil
	}
}
