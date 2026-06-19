package app

import (
	"sync"
	"time"

	"vidc/internal/probe"
	"vidc/internal/scan"
)

func probeAll(files []scan.VideoFile, ffprobePath string, workers int, probeFile func(string, string, time.Duration) (float64, error), onProbe func()) []ProbeResult {
	jobs := make(chan scan.VideoFile, len(files))
	results := make(chan ProbeResult, len(files))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				dur, err := probeFile(ffprobePath, f.Path, probe.DefaultTimeout)
				results <- ProbeResult{
					Path:     f.Path,
					Size:     f.Size,
					Duration: dur,
					Error:    err,
				}
				if onProbe != nil {
					onProbe()
				}
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var out []ProbeResult
	for r := range results {
		out = append(out, r)
	}
	return out
}
