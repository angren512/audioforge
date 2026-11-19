package media

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Info 是 ffprobe 探测出的音频信息。
type Info struct {
	Container  string  `json:"container"`
	Codec      string  `json:"codec"`
	Duration   float64 `json:"duration"`
	SampleRate int     `json:"sampleRate"`
	Channels   int     `json:"channels"`
	BitRate    int64   `json:"bitRate"`
	HasCover   bool    `json:"hasCover"`
}

type probeResult struct {
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		CodecName   string `json:"codec_name"`
		CodecType   string `json:"codec_type"`
		SampleRate  string `json:"sample_rate"`
		Channels    int    `json:"channels"`
		BitRate     string `json:"bit_rate"`
		Disposition struct {
			AttachedPic int `json:"attached_pic"`
		} `json:"disposition"`
	} `json:"streams"`
}

// Probe 调用 ffprobe 读取音频信息。
func Probe(ctx context.Context, ffprobe, path string) (*Info, error) {
	cmd := exec.CommandContext(ctx, ffprobe,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe 无法识别该文件: %w", err)
	}

	var pr probeResult
	if err := json.Unmarshal(out, &pr); err != nil {
		return nil, fmt.Errorf("解析 ffprobe 输出失败: %w", err)
	}

	info := &Info{Container: pr.Format.FormatName}
	info.Duration = parseFloat(pr.Format.Duration)
	info.BitRate = parseInt(pr.Format.BitRate)

	for _, s := range pr.Streams {
		switch s.CodecType {
		case "audio":
			if info.Codec == "" {
				info.Codec = s.CodecName
				info.SampleRate = int(parseInt(s.SampleRate))
				info.Channels = s.Channels
				if info.BitRate == 0 {
					info.BitRate = parseInt(s.BitRate)
				}
			}
		case "video":
			if s.Disposition.AttachedPic == 1 {
				info.HasCover = true
			}
		}
	}

	if info.Codec == "" {
		return nil, fmt.Errorf("文件中没有可用的音频流")
	}
	return info, nil
}

func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return v
}

func parseInt(s string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// ExtFromContainer 把 ffprobe 的 format_name 映射为常见扩展名。
func ExtFromContainer(container, codec string) string {
	c := strings.ToLower(container)
	switch {
	case strings.Contains(c, "flac"):
		return ".flac"
	case strings.Contains(c, "ogg"):
		return ".ogg"
	case strings.Contains(c, "mp3"):
		return ".mp3"
	case strings.Contains(c, "wav"):
		return ".wav"
	case strings.Contains(c, "aiff"):
		return ".aiff"
	case strings.Contains(c, "asf"):
		return ".wma"
	case strings.Contains(c, "dsf"):
		return ".dsf"
	case strings.Contains(c, "dsdiff"):
		return ".dff"
	case strings.Contains(c, "mp4"), strings.Contains(c, "mov"):
		if strings.Contains(strings.ToLower(codec), "alac") {
			return ".m4a"
		}
		return ".m4a"
	case strings.Contains(c, "ape"):
		return ".ape"
	}
	return ".bin"
}
