// Package view provides console output helpers for ass2sup, styled with the
// Charm lipgloss library to match the huh TUI form's visual identity.
package view

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"

	"ass2sup/model"
)

// ── Charm-theme palette (mirrors huh.ThemeCharm exactly) ────────────────────

var (
	colIndigo  = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
	colGreen   = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
	colRed     = lipgloss.AdaptiveColor{Light: "#FF4672", Dark: "#ED567A"}
	colYellow  = lipgloss.AdaptiveColor{Light: "#F1A623", Dark: "#F1A623"}
	colFuchsia = lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"} //nolint:unused
	colMuted   = lipgloss.AdaptiveColor{Light: "#747474", Dark: "#8A8A8A"}
)

// ── Shared styles ────────────────────────────────────────────────────────────

var (
	styleInfo    = lipgloss.NewStyle().Foreground(colIndigo)
	styleSuccess = lipgloss.NewStyle().Foreground(colGreen)
	styleError   = lipgloss.NewStyle().Foreground(colRed)
	styleWarn    = lipgloss.NewStyle().Foreground(colYellow)
	styleMuted   = lipgloss.NewStyle().Foreground(colMuted)
	styleBold    = lipgloss.NewStyle().Bold(true)

	// Rounded box with an indigo border — used for the banner and summary.
	styleIndigoBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colIndigo).
			Padding(0, 2)

	// Rounded box with a muted border — used for ShowPair (dry-run mode).
	styleMutedBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colMuted).
			Padding(0, 1)
)

// ── fmtDuration ──────────────────────────────────────────────────────────────

// fmtDuration formats a time.Duration as a compact human-readable string:
//   - >= 1 minute  → "5m 23s"
//   - >= 1 second  → "45s" or "1.2s"
//   - < 1 second   → "120ms"
func fmtDuration(d time.Duration) string {
	switch {
	case d >= time.Minute:
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", m, s)
	case d >= time.Second:
		if d%time.Second == 0 {
			return fmt.Sprintf("%ds", int(d.Seconds()))
		}
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
}

// ── Package-level functions ───────────────────────────────────────────────────

// PrintBanner displays the application banner inside a rounded indigo-bordered
// box.
func PrintBanner(name, version string) {
	title := styleBold.Render(name + " v" + version)
	sub := styleMuted.Render("ASS to PGS Subtitle Converter")
	fmt.Println()
	fmt.Println(styleIndigoBox.Render(title + "\n" + sub))
	fmt.Println()
}

// PrintInfo displays an informational message on stdout.
func PrintInfo(message string) {
	fmt.Println(styleInfo.Render("ℹ " + message))
}

// PrintSuccess displays a success message on stdout.
func PrintSuccess(message string) {
	fmt.Println(styleSuccess.Render("✓ " + message))
}

// PrintError displays an error message on stderr.
func PrintError(message string) {
	fmt.Fprintln(os.Stderr, styleError.Render("✗ "+message))
}

// PrintWarning displays a warning message on stdout.
func PrintWarning(message string) {
	fmt.Println(styleWarn.Render("⚠ " + message))
}

// ── ConsoleView ───────────────────────────────────────────────────────────────

// ConsoleView handles all console output. All methods are safe to call
// concurrently from multiple goroutines.
type ConsoleView struct {
	mu sync.Mutex
}

// NewConsoleView creates a new ConsoleView.
func NewConsoleView() *ConsoleView {
	return &ConsoleView{}
}

// ShowInfo displays an informational message on stdout.
func (v *ConsoleView) ShowInfo(format string, args ...interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	fmt.Println(styleInfo.Render("ℹ " + fmt.Sprintf(format, args...)))
}

// ShowSuccess displays a success message on stdout.
func (v *ConsoleView) ShowSuccess(format string, args ...interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	fmt.Println(styleSuccess.Render("✓ " + fmt.Sprintf(format, args...)))
}

// ShowError displays an error message on stderr.
func (v *ConsoleView) ShowError(format string, args ...interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	fmt.Fprintln(os.Stderr, styleError.Render("✗ "+fmt.Sprintf(format, args...)))
}

// ShowWarning displays a warning message on stdout.
func (v *ConsoleView) ShowWarning(format string, args ...interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	fmt.Println(styleWarn.Render("⚠ " + fmt.Sprintf(format, args...)))
}

// ShowPair displays a subtitle pair inside a muted rounded-border box. Intended
// for dry-run mode where no actual conversion is performed.
func (v *ConsoleView) ShowPair(pair *model.SubtitlePair) {
	v.mu.Lock()
	defer v.mu.Unlock()

	label := func(s string) string { return styleMuted.Render(s) }

	content := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %dx%d @ %.3f fps",
		label("Video:   "), filepath.Base(pair.VideoPath),
		label("Subtitle:"), filepath.Base(pair.SubtitlePath),
		label("Output:  "), filepath.Base(pair.OutputPath),
		label("Format:  "),
		pair.Metadata.Width,
		pair.Metadata.Height,
		pair.Metadata.FrameRate,
	)

	fmt.Println(styleMutedBox.Render(content))
}

// ShowSummary displays the final conversion summary inside a rounded
// indigo-bordered box. The caller is responsible for any exit-code logic
// based on the failed count.
func (v *ConsoleView) ShowSummary(succeeded, failed int, elapsed time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()

	total := succeeded + failed
	avgStr := "N/A"
	if total > 0 {
		avgStr = fmtDuration(elapsed / time.Duration(total))
	}

	succLine := fmt.Sprintf("%s  Succeeded: %s",
		styleSuccess.Render("✓"),
		styleSuccess.Render(fmt.Sprintf("%d", succeeded)),
	)
	failLine := fmt.Sprintf("%s  Failed:    %s",
		styleError.Render("✗"),
		styleError.Render(fmt.Sprintf("%d", failed)),
	)

	var statusLine string
	if failed == 0 {
		statusLine = styleSuccess.Render("✓ All conversions completed successfully!")
	} else {
		statusLine = styleWarn.Render(fmt.Sprintf("⚠  Completed with %d error(s)", failed))
	}

	content := fmt.Sprintf(
		"%s\n%s\n\n  Total time:   %s\n  Avg per file: %s\n\n%s",
		succLine,
		failLine,
		fmtDuration(elapsed),
		avgStr,
		statusLine,
	)

	fmt.Println()
	fmt.Println(styleIndigoBox.Render(content))
	fmt.Println()
}
