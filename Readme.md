# vidc

A Go command-line tool that scans directories for video files and removes those shorter than a user-specified duration threshold.

## Prerequisites

- [Go](https://go.dev/dl/) 1.21+
- [FFmpeg](https://ffmpeg.org/download.html) (provides `ffprobe`)

Make sure `ffprobe` is in your PATH, or specify its location with `--ffprobe`.

## Install

```bash
go build -o bin/vidc ./cmd/vidc
```

## Usage

```
vidc -d <directory> -m <seconds> [flags]
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dir` | `-d` | string | *(required)* | Directory to scan |
| `--max-duration` | `-m` | float | *(required)* | Delete threshold in seconds |
| `--recursive` | `-r` | bool | `false` | Scan subdirectories recursively |
| `--dry-run` | | bool | `true` | Preview only, no files are deleted |
| `--yes` | `-y` | bool | `false` | Confirm deletion in non-interactive mode |
| `--workers` | `-w` | int | CPU count | Number of concurrent ffprobe workers |
| `--extensions` | `-e` | string | `.mp4,.mov,.mkv,...` | Comma-separated video extensions |
| `--ffprobe` | | string | `ffprobe` | Path to ffprobe executable |
| `--verbose` | `-v` | bool | `false` | Verbose output |
| `--select` | | string | `""` | Non-interactive selection: `all`, `1,2,5`, `1-5`, `all,-2` |
| `--confirm-delete` | | bool | `false` | Non-interactive deletion confirmation |
| `--no-progress` | | bool | `false` | Disable progress display |

### Examples

```bash
# Dry-run: preview short videos in a directory
vidc -d "D:\videos" -m 10

# Recursive dry-run
vidc -d "D:\videos" -m 10 -r

# Interactive delete with selection prompt and confirmation
vidc -d "D:\videos" -m 10 -r --dry-run=false

# Scripted delete all matching files (non-interactive)
vidc -d "D:\videos" -m 10 -r --dry-run=false --select all --confirm-delete

# Scripted delete by range (files 1 to 5)
vidc -d "D:\videos" -m 10 -r --dry-run=false --select 1-5 --confirm-delete

# Scripted delete all except file 3
vidc -d "D:\videos" -m 10 -r --dry-run=false --select all,-3 --confirm-delete

# Backward-compatible scripted delete (--yes implies confirm and defaults to all)
vidc -d "D:\videos" -m 10 -r -y --dry-run=false

# Only .mp4 and .mov files, with 8 workers
vidc -d "D:\videos" -m 5 -e "mp4,mov" -w 8

# Custom ffprobe path
vidc -d "D:\videos" -m 10 --ffprobe "C:\tools\ffprobe.exe"
```

## How It Works

1. **Scan** - Walks the directory (optionally recursive) and filters files by video extension whitelist. Progress is shown in real-time.
2. **Probe** - Uses `ffprobe` to extract the duration of each video candidate. Probing runs concurrently via a worker pool with a progress bar.
3. **List** - Prints matched short videos with numbering, duration, size, and path.
4. **Select** - In interactive mode, choose files by number, range, or `all`. In script mode, use `--select` with an expression.
5. **Confirm** - Interactive mode requires typing `delete` to confirm. Script mode requires `--confirm-delete` (or `--yes`).
6. **Delete** - Selected files are permanently removed with `os.Remove`. Progress is shown during deletion.
7. **Report** - Prints a summary of scanned, matched, deleted, and failed counts.

By default (`--dry-run=true`), the tool previews only and never deletes files.

### Selection Expressions

In non-interactive mode, use `--select` with these expressions:

| Expression | Meaning |
|-----------|---------|
| `all` | Select all matched files |
| `none` / `q` | Cancel (select nothing) |
| `1` | Select file #1 |
| `1,2,3,5,10` | Select specific numbers |
| `1-5` | Select range 1 through 5 |
| `all,-2,-4` | Select all except #2 and #4 |

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success (or user cancelled) |
| `1` | Parameter error, invalid selection, or missing confirmation |
| `2` | Completed with scan, probe, or delete failures |

## Project Structure

```
.
|-- cmd/vidc/main.go          # Entry point and flag parsing
|-- internal/
|   |-- app/run.go            # Orchestration, worker pool, reporting
|   |-- probe/ffprobe.go      # ffprobe invocation and duration parsing
|   |-- scan/scanner.go       # Directory walk and extension filtering
|   |-- cleanup/remover.go    # File deletion
|   |-- progress/             # Progress bars, spinner, TTY detection
|   |-- selecting/            # Selection expression parsing, interactive prompt
|-- go.mod
|-- Readme.md
```

## Testing

```bash
go test ./...
```

Tests use fake ffprobe scripts and temporary directories. No real video files or real ffprobe installation is required.

## License

MIT
