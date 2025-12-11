package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getVideoAspectRatio(filepath string) (string, error) {

	stream := &bytes.Buffer{}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)
	cmd.Stdout = stream
	cmd.Run()

	type Stream struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	type FFProbeOutput struct {
		Streams []Stream `json:"streams"`
	}

	stats := &FFProbeOutput{}

	err := json.Unmarshal(stream.Bytes(), stats)
	if err != nil {
		return "", fmt.Errorf("unmarshal err %v", err)
	}

	if len(stats.Streams) == 0 {
		return "", fmt.Errorf("streams empty")
	}

	wideRatio := math.Round(16.0/9.0*10) / 10
	longRatio := math.Round(9.0/16.0*10) / 10

	videoRatio := math.Round(float64(stats.Streams[0].Width)/float64(stats.Streams[0].Height)*10) / 10

	switch videoRatio {
	case wideRatio:
		return "landscape/", nil
	case longRatio:
		return "portrait/", nil
	default:
		return "other/", nil
	}

}

func processVideoForFastStart(filepath string) (string, error) {
	output := fmt.Sprintf("tubely.processing")

	cmd := exec.Command("ffmpeg", "-i", filepath, "-movflags", "faststart", "-codec", "copy", "-f", "mp4", output)
	var stderr bytes.Buffer // Declare a buffer to capture stderr
	cmd.Stderr = &stderr    // Assign the buffer to cmd.Stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ffmpeg err %w", err)
	}

	return output, nil
}
