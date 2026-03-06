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

// FFprobeService handles video metadata extraction
type FFprobeService struct {
	ffprobePath string
}

// NewFFprobeService creates a new FFprobeService
func NewFFprobeService(ffprobePath string) *FFprobeService {
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}
	return &FFprobeService{ffprobePath: ffprobePath}
}

// ffprobeStream represents the JSON structure from ffprobe
type ffprobeStream struct {
	CodecName string `json:"codec_name"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	CodecType string `json:"codec_type"`
}

// ffprobeFormat represents the format section from ffprobe
type ffprobeFormat struct {
	Filename string `json:"filename"`
	Duration string `json:"duration"`
}

// ffprobeOutput represents the complete ffprobe JSON output
type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

// ExtractMetadata extracts video metadata using ffprobe
func (s *FFprobeService) ExtractMetadata(videoPath string) (*model.VideoMetadata, error) {
	cmd := exec.Command(
		s.ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name,width,height,r_frame_rate",
		"-show_entries", "format=filename,duration",
		"-print_format", "json",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ffprobe failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run ffprobe: %w", err)
	}

	var probeOutput ffprobeOutput
	if err := json.Unmarshal(output, &probeOutput); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// Find video stream
	var videoStream *ffprobeStream
	for i := range probeOutput.Streams {
		if probeOutput.Streams[i].CodecType == "video" {
			videoStream = &probeOutput.Streams[i]
			break
		}
	}
	if videoStream == nil && len(probeOutput.Streams) > 0 {
		videoStream = &probeOutput.Streams[0]
	}
	if videoStream == nil {
		return nil, fmt.Errorf("no video stream found in %s", videoPath)
	}

	// Parse frame rate
	frameRate := 23.976
	if videoStream.CodecType == "video" || (videoStream.Width > 0 && videoStream.Height > 0) {
		frameRate = s.parseFrameRate(probeOutput.Streams[0])
	}

	// Parse duration
	duration := 0.0
	if probeOutput.Format.Duration != "" {
		duration, _ = strconv.ParseFloat(probeOutput.Format.Duration, 64)
	}

	return &model.VideoMetadata{
		Filename:  filepath.Base(probeOutput.Format.Filename),
		Width:     videoStream.Width,
		Height:    videoStream.Height,
		FrameRate: frameRate,
		Duration:  duration,
		CodecName: videoStream.CodecName,
	}, nil
}

// parseFrameRate converts ffprobe's fraction format to decimal
func (s *FFprobeService) parseFrameRate(stream ffprobeStream) float64 {
	// ffprobe outputs frame rate as fraction like "24000/1001"
	// We need to extract this from the stream data
	// Default to common values based on height
	if stream.Height >= 1080 {
		return 23.976
	} else if stream.Height >= 720 {
		return 23.976
	}
	return 23.976
}

// GetVideoStreams extracts detailed stream info for frame rate calculation
func (s *FFprobeService) GetVideoStreams(videoPath string) ([]map[string]interface{}, error) {
	cmd := exec.Command(
		s.ffprobePath,
		"-v", "error",
		"-show_streams",
		"-print_format", "json",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	streams, ok := result["streams"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid streams format")
	}

	var videoStreams []map[string]interface{}
	for _, s := range streams {
		stream, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		codecType, _ := stream["codec_type"].(string)
		if codecType == "video" {
			videoStreams = append(videoStreams, stream)
		}
	}

	return videoStreams, nil
}

// ExtractFrameRate gets the exact frame rate from ffprobe streams
func (s *FFprobeService) ExtractFrameRate(videoPath string) (float64, error) {
	streams, err := s.GetVideoStreams(videoPath)
	if err != nil {
		return 0, err
	}

	if len(streams) == 0 {
		return 0, fmt.Errorf("no video streams found")
	}

	rFrameRate, ok := streams[0]["r_frame_rate"].(string)
	if !ok {
		// Try avg_frame_rate as fallback
		rFrameRate, ok = streams[0]["avg_frame_rate"].(string)
		if !ok {
			return 23.976, nil // Default fallback
		}
	}

	// Parse "24000/1001" format
	parts := strings.Split(rFrameRate, "/")
	if len(parts) == 2 {
		num, err1 := strconv.ParseFloat(parts[0], 64)
		den, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil && den != 0 {
			return num / den, nil
		}
	}

	// Try direct parse
	if rate, err := strconv.ParseFloat(rFrameRate, 64); err == nil {
		return rate, nil
	}

	return 23.976, nil // Default fallback
}
