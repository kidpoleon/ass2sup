package model

// Config holds application configuration
type Config struct {
	Directory   string // Working directory (where files are located)
	Spp2PgsPath string // Path to extracted Spp2Pgs executable
	FFprobePath string // Path to ffprobe (optional, uses PATH if empty)
	Workers     int    // Number of parallel workers (0 = auto)
	DryRun      bool   // Show what would be done without converting
	Overwrite   bool   // Overwrite existing .sup files
	Verbose     bool   // Enable verbose output
}

// VideoMetadata holds ffprobe-extracted video information
type VideoMetadata struct {
	Filename   string  `json:"filename"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	FrameRate  float64 `json:"frame_rate"`
	Duration   float64 `json:"duration"`
	CodecName  string  `json:"codec_name"`
}

// SubtitlePair represents a matched .ass and video file
type SubtitlePair struct {
	VideoPath    string
	SubtitlePath string
	OutputPath   string
	Metadata     VideoMetadata
}

// ConversionResult represents the outcome of a conversion
type ConversionResult struct {
	Pair      SubtitlePair
	Success   bool
	Error     error
	Duration  float64 // conversion time in seconds
}
