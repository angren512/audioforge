package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"audioforge/internal/assets"
	"audioforge/internal/decrypt"
	"audioforge/internal/media"
	"audioforge/internal/store"
	"audioforge/internal/task"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 普通音频扩展名，用于筛选拖入的文件
var plainExts = map[string]bool{
	".mp3": true, ".flac": true, ".wav": true, ".ape": true, ".m4a": true,
	".aac": true, ".ogg": true, ".opus": true, ".wma": true, ".aiff": true,
	".aif": true, ".alac": true, ".dsf": true, ".dff": true, ".ac3": true,
	".amr": true, ".mp2": true, ".mka": true, ".tak": true, ".wv": true, ".tta": true,
}

// App 是绑定到前端的应用对象。
type App struct {
	ctx      context.Context
	mu       sync.Mutex
	settings store.Settings
	pool     *task.Pool
	logger   *zap.Logger
}

// Bootstrap 是前端启动时需要的全部数据。
type Bootstrap struct {
	Settings      store.Settings     `json:"settings"`
	Formats       []media.FormatSpec `json:"formats"`
	EncryptedExts []string           `json:"encryptedExts"`
	PlainExts     []string           `json:"plainExts"`
	Version       string             `json:"version"`
	LogPath       string             `json:"logPath"`
}

// StartOptions 是前端点击「开始」时提交的参数。
type StartOptions struct {
	OutputDir   string `json:"outputDir"`
	Format      string `json:"format"`
	Bitrate     string `json:"bitrate"`
	UseVBR      bool   `json:"useVbr"`
	Quality     int    `json:"quality"`
	SampleRate  int    `json:"sampleRate"`
	Channels    int    `json:"channels"`
	Compression int    `json:"compression"`
	BitDepth    int    `json:"bitDepth"`
	Workers     int    `json:"workers"`
	KggDbPath   string `json:"kggDbPath"`
}

// NewApp 创建应用实例。
func NewApp() *App {
	return &App{settings: store.Load(), logger: zap.NewNop()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.logger = newLogger()
	a.logger.Info("AudioForge 启动", zap.String("version", Version))
}

func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	pool := a.pool
	a.mu.Unlock()
	if pool != nil {
		pool.Close()
	}
	a.logger.Info("AudioForge 退出")
}

func newLogger() *zap.Logger {
	path := store.LogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return zap.NewNop()
	}
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{path}
	cfg.Encoding = "console"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	lg, err := cfg.Build()
	if err != nil {
		return zap.NewNop()
	}
	return lg
}

// GetBootstrap 返回前端初始化数据。
func (a *App) GetBootstrap() Bootstrap {
	a.mu.Lock()
	s := a.settings
	a.mu.Unlock()

	exts := decrypt.SupportedExts()
	plain := make([]string, 0, len(plainExts))
	for e := range plainExts {
		plain = append(plain, e)
	}
	return Bootstrap{
		Settings:      s,
		Formats:       media.Formats,
		EncryptedExts: exts,
		PlainExts:     plain,
		Version:       Version,
		LogPath:       store.LogPath(),
	}
}

// SaveSettings 持久化用户配置。
func (a *App) SaveSettings(s store.Settings) error {
	a.mu.Lock()
	a.settings = s
	a.mu.Unlock()
	return store.Save(s)
}

// PickFiles 打开文件选择框。
func (a *App) PickFiles() []string {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择音频文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "音频文件", Pattern: "*.mp3;*.flac;*.wav;*.ape;*.m4a;*.aac;*.ogg;*.opus;*.wma;*.aiff;*.dsf;*.dff;*.mgg;*.mflac;*.qmc0;*.qmc3;*.qmcflac;*.qmcogg;*.kgm;*.kgma;*.kgg;*.vpr;*.kwm;*.tm0;*.tm2;*.tm3;*.ncm;*.xm;*.x2m;*.x3m"},
			{DisplayName: "所有文件", Pattern: "*.*"},
		},
	})
	if err != nil {
		a.logger.Error("打开文件选择框失败", zap.Error(err))
		return nil
	}
	return files
}

// PickDir 打开目录选择框。
func (a *App) PickDir() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "选择输出目录"})
	if err != nil {
		return ""
	}
	return dir
}

// AddFiles 把文件或目录加入队列，目录会递归展开。
func (a *App) AddFiles(paths []string) []task.Task {
	files := expand(paths)
	if len(files) == 0 {
		return nil
	}
	a.mu.Lock()
	s := a.settings
	a.mu.Unlock()

	pool := a.ensurePool(s)
	return pool.Add(files)
}

// Start 按当前参数开始转换。
func (a *App) Start(opt StartOptions) error {
	a.mu.Lock()
	s := a.settings
	a.mu.Unlock()

	if !media.ValidFormat(opt.Format) {
		return fmt.Errorf("输出格式无效: %s", opt.Format)
	}

	ffmpeg, err := assets.FFmpeg()
	if err != nil {
		return fmt.Errorf("ffmpeg 不可用: %w", err)
	}
	ffprobe, err := assets.FFprobe()
	if err != nil {
		return fmt.Errorf("ffprobe 不可用: %w", err)
	}

	pool := a.ensurePool(s)
	pool.Update(task.Config{
		FFmpeg:    ffmpeg,
		FFprobe:   ffprobe,
		OutputDir: opt.OutputDir,
		Media: media.Options{
			Format:      opt.Format,
			Bitrate:     opt.Bitrate,
			UseVBR:      opt.UseVBR,
			Quality:     opt.Quality,
			SampleRate:  opt.SampleRate,
			Channels:    opt.Channels,
			Compression: opt.Compression,
			BitDepth:    opt.BitDepth,
		},
		Decrypt: decrypt.Params{KggDb: opt.KggDbPath},
		Workers: opt.Workers,
	})
	pool.Start()
	return nil
}

// CancelTask 取消单个任务。
func (a *App) CancelTask(id string) {
	a.mu.Lock()
	pool := a.pool
	a.mu.Unlock()
	if pool != nil {
		pool.CancelTask(id)
	}
}

// CancelAll 取消全部任务。
func (a *App) CancelAll() {
	a.mu.Lock()
	pool := a.pool
	a.mu.Unlock()
	if pool != nil {
		pool.CancelAll()
	}
}

// ClearFinished 清空已完成的任务。
func (a *App) ClearFinished() {
	a.mu.Lock()
	pool := a.pool
	a.mu.Unlock()
	if pool != nil {
		pool.ClearFinished()
	}
}

// Snapshot 返回当前任务列表。
func (a *App) Snapshot() []task.Task {
	a.mu.Lock()
	pool := a.pool
	a.mu.Unlock()
	if pool == nil {
		return nil
	}
	return pool.Snapshot()
}

// RevealInExplorer 在资源管理器中定位文件。
func (a *App) RevealInExplorer(path string) {
	if path == "" {
		return
	}
	revealInExplorer(path)
}

// OpenOutputDir 打开输出目录。
func (a *App) OpenOutputDir(dir string) {
	if dir == "" {
		return
	}
	runtime.BrowserOpenURL(a.ctx, dir)
}

// OpenLog 打开日志文件所在目录。
func (a *App) OpenLog() {
	runtime.BrowserOpenURL(a.ctx, filepath.Dir(store.LogPath()))
}

func (a *App) ensurePool(s store.Settings) *task.Pool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pool != nil {
		return a.pool
	}
	ffmpeg, _ := assets.FFmpeg()
	ffprobe, _ := assets.FFprobe()
	a.pool = task.New(task.Config{
		FFmpeg:    ffmpeg,
		FFprobe:   ffprobe,
		TmpDir:    os.TempDir(),
		OutputDir: s.OutputDir,
		Media: media.Options{
			Format:      s.Format,
			Bitrate:     s.Bitrate,
			UseVBR:      s.UseVBR,
			Quality:     s.Quality,
			SampleRate:  s.SampleRate,
			Channels:    s.Channels,
			Compression: s.Compression,
			BitDepth:    s.BitDepth,
		},
		Workers:  s.Workers,
		Decrypt:  decrypt.Params{KggDb: s.KggDbPath},
		Logger:   a.logger,
		OnUpdate: a.onTaskUpdate,
		OnStats:  a.onStats,
	})
	return a.pool
}

func (a *App) onTaskUpdate(t task.Task) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "task:update", t)
	}
}

func (a *App) onStats(s task.Stats) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "stats", s)
	}
}

// expand 展开目录并过滤出可处理的音频文件。
func expand(paths []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			_ = filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if accept(path) && !seen[path] {
					seen[path] = true
					out = append(out, path)
				}
				return nil
			})
			continue
		}
		if accept(p) && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

func accept(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return decrypt.IsEncrypted(path) || plainExts[ext]
}

// onFileDrop 处理拖拽进来的文件与目录。
func (a *App) onFileDrop(_ int, _ int, paths []string) {
	a.AddFiles(paths)
}
