package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds application configuration
type Config struct {
	InputDir      string
	Spp2PgsPath   string
	FFprobePath   string
	DryRun        bool
	Verbose       bool
	Overwrite     bool
	Workers       int
}

// ParseFlags parses command-line flags and returns Config
func ParseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.InputDir, "dir", ".", "Directory containing video and subtitle files")
	flag.StringVar(&cfg.Spp2PgsPath, "spp2pgs", "Spp2Pgs.exe", "Path to Spp2Pgs executable")
	flag.StringVar(&cfg.FFprobePath, "ffprobe", "ffprobe", "Path to ffprobe executable")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Show what would be done without converting")
	flag.BoolVar(&cfg.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&cfg.Overwrite, "force", false, "Overwrite existing .sup files")
	flag.IntVar(&cfg.Workers, "workers", 1, "Number of parallel conversions")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "ass2sup - Convert ASS subtitles to Blu-ray PGS format\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -dir=\"C:\\Videos\" -spp2pgs=\"Spp2Pgs.exe\"\n", os.Args[0])
	}

	flag.Parse()

	// Resolve absolute paths
	if cfg.InputDir != "." {
		abs, err := filepath.Abs(cfg.InputDir)
		if err == nil {
			cfg.InputDir = abs
		}
	}

	return cfg
}

// Validate checks if configuration is valid
func (c *Config) Validate() error {
	if c.InputDir == "" {
		return fmt.Errorf("input directory is required")
	}

	info, err := os.Stat(c.InputDir)
	if err != nil {
		return fmt.Errorf("cannot access input directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("input path is not a directory: %s", c.InputDir)
	}

	if c.Workers < 1 {
		c.Workers = 1
	}

	return nil
}
