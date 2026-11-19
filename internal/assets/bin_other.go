//go:build !windows

package assets

import (
	"errors"
	"os"
)

// FFmpeg 返回 ffmpeg 可执行文件路径。
// 非 Windows 平台仅用于开发机跑测试，需通过环境变量显式指定。
func FFmpeg() (string, error) {
	return lookup("AUDIOFORGE_FFMPEG", "ffmpeg")
}

// FFprobe 返回 ffprobe 可执行文件路径。
func FFprobe() (string, error) {
	return lookup("AUDIOFORGE_FFPROBE", "ffprobe")
}

func lookup(env, name string) (string, error) {
	if p := os.Getenv(env); p != "" {
		return p, nil
	}
	return "", errors.New("当前平台未内置 " + name + "，请设置环境变量 " + env)
}
