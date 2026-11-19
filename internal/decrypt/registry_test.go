package decrypt

import "testing"

func TestSuffix(t *testing.T) {
	cases := map[string]string{
		"a.qmc0":     ".qmc0",
		"a.QMC3":     ".qmc3",
		"A.MFLAC":    ".mflac",
		"a.mgg":      ".mgg",
		"a.mgg1":     ".mgg1",
		"a.kgm":      ".kgm",
		"a.kgm.flac": ".kgm.flac",
		"a.kgg":      ".kgg",
		"a.kwm":      ".kwm",
		"a.tm0":      ".tm0",
		"a.ncm":      ".ncm",
		"a.xm":       ".xm",
		"a.x2m":      ".x2m",
		"a.mdl":      ".mdl",
		"普通歌曲.mp3":   "",
		"a.flac":     "",
		"a.mp3":      "",
		"a.txt":      "",
		"":           "",
	}
	for name, want := range cases {
		if got := Suffix(name); got != want {
			t.Errorf("Suffix(%q) = %q, 期望 %q", name, got, want)
		}
	}
}

func TestIsEncrypted(t *testing.T) {
	for _, n := range []string{"x.ncm", "y.mgg", "z.kgm.flac", "w.mdl"} {
		if !IsEncrypted(n) {
			t.Errorf("%s 应判定为加密文件", n)
		}
	}
	for _, n := range []string{"x.mp3", "y.flac", "z.wav"} {
		if IsEncrypted(n) {
			t.Errorf("%s 不应判定为加密文件", n)
		}
	}
}

func TestBaseName(t *testing.T) {
	cases := map[string]string{
		"周杰伦 - 晴天.mflac": "周杰伦 - 晴天",
		"song.kgm.flac":  "song",
		"plain.mp3":      "plain.mp3",
	}
	for in, want := range cases {
		if got := BaseName(in); got != want {
			t.Errorf("BaseName(%q) = %q, 期望 %q", in, got, want)
		}
	}
}

func TestIsSoda(t *testing.T) {
	if !IsSoda("a.mdl") || !IsSoda("A.MDL") {
		t.Error(".mdl 应判定为汽水音乐格式")
	}
	if IsSoda("a.mp3") {
		t.Error(".mp3 不是汽水音乐格式")
	}
}

func TestSupportedExts(t *testing.T) {
	exts := SupportedExts()
	if len(exts) == 0 {
		t.Fatal("加密后缀列表为空")
	}
	has := map[string]bool{}
	for _, e := range exts {
		has[e] = true
	}
	for _, want := range []string{".ncm", ".mgg", ".kgm", ".kwm", ".tm0", ".qmc0", ".mdl"} {
		if !has[want] {
			t.Errorf("缺少后缀 %s", want)
		}
	}
}
