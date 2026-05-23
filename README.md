# ass2sup

<p align="center">
  <img src="icon/ass2sup_png.png" alt="ass2sup icon" width="128" height="128">
</p>

<p align="center">
  <a href="https://github.com/kidpoleon/ass2sup/releases"><img src="https://img.shields.io/badge/version-2.0.0-7571F9.svg" alt="Version"></a>
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/go-1.22+-00ADD8.svg" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-GPLv3-02BA84.svg" alt="License"></a>
  <a href="https://github.com/kidpoleon/ass2sup/releases"><img src="https://img.shields.io/badge/platform-Windows-lightgrey.svg" alt="Platform"></a>
</p>

> A self-contained Windows tool that converts `.ass` subtitle files to `.sup` (Blu-ray PGS) format — with a TUI, recursive directory scanning, and up to 16 parallel conversions.

---

## Features

- **Self-Contained** — embeds Spp2Pgs and its required DLLs; no side-by-side installation needed
- **TUI Prompt** — a clean terminal form asks for the source directory at launch
- **Recursive Scan** — walks the full directory tree and matches video/subtitle pairs folder-by-folder
- **Smart Matching** — pairs files by episode number (`S01E01`, `1x01`, `Episode N`) or name similarity
- **16 Parallel Workers** — up to 16 Spp2Pgs processes run simultaneously; parallel ffprobe during scan
- **Accurate Metadata** — single ffprobe call per video extracts resolution and exact frame rate
- **Real-Time Progress** — animated bar with live elapsed time and ETA
- **Proper Timing** — total time and per-file average shown in the summary
- **Graceful Shutdown** — Ctrl-C cancels queued work cleanly

---

## Requirements

| Requirement | Details |
|---|---|
| **Windows 10/11** (64-bit) | Spp2Pgs.exe and xy-VSSppf.dll are Windows PE binaries |
| **ffprobe** in PATH | Part of the FFmpeg suite — see installation guide below |

> Only `ffprobe` is needed from FFmpeg; the full `ffmpeg` encoder is not required.

---

## Installing FFmpeg (ffprobe)

Choose any one method. After installation, open a new terminal and verify with `ffprobe -version`.

### Option 1 — winget (built into Windows 10/11)

```powershell
winget install -e --id Gyan.FFmpeg
```

`ffprobe` will be on your PATH automatically after the install completes.

---

### Option 2 — Chocolatey

> Install Chocolatey first if you don't have it: https://chocolatey.org/install

```powershell
choco install ffmpeg -y
```

---

### Option 3 — Scoop

> Install Scoop first if you don't have it: https://scoop.sh

```powershell
scoop install ffmpeg
```

---

### Option 4 — Manual (no package manager)

1. Go to **https://ffmpeg.org/download.html** → click **Windows** → choose the **gyan.dev** or **BtbN** build
2. Download the **essentials** or **full** build (`.zip` or `.7z`)
3. Extract the archive — you will find a `bin\` folder containing `ffprobe.exe`
4. Add that `bin\` folder to your system PATH:
   - Press **Win + S** → search **"Edit the system environment variables"** → open it
   - Click **Environment Variables…**
   - Under **System variables**, select **Path** → click **Edit**
   - Click **New** and paste the full path to the `bin\` folder (e.g. `C:\ffmpeg\bin`)
   - Click **OK** on all dialogs
5. Open a **new** PowerShell or Command Prompt window and run:

```powershell
ffprobe -version
```

You should see version output. If you see `'ffprobe' is not recognized`, the `bin\` folder is not in PATH — repeat step 4.

---

## Usage

1. Download `ass2sup.exe` from the [Releases](https://github.com/kidpoleon/ass2sup/releases) page
2. Run `ass2sup.exe`
3. A TUI prompt appears — type or paste the path to your media folder
4. The program recursively scans for video + `.ass` pairs, extracts metadata, converts, and reports

```
╭─────────────────────────────────╮
│  ass2sup v2.0.0                 │
│  ASS → PGS Subtitle Converter   │
╰─────────────────────────────────╯

ℹ Extracting converter...
✓ Ready

  Source Directory
  ❯ C:\Media\Show S01

ℹ 12 video(s) · 12 subtitle(s) · 1 director(ies)
ℹ 12 pair(s) queued · 16 workers
  Converting  ▕━━━━━━━━━━━━━━━━━━╸──────────────────────────▏  58%  (7/12)  [1m12s:52s]

╭─────────────────────────────────────────────╮
│  ✓  Succeeded: 12                           │
│  ✗  Failed:    0                            │
│                                             │
│    Total time:   2m 04s                     │
│    Avg per file: 10.3s                      │
│                                             │
│  ✓ All conversions completed successfully!  │
╰─────────────────────────────────────────────╯
```

---

## How It Works

### File Matching

Each subdirectory is matched independently — files from `/Season 1/` never pair with subtitles from `/Season 2/`.

Matching score (highest wins, minimum 60 required):

| Score | Condition |
|---|---|
| 100 | Exact name match (after normalisation) |
| 80 | One name contains the other |
| 60 | Identical episode number (`S01E01`, `1x01`, `Episode N`) |

Language suffixes (`.en`, `.english`, `.eng`, `.sub`) are stripped before comparison.

### Metadata Extraction

One `ffprobe` call per video retrieves width, height, codec, duration, and `r_frame_rate` (as an exact rational like `24000/1001`). Up to **8 concurrent ffprobe** processes run during the scan phase.

### Conversion

Resolution mapping:

| Video height | Spp2Pgs `-s` flag |
|---|---|
| ≥ 2160 (4K) | `1080` (tool maximum) |
| ≥ 1080 | `1080` |
| ≥ 720 | `720` |
| ≥ 576 | `576` |
| anything else | `480` |

Frame-rate mapping: `23.976→23` · `24→24` · `25→25` · `29.97→29` · `30→30` · `50→50` · `59.94→59` · `60→60`

Spp2Pgs runs with `-v127` (errors and warnings only) to reduce subprocess I/O overhead.  
Success is determined by output file existence — Spp2Pgs exits 1 even on success.

---

## Supported Formats

**Video input:** `.mp4` `.mkv` `.avi` `.mov` `.m4v` `.webm` `.ts` `.m2ts`

**Subtitle input:** `.ass`

**Output:** `.sup` (Blu-ray PGS / HDMV PG stream)

---

## Architecture

```
ass2sup/
├── main.go              # Entry point, TUI prompt (charmbracelet/huh)
├── main_unsupported.go  # Non-Windows stub (prints error, exits)
├── embed/               # //go:embed — bundles Spp2Pgs.exe + DLLs, extracts to temp
├── model/               # Shared data structures (Config, VideoMetadata, SubtitlePair, …)
├── service/
│   ├── ffprobe.go       # Single-call ffprobe wrapper, rational FPS parsing
│   └── spp2pgs.go       # Spp2Pgs process management, format/rate mapping
├── controller/
│   └── converter.go     # Worker pool, recursive WalkDir scan, parallel ffprobe, progress bar
├── view/
│   └── console.go       # lipgloss-styled output, thread-safe ConsoleView
└── util/
    └── file.go          # Extension checks, episode-number extraction, match scoring
```

---

## Building from Source

### Requirements

- Go 1.22+
- Windows (build target)
- [`goversioninfo`](https://github.com/josephspurrier/goversioninfo) for regenerating the icon resource (optional — `resource.syso` is committed)
- The four Spp2Pgs binaries in `embed/` (see below)

### Obtaining the Spp2Pgs binaries

Download [`160506.EXE.Spp2Pgs.0_9_3_7.7z`](https://github.com/subelf/Spp2Pgs/releases/tag/0.9.3.7) and extract into `embed/`:

```
Spp2Pgs.exe      → embed/Spp2Pgs.exe
Spp2Pgs64.exe    → embed/Spp2Pgs64.exe
xy-VSSppf.dll    → embed/xy-VSSppf.dll
xy-VSSppf64.dll  → embed/xy-VSSppf64.dll
```

### Build

```powershell
go build -o ass2sup.exe .
```

### Regenerate icon / version info (optional)

```powershell
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
go generate
go build -o ass2sup.exe .
```

---

## License

**GNU General Public License v3.0** — see [LICENSE](LICENSE).

### Credits & Attribution

- **[subelf/Spp2Pgs](https://github.com/subelf/Spp2Pgs)** (GPL-3.0) — the subtitle conversion engine
  - Based on [avs2bdnxml](http://www.ps-auxw.de/avs2bdnxml/) and [xy-VSFilter](https://github.com/Cyberbeing/xy-VSFilter)
- **[charmbracelet/huh](https://github.com/charmbracelet/huh)** — TUI form library
- **[charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)** — terminal styling
- **[schollz/progressbar](https://github.com/schollz/progressbar)** — progress bar

---

<p align="center">Made by <a href="https://github.com/kidpoleon">kidpoleon</a></p>
