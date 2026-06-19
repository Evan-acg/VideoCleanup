package app

// Run executes the cleanup with the given configuration using default dependencies.
// Returns an exit code: 0=success, 1=config error, 2=partial failure.
func Run(cfg Config) int {
	return NewRunner(DefaultDeps()).Run(cfg)
}
