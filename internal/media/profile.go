package media

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatSpec 描述一种可选输出格式。
type FormatSpec struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Ext      string `json:"ext"`
	Lossless bool   `json:"lossless"`
	Cover    bool   `json:"cover"`
	Hint     string `json:"hint"`
}

// Formats 是 UI 中可选的输出格式集合。
var Formats = []FormatSpec{
	{ID: "mp3", Name: "MP3", Ext: ".mp3", Lossless: false, Cover: true, Hint: "兼容性最好"},
	{ID: "flac", Name: "FLAC", Ext: ".flac", Lossless: true, Cover: true, Hint: "无损，体积约减半"},
	{ID: "wav", Name: "WAV", Ext: ".wav", Lossless: true, Cover: false, Hint: "无损未压缩，体积最大"},
	{ID: "alac", Name: "ALAC", Ext: ".m4a", Lossless: true, Cover: true, Hint: "苹果无损"},
	{ID: "aac", Name: "AAC", Ext: ".m4a", Lossless: false, Cover: true, Hint: "同码率优于 MP3"},
	{ID: "ogg", Name: "OGG", Ext: ".ogg", Lossless: false, Cover: true, Hint: "开源有损"},
	{ID: "opus", Name: "Opus", Ext: ".opus", Lossless: false, Cover: false, Hint: "低码率音质最佳"},
	{ID: "aiff", Name: "AIFF", Ext: ".aiff", Lossless: true, Cover: false, Hint: "苹果无损未压缩"},
	{ID: "wma", Name: "WMA", Ext: ".wma", Lossless: false, Cover: true, Hint: "Windows 生态"},
}

// FindFormat 按 id 查找输出格式。
func FindFormat(id string) (FormatSpec, bool) {
	for _, f := range Formats {
		if f.ID == id {
			return f, true
		}
	}
	return FormatSpec{}, false
}

// ValidFormat 判断输出格式是否受支持。
func ValidFormat(id string) bool {
	_, ok := FindFormat(id)
	return ok
}

// Options 描述一次转换的输出参数。
type Options struct {
	Format      string `json:"format"`
	Bitrate     string `json:"bitrate"`
	UseVBR      bool   `json:"useVbr"`
	Quality     int    `json:"quality"`
	SampleRate  int    `json:"sampleRate"`
	Channels    int    `json:"channels"`
	Compression int    `json:"compression"`
	BitDepth    int    `json:"bitDepth"`
}

// BuildArgs 生成完整的 ffmpeg 参数（不含可执行文件本身）。
func BuildArgs(in, out string, opt Options, hasCover bool) ([]string, error) {
	spec, ok := FindFormat(opt.Format)
	if !ok {
		return nil, fmt.Errorf("不支持的输出格式: %s", opt.Format)
	}

	args := []string{"-hide_banner", "-nostdin", "-y", "-i", in, "-map", "0:a:0"}
	if spec.Cover && hasCover {
		args = append(args, "-map", "0:v?", "-c:v", "copy", "-disposition:v", "attached_pic")
	} else {
		args = append(args, "-vn")
	}
	args = append(args, "-map_metadata", "0")

	codec, err := codecArgs(spec.ID, opt)
	if err != nil {
		return nil, err
	}
	args = append(args, codec...)

	if opt.SampleRate > 0 {
		args = append(args, "-ar", strconv.Itoa(opt.SampleRate))
	}
	if opt.Channels > 0 {
		args = append(args, "-ac", strconv.Itoa(opt.Channels))
	}
	if spec.ID == "mp3" {
		args = append(args, "-id3v2_version", "3", "-write_id3v1", "1")
	}

	args = append(args, "-progress", "pipe:1", "-nostats", out)
	return args, nil
}

func codecArgs(id string, opt Options) ([]string, error) {
	switch id {
	case "mp3":
		if opt.UseVBR {
			q := opt.Quality
			if q < 0 || q > 9 {
				q = 2
			}
			return []string{"-c:a", "libmp3lame", "-q:a", strconv.Itoa(q)}, nil
		}
		return []string{"-c:a", "libmp3lame", "-b:a", bitrateOr(opt, "320k")}, nil
	case "flac":
		c := opt.Compression
		if c < 0 || c > 12 {
			c = 5
		}
		return []string{"-c:a", "flac", "-compression_level", strconv.Itoa(c)}, nil
	case "wav":
		if opt.BitDepth == 24 {
			return []string{"-c:a", "pcm_s24le"}, nil
		}
		return []string{"-c:a", "pcm_s16le"}, nil
	case "alac":
		return []string{"-c:a", "alac"}, nil
	case "aac":
		return []string{"-c:a", "aac", "-b:a", bitrateOr(opt, "256k")}, nil
	case "ogg":
		q := opt.Quality
		if q < 0 || q > 10 {
			q = 6
		}
		return []string{"-c:a", "libvorbis", "-q:a", strconv.Itoa(q)}, nil
	case "opus":
		return []string{"-c:a", "libopus", "-b:a", clampBitrate(opt, "160k", 256)}, nil
	case "aiff":
		if opt.BitDepth == 24 {
			return []string{"-c:a", "pcm_s24be"}, nil
		}
		return []string{"-c:a", "pcm_s16be"}, nil
	case "wma":
		return []string{"-c:a", "wmav2", "-b:a", clampBitrate(opt, "192k", 192)}, nil
	}
	return nil, fmt.Errorf("不支持的输出格式: %s", id)
}

// clampBitrate 把码率限制在编码器支持的范围内，超出时回退到上限。
// 例如 libopus 最高 256kbps、wmav2 最高 192kbps，直接传 320k 会让 ffmpeg 报错。
func clampBitrate(opt Options, def string, maxKbps int) string {
	if opt.Bitrate == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSuffix(strings.ToLower(opt.Bitrate), "k"))
	if err != nil || n <= 0 {
		return def
	}
	if n > maxKbps {
		return strconv.Itoa(maxKbps) + "k"
	}
	return opt.Bitrate
}

func bitrateOr(opt Options, def string) string {
	if opt.Bitrate == "" {
		return def
	}
	return opt.Bitrate
}
