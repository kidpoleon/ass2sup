// Package controller orchestrates the subtitle conversion process with parallel
// processing and progress tracking. It implements a worker-pool pattern for
// efficient concurrent conversions while maintaining thread-safe progress
// reporting.
//
// Directory scanning is fully recursive: the entire tree under Config.Directory
// is walked, and video/subtitle files are matched within each individual
// subdirectory so that files from different seasons or shows never interfere
// with each other.
//
// Metadata extraction (ffprobe) is parallelised during the scan phase — up to
// maxProbeWorkers ffprobe processes run simultaneously so that scanning a large
// library is fast even when each probe call takes a second or two.
package controller

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/schollz/progressbar/v3"

	"ass2sup/model"
	"ass2sup/service"
	"ass2sup/util"
	"ass2sup/view"
)

const (
	// defaultWorkers is the number of concurrent Spp2Pgs processes to run.
	// Each worker spawns one Spp2Pgs.exe subprocess, so 16 means up to 16
	// subtitle files are rendered in parallel.
	defaultWorkers = 16

	// maxWorkers is an absolute safety ceiling.
	maxWorkers = 32

	// maxProbeWorkers limits concurrent ffprobe processes during the scan phase.
	// ffprobe is I/O-bound; 8 concurrent probes saturates typical storage without
	// overwhelming the OS scheduler.
	maxProbeWorkers = 8

	// conversionTimeout is the per-file deadline passed to Spp2Pgs.
	conversionTimeout = 30 * time.Minute
)

// ConverterController orchestrates the subtitle conversion process.
type ConverterController struct {
	config     *model.Config
	ffprobeSvc *service.FFprobeService
	spp2pgsSvc *service.Spp2PgsService
	view       *view.ConsoleView

	// Concurrency control
	workQueue chan *model.SubtitlePair
	results   chan *model.ConversionResult
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc

	// Progress tracking (all updated via atomic operations)
	totalFiles int32
	completed  int32
	succeeded  int32
	failed     int32
}

// NewConverterController creates a new controller instance.
//
// Worker count resolution:
//  1. cfg.Workers if > 0 (caller override)
//  2. defaultWorkers (16) otherwise
//
// The value is clamped to [1, maxWorkers].
func NewConverterController(cfg *model.Config) *ConverterController {
	workers := cfg.Workers
	if workers <= 0 {
		workers = defaultWorkers
	}
	if workers > maxWorkers {
		workers = maxWorkers
	}
	if workers < 1 {
		workers = 1
	}
	cfg.Workers = workers

	ctx, cancel := context.WithCancel(context.Background())

	return &ConverterController{
		config:     cfg,
		ffprobeSvc: service.NewFFprobeService(cfg.FFprobePath),
		spp2pgsSvc: service.NewSpp2PgsService(cfg.Spp2PgsPath),
		view:       view.NewConsoleView(),
		workQueue:  make(chan *model.SubtitlePair, workers*4),
		results:    make(chan *model.ConversionResult, workers*4),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Shutdown signals all workers to stop after their current job finishes.
func (c *ConverterController) Shutdown() {
	c.cancel()
}

// Run executes the complete conversion workflow and returns any fatal error.
func (c *ConverterController) Run() error {
	defer c.cancel()

	if err := c.verifyDependencies(); err != nil {
		return err
	}

	pairs, err := c.scanDirectory()
	if err != nil {
		return err
	}

	if len(pairs) == 0 {
		c.view.ShowInfo("No video/subtitle pairs found — nothing to convert")
		return nil
	}

	c.view.ShowInfo("%d pair(s) queued · %d workers", len(pairs), c.config.Workers)
	atomic.StoreInt32(&c.totalFiles, int32(len(pairs)))

	if c.config.DryRun {
		c.view.ShowInfo("=== DRY RUN MODE — no files will be written ===")
		for _, pair := range pairs {
			c.view.ShowPair(pair)
		}
		return nil
	}

	return c.processConversionsWithProgress(pairs)
}

// verifyDependencies checks that the required executables are accessible.
func (c *ConverterController) verifyDependencies() error {
	if err := c.spp2pgsSvc.VerifyExecutable(); err != nil {
		return fmt.Errorf("Spp2Pgs not accessible: %w", err)
	}
	return nil
}

// ── Scan ─────────────────────────────────────────────────────────────────────

// probeJob pairs a matched video+subtitle before metadata has been extracted.
type probeJob struct {
	videoPath    string
	subtitlePath string
	outputPath   string
}

// probeResult is the outcome of one ffprobe call.
type probeResult struct {
	job      probeJob
	metadata *model.VideoMetadata
	err      error
}

// scanDirectory walks the entire directory tree rooted at Config.Directory.
//
// Files are collected into per-directory buckets so that matching only ever
// happens between files that share the same parent folder. This prevents an
// episode in /Season 1/ from accidentally being paired with a subtitle in
// /Season 2/.
//
// After matching, metadata is extracted in parallel (up to maxProbeWorkers
// concurrent ffprobe processes) to keep scan time low for large libraries.
func (c *ConverterController) scanDirectory() ([]*model.SubtitlePair, error) {
	dirVideos := make(map[string][]string)
	dirSubs := make(map[string][]string)

	err := filepath.WalkDir(c.config.Directory, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			c.view.ShowWarning("Skipping inaccessible path: %s (%v)", path, walkErr)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		dir := filepath.Dir(path)
		switch {
		case util.IsVideoFile(name):
			dirVideos[dir] = append(dirVideos[dir], path)
		case util.IsASSFile(name):
			dirSubs[dir] = append(dirSubs[dir], path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree: %w", err)
	}

	// Count totals for the scan summary line.
	totalVideos, totalSubs, totalDirs := 0, 0, 0
	for dir, v := range dirVideos {
		totalVideos += len(v)
		if len(dirSubs[dir]) > 0 {
			totalDirs++
		}
	}
	for _, s := range dirSubs {
		totalSubs += len(s)
	}
	if totalDirs == 0 {
		totalDirs = len(dirVideos)
	}
	c.view.ShowInfo("%d video(s) · %d subtitle(s) · %d director(ies)", totalVideos, totalSubs, totalDirs)

	// Match video files to subtitle files within each directory.
	var jobs []probeJob
	for dir, videoFiles := range dirVideos {
		subtitleFiles := dirSubs[dir]
		if len(subtitleFiles) == 0 {
			c.view.ShowWarning("No .ass subtitles in: %s — skipping %d video file(s)", dir, len(videoFiles))
			continue
		}
		for _, videoPath := range videoFiles {
			subtitlePath, found := util.FindBestSubtitleMatch(videoPath, subtitleFiles)
			if !found {
				c.view.ShowWarning("No matching subtitle for: %s", filepath.Base(videoPath))
				continue
			}
			supName := util.ReplaceExtension(filepath.Base(subtitlePath), ".sup")
			outputPath := filepath.Join(dir, supName)
			if !c.config.Overwrite {
				if _, err := os.Stat(outputPath); err == nil {
					c.view.ShowInfo("Skipping %s (output already exists)", supName)
					continue
				}
			}
			jobs = append(jobs, probeJob{
				videoPath:    videoPath,
				subtitlePath: subtitlePath,
				outputPath:   outputPath,
			})
		}
	}

	if len(jobs) == 0 {
		return nil, nil
	}

	// Extract metadata for all matched videos in parallel.
	// Up to maxProbeWorkers ffprobe processes run simultaneously.
	numProbers := maxProbeWorkers
	if len(jobs) < numProbers {
		numProbers = len(jobs)
	}

	jobCh := make(chan probeJob, len(jobs))
	resultCh := make(chan probeResult, len(jobs))

	var probeWg sync.WaitGroup
	for i := 0; i < numProbers; i++ {
		probeWg.Add(1)
		go func() {
			defer probeWg.Done()
			for job := range jobCh {
				meta, err := c.ffprobeSvc.ExtractMetadata(job.videoPath)
				resultCh <- probeResult{job: job, metadata: meta, err: err}
			}
		}()
	}

	for _, job := range jobs {
		jobCh <- job
	}
	close(jobCh)

	probeWg.Wait()
	close(resultCh)

	// Collect results and build the final pair list.
	var pairs []*model.SubtitlePair
	for r := range resultCh {
		if r.err != nil {
			c.view.ShowError("Metadata error for %s: %v", filepath.Base(r.job.videoPath), r.err)
			continue
		}
		pairs = append(pairs, &model.SubtitlePair{
			VideoPath:    r.job.videoPath,
			SubtitlePath: r.job.subtitlePath,
			OutputPath:   r.job.outputPath,
			Metadata:     *r.metadata,
		})
	}

	// Sort for a deterministic processing order regardless of goroutine scheduling.
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].OutputPath < pairs[j].OutputPath
	})

	return pairs, nil
}

// ── Conversion ────────────────────────────────────────────────────────────────

// processConversionsWithProgress spins up the worker pool, queues all pairs,
// and displays a live progress bar with elapsed time and ETA.
func (c *ConverterController) processConversionsWithProgress(pairs []*model.SubtitlePair) error {
	mainBar := progressbar.NewOptions(
		len(pairs),
		progressbar.OptionSetDescription("  Converting"),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(45),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "━",
			SaucerHead:    "╸",
			SaucerPadding: "─",
			BarStart:      "▕",
			BarEnd:        "▏",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
	)

	// Collector goroutine — drains results and ticks the progress bar.
	resultDone := make(chan struct{})
	go func() {
		defer close(resultDone)
		for range c.results {
			mainBar.Add(1) //nolint:errcheck
		}
	}()

	// Show the bar immediately (sets its internal startTime) and keep the
	// elapsed-time display ticking while workers are running. Without this,
	// the bar only becomes visible when the first file finishes — which for
	// a long single-file conversion means the bar appears at 100% instantly.
	mainBar.Add(0) //nolint:errcheck // initial render, sets startTime
	tickStop := make(chan struct{})
	go func() {
		t := time.NewTicker(250 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				mainBar.Add(0) //nolint:errcheck
			case <-tickStop:
				return
			case <-c.ctx.Done():
				return
			}
		}
	}()

	start := time.Now()

	// Start the worker pool.
	for i := 0; i < c.config.Workers; i++ {
		c.wg.Add(1)
		go c.worker()
	}

	// Feed the work queue from a separate goroutine so we don't block here.
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

	c.wg.Wait()
	close(c.results)
	<-resultDone
	close(tickStop) // stop the elapsed-time ticker

	elapsed := time.Since(start)
	succeeded := atomic.LoadInt32(&c.succeeded)
	failed := atomic.LoadInt32(&c.failed)
	c.view.ShowSummary(int(succeeded), int(failed), elapsed)

	if failed > 0 {
		return fmt.Errorf("%d conversion(s) failed", failed)
	}
	return nil
}

// worker pulls jobs from the work queue and converts them one at a time.
// Multiple workers run concurrently, each driving its own Spp2Pgs process.
func (c *ConverterController) worker() {
	defer c.wg.Done()

	for pair := range c.workQueue {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		fileName := filepath.Base(pair.SubtitlePath)
		result, err := c.spp2pgsSvc.ConvertWithProgress(pair, conversionTimeout, false)

		switch {
		case err != nil:
			atomic.AddInt32(&c.failed, 1)
			c.view.ShowError("%s: %v", fileName, err)
		case result.Success:
			atomic.AddInt32(&c.succeeded, 1)
		default:
			atomic.AddInt32(&c.failed, 1)
			c.view.ShowError("%s: conversion produced no output", fileName)
		}

		atomic.AddInt32(&c.completed, 1)
		c.results <- result
	}
}

// IsFFprobeAvailable reports whether ffprobe is reachable via PATH.
func IsFFprobeAvailable() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}
