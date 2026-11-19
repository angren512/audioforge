package media

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ffmpegPaths 读取集成测试所需的 ffmpeg / ffprobe，未配置则跳过。
func ffmpegPaths(t *testing.T) (string, string) {
	t.Helper()
	ff := os.Getenv("AUDIOFORGE_FFMPEG")
	fp := os.Getenv("AUDIOFORGE_FFPROBE")
	if ff == "" || fp == "" {
		t.Skip("未设置 AUDIOFORGE_FFMPEG / AUDIOFORGE_FFPROBE，跳过真实转码测试")
	}
	return ff, fp
}

// makeSample 用 ffmpeg 生成一个 3 秒 440Hz 正弦波源文件。
func makeSample(t *testing.T, ffmpeg, dir, name string, extra ...string) string {
	t.Helper()
	out := filepath.Join(dir, "src."+name)
	args := []string{"-hide_banner", "-nostdin", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=3"}
	args = append(args, extra...)
	args = append(args, out)
	if b, err := exec.Command(ffmpeg, args...).CombinedOutput(); err != nil {
		t.Fatalf("生成样本 %s 失败: %v %s", name, err, string(b))
	}
	return out
}

func TestConvertAllFormats(t *testing.T) {
	ffmpeg, ffprobe := ffmpegPaths(t)
	dir := t.TempDir()
	src := makeSample(t, ffmpeg, dir, "flac", "-c:a", "flac")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	expect := map[string]string{
		"mp3": "mp3", "flac": "flac", "wav": "pcm_s16le", "alac": "alac",
		"aac": "aac", "ogg": "vorbis", "opus": "opus", "aiff": "pcm_s16be", "wma": "wmav2",
	}

	for _, f := range Formats {
		f := f
		t.Run(f.ID, func(t *testing.T) {
			out := filepath.Join(dir, "out-"+f.ID+f.Ext)
			var progress []float64
			err := Convert(ctx, ffmpeg, src, out, Options{
				Format: f.ID, Bitrate: "320k", Compression: 5, BitDepth: 16,
			}, 3, false, func(p float64) { progress = append(progress, p) })
			if err != nil {
				t.Fatalf("转换失败: %v", err)
			}
			if fi, err := os.Stat(out); err != nil || fi.Size() == 0 {
				t.Fatalf("输出文件不存在或为空: %v", err)
			}
			info, err := Probe(ctx, ffprobe, out)
			if err != nil {
				t.Fatalf("探测失败: %v", err)
			}
			if want := expect[f.ID]; info.Codec != want {
				t.Errorf("codec = %q, 期望 %q", info.Codec, want)
			}
			if info.Duration < 2.5 || info.Duration > 3.5 {
				t.Errorf("时长异常: %.2fs", info.Duration)
			}
			if len(progress) == 0 || progress[len(progress)-1] != 100 {
				t.Errorf("进度回调异常: %v", progress)
			}
		})
	}
}

func TestConvertFromMultipleSources(t *testing.T) {
	ffmpeg, ffprobe := ffmpegPaths(t)
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	sources := []struct {
		ext  string
		args []string
	}{
		{"wav", []string{"-c:a", "pcm_s16le"}},
		{"ogg", []string{"-c:a", "libvorbis", "-q:a", "6"}},
		{"m4a", []string{"-c:a", "aac", "-b:a", "192k"}},
		{"opus", []string{"-c:a", "libopus", "-b:a", "128k"}},
	}

	for _, s := range sources {
		s := s
		t.Run(s.ext, func(t *testing.T) {
			src := makeSample(t, ffmpeg, dir, s.ext, s.args...)
			out := filepath.Join(dir, "to-mp3-"+s.ext+".mp3")
			if err := Convert(ctx, ffmpeg, src, out, Options{Format: "mp3", Bitrate: "320k"}, 3, false, nil); err != nil {
				t.Fatalf("转 MP3 失败: %v", err)
			}
			info, err := Probe(ctx, ffprobe, out)
			if err != nil {
				t.Fatalf("探测失败: %v", err)
			}
			if info.Codec != "mp3" {
				t.Errorf("codec = %q, 期望 mp3", info.Codec)
			}
			// libopus 只支持 48kHz 系采样率，源为 44100 时会被自动重采样
			wantRate := 44100
			if s.ext == "opus" {
				wantRate = 48000
			}
			if info.SampleRate != wantRate {
				t.Errorf("采样率 = %d, 期望 %d", info.SampleRate, wantRate)
			}
		})
	}
}

func TestConvertErrorOnBadInput(t *testing.T) {
	ffmpeg, _ := ffmpegPaths(t)
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.mp3")
	if err := os.WriteFile(bad, []byte(strings.Repeat("not audio", 100)), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Convert(context.Background(), ffmpeg, bad, filepath.Join(dir, "x.mp3"),
		Options{Format: "mp3"}, 0, false, nil)
	if err == nil {
		t.Fatal("损坏输入应返回错误")
	}
}

func TestProbeRejectsNonAudio(t *testing.T) {
	_, ffprobe := ffmpegPaths(t)
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.bin")
	if err := os.WriteFile(bad, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Probe(context.Background(), ffprobe, bad); err == nil {
		t.Fatal("非音频文件应探测失败")
	}
}

func TestExtFromCoverDetection(t *testing.T) {
	ffmpeg, ffprobe := ffmpegPaths(t)
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	src := makeSample(t, ffmpeg, dir, "flac", "-c:a", "flac")
	noCover, err := Probe(ctx, ffprobe, src)
	if err != nil {
		t.Fatal(err)
	}
	if noCover.HasCover {
		t.Error("纯音频不应被识别出封面")
	}
}
