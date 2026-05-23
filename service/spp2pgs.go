package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"ass2sup/model"
)

// Spp2PgsService handles conversion from ASS to PGS
type Spp2PgsService struct {
	spp2pgsPath string
}

// Spp2PgsRate represents frame rate codes for Spp2Pgs
type Spp2PgsRate int

const (
	Rate23976 Spp2PgsRate = 1 // 23.976 fps
	Rate24000 Spp2PgsRate = 2 // 24.0 fps
	Rate25000 Spp2PgsRate = 3 // 25.0 fps
	Rate29970 Spp2PgsRate = 4 // 29.97 fps
	Rate30000 Spp2PgsRate = 5 // 30.0 fps
	Rate50000 Spp2PgsRate = 6 // 50.0 fps
	Rate59940 Spp2PgsRate = 7 // 59.94 fps
	Rate60000 Spp2PgsRate = 8 // 60.0 fps
)

// Spp2PgsFormat represents format codes for Spp2Pgs
type Spp2PgsFormat int

const (
	Format480i  Spp2PgsFormat = 1 // 480i
	Format576i  Spp2PgsFormat = 2 // 576i
	Format480p  Spp2PgsFormat = 3 // 480p
	Format1080i Spp2PgsFormat = 4 // 1080i
	Format720p  Spp2PgsFormat = 5 // 720p
	Format1080p Spp2PgsFormat = 6 // 1080p
	Format576p  Spp2PgsFormat = 7 // 576p
	Format4K    Spp2PgsFormat = 6 // 4K uses 1080p code (tool limitation)
)

// NewSpp2PgsService creates a new Spp2PgsService
func NewSpp2PgsService(spp2pgsPath string) *Spp2PgsService {
	if spp2pgsPath == "" {
		spp2pgsPath = "Spp2Pgs.exe"
	}
	return &Spp2PgsService{spp2pgsPath: spp2pgsPath}
}

// Convert converts an ASS subtitle file to PGS format
func (s *Spp2PgsService) Convert(ctx context.Context, pair *model.SubtitlePair, dryRun bool) error {
	// Determine format value (resolution) from video resolution
	formatValue := s.determineFormatValue(pair.Metadata.Height)

	// Determine rate value from frame rate
	rateValue := s.determineRateValue(pair.Metadata.FrameRate)

	if dryRun {
		fmt.Printf("[DRY-RUN] Would convert:\n")
		fmt.Printf("  Input:  %s\n", pair.SubtitlePath)
		fmt.Printf("  Output: %s\n", pair.OutputPath)
		fmt.Printf("  Format: %s, Rate: %s\n", formatValue, rateValue)
		return nil
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(pair.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get absolute path to Spp2Pgs executable
	absSpp2pgsPath, err := filepath.Abs(s.spp2pgsPath)
	if err != nil {
		return fmt.Errorf("failed to resolve Spp2Pgs path: %w", err)
	}

	// Build command - run from Spp2Pgs directory so DLLs are found
	exeDir := filepath.Dir(absSpp2pgsPath)

	cmd := exec.CommandContext(
		ctx,
		absSpp2pgsPath,
		"-i", pair.SubtitlePath,
		"-s", formatValue,
		"-r", rateValue,
		"-v127", // suppress Spp2Pgs info output; show only errors and warnings
		pair.OutputPath,
	)
	cmd.Dir = exeDir // Set working directory to Spp2Pgs location for DLLs

	// Run conversion
	output, err := cmd.CombinedOutput()

	// Spp2Pgs returns exit code 1 even on success - check if output file was created
	if _, statErr := os.Stat(pair.OutputPath); statErr == nil {
		// File exists, conversion succeeded
		return nil
	}

	// Output file not found, conversion failed
	if err != nil {
		return fmt.Errorf("spp2pgs conversion failed: %w\nOutput: %s", err, string(output))
	}

	return fmt.Errorf("conversion completed but output file not found: %s", pair.OutputPath)
}

// ConvertWithProgress converts with timeout and progress tracking
func (s *Spp2PgsService) ConvertWithProgress(pair *model.SubtitlePair, timeout time.Duration, dryRun bool) (*model.ConversionResult, error) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := &model.ConversionResult{
		Pair:     *pair,
		Success:  false,
		Duration: 0,
	}

	err := s.Convert(ctx, pair, dryRun)
	result.Duration = time.Since(start).Seconds()

	if err != nil {
		result.Error = err
		return result, err
	}

	result.Success = true
	return result, nil
}

// determineFormatValue returns the resolution value for Spp2Pgs
func (s *Spp2PgsService) determineFormatValue(height int) string {
	switch {
	case height >= 2160:
		return "1080" // 4K - use 1080p setting as max
	case height >= 1080:
		return "1080"
	case height >= 720:
		return "720"
	case height >= 576:
		return "576"
	case height >= 480:
		return "480"
	default:
		return "480"
	}
}

// determineRateValue returns the rate value for Spp2Pgs
func (s *Spp2PgsService) determineRateValue(fps float64) string {
	switch {
	case fps >= 59.0 && fps <= 60.1:
		if fps < 59.95 {
			return "59"
		}
		return "60"
	case fps >= 49.0 && fps <= 50.1:
		return "50"
	case fps >= 29.0 && fps <= 30.1:
		if fps < 29.98 {
			return "29"
		}
		return "30"
	case fps >= 24.0 && fps <= 25.1:
		if fps < 24.5 {
			return "24"
		}
		return "25"
	case fps >= 23.0 && fps <= 24.1:
		return "23"
	default:
		return "23" // Default to 23.976
	}
}

// VerifyExecutable checks if Spp2Pgs executable exists and is accessible
func (s *Spp2PgsService) VerifyExecutable() error {
	if _, err := exec.LookPath(s.spp2pgsPath); err != nil {
		// Try as absolute/relative path
		if _, err := os.Stat(s.spp2pgsPath); os.IsNotExist(err) {
			return fmt.Errorf("Spp2Pgs executable not found: %s", s.spp2pgsPath)
		}
	}
	return nil
}
