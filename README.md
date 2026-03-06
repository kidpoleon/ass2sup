# ass2sup

<p align="center">
  <img src="icon/ass2sup_png.png" alt="ass2sup icon" width="128" height="128">
</p>

[![Version](https://img.shields.io/badge/version-1.0.1-blue.svg)](https://github.com/kidpoleon/ass2sup/releases)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows-lightgrey.svg)](https://github.com/kidpoleon/ass2sup/releases)

> A self-contained Windows application that converts `.ass` (Advanced SubStation Alpha) subtitle files to `.sup` (Blu-ray PGS - Presentation Graphic Stream) format.

<p align="center">
  <b>Drag. Drop. Convert. It's that simple.</b>
</p>

---

## ✨ Features

- **Self-Contained**: Embeds Spp2Pgs converter and all required DLLs internally - no external dependencies needed
- **Drag-and-Drop Workflow**: Simply copy the executable to any folder with video/subtitle files and double-click to run
- **Auto-Detection**: Automatically finds and matches video files with their corresponding subtitle files
- **Parallel Processing**: Uses multiple workers for concurrent conversions (up to 8 parallel conversions)
- **Video Metadata**: Uses ffprobe to detect resolution and frame rate from video files
- **Progress Tracking**: Real-time progress bars showing conversion status
- **Robust Error Handling**: Graceful handling of errors with informative messages
- **Smart Matching**: Matches subtitles to videos using episode number extraction

## Usage

### Method 1: Drag and Drop (Recommended)
1. Copy `ass2sup.exe` to any folder containing:
   - Video files (`.mp4`, `.mkv`, `.avi`, `.mov`, `.m4v`, `.webm`)
   - Matching ASS subtitle files (`.ass`)
2. Double-click `ass2sup.exe`
3. The program will:
   - Extract the embedded converter to a temp location
   - Scan for video/subtitle pairs
   - Convert all matching files concurrently
   - Display progress bars
   - Show a summary when complete

### Method 2: Command Line
```bash
# Simply run in current directory
ass2sup.exe

# The program will auto-detect files in the current working directory
```

## Requirements

- **Windows**: Windows 7 or later (64-bit recommended)
- **FFmpeg**: ffprobe must be installed and available in PATH
  - Download from: https://ffmpeg.org/download.html
  - Add `bin` folder to your system PATH

## How It Works

### File Matching
The program uses intelligent filename matching to pair video files with subtitle files:

1. **Episode Number Extraction**: Extracts episode numbers from filenames using patterns like:
   - `1x01`, `S01E01`, `EP01`, `Episode 01`, etc.
   
2. **Similarity Scoring**: If no episode number found, uses string similarity

3. **Exact Match**: Falls back to exact filename matching (minus extension)

### Conversion Process

1. **Metadata Extraction**: Uses ffprobe to get video resolution and frame rate
2. **Format Detection**: Maps video resolution to appropriate Spp2Pgs format:
   - 2160p+ (4K) → 1080p mode (tool limitation)
   - 1080p → 1080p mode
   - 720p → 720p mode
   - 576p → 576p mode
   - 480p → 480p mode

3. **Frame Rate Detection**: Maps frame rates:
   - 23.976 fps → 23
   - 24 fps → 24
   - 25 fps → 25
   - 29.97 fps → 29
   - 30 fps → 30
   - 50 fps → 50
   - 59.94 fps → 59
   - 60 fps → 60

4. **Parallel Conversion**: Runs conversions in parallel using worker pool pattern

## Architecture

The application follows an MVC (Model-View-Controller) pattern with additional service layers:

```
ass2sup/
├── embed/           # Embedded binary extraction
├── model/           # Data structures
├── service/         # Business logic (ffprobe, Spp2Pgs)
├── controller/      # Workflow orchestration
├── view/            # Console output
└── main.go          # Entry point
```

### Key Components

- **embed/**: Uses Go 1.16+ `//go:embed` to bundle Spp2Pgs.exe and xy-VSSppf.dll
- **service/ffprobe.go**: JSON parsing for ffprobe output
- **service/spp2pgs.go**: Handles Spp2Pgs execution with proper working directory
- **controller/converter.go**: Worker pool implementation with progress tracking
- **view/console.go**: Colored console output with progress bars

## Technical Details

### Embedded Binaries
The application embeds:
- `Spp2Pgs.exe` / `Spp2Pgs64.exe` - The subtitle converter (32/64-bit)
- `xy-VSSppf.dll` / `xy-VSSppf64.dll` - Required VSFilter DLL

At runtime, these are extracted to a temporary directory and cleaned up on exit.

### Concurrency
- Default worker count: `min(CPU_COUNT, 8)`
- Configurable via code (auto-detected based on CPU cores)
- Thread-safe progress reporting using atomic counters
- Context-based cancellation for graceful shutdown

### Error Handling
- Spp2Pgs returns exit code 1 even on success - we check for output file existence
- Graceful handling of missing ffprobe
- Per-file error reporting without stopping other conversions
- Signal handling for Ctrl+C interruption

## Building from Source

### Requirements
- Go 1.21 or later
- Windows (for Spp2Pgs binaries)

### Build
```bash
go build -o ass2sup.exe
```

### Dependencies
```bash
go get github.com/fatih/color
go get github.com/schollz/progressbar/v3
```

## 📄 License

**MIT License** - See [LICENSE](LICENSE) file for details

### Credits & Attribution

This project would not be possible without:

- **[subelf/Spp2Pgs](https://github.com/subelf/Spp2Pgs)** - The core subtitle conversion engine
  - Converts ASS subtitles to Blu-ray PGS format
  - Includes xy-VSSppf.dll for VSFilter rendering
  - Licensed under its own terms (see original repository)

- **[schollz/progressbar](https://github.com/schollz/progressbar)** - Terminal progress bar library
- **[fatih/color](https://github.com/fatih/color)** - Terminal color output library

The Spp2Pgs project includes components based on:
- [avs2bdnxml](http://www.ps-auxw.de/avs2bdnxml/) by xibeifeng
- [xy-VSFilter](https://github.com/Cyberbeing/xy-VSFilter) fork by Cyberbeing

This application merely packages Spp2Pgs for easier use and does not modify its functionality. All credit for the actual subtitle conversion goes to the original Spp2Pgs authors.

---

## 👤 Author

**kidpoleon** - [github.com/kidpoleon](https://github.com/kidpoleon)

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 💡 Feature Requests & Bug Reports

Please use [GitHub Issues](https://github.com/kidpoleon/ass2sup/issues) for:
- Bug reports
- Feature requests
- Questions about usage

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/kidpoleon">kidpoleon</a>
</p>
