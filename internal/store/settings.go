package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// Settings 是持久化的用户配置。
type Settings struct {
	OutputDir          string `json:"outputDir"`
	Format             string `json:"format"`
	Bitrate            string `json:"bitrate"`
	UseVBR             bool   `json:"useVbr"`
	Quality            int    `json:"quality"`
	SampleRate         int    `json:"sampleRate"`
	Channels           int    `json:"channels"`
	Compression        int    `json:"compression"`
	BitDepth           int    `json:"bitDepth"`
	Workers            int    `json:"workers"`
	AcceptedDisclaimer bool   `json:"acceptedDisclaimer"`
	KggDbPath          string `json:"kggDbPath"`
}

// Default 返回默认配置。
func Default() Settings {
	w := runtime.NumCPU() / 2
	if w < 2 {
		w = 2
	}
	if w > 8 {
		w = 8
	}
	return Settings{
		Format:      "mp3",
		Bitrate:     "320k",
		Quality:     2,
		Compression: 5,
		BitDepth:    16,
		Workers:     w,
	}
}

// Dir 返回应用数据目录（Windows 下为 %APPDATA%\AudioForge）。
func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return "."
	}
	return filepath.Join(base, "AudioForge")
}

// LogPath 返回日志文件路径。
func LogPath() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = Dir()
	}
	return filepath.Join(base, "AudioForge", "logs", "app.log")
}

func settingsPath() string {
	return filepath.Join(Dir(), "settings.json")
}

// Load 读取配置，文件不存在时返回默认值。
func Load() Settings {
	s := Default()
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

// Save 写入配置。
func Save(s Settings) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(), data, 0o644)
}
