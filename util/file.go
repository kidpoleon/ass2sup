package util

import (
	"path/filepath"
	"regexp"
	"strings"
)

// VideoExtensions contains supported video file extensions
var VideoExtensions = []string{
	".mp4", ".mkv", ".avi", ".mov", ".m4v", ".webm", ".ts", ".m2ts",
}

// SubtitleExtensions contains supported subtitle file extensions
var SubtitleExtensions = []string{
	".ass", ".ssa", ".srt",
}

// IsVideoFile checks if a file has a video extension
func IsVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, vext := range VideoExtensions {
		if ext == vext {
			return true
		}
	}
	return false
}

// IsSubtitleFile checks if a file has a subtitle extension
func IsSubtitleFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, sext := range SubtitleExtensions {
		if ext == sext {
			return true
		}
	}
	return false
}

// IsASSFile checks if a file is an ASS subtitle file
func IsASSFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".ass"
}

// GetBaseName returns the filename without extension
func GetBaseName(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

// ReplaceExtension replaces the file extension
func ReplaceExtension(filename, newExt string) string {
	if !strings.HasPrefix(newExt, ".") {
		newExt = "." + newExt
	}
	return GetBaseName(filename) + newExt
}

// MatchSubtitleToVideo tries to find a matching subtitle file for a video
func MatchSubtitleToVideo(videoPath string, subtitleFiles []string) (string, bool) {
	videoBase := filepath.Base(videoPath)
	videoName := GetBaseName(videoBase)
	videoNameNorm := normalizeForMatch(videoName)

	for _, subPath := range subtitleFiles {
		subBase := filepath.Base(subPath)
		subName := GetBaseName(subBase)
		subNameNorm := normalizeForMatch(subName)

		if videoName == subName || videoNameNorm == subNameNorm {
			return subPath, true
		}
		if strings.Contains(subNameNorm, videoNameNorm) || strings.Contains(videoNameNorm, subNameNorm) {
			return subPath, true
		}
	}
	return "", false
}

// normalizeForMatch normalizes a filename for matching
func normalizeForMatch(name string) string {
	s := strings.ToLower(name)
	replacements := []string{
		".en", ".english", ".eng",
		".sub", ".subtitle",
		"_en", "_english", "_eng",
	}
	for _, r := range replacements {
		s = strings.ReplaceAll(s, r, "")
	}
	return s
}

// FindBestSubtitleMatch finds the best matching subtitle for a video file
func FindBestSubtitleMatch(videoFile string, subtitleFiles []string) (string, bool) {
	videoBase := filepath.Base(videoFile)
	videoNameNoExt := GetBaseName(videoBase)

	var bestMatch string
	bestScore := -1

	for _, subFile := range subtitleFiles {
		subBase := filepath.Base(subFile)
		subNameNoExt := GetBaseName(subBase)

		score := calculateMatchScore(videoNameNoExt, subNameNoExt)
		if score > bestScore {
			bestScore = score
			bestMatch = subFile
		}
	}

	if bestScore >= 60 {
		return bestMatch, true
	}
	return "", false
}

// calculateMatchScore calculates a matching score between video and subtitle names
func calculateMatchScore(videoName, subName string) int {
	vNorm := normalizeForMatch(videoName)
	sNorm := normalizeForMatch(subName)

	if vNorm == sNorm {
		return 100
	}
	if strings.Contains(sNorm, vNorm) || strings.Contains(vNorm, sNorm) {
		return 80
	}

	vEp := extractEpisodeNumber(vNorm)
	sEp := extractEpisodeNumber(sNorm)
	if vEp != "" && vEp == sEp {
		return 60
	}

	return -1
}

// Episode regex patterns
var episodePatterns = []*regexp.Regexp{
	regexp.MustCompile(`[Ss](\d+)[Ee](\d+)`),  // S01E01
	regexp.MustCompile(`(\d+)[Xx](\d+)`),     // 1x01
	regexp.MustCompile(`[Ee]pisode\s*(\d+)`),  // Episode 1
}

// extractEpisodeNumber extracts S01E01 style episode numbers
func extractEpisodeNumber(name string) string {
	for _, re := range episodePatterns {
		matches := re.FindString(name)
		if matches != "" {
			return strings.ToLower(matches)
		}
	}
	return ""
}
