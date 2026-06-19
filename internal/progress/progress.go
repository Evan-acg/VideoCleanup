package progress

import (
	"fmt"
	"io"
	"strings"
)

// Spinner shows indeterminate progress updates, typically for the scan phase.
type Spinner struct {
	w      io.Writer
	active bool
}

// NewSpinner creates a Spinner that writes to w when active is true.
func NewSpinner(w io.Writer, active bool) *Spinner {
	return &Spinner{w: w, active: active}
}

// Update writes a progress line with file counts.
func (s *Spinner) Update(totalFiles int, videoCount int) {
	if !s.active {
		return
	}
	fmt.Fprintf(s.w, "\rScanning... %d files, %d video candidates", totalFiles, videoCount)
}

// Stop clears the line so subsequent output starts clean.
func (s *Spinner) Stop() {
	if !s.active {
		return
	}
	fmt.Fprint(s.w, "\r"+strings.Repeat(" ", 80)+"\r")
}

// Bar shows determinate progress with a visual bar, typically for probe/delete phases.
type Bar struct {
	w      io.Writer
	active bool
	label  string
	total  int
	width  int
}

// NewBar creates a Bar that writes to w when active is true.
func NewBar(w io.Writer, active bool) *Bar {
	return &Bar{w: w, active: active, width: 30}
}

// Start initiates a new progress bar with the given label and total steps.
func (b *Bar) Start(label string, total int) {
	b.label = label
	b.total = total
	b.render(0)
}

// Advance updates the bar to the given current step.
func (b *Bar) Advance(current int) {
	b.render(current)
}

// Finish completes the bar and moves to the next line.
func (b *Bar) Finish() {
	if !b.active {
		return
	}
	fmt.Fprint(b.w, "\n")
}

func (b *Bar) render(current int) {
	if !b.active || b.total <= 0 {
		return
	}
	if current > b.total {
		current = b.total
	}
	filled := b.width * current / b.total
	bar := strings.Repeat("#", filled) + strings.Repeat("-", b.width-filled)
	fmt.Fprintf(b.w, "\r%s [%s] %d/%d", b.label, bar, current, b.total)
}
