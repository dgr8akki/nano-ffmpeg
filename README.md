<p align="center">
  <br>
  <strong>nano-ffmpeg</strong>
  <br>
  <em>Every ffmpeg feature. Zero flags to remember.</em>
  <br><br>
  <a href="https://nano-ffmpeg.vercel.app">Website</a> &bull;
  <a href="#quick-start">Quick Start</a> &bull;
  <a href="#install">Install</a> &bull;
  <a href="#features">Features</a> &bull;
  <a href="#usage">Usage</a> &bull;
  <a href="#cli-options">CLI Options</a> &bull;
  <a href="#operations">Operations</a> &bull;
  <a href="#keybindings">Keybindings</a> &bull;
  <a href="#releasing">Releasing</a> &bull;
  <a href="#contributing">Contributing</a> &bull;
  <a href="#license">License</a>
</p>

<p align="center">
  <a href="https://github.com/dgr8akki/nano-ffmpeg/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/dgr8akki/nano-ffmpeg/ci.yml?branch=main&label=CI" alt="CI"></a>
  <a href="https://github.com/dgr8akki/nano-ffmpeg/releases/latest"><img src="https://img.shields.io/github/v/release/dgr8akki/nano-ffmpeg?sort=semver" alt="Latest release"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/dgr8akki/nano-ffmpeg" alt="Go version"></a>
  <a href="#license"><img src="https://img.shields.io/github/license/dgr8akki/nano-ffmpeg" alt="License"></a>
</p>

---

nano-ffmpeg wraps the full power of ffmpeg in a beautiful, keyboard-driven terminal dashboard. No more googling flags. Browse your files, pick what you want to do, tweak settings with presets, and watch a live progress bar while it encodes.

Built for people who know they need ffmpeg but can't remember how to use it.

```
╭─────────────────────────────────────────────────────────────────────╮
│  nano-ffmpeg > Home                                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ╭──────────────────────────────────────────────────────────────╮   │
│  │  ffmpeg 8.1                                                  │   │
│  │  497 codecs  |  231 encoders  |  234 formats  |  489 filters │   │
│  │  HW Accel: videotoolbox                                      │   │
│  ╰──────────────────────────────────────────────────────────────╯   │
│                                                                     │
│  RECENT FILES                                                       │
│     interview.mp4    ~/Videos                                       │
│     concert.mkv      ~/Downloads                                    │
│                                                                     │
│  OPERATIONS                                                         │
│   > Convert Format     Change container or codec                    │
│     Extract Audio      Strip video, keep audio                      │
│     Resize / Scale     Change resolution                            │
│     Trim / Cut         Cut segments by time                         │
│     Compress           Reduce file size                             │
│     ...                                                             │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  ↑↓ Navigate   Enter Select   q Quit   ? Help                      │
╰─────────────────────────────────────────────────────────────────────╯
```

## Quick Start

If you already have `ffmpeg` and `ffprobe` on your `PATH`:

```bash
# macOS (Homebrew tap -- also pulls ffmpeg-full):
brew install dgr8akki/tap/nano-ffmpeg
nano-ffmpeg
```

```powershell
# Windows (Scoop -- pulls ffmpeg from the extras bucket):
scoop bucket add extras
scoop bucket add nano-ffmpeg https://github.com/dgr8akki/scoop-bucket
scoop install nano-ffmpeg
nano-ffmpeg
```

```bash
# Any Go toolchain:
go install github.com/dgr8akki/nano-ffmpeg@latest
nano-ffmpeg
```

Jump straight into a specific file without clicking through the file picker:

```bash
nano-ffmpeg -d ~/Videos/interview.mp4
```

The TUI takes you from there: pick an operation, tweak the pre-filled defaults, hit Enter. See [Usage](#usage) for the full flow and [CLI Options](#cli-options) for every flag.

## Features

**Core**
- 12 ffmpeg operations accessible through guided, multi-screen workflows
- Pre-filled defaults for every operation so you can hit Enter without thinking about flags
- Command preview on every settings screen -- see the exact `ffmpeg` command before it runs
- Trim pre-fills the input's total duration; Stabilize automatically falls back to `deshake` if `vidstab` isn't in your ffmpeg build

**Progress Tracking**
- Gradient progress bar (green-to-cyan) with percentage
- Real-time stats: elapsed, ETA (smoothed over rolling window), speed, FPS, bitrate, frames, output size
- Braille-dot spinner for indeterminate operations (stream copy, concat)
- Scrollable live log of raw ffmpeg output
- Cancel with confirmation (`Esc` > `y`)

**File Handling**
- Built-in file browser with directory navigation
- Path input mode (toggle with `/`) for when you know exactly where your file is
- Inline `ffprobe` metadata preview: codec, resolution, framerate, audio, duration, size
- Recent files list on the home screen

**Intelligence**
- Capability detection: probes your ffmpeg build on startup and reports codec/format/filter/HW-accel counts on the Home screen
- Hardware acceleration detection: shows available accelerators (VideoToolbox, NVENC, VAAPI) on Home (note: detected accelerators are not yet applied to encode commands -- see `docs/future_scope.md`)
- Human-readable error translation: converts cryptic ffmpeg errors into actionable messages
- Capability cache at `~/.config/nano-ffmpeg/capabilities.json` (invalidated on version change)

**Polish**
- Context-sensitive help overlay (`?` on any screen)
- Persistent config: recent files, preferences at `~/.config/nano-ffmpeg/config.json`
- Responsive layout with 80x24 minimum terminal size detection
- Keyboard-first design with vim-style navigation (`j`/`k`)

## Requirements

- **ffmpeg** and **ffprobe** installed and available in `$PATH`
- For full Stabilize support (`vidstabdetect`/`vidstabtransform`), use an ffmpeg build with `libvidstab` (Homebrew: `ffmpeg-full`)
- Go 1.22+ (for building from source)
- Terminal: 80x24 minimum

### Installing ffmpeg

```bash
# macOS
brew install ffmpeg-full

# macOS (minimal build, Stabilize falls back to deshake)
brew install ffmpeg

# Ubuntu / Debian
sudo apt install ffmpeg

# Fedora
sudo dnf install ffmpeg

# Arch
sudo pacman -S ffmpeg

# Windows (Scoop, recommended -- matches what nano-ffmpeg pulls in)
scoop bucket add extras
scoop install extras/ffmpeg

# Windows (winget)
winget install ffmpeg

# Windows (Chocolatey)
choco install ffmpeg
```

## Install

**Homebrew -- macOS / Linux (recommended):**

```bash
brew install dgr8akki/tap/nano-ffmpeg
```

The Homebrew tap installs `ffmpeg-full` as a dependency so Stabilize, thumbnails, etc. work out of the box.

**Scoop -- Windows (recommended):**

```powershell
scoop bucket add extras
scoop bucket add nano-ffmpeg https://github.com/dgr8akki/scoop-bucket
scoop install nano-ffmpeg
```

The Scoop manifest declares `extras/ffmpeg` as a dependency, so Scoop pulls ffmpeg/ffprobe for you. Installs are user-scope (no admin prompt).

**Arch Linux (AUR):**

```bash
yay -S nano-ffmpeg
```

**Download binary:**

Grab a prebuilt binary from [GitHub Releases](https://github.com/dgr8akki/nano-ffmpeg/releases/latest) for your platform (macOS, Linux, Windows).

**Go install:**

```bash
go install github.com/dgr8akki/nano-ffmpeg@latest
```

**Build from source:**

```bash
git clone https://github.com/dgr8akki/nano-ffmpeg.git
cd nano-ffmpeg
go build -o nano-ffmpeg .
./nano-ffmpeg
```

## Usage

Run with no arguments to open the TUI:

```bash
nano-ffmpeg
```

The TUI guides you through the full flow:

```
Home  -->  File Picker  -->  Operations  -->  Settings  -->  Progress  -->  Result
                                                                              |
                                                                         Back to Home
```

1. **Home** -- See your ffmpeg version, capabilities, and recent files. Pick an operation.
2. **File Picker** -- Browse to your file or type a path. See metadata inline.
3. **Operations** -- Choose what to do (convert, compress, trim, etc.).
4. **Settings** -- Configure with pre-filled defaults. See the ffmpeg command live.
5. **Progress** -- Watch encoding with a live progress bar, ETA, and stats.
6. **Result** -- See output path, before/after size comparison. Do another or quit.

All flags are optional; see [CLI Options](#cli-options) for the full list and examples.

## CLI Options

| Flag | Short | Value | Description |
|------|-------|-------|-------------|
| `--theme` | `-t` | `dark` \| `light` | Theme override for this run. Without the flag, the theme from `~/.config/nano-ffmpeg/config.json` is used. |
| `--dir` | `-d` | `<directory>` | Open the File Picker pre-focused on this directory. |
| `--dir` | `-d` | `<file>` | Skip the File Picker and jump straight to Operations with this file preloaded (the file is probed and added to the recent-files list). |
| `--version` | -- | -- | Print the version and exit. |
| `--help` | `-h` | -- | Print usage and exit. |

Examples:

```bash
# Force a theme for a single run
nano-ffmpeg --theme light
nano-ffmpeg -t dark

# Open the File Picker at a folder
nano-ffmpeg -d ~/Videos

# Skip the File Picker entirely
nano-ffmpeg -d ~/Videos/interview.mp4
```

## Operations

| Operation | What it does | Key settings |
|-----------|-------------|--------------|
| **Convert Format** | Change container/codec | MP4, MKV, WebM, AVI, MOV; H.264, H.265, AV1, VP9; CRF quality, preset speed, audio codec |
| **Extract Audio** | Strip video, keep audio track | MP3, AAC, FLAC, WAV, OGG, Opus; bitrate presets (64k-320k) |
| **Resize / Scale** | Change output height | 4K, 1080p, 720p, 480p, 360p; H.264 or H.265 (aspect ratio field is shown but currently has no effect -- see `docs/future_scope.md`) |
| **Trim / Cut** | Cut segments by time | Start/end time (end pre-filled from ffprobe); lossless cut (stream copy) toggle |
| **Compress** | Reduce file size | CRF quality; H.264/H.265/AV1; preset speed (the Two-Pass toggle is shown but currently inert -- see `docs/future_scope.md`) |
| **Merge / Concat** | Join multiple files in the same folder with the same extension | Alphabetical order; stream copy or re-encode to H.264/AAC |
| **Add Subtitles** | Burn-in or embed existing subtitle streams from the input | Picks a subtitle track from the input file; font/size/position customization is not yet exposed |
| **Create GIF** | Animated GIF from video | 10/15/24 fps; width presets; palette optimization (only GIF output today; WebP planned) |
| **Extract Thumbnails** | Grab frames as images (PNG) | Single frame at a timestamp, 4x4 contact sheet, or every 5 seconds |
| **Watermark** | Overlay a solid white color box | 5-position grid (corners + center), opacity, size presets (image and text overlays planned) |
| **Audio Adjustments** | Normalize, volume, fade | loudnorm, dB boost/reduce, fade in/out, remove audio |
| **Video Filters** | Stabilize, deinterlace, speed, rotate, flip | vidstab (or deshake fallback), yadif, 2x/0.5x speed, rotate 90°, horizontal/vertical flip |

## Progress Screen

```
  input.mkv  -->  input_compressed.mp4

  ████████████████████████████░░░░░░░░░░░░  63.4%

  Elapsed   00:01:23        Frames    4,521
  ETA       00:00:48        Size      142.3 MB
  Speed     2.3x            Bitrate   8241 kbps
  FPS       54.2
  
  ╭─ Live Log ─────────────────────────────────────────────╮
  │ frame= 4521 fps=54.2 q=28.0 size= 148736kB time=...   │
  │ frame= 4548 fps=54.1 q=28.0 size= 149120kB time=...   │
  ╰────────────────────────────────────────────────────────╯
```

- Progress bar gradient: green (0%) to cyan (100%)
- ETA smoothed with rolling average over last 5 updates (no jitter)
- Braille spinner (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`) for indeterminate operations
- Cancel with `Esc` > confirm with `y`

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `q` | Quit |
| `Ctrl+C` | Force quit |
| `?` | Toggle help overlay |

### Navigation

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Select / confirm / execute |
| `Esc` | Go back one screen |

### File Picker

| Key | Action |
|-----|--------|
| `Enter` | Open directory / select file |
| `Backspace` | Go to parent directory |
| `/` | Toggle path input mode |

### Settings

| Key | Action |
|-----|--------|
| `←` / `→` | Change field value (select/toggle) or move the text cursor |
| Typing | Edit text fields (Start Time, End Time, Duration, Timestamp) |
| `Enter` | Execute the ffmpeg command |

### Progress

| Key | Action |
|-----|--------|
| `Esc` | Cancel (with confirmation) |
| `y` / `n` | Confirm or deny cancellation |

## Configuration

Config is stored at `~/.config/nano-ffmpeg/config.json`:

```json
{
  "default_output_dir": "",
  "theme": "dark",
  "recent_files": [
    "/Users/you/Videos/interview.mp4",
    "/Users/you/Downloads/concert.mkv"
  ],
  "hw_accel": "auto",
  "ffmpeg_path": ""
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `default_output_dir` | `""` (same as input) | Where output files are saved |
| `theme` | `"dark"` | Color theme: `dark` or `light` |
| `recent_files` | `[]` | Last 10 files used (auto-populated) |
| `hw_accel` | `"auto"` | Hardware acceleration: `auto`, `off`, `videotoolbox`, `nvenc`, `vaapi` |
| `ffmpeg_path` | `""` (auto-detect) | Override ffmpeg binary path |

Capabilities are cached separately at `~/.config/nano-ffmpeg/capabilities.json` and auto-invalidated when your ffmpeg version changes.

If you pass `--theme dark|light` (or `-t dark|light`), it overrides the config theme for that run.
If you pass `--dir <directory|file>` (or `-d <directory|file>`), it overrides startup location for that run.

## Project Structure

```
nano-ffmpeg/
├── main.go                              # Entry point
├── cmd/
│   ├── root.go                          # Cobra CLI, --theme/--dir/--version flags
│   └── root_test.go
├── internal/
│   ├── app/
│   │   ├── app.go                       # Top-level Bubble Tea model, screen router
│   │   ├── config.go                    # Config load/save, recent files
│   │   ├── app_test.go
│   │   └── config_test.go
│   ├── ffmpeg/
│   │   ├── detect.go                    # Find ffmpeg/ffprobe binaries, parse version
│   │   ├── capabilities.go              # Probe codecs, formats, filters, hwaccels; cache
│   │   ├── probe.go                     # Run ffprobe, parse JSON into Go structs
│   │   ├── command.go                   # Struct-based ffmpeg command builder
│   │   ├── runner.go                    # Process management, stderr streaming
│   │   ├── progress.go                  # Parse ffmpeg progress output, ETA calculation
│   │   ├── errors.go                    # Translate ffmpeg errors to human-readable
│   │   └── *_test.go                    # Full unit suite per file above
│   ├── preset/
│   │   ├── preset.go                    # Quality / resolution / format preset catalog
│   │   └── preset_test.go
│   ├── screens/
│   │   ├── screen.go                    # Screen interface definition
│   │   ├── messages.go                  # Shared navigation/status messages
│   │   ├── screens_test.go
│   │   ├── home/home.go                 # Dashboard: ffmpeg info, recent files, operations
│   │   ├── filepicker/filepicker.go     # File browser + path input + ffprobe preview
│   │   ├── operations/operations.go     # Operation category picker
│   │   ├── settings/settings.go         # Dynamic form per operation, live command preview
│   │   ├── progress/progress.go         # Progress bar, stats, live log, cancel flow
│   │   └── result/result.go             # Output summary, size comparison
│   └── ui/
│       ├── theme.go                     # Color palette and shared styles (dark/light)
│       ├── frame.go                     # Top bar, bottom bar, status line
│       ├── help.go                      # Context-sensitive help overlay
│       └── responsive.go                # Terminal size detection
├── website/                             # Next.js marketing site (deployed to Vercel)
│   ├── app/                             # Landing page + /docs page
│   ├── components/                      # Navbar, Footer, TerminalDemo
│   └── README.md                        # Contributor doc for the site
├── .github/workflows/
│   ├── ci.yml                           # Build + vet + test on push/PR
│   └── release.yml                      # GoReleaser on tag push
├── .goreleaser.yaml                     # Cross-platform build + Homebrew tap + Scoop bucket config
├── homebrew/nano-ffmpeg.rb              # Formula template (reference)
├── docs/
│   ├── design/                          # Original design spec and implementation plan
│   ├── future_scope.md                  # Gap-closing roadmap (see Future Roadmap below)
│   ├── release.sh                       # Single-command tag + push + workflow watch
│   └── Makefile                         # `make release[-minor|-major]` wrappers
├── go.mod
├── go.sum
└── README.md
```

## Tech Stack

| Component | Library | Purpose |
|-----------|---------|---------|
| Language | Go 1.22+ | Single binary, no runtime dependency |
| TUI framework | [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Elm-architecture terminal UI |
| Styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) | Composable terminal styles |
| Components | [Bubbles](https://github.com/charmbracelet/bubbles) | Pre-built TUI components |
| CLI | [Cobra](https://github.com/spf13/cobra) | Argument parsing, `--version`, `--help` |
| ffmpeg | `os/exec` | Shell out to the user's installed ffmpeg (no CGo bindings) |
| Release | [GoReleaser](https://goreleaser.com/) | Cross-compile + GitHub Release + Homebrew tap + Scoop bucket |

## Testing

```bash
# Fast path
go test ./...

# Verbose
go test ./... -v

# A single package
go test ./internal/ffmpeg/ -v
go test ./internal/screens/settings/ -v

# Ignore the cache
go test -count=1 ./...
```

Snapshot of coverage by area:

- **CLI (`cmd/`)** -- flag parsing for `--theme`, `--dir` (directory vs file), error paths.
- **App / config (`internal/app/`)** -- recent-files dedup and cap, config load/save defaults, initial-file startup path.
- **ffmpeg (`internal/ffmpeg/`)** -- command builder (convert/trim/extract/resize + extras), capability parsing, ffprobe JSON parsing, runner lifecycle, progress/ETA smoothing, error translation, detect helpers.
- **Preset catalog (`internal/preset/`)** -- option tables for video/audio/gif/compress presets.
- **Screens (`internal/screens/*`)** -- filepicker, home, operations, settings (per-op form building + command assembly), progress, result, screen-router messages.
- **UI (`internal/ui/`)** -- theme palette/style build, responsive size checks, help overlay layout, frame rendering.

## Releasing

Releases are driven by pushing a `v*` tag from `main`. The [`Release workflow`](.github/workflows/release.yml) then runs [GoReleaser](https://goreleaser.com) to publish a GitHub Release, update the Homebrew tap (`dgr8akki/homebrew-tap`), and update the Scoop bucket (`dgr8akki/scoop-bucket`). Tap/bucket updates require `HOMEBREW_TAP_TOKEN` and `SCOOP_BUCKET_TOKEN` repo secrets.

From a clean `main`:

```bash
# 1. Sanity checks
go test ./...
go vet ./...

# 2. (Optional) Validate the GoReleaser config locally
goreleaser check
goreleaser release --snapshot --clean --skip=publish

# 3. Pick the next version (last tag + bump). Example: v0.4.0 -> v0.5.0
PREV=$(git describe --tags --abbrev=0)
NEXT=v0.5.0

# 4. Annotated tag whose body is the changelog since the previous tag
git tag -a "$NEXT" -m "$NEXT

$(git log "$PREV"..HEAD --pretty=format:'- %s' --reverse)"

# 5. Push the tag to trigger the release workflow
git push origin "$NEXT"

# 6. Tail the workflow
gh run watch
```

## Future Roadmap

See [`docs/future_scope.md`](docs/future_scope.md) for the full plan, including the feature gaps surfaced by the README/website sync audit (watermark image/text overlays, subtitle styling, crop/color filters, two-pass encoding, clipboard copy, capability-driven filtering, hardware-accelerated encoding, WebP output, aspect-ratio handling, merge reordering, smart defaults).

Longer-term ideas tracked but not in v0.1.0:

- [ ] Batch processing (apply same operation to multiple files)
- [ ] Custom preset save/load
- [ ] Operation queue (line up multiple jobs)
- [ ] Watch folder (auto-process new files)
- [ ] FFplay preview before full encode
- [ ] Scene detection / smart split
- [ ] Whisper-based auto-subtitle generation
- [ ] Plugin system for custom operations
- [ ] Remote file support (URL / S3 input)
- [ ] Localization / i18n

## Contributing

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/awesome`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Commit and push
6. Open a PR

Please follow existing code structure -- one package per screen, logic in `internal/ffmpeg/`, UI in `internal/ui/`.

## License

Released under the [MIT License](LICENSE).
