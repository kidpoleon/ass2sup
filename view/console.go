package view

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"ass2sup/model"
)

// ConsoleView handles all console output
type ConsoleView struct {
	infoColor    *color.Color
	successColor *color.Color
	errorColor   *color.Color
	warnColor    *color.Color
	headerColor  *color.Color
}

// package-level colors for global functions
var (
	pkgInfoColor    = color.New(color.FgCyan)
	pkgSuccessColor = color.New(color.FgGreen)
	pkgErrorColor   = color.New(color.FgRed)
	pkgWarnColor    = color.New(color.FgYellow)
	pkgHeaderColor  = color.New(color.FgWhite, color.Bold)
)

// PrintBanner displays the application banner with version info
func PrintBanner(name, version string) {
	fmt.Println()
	pkgHeaderColor.Println("╔══════════════════════════════════════════════════════════╗")
	pkgHeaderColor.Printf("║  %-56s║\n", name+" v"+version)
	pkgHeaderColor.Println("║  ASS to PGS Subtitle Converter                           ║")
	pkgHeaderColor.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// PrintInfo displays an informational message
func PrintInfo(message string) {
	pkgInfoColor.Println("ℹ " + message)
}

// PrintSuccess displays a success message
func PrintSuccess(message string) {
	pkgSuccessColor.Println("✓ " + message)
}

// PrintError displays an error message
func PrintError(message string) {
	pkgErrorColor.Println("✗ " + message)
}

// PrintWarning displays a warning message
func PrintWarning(message string) {
	pkgWarnColor.Println("⚠ " + message)
}

// NewConsoleView creates a new ConsoleView with colored output
func NewConsoleView() *ConsoleView {
	return &ConsoleView{
		infoColor:    color.New(color.FgCyan),
		successColor: color.New(color.FgGreen),
		errorColor:   color.New(color.FgRed),
		warnColor:    color.New(color.FgYellow),
		headerColor:  color.New(color.FgWhite, color.Bold),
	}
}

// ShowInfo displays an informational message
func (v *ConsoleView) ShowInfo(format string, args ...interface{}) {
	v.infoColor.Println(fmt.Sprintf(format, args...))
}

// ShowSuccess displays a success message
func (v *ConsoleView) ShowSuccess(format string, args ...interface{}) {
	v.successColor.Println(fmt.Sprintf(format, args...))
}

// ShowError displays an error message
func (v *ConsoleView) ShowError(format string, args ...interface{}) {
	v.errorColor.Println(fmt.Sprintf(format, args...))
}

// ShowWarning displays a warning message
func (v *ConsoleView) ShowWarning(format string, args ...interface{}) {
	v.warnColor.Println(fmt.Sprintf(format, args...))
}

// ShowHeader displays a header
func (v *ConsoleView) ShowHeader(title string) {
	fmt.Println()
	v.headerColor.Println("═══════════════════════════════════════════")
	v.headerColor.Println("  " + title)
	v.headerColor.Println("═══════════════════════════════════════════")
	fmt.Println()
}

// ShowPair displays a subtitle pair info
func (v *ConsoleView) ShowPair(pair *model.SubtitlePair) {
	fmt.Printf("  Video:    %s\n", filepath.Base(pair.VideoPath))
	fmt.Printf("  Subtitle: %s\n", filepath.Base(pair.SubtitlePath))
	fmt.Printf("  Output:   %s\n", filepath.Base(pair.OutputPath))
	fmt.Printf("  Format:   %dx%d @ %.3f fps\n",
		pair.Metadata.Width,
		pair.Metadata.Height,
		pair.Metadata.FrameRate)
	fmt.Println()
}

// ShowProgress displays a progress message
func (v *ConsoleView) ShowProgress(format string, args ...interface{}) {
	fmt.Printf("  → %s\n", fmt.Sprintf(format, args...))
}

// ShowSummary displays the final conversion summary
func (v *ConsoleView) ShowSummary(succeeded, failed int, totalDuration float64) {
	fmt.Println()
	v.headerColor.Println("═══════════════════════════════════════════")
	v.headerColor.Println("  CONVERSION SUMMARY")
	v.headerColor.Println("═══════════════════════════════════════════")
	fmt.Println()

	if succeeded > 0 {
		v.successColor.Printf("  ✓ Succeeded: %d\n", succeeded)
	}
	if failed > 0 {
		v.errorColor.Printf("  ✗ Failed: %d\n", failed)
	}

	fmt.Printf("\n  Total time: %.1f seconds\n", totalDuration)
	fmt.Printf("  Average: %.1f seconds per file\n", totalDuration/float64(succeeded+failed))
	fmt.Println()

	if failed == 0 {
		v.successColor.Println("  All conversions completed successfully!")
	} else {
		v.warnColor.Printf("  Completed with %d errors\n", failed)
		os.Exit(1)
	}
}

// ShowUsage displays usage information
func (v *ConsoleView) ShowUsage() {
	v.ShowHeader("ASS to SUP Converter")
	fmt.Println("Converts ASS subtitle files to Blu-ray PGS (.sup) format")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ass2sup [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -dir string        Input directory (default \".\")")
	fmt.Println("  -spp2pgs string    Path to Spp2Pgs.exe (default \"Spp2Pgs.exe\")")
	fmt.Println("  -ffprobe string    Path to ffprobe (default \"ffprobe\")")
	fmt.Println("  -dry-run           Show what would be done without converting")
	fmt.Println("  -force             Overwrite existing .sup files")
	fmt.Println("  -workers int       Parallel conversions (default 1)")
	fmt.Println("  -v                 Verbose output")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ass2sup -dir=\"C:\\Videos\" -spp2pgs=\"Spp2Pgs.exe\"")
	fmt.Println("  ass2sup -dir=\".\" -dry-run -v")
	fmt.Println()
}
