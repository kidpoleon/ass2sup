package service

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"ass2sup/model"
)

// FFprobeService handles video metadata extraction.
type FFprobeService struct {
	ffprobePath string
}

// NewFFprobeService creates a new FFprobeService.
// If ffprobePath is empty, "ffprobe" (from PATH) is used.
func NewFFprobeService(ffprobePath string) *FFprobeService {
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}
	return &FFprobeService{ffprobePath: ffprobePath}
}

// ffprobeJSON is the subset of ffprobe's JSON output we care about.
type ffprobeJSON struct {
	Streams []struct {
		CodecName    string `json:"codec_name"`
		CodecType    string `json:"codec_type"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		RFrameRate   string `json:"r_frame_rate"`
		AvgFrameRate string `json:"avg_frame_rate"`
	} `json:"streams"`
	Format struct {
		Filename string `json:"filename"`
		Duration string `json:"duration"`
	} `json:"format"`
}

// ExtractMetadata extracts all needed video metadata in a single ffprobe call.
// It reads codec, resolution, frame rate (as an exact rational), and duration.
// This replaces the previous two-call pattern (ExtractMetadata + ExtractFrameRate).
func (s *FFprobeService) ExtractMetadata(videoPath string) (*model.VideoMetadata, error) {
	cmd := exec.Command(
		s.ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name,codec_type,width,height,r_frame_rate,avg_frame_rate",
		"-show_entries", "format=filename,duration",
		"-print_format", "json",
		videoPath,
	)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ffprobe error: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("failed to run ffprobe: %w", err)
	}

	var probe ffprobeJSON
	if err := json.Unmarshal(out, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	if len(probe.Streams) == 0 {
		return nil, fmt.Errorf("no video stream found in %s", filepath.Base(videoPath))
	}
	stream := probe.Streams[0]

	// Parse frame rate from rational "24000/1001"; fall back to avg_frame_rate.
	fps := parseRational(stream.RFrameRate)
	if fps <= 0 {
		fps = parseRational(stream.AvgFrameRate)
	}
	if fps <= 0 {
		fps = 23.976 // safe default
	}

	duration := 0.0
	if probe.Format.Duration != "" {
		duration, _ = strconv.ParseFloat(probe.Format.Duration, 64)
	}

	filename := filepath.Base(videoPath)
	if probe.Format.Filename != "" {
		filename = filepath.Base(probe.Format.Filename)
	}

	return &model.VideoMetadata{
		Filename:  filename,
		Width:     stream.Width,
		Height:    stream.Height,
		FrameRate: fps,
		Duration:  duration,
		CodecName: stream.CodecName,
	}, nil
}

// parseRational converts a rational string like "24000/1001" to float64.
// Also handles plain numeric strings like "25".
func parseRational(s string) float64 {
	if s == "" {
		return 0
	}
	if parts := strings.SplitN(s, "/", 2); len(parts) == 2 {
		num, e1 := strconv.ParseFloat(parts[0], 64)
		den, e2 := strconv.ParseFloat(parts[1], 64)
		if e1 == nil && e2 == nil && den > 0 {
			return num / den
		}
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
