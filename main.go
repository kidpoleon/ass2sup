// ass2sup - A self-contained ASS to PGS subtitle converter
// This program automatically converts .ass subtitle files to .sup (Blu-ray PGS) format
// when placed in a media folder alongside video files.
//
// Usage:
//   Simply drag and drop ass2sup.exe into any folder containing video files
//   and matching .ass subtitle files, then double-click to run.
//
// Features:
//   - Self-contained: Embeds Spp2Pgs and required DLLs
//   - Auto-detection: Finds video/subtitle pairs automatically
//   - Parallel processing: Converts multiple files concurrently
//   - Progress tracking: Real-time progress bars for each conversion
//   - Video metadata: Uses ffprobe to detect resolution and frame rate
//
// Author: Cascade
// License: MIT

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ass2sup/controller"
	"ass2sup/embed"
	"ass2sup/model"
	"ass2sup/view"
)

const (
	appName    = "ass2sup"
	appVersion = "1.0.0"
)

func main() {
	// Print banner
	view.PrintBanner(appName, appVersion)

	// Extract embedded Spp2Pgs binaries
	view.PrintInfo("Extracting embedded converter...")
	binaries, err := embed.Extract()
	if err != nil {
		view.PrintError(fmt.Sprintf("Failed to extract binaries: %v", err))
		waitForExit()
		os.Exit(1)
	}
	defer binaries.Cleanup()
	view.PrintSuccess("Converter ready")

	// Get working directory (where the exe was launched from)
	workDir, err := os.Getwd()
	if err != nil {
		view.PrintError(fmt.Sprintf("Failed to get working directory: %v", err))
		waitForExit()
		os.Exit(1)
	}

	// Check if ffprobe is available
	if !controller.IsFFprobeAvailable() {
		view.PrintError("ffprobe not found in PATH")
		view.PrintInfo("Please install FFmpeg and ensure ffprobe is in your system PATH")
		view.PrintInfo("Download: https://ffmpeg.org/download.html")
		waitForExit()
		os.Exit(1)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create converter with default settings
	config := &model.Config{
		Directory:   workDir,
		Spp2PgsPath: binaries.Spp2PgsPath,
		Workers:     0, // Auto-detect based on CPU
		DryRun:      false,
		Verbose:     true,
	}

	// Create controller
	ctrl := controller.NewConverterController(config)

	// Handle interrupt signals
	go func() {
		<-sigChan
		view.PrintWarning("\nInterrupted! Shutting down gracefully...")
		ctrl.Shutdown()
	}()

	// Run conversion
	view.PrintInfo(fmt.Sprintf("Scanning directory: %s", workDir))
	if err := ctrl.Run(); err != nil {
		view.PrintError(fmt.Sprintf("Conversion failed: %v", err))
		waitForExit()
		os.Exit(1)
	}

	// Success - wait for user input before closing (when double-clicked)
	waitForExit()
}

func waitForExit() {
	fmt.Println("\nPress Enter to exit...")
	fmt.Scanln()
}
