// Package controller orchestrates the subtitle conversion process with parallel processing
// and progress tracking. It implements a worker pool pattern for efficient concurrent
// conversions while maintaining thread-safe progress reporting.
package controller

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/schollz/progressbar/v3"

	"ass2sup/model"
	"ass2sup/service"
	"ass2sup/util"
	"ass2sup/view"
)

// ConverterController orchestrates the subtitle conversion process
type ConverterController struct {
	config      *model.Config
	ffprobeSvc  *service.FFprobeService
	spp2pgsSvc  *service.Spp2PgsService
	view        *view.ConsoleView
	
	// Concurrency control
	workQueue   chan *model.SubtitlePair
	results     chan *model.ConversionResult
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	
	// Progress tracking
	totalFiles  int32
	completed   int32
	succeeded   int32
	failed      int32
	skipped     int32
}

// NewConverterController creates a new controller instance with optimized worker count
func NewConverterController(cfg *model.Config) *ConverterController {
	// Auto-detect worker count if not specified
	workers := cfg.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
		// Cap at reasonable max for I/O bound operations
		if workers > 8 {
			workers = 8
		}
	}
	cfg.Workers = workers

	ctx, cancel := context.WithCancel(context.Background())

	return &ConverterController{
		config:     cfg,
		ffprobeSvc: service.NewFFprobeService(cfg.FFprobePath),
		spp2pgsSvc: service.NewSpp2PgsService(cfg.Spp2PgsPath),
		view:       view.NewConsoleView(),
		workQueue:  make(chan *model.SubtitlePair, 100),
		results:    make(chan *model.ConversionResult, 100),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Shutdown gracefully cancels ongoing operations
func (c *ConverterController) Shutdown() {
	c.cancel()
}

// Run executes the complete conversion workflow
func (c *ConverterController) Run() error {
	defer c.cancel()

	// Verify dependencies
	if err := c.verifyDependencies(); err != nil {
		return err
	}

	// Scan directory for files
	pairs, err := c.scanDirectory()
	if err != nil {
		return err
	}

	if len(pairs) == 0 {
		c.view.ShowInfo("No matching video/subtitle pairs found in: %s", c.config.Directory)
		c.view.ShowInfo("Make sure you have both video files (.mp4, .mkv, etc.) and subtitle files (.ass) in this folder")
		return nil
	}

	c.view.ShowInfo("Found %d video/subtitle pairs to process", len(pairs))
	c.view.ShowInfo("Using %d parallel workers", c.config.Workers)
	
	atomic.StoreInt32(&c.totalFiles, int32(len(pairs)))

	// Dry run mode - just show what would be done
	if c.config.DryRun {
		c.view.ShowInfo("=== DRY RUN MODE ===")
		for _, pair := range pairs {
			c.view.ShowPair(pair)
		}
		return nil
	}

	// Process conversions with progress bars
	return c.processConversionsWithProgress(pairs)
}

// verifyDependencies checks required executables
func (c *ConverterController) verifyDependencies() error {
	if err := c.spp2pgsSvc.VerifyExecutable(); err != nil {
		return fmt.Errorf("Spp2Pgs not accessible: %w", err)
	}
	return nil
}

// scanDirectory finds video files and matches them with subtitles
func (c *ConverterController) scanDirectory() ([]*model.SubtitlePair, error) {
	entries, err := os.ReadDir(c.config.Directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var videoFiles []string
	var subtitleFiles []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(c.config.Directory, name)

		if util.IsVideoFile(name) {
			videoFiles = append(videoFiles, path)
		} else if util.IsASSFile(name) {
			subtitleFiles = append(subtitleFiles, path)
		}
	}

	c.view.ShowInfo("Found %d video files and %d ASS subtitle files", len(videoFiles), len(subtitleFiles))

	// Match videos with subtitles
	var pairs []*model.SubtitlePair
	for _, videoPath := range videoFiles {
		subtitlePath, found := util.FindBestSubtitleMatch(videoPath, subtitleFiles)
		if !found {
			c.view.ShowWarning("No matching subtitle found for: %s", filepath.Base(videoPath))
			continue
		}

		// Extract metadata
		metadata, err := c.extractMetadata(videoPath)
		if err != nil {
			c.view.ShowError("Failed to extract metadata from %s: %v", filepath.Base(videoPath), err)
			continue
		}

		// Determine output path
		supName := util.ReplaceExtension(filepath.Base(subtitlePath), ".sup")
		outputPath := filepath.Join(c.config.Directory, supName)

		// Check if output exists and skip if not overwriting
		if !c.config.Overwrite {
			if _, err := os.Stat(outputPath); err == nil {
				c.view.ShowInfo("Skipping %s (already exists)", supName)
				continue
			}
		}

		pair := &model.SubtitlePair{
			VideoPath:    videoPath,
			SubtitlePath: subtitlePath,
			OutputPath:   outputPath,
			Metadata:     *metadata,
		}
		pairs = append(pairs, pair)
	}

	return pairs, nil
}

// extractMetadata extracts video metadata with frame rate detection
func (c *ConverterController) extractMetadata(videoPath string) (*model.VideoMetadata, error) {
	metadata, err := c.ffprobeSvc.ExtractMetadata(videoPath)
	if err != nil {
		return nil, err
	}

	// Get precise frame rate
	fps, err := c.ffprobeSvc.ExtractFrameRate(videoPath)
	if err == nil && fps > 0 {
		metadata.FrameRate = fps
	}

	return metadata, nil
}

// processConversionsWithProgress handles parallel conversions with visual progress tracking
func (c *ConverterController) processConversionsWithProgress(pairs []*model.SubtitlePair) error {
	// Create main progress bar
	mainBar := progressbar.NewOptions(
		len(pairs),
		progressbar.OptionSetDescription("Converting subtitles"),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
	)

	// Start result collector
	resultDone := make(chan bool)
	go c.collectResultsWithProgress(len(pairs), resultDone, mainBar)

	// Start workers
	for i := 0; i < c.config.Workers; i++ {
		c.wg.Add(1)
		go c.workerWithProgress()
	}

	// Queue all work
	go func() {
		for _, pair := range pairs {
			select {
			case c.workQueue <- pair:
			case <-c.ctx.Done():
				return
			}
		}
		close(c.workQueue)
	}()

	// Wait for workers to complete
	c.wg.Wait()
	close(c.results)
	<-resultDone

	// Show final summary
	succeeded := atomic.LoadInt32(&c.succeeded)
	failed := atomic.LoadInt32(&c.failed)
	c.view.ShowSummary(int(succeeded), int(failed), 0)

	if failed > 0 {
		return fmt.Errorf("%d conversion(s) failed", failed)
	}
	return nil
}

// workerWithProgress processes conversion jobs with individual progress tracking
func (c *ConverterController) workerWithProgress() {
	defer c.wg.Done()

	for pair := range c.workQueue {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		fileName := filepath.Base(pair.SubtitlePath)
		
		result, err := c.spp2pgsSvc.ConvertWithProgress(pair, 30*time.Minute, false)
		
		if err != nil {
			atomic.AddInt32(&c.failed, 1)
			c.view.ShowError("✗ %s: %v", fileName, err)
		} else if result.Success {
			atomic.AddInt32(&c.succeeded, 1)
		} else {
			atomic.AddInt32(&c.failed, 1)
		}
		
		atomic.AddInt32(&c.completed, 1)
		c.results <- result
	}
}

// collectResultsWithProgress aggregates conversion results and updates main progress bar
func (c *ConverterController) collectResultsWithProgress(expected int, done chan bool, mainBar *progressbar.ProgressBar) {
	completed := 0
	
	for result := range c.results {
		completed++
		mainBar.Add(1)
		
		if !result.Success && result.Error != nil {
			// Error already shown by worker
		}
	}
	
	done <- true
}

// IsFFprobeAvailable checks if ffprobe is available in the system PATH
func IsFFprobeAvailable() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}
