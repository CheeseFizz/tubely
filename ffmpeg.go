package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type ffprobe_stream struct {
	Index        int    `json:"index"`
	Codec        string `json:"codec_name"`
	Codec_type   string `json:"codec_type"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	Aspect_ratio string `json:"display_aspect_ratio,omitempty"`
}

type ffprobe_out struct {
	Streams []ffprobe_stream `json:"streams"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	var aspect_ratio string
	zero_stream := ffprobe_stream{}

	_, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	ex := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	out := bytes.Buffer{}
	ex.Stdout = &out
	err = ex.Run()
	if err != nil {
		return "", err
	}

	video_data := ffprobe_out{make([]ffprobe_stream, 0)}
	json.Unmarshal(out.Bytes(), &video_data)

	var video_stream ffprobe_stream
	for _, item := range video_data.Streams {
		if item.Index == 0 {
			video_stream = item
			break
		}
	}
	if video_stream == zero_stream {
		return "", fmt.Errorf("video stream not found")
	}

	switch video_stream.Aspect_ratio {
	case "16:9", "9:16":
		aspect_ratio = video_stream.Aspect_ratio
	case "4:3":
		aspect_ratio = "other"
	default:
		if (video_stream.Width / 16) == (video_stream.Height / 9) {
			aspect_ratio = "16:9"
		} else if (video_stream.Width / 9) == (video_stream.Height / 16) {
			aspect_ratio = "9:16"
		} else {
			aspect_ratio = "other"
		}
	}

	return aspect_ratio, nil
}

func processVideoForFastStart(filePath string) (string, error) {
	outfile := filePath + ".processing"
	ex := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outfile)
	out := bytes.Buffer{}
	ex.Stdout = &out
	err := ex.Run()
	if err != nil {
		return "", err
	}

	return outfile, nil
}
