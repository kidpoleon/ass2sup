//go:build windows

// Package embed provides embedded Spp2Pgs binaries for self-contained operation.
// The binaries are extracted to a temporary directory at runtime and cleaned up on exit.
// This package is Windows-only: it embeds Windows PE executables and DLLs.
package embed

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed Spp2Pgs.exe Spp2Pgs64.exe xy-VSSppf.dll xy-VSSppf64.dll
var embeddedFiles embed.FS

// ExtractedBinaries holds paths to extracted Spp2Pgs executables
type ExtractedBinaries struct {
	Spp2PgsPath string
	TempDir     string
}

// Extract extracts the appropriate Spp2Pgs binary and its DLL to a temp directory.
// The caller should call Cleanup() when done to remove the temp files.
func Extract() (*ExtractedBinaries, error) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "ass2sup-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Determine which binary to use based on architecture
	var exeName, dllName string
	if runtime.GOARCH == "amd64" {
		exeName = "Spp2Pgs64.exe"
		dllName = "xy-VSSppf64.dll"
	} else {
		exeName = "Spp2Pgs.exe"
		dllName = "xy-VSSppf.dll"
	}

	// Extract executable
	exePath := filepath.Join(tempDir, exeName)
	if err := extractFile(exeName, exePath); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to extract %s: %w", exeName, err)
	}

	// Extract DLL
	dllPath := filepath.Join(tempDir, dllName)
	if err := extractFile(dllName, dllPath); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to extract %s: %w", dllName, err)
	}

	return &ExtractedBinaries{
		Spp2PgsPath: exePath,
		TempDir:     tempDir,
	}, nil
}

// extractFile extracts a single file from embedded FS to disk
func extractFile(srcName, dstPath string) error {
	srcFile, err := embeddedFiles.Open(srcName)
	if err != nil {
		return fmt.Errorf("failed to open embedded file %s: %w", srcName, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", dstPath, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

// Cleanup removes the temporary directory and extracted files
func (e *ExtractedBinaries) Cleanup() error {
	if e.TempDir != "" {
		return os.RemoveAll(e.TempDir)
	}
	return nil
}
