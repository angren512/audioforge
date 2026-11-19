package decrypt

import (
	"sort"
	"strings"
)

// 各音乐平台的加密文件后缀（按长度降序匹配，避免 .kgm.flac 与 .flac 混淆）
var encryptedSuffixes = []string{
	".kgm.flac", ".vpr.flac",
	".qmcflac", ".qmcogg",
	".mflac0", ".mflac1", ".mflac2", ".mflaca", ".mflach", ".mflacl", ".mflacm", ".mflac",
	".mgg0", ".mgg1", ".mgg2", ".mgga", ".mggh", ".mggl", ".mggm", ".mgg",
	".mmp4",
	".bkcmp3", ".bkcm4a", ".bkcflac", ".bkcwav", ".bkcape", ".bkcogg", ".bkcwma",
	".666c6163", ".6d7033", ".6f6767", ".6d3461", ".776176",
	".qmc0", ".qmc2", ".qmc3", ".qmc4", ".qmc6", ".qmc8", ".qmcflac2",
	".ncm",
	".kgm", ".kgma", ".kgg", ".vpr",
	".kwm",
	".tm0", ".tm2", ".tm3", ".tm6",
	".x2m", ".x3m", ".xm",
	".tkm",
	".mdl",
}

var sortedSuffixes []string

func init() {
	sortedSuffixes = append(sortedSuffixes, encryptedSuffixes...)
	sort.Slice(sortedSuffixes, func(i, j int) bool {
		return len(sortedSuffixes[i]) > len(sortedSuffixes[j])
	})
}

// Suffix 返回文件名命中的加密后缀，未命中返回空串。
func Suffix(name string) string {
	lower := strings.ToLower(name)
	for _, s := range sortedSuffixes {
		if strings.HasSuffix(lower, s) {
			return s
		}
	}
	return ""
}

// IsEncrypted 判断文件是否为需要解密的加密音频。
func IsEncrypted(name string) bool {
	return Suffix(name) != ""
}

// IsSoda 判断是否为汽水音乐文件（当前无法离线解密）。
func IsSoda(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".mdl")
}

// BaseName 去掉加密后缀，得到干净的歌曲名。
func BaseName(name string) string {
	if s := Suffix(name); s != "" {
		return name[:len(name)-len(s)]
	}
	return name
}

// SupportedExts 返回全部加密后缀，供前端展示。
func SupportedExts() []string {
	return append([]string(nil), encryptedSuffixes...)
}
