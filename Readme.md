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
| `--yes` | `-y` | bool | `false` | Actually delete matched files |
| `--workers` | `-w` | int | CPU count | Number of concurrent ffprobe workers |
| `--extensions` | `-e` | string | `.mp4,.mov,.mkv,...` | Comma-separated video extensions |
| `--ffprobe` | | string | `ffprobe` | Path to ffprobe executable |
| `--verbose` | `-v` | bool | `false` | Verbose output |

### Examples

```bash
# Dry-run: preview short videos in a directory
vidc -d "D:\videos" -m 10

# Recursive dry-run
vidc -d "D:\videos" -m 10 -r

# Actually delete short videos (must explicitly disable dry-run)
vidc -d "D:\videos" -m 10 -r -y --dry-run=false

# Only .mp4 and .mov files, with 8 workers
vidc -d "D:\videos" -m 5 -e "mp4,mov" -w 8

# Custom ffprobe path
vidc -d "D:\videos" -m 10 --ffprobe "C:\tools\ffprobe.exe"
```

## How It Works

1. **Scan** - Walks the directory (optionally recursive) and filters files by video extension whitelist.
2. **Probe** - Uses `ffprobe` to extract the duration of each video candidate. Probing runs concurrently via a worker pool.
3. **Report** - Prints matched short videos and a summary. By default (`--dry-run=true`), no files are deleted.
4. **Delete** - With `--dry-run=false --yes`, matched files are permanently removed with `os.Remove`.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Parameter or environment error |
| `2` | Completed with probe or delete failures |

## Project Structure

```
.
|-- cmd/vidc/main.go        # Entry point and flag parsing
|-- internal/
|   |-- app/run.go          # Orchestration, worker pool, reporting
|   |-- probe/ffprobe.go    # ffprobe invocation and duration parsing
|   |-- scan/scanner.go     # Directory walk and extension filtering
|   |-- cleanup/remover.go  # File deletion
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
