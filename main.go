//go:build windows

// ass2sup - A self-contained ASS to PGS subtitle converter for Windows.
//
// Usage:
//
//	Run ass2sup.exe. A TUI prompt will ask for the source directory.
//	The program scans that directory recursively, pairs every video file
//	with its matching .ass subtitle, and converts them all to .sup
//	(Blu-ray PGS) format using the embedded Spp2Pgs converter.
//
// Requirements:
//
//	ffprobe must be available in PATH (part of the FFmpeg suite).
//	Download: https://ffmpeg.org/download.html
//
// Author: kidpoleon
// License: MIT
package main

//go:generate goversioninfo -64 -icon=icon/ass2sup_ico.ico

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/huh"

	"ass2sup/controller"
	"ass2sup/embed"
	"ass2sup/model"
	"ass2sup/view"
)

const (
	appName    = "ass2sup"
	appVersion = "2.0.0"
)

func main() {
	view.PrintBanner(appName, appVersion)

	// Extract the embedded Spp2Pgs binaries to a temp directory.
	// They are cleaned up via defer when main returns.
	view.PrintInfo("Extracting converter...")
	binaries, err := embed.Extract()
	if err != nil {
		view.PrintError(fmt.Sprintf("Failed to extract binaries: %v", err))
		waitForExit()
		os.Exit(1)
	}
	defer binaries.Cleanup()
	view.PrintSuccess("Ready")

	// ffprobe must be on PATH for video metadata extraction.
	if !controller.IsFFprobeAvailable() {
		view.PrintError("ffprobe not found in PATH")
		view.PrintInfo("Please install FFmpeg and ensure ffprobe is in your system PATH")
		view.PrintInfo("Download: https://ffmpeg.org/download.html")
		waitForExit()
		os.Exit(1)
	}

	// Show the TUI and ask the user where their media lives.
	sourceDir, err := promptSourceDirectory()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("\nCancelled.")
			os.Exit(0)
		}
		view.PrintError(fmt.Sprintf("Input error: %v", err))
		waitForExit()
		os.Exit(1)
	}

	// Graceful shutdown on Ctrl-C / SIGTERM.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cfg := &model.Config{
		Directory:   sourceDir,
		Spp2PgsPath: binaries.Spp2PgsPath,
		Workers:     0, // 0 → auto → 16 (see controller)
		DryRun:      false,
		Verbose:     true,
	}

	ctrl := controller.NewConverterController(cfg)

	go func() {
		<-sigChan
		view.PrintWarning("\nInterrupted! Shutting down gracefully...")
		ctrl.Shutdown()
	}()

	view.PrintInfo(fmt.Sprintf("Scanning recursively: %s", sourceDir))
	if err := ctrl.Run(); err != nil {
		view.PrintError(fmt.Sprintf("Conversion failed: %v", err))
		waitForExit()
		os.Exit(1)
	}

	waitForExit()
}

// promptSourceDirectory renders a charmbracelet/huh TUI form that asks the
// user to type the path to the folder they want to process. The path is
// validated before the form closes: it must exist and be a directory.
func promptSourceDirectory() (string, error) {
	var dir string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Source Directory").
				Description("Path to folder with video files and .ass subtitles (searched recursively)").
				Placeholder(`C:\Videos`).
				Value(&dir).
				Validate(func(s string) error {
					s = cleanPath(s)
					if s == "" {
						return fmt.Errorf("path cannot be empty")
					}
					info, err := os.Stat(s)
					if err != nil {
						return fmt.Errorf("cannot access directory: %v", err)
					}
					if !info.IsDir() {
						return fmt.Errorf("path is not a directory")
					}
					return nil
				}),
		),
	).WithTheme(huh.ThemeCharm()).WithWidth(64)

	if err := form.Run(); err != nil {
		return "", err
	}

	fmt.Println() // ensure a clean newline after the form closes
	return cleanPath(dir), nil
}

// cleanPath strips surrounding whitespace and stray quotes that Windows
// Explorer sometimes adds when a user drags a path into the terminal.
func cleanPath(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	return strings.TrimSpace(s)
}

func waitForExit() {
	fmt.Println("\nPress Enter to exit...")
	fmt.Scanln()
}
