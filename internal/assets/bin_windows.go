//go:build windows

package assets

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// 两个静态二进制直接打进 exe，保证双击即用、无需任何外部依赖。
//
//go:embed bin/ffmpeg.exe bin/ffprobe.exe
var binFS embed.FS

var (
	prepOnce sync.Once
	prepErr  error
	binDir   string
)

// FFmpeg 返回内置 ffmpeg.exe 的本地路径（首次调用时释放到缓存目录）。
func FFmpeg() (string, error) {
	return ensure("ffmpeg.exe")
}

// FFprobe 返回内置 ffprobe.exe 的本地路径（首次调用时释放到缓存目录）。
func FFprobe() (string, error) {
	return ensure("ffprobe.exe")
}

func ensure(name string) (string, error) {
	prepOnce.Do(prepare)
	if prepErr != nil {
		return "", prepErr
	}
	p := filepath.Join(binDir, name)
	if _, err := os.Stat(p); err != nil {
		return "", errors.New("内置 " + name + " 释放失败: " + err.Error())
	}
	return p, nil
}

func prepare() {
	base, err := os.UserCacheDir()
	if err != nil {
		prepErr = err
		return
	}
	binDir = filepath.Join(base, "AudioForge", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		prepErr = err
		return
	}
	for _, name := range []string{"ffmpeg.exe", "ffprobe.exe"} {
		if err := extract(name); err != nil {
			prepErr = err
			return
		}
	}
}

func extract(name string) error {
	target := filepath.Join(binDir, name)
	st, err := fs.Stat(binFS, "bin/"+name)
	if err != nil {
		return err
	}
	// 大小一致即认为已释放过，避免每次启动重复写 90MB
	if fi, err := os.Stat(target); err == nil && fi.Size() == st.Size() {
		return nil
	}
	src, err := binFS.Open("bin/" + name)
	if err != nil {
		return err
	}
	defer src.Close()

	tmp := target + ".tmp"
	dst, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(tmp)
		return err
	}
	if err := dst.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, target)
}
