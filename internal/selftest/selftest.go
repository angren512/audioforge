package selftest

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"audioforge/internal/assets"
	"audioforge/internal/media"
)

type line struct {
	ok   bool
	text string
}

// Run 执行无界面自检，返回进程退出码。
func Run(w io.Writer) int {
	var lines []line
	add := func(ok bool, format string, args ...interface{}) {
		lines = append(lines, line{ok: ok, text: fmt.Sprintf(format, args...)})
	}

	ffmpeg, err := assets.FFmpeg()
	if err != nil {
		add(false, "ffmpeg 不可用: %v", err)
		return finish(w, lines)
	}
	ffprobe, err := assets.FFprobe()
	if err != nil {
		add(false, "ffprobe 不可用: %v", err)
		return finish(w, lines)
	}
	add(true, "ffmpeg: %s", ffmpeg)
	add(true, "ffprobe: %s", ffprobe)

	dir, err := os.MkdirTemp("", "audioforge-selftest-*")
	if err != nil {
		add(false, "创建临时目录失败: %v", err)
		return finish(w, lines)
	}
	defer os.RemoveAll(dir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 1. 生成不同来源格式的样本
	sources := []struct {
		name string
		args []string
	}{
		{"wav", []string{"-c:a", "pcm_s16le"}},
		{"flac", []string{"-c:a", "flac"}},
		{"ogg", []string{"-c:a", "libvorbis", "-q:a", "6"}},
		{"m4a", []string{"-c:a", "aac", "-b:a", "192k"}},
		{"opus", []string{"-c:a", "libopus", "-b:a", "128k"}},
	}

	var srcFlac string
	for _, s := range sources {
		out := filepath.Join(dir, "sample."+s.name)
		args := []string{"-hide_banner", "-nostdin", "-y",
			"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=3"}
		args = append(args, s.args...)
		args = append(args, out)
		cmd := exec.CommandContext(ctx, ffmpeg, args...)
		hideConsole(cmd)
		if outBytes, err := cmd.CombinedOutput(); err != nil {
			add(false, "生成样本 %s 失败: %v %s", s.name, err, tail(string(outBytes), 2))
			continue
		}
		add(true, "生成样本 %s", s.name)
		if s.name == "flac" {
			srcFlac = out
		}
	}

	if srcFlac == "" {
		return finish(w, lines)
	}

	// 2. 输入格式 → MP3
	for _, s := range sources {
		in := filepath.Join(dir, "sample."+s.name)
		if _, err := os.Stat(in); err != nil {
			continue
		}
		out := filepath.Join(dir, "from-"+s.name+".mp3")
		err := media.Convert(ctx, ffmpeg, in, out, media.Options{Format: "mp3", Bitrate: "320k"}, 3, false, nil)
		if err != nil {
			add(false, "%s → MP3 失败: %v", s.name, err)
			continue
		}
		info, err := media.Probe(ctx, ffprobe, out)
		if err != nil {
			add(false, "%s → MP3 校验失败: %v", s.name, err)
			continue
		}
		// Opus 源本身是 48kHz，转 MP3 会保留源采样率，因此这里只校验编码与时长
		ok := info.Codec == "mp3" && info.Channels == 1 && info.Duration > 2.5
		add(ok, "%s → MP3  codec=%s %dHz %dch %.2fs",
			s.name, info.Codec, info.SampleRate, info.Channels, info.Duration)
	}

	// 3. FLAC 样本 → 全部输出格式
	expected := map[string]string{
		"mp3": "mp3", "flac": "flac", "wav": "pcm_s16le", "alac": "alac",
		"aac": "aac", "ogg": "vorbis", "opus": "opus", "aiff": "pcm_s16be", "wma": "wmav2",
	}
	for _, f := range media.Formats {
		out := filepath.Join(dir, "out"+f.Ext)
		err := media.Convert(ctx, ffmpeg, srcFlac, out, media.Options{
			Format: f.ID, Bitrate: "320k", Compression: 5, BitDepth: 16,
		}, 3, false, nil)
		if err != nil {
			add(false, "FLAC → %s 失败: %v", f.Name, err)
			continue
		}
		info, err := media.Probe(ctx, ffprobe, out)
		if err != nil {
			add(false, "FLAC → %s 校验失败: %v", f.Name, err)
			continue
		}
		want := expected[f.ID]
		ok := info.Codec == want && info.Duration > 2.5 && info.Duration < 3.5
		add(ok, "FLAC → %-4s codec=%-10s %dHz %dch %.2fs",
			f.Name, info.Codec, info.SampleRate, info.Channels, info.Duration)
	}

	return finish(w, lines)
}

func finish(w io.Writer, lines []line) int {
	failed := 0
	var sb strings.Builder
	sb.WriteString("AudioForge 自检报告\n")
	sb.WriteString(time.Now().Format("2006-01-02 15:04:05"))
	sb.WriteString("\n\n")
	for _, l := range lines {
		mark := "[OK]  "
		if !l.ok {
			mark = "[FAIL]"
			failed++
		}
		sb.WriteString(mark + " " + l.text + "\n")
	}
	sb.WriteString(fmt.Sprintf("\n合计 %d 项，失败 %d 项\n", len(lines), failed))
	report := sb.String()

	fmt.Fprint(w, report)
	if path := reportPath(); path != "" {
		if err := os.WriteFile(path, []byte(report), 0o644); err == nil {
			fmt.Fprintf(w, "\n报告已写入: %s\n", path)
		}
	}
	if failed > 0 {
		return 1
	}
	return 0
}

func reportPath() string {
	// 优先写到 exe 同目录，方便用户直接找到；目录不可写时退回缓存目录
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if info, err := os.Stat(dir); err == nil && info.IsDir() && writable(dir) {
			return filepath.Join(dir, "selftest-report.txt")
		}
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	out := filepath.Join(base, "AudioForge")
	if err := os.MkdirAll(out, 0o755); err != nil {
		return ""
	}
	return filepath.Join(out, "selftest-report.txt")
}

func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".af-write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " | ")
}
