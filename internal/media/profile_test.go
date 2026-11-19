package media

import (
	"strings"
	"testing"
)

func TestFormatsUnique(t *testing.T) {
	ids := map[string]bool{}
	// ALAC 与 AAC 都是 MP4 容器，共用 .m4a 属于预期
	shareable := map[string]bool{".m4a": true}
	exts := map[string]bool{}
	for _, f := range Formats {
		if ids[f.ID] {
			t.Errorf("格式 id 重复: %s", f.ID)
		}
		ids[f.ID] = true
		if exts[f.Ext] && !shareable[f.Ext] {
			t.Errorf("输出扩展名重复: %s", f.Ext)
		}
		exts[f.Ext] = true
		if !strings.HasPrefix(f.Ext, ".") {
			t.Errorf("%s 的扩展名缺少点号: %s", f.ID, f.Ext)
		}
	}
	if len(Formats) != 9 {
		t.Errorf("期望 9 种输出格式，实际 %d", len(Formats))
	}
}

func TestValidFormat(t *testing.T) {
	for _, id := range []string{"mp3", "flac", "wav", "alac", "aac", "ogg", "opus", "aiff", "wma"} {
		if !ValidFormat(id) {
			t.Errorf("%s 应为合法格式", id)
		}
	}
	for _, id := range []string{"", "MP3", "mp4", "xxx"} {
		if ValidFormat(id) {
			t.Errorf("%s 不应为合法格式", id)
		}
	}
}

func TestBuildArgsContainsCodec(t *testing.T) {
	cases := []struct {
		opt  Options
		want []string
	}{
		{Options{Format: "mp3", Bitrate: "320k"}, []string{"-c:a", "libmp3lame", "-b:a", "320k", "-id3v2_version", "3"}},
		{Options{Format: "mp3", UseVBR: true, Quality: 0}, []string{"-c:a", "libmp3lame", "-q:a", "0"}},
		{Options{Format: "flac", Compression: 8}, []string{"-c:a", "flac", "-compression_level", "8"}},
		{Options{Format: "wav", BitDepth: 24}, []string{"-c:a", "pcm_s24le"}},
		{Options{Format: "wav", BitDepth: 16}, []string{"-c:a", "pcm_s16le"}},
		{Options{Format: "alac"}, []string{"-c:a", "alac"}},
		{Options{Format: "aac", Bitrate: "256k"}, []string{"-c:a", "aac", "-b:a", "256k"}},
		{Options{Format: "ogg", Quality: 6}, []string{"-c:a", "libvorbis", "-q:a", "6"}},
		{Options{Format: "opus", Bitrate: "160k"}, []string{"-c:a", "libopus", "-b:a", "160k"}},
		{Options{Format: "aiff", BitDepth: 24}, []string{"-c:a", "pcm_s24be"}},
		{Options{Format: "wma", Bitrate: "192k"}, []string{"-c:a", "wmav2", "-b:a", "192k"}},
	}

	for _, c := range cases {
		args, err := BuildArgs("in.flac", "out.bin", c.opt, false)
		if err != nil {
			t.Fatalf("%s: %v", c.opt.Format, err)
		}
		joined := strings.Join(args, " ")
		for i := 0; i+1 < len(c.want); i += 2 {
			kv := c.want[i] + " " + c.want[i+1]
			if !strings.Contains(joined, kv) {
				t.Errorf("%s 参数缺少 %q，实际: %s", c.opt.Format, kv, joined)
			}
		}
		if args[len(args)-1] != "out.bin" {
			t.Errorf("%s 输出路径不在末尾: %v", c.opt.Format, args)
		}
		if !strings.Contains(joined, "-progress pipe:1") {
			t.Errorf("%s 缺少进度管道参数", c.opt.Format)
		}
	}
}

func TestBuildArgsCover(t *testing.T) {
	withCover, err := BuildArgs("in.flac", "out.mp3", Options{Format: "mp3"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(withCover, " "), "-c:v copy") {
		t.Errorf("有封面时应复制封面: %v", withCover)
	}

	noCover, err := BuildArgs("in.flac", "out.mp3", Options{Format: "mp3"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(noCover, " "), "-vn") {
		t.Errorf("无封面时应禁用视频流: %v", noCover)
	}

	// WAV 不支持封面，即便源有封面也要加 -vn
	wav, err := BuildArgs("in.flac", "out.wav", Options{Format: "wav"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(wav, " "), "-vn") {
		t.Errorf("WAV 不应携带封面: %v", wav)
	}
}

func TestBuildArgsInvalid(t *testing.T) {
	if _, err := BuildArgs("a", "b", Options{Format: "mp4"}, false); err == nil {
		t.Fatal("非法格式应返回错误")
	}
}

func TestBuildArgsQualityClamp(t *testing.T) {
	args, err := BuildArgs("in.flac", "out.mp3", Options{Format: "mp3", UseVBR: true, Quality: 42}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(args, " "), "-q:a 2") {
		t.Errorf("超范围的质量参数应回退到 2: %v", args)
	}
}

func TestBuildArgsBitrateClamp(t *testing.T) {
	opus, err := BuildArgs("in.flac", "out.opus", Options{Format: "opus", Bitrate: "320k"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(opus, " "), "-b:a 256k") {
		t.Errorf("Opus 码率应被限制到 256k: %v", opus)
	}

	wma, err := BuildArgs("in.flac", "out.wma", Options{Format: "wma", Bitrate: "320k"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(wma, " "), "-b:a 192k") {
		t.Errorf("WMA 码率应被限制到 192k: %v", wma)
	}

	// 未超出上限时保持用户选择
	fine, err := BuildArgs("in.flac", "out.opus", Options{Format: "opus", Bitrate: "128k"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(fine, " "), "-b:a 128k") {
		t.Errorf("范围内码率不应被改动: %v", fine)
	}
}
