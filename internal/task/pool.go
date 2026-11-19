package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"audioforge/internal/decrypt"
	"audioforge/internal/media"

	"go.uber.org/zap"
)

// Config 是一次批量转换的完整配置。
type Config struct {
	FFmpeg    string
	FFprobe   string
	TmpDir    string
	OutputDir string
	Media     media.Options
	Workers   int
	Decrypt   decrypt.Params
	Logger    *zap.Logger
	OnUpdate  func(Task)
	OnStats   func(Stats)
}

// Pool 是无长度上限的任务队列。
type Pool struct {
	cfg Config

	mu      sync.Mutex
	cond    *sync.Cond
	rootCtx context.Context
	rootCxl context.CancelFunc

	queue   []*Task
	all     []*Task
	cancels map[string]context.CancelFunc
	running int
	spawned int
	closed  bool
	seq     int
}

// New 创建任务池。
func New(cfg Config) *Pool {
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if cfg.Workers > 32 {
		cfg.Workers = 32
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	ctx, cxl := context.WithCancel(context.Background())
	p := &Pool{
		cfg:     cfg,
		rootCtx: ctx,
		rootCxl: cxl,
		cancels: make(map[string]context.CancelFunc),
	}
	p.cond = sync.NewCond(&p.mu)
	return p
}

// Update 更新运行参数；并发数变化会在下次 Start 时生效。
func (p *Pool) Update(cfg Config) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cfg.FFmpeg = cfg.FFmpeg
	p.cfg.FFprobe = cfg.FFprobe
	p.cfg.OutputDir = cfg.OutputDir
	p.cfg.Media = cfg.Media
	p.cfg.Decrypt = cfg.Decrypt
	if cfg.Workers > 0 {
		if cfg.Workers > 32 {
			cfg.Workers = 32
		}
		p.cfg.Workers = cfg.Workers
	}
}

// Add 追加文件，返回新建的任务快照。
func (p *Pool) Add(paths []string) []Task {
	p.mu.Lock()

	added := make([]Task, 0, len(paths))
	for _, path := range paths {
		p.seq++
		t := &Task{
			ID:        "t" + strconv.Itoa(p.seq),
			Src:       path,
			Name:      filepath.Base(path),
			Ext:       strings.ToLower(filepath.Ext(path)),
			Encrypted: decrypt.IsEncrypted(path),
			State:     StateQueued,
		}
		p.all = append(p.all, t)
		p.queue = append(p.queue, t)
		added = append(added, *t)
	}
	p.cond.Broadcast()
	stats := p.statsLocked()
	p.mu.Unlock()

	p.emitStats(stats)
	return added
}

// Start 启动 worker（按当前并发数补齐），开始消费队列。
func (p *Pool) Start() {
	p.mu.Lock()
	for p.spawned < p.cfg.Workers {
		p.spawned++
		go p.worker()
	}
	p.mu.Unlock()
	p.cond.Broadcast()
}

// Snapshot 返回全部任务快照。
func (p *Pool) Snapshot() []Task {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Task, len(p.all))
	for i, t := range p.all {
		out[i] = *t
	}
	return out
}

// Stats 返回当前统计。
func (p *Pool) Stats() Stats {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.statsLocked()
}

// CancelTask 取消单个任务。
func (p *Pool) CancelTask(id string) {
	p.mu.Lock()
	var queued *Task
	for i, t := range p.queue {
		if t.ID == id {
			queued = t
			p.queue = append(p.queue[:i], p.queue[i+1:]...)
			break
		}
	}
	cancel := p.cancels[id]
	p.mu.Unlock()

	if queued != nil {
		p.mutate(id, func(t *Task) {
			t.State = StateCanceled
			t.Message = "已取消"
		})
	}
	if cancel != nil {
		cancel()
	}
}

// CancelAll 取消队列中与正在执行的全部任务。
func (p *Pool) CancelAll() {
	p.mu.Lock()
	queued := p.queue
	p.queue = nil
	cancels := make([]context.CancelFunc, 0, len(p.cancels))
	for _, c := range p.cancels {
		cancels = append(cancels, c)
	}
	p.mu.Unlock()

	for _, t := range queued {
		p.mutate(t.ID, func(t *Task) {
			t.State = StateCanceled
			t.Message = "已取消"
		})
	}
	for _, c := range cancels {
		c()
	}
}

// ClearFinished 移除已结束的任务。
func (p *Pool) ClearFinished() {
	p.mu.Lock()
	kept := p.all[:0]
	for _, t := range p.all {
		if !t.State.IsFinished() {
			kept = append(kept, t)
		}
	}
	p.all = kept
	stats := p.statsLocked()
	p.mu.Unlock()
	p.emitStats(stats)
}

func (p *Pool) worker() {
	for {
		t := p.pop()
		if t == nil {
			return
		}
		p.process(t)
		p.mu.Lock()
		p.running--
		stats := p.statsLocked()
		p.mu.Unlock()
		p.emitStats(stats)
	}
}

func (p *Pool) pop() *Task {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.queue) == 0 && !p.closed {
		p.cond.Wait()
	}
	if p.closed {
		return nil
	}
	t := p.queue[0]
	p.queue = p.queue[1:]
	p.running++
	return t
}

func (p *Pool) process(t *Task) {
	// 快照配置，避免 Update 并发写入导致数据竞争
	p.mu.Lock()
	cfg := p.cfg
	p.mu.Unlock()

	ctx, cancel := context.WithCancel(p.rootCtx)
	p.mu.Lock()
	p.cancels[t.ID] = cancel
	p.mu.Unlock()
	defer func() {
		cancel()
		p.mu.Lock()
		delete(p.cancels, t.ID)
		p.mu.Unlock()
	}()

	src := t.Src
	if t.Encrypted {
		p.mutate(t.ID, func(t *Task) {
			t.State = StateDecrypting
			t.Message = "正在解密"
		})
		tmp, err := decrypt.Decode(ctx, t.Src, cfg.TmpDir, cfg.Decrypt, cfg.Logger)
		if err != nil {
			p.finish(t.ID, StateFailed, err.Error())
			return
		}
		defer os.Remove(tmp)
		src = tmp
	}

	if ctx.Err() != nil {
		p.finish(t.ID, StateCanceled, "已取消")
		return
	}

	p.mutate(t.ID, func(t *Task) {
		t.State = StateProbing
		t.Message = "正在分析"
	})
	info, err := media.Probe(ctx, cfg.FFprobe, src)
	if err != nil {
		p.finish(t.ID, StateFailed, err.Error())
		return
	}
	p.mutate(t.ID, func(t *Task) {
		t.Info = info
	})

	spec, ok := media.FindFormat(cfg.Media.Format)
	if !ok {
		p.finish(t.ID, StateFailed, "输出格式无效: "+cfg.Media.Format)
		return
	}

	base := t.Name
	if t.Encrypted {
		base = decrypt.BaseName(t.Name)
	} else {
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}

	outDir := cfg.OutputDir
	if outDir == "" {
		outDir = filepath.Dir(t.Src)
	}
	outPath := uniquePath(filepath.Join(outDir, base+spec.Ext))

	p.mutate(t.ID, func(t *Task) {
		t.State = StateConverting
		t.Message = "正在转换"
		t.Output = outPath
	})

	last := time.Now()
	err = media.Convert(ctx, cfg.FFmpeg, src, outPath, cfg.Media, info.Duration, info.HasCover, func(pr float64) {
		if time.Since(last) < 120*time.Millisecond && pr < 100 {
			return
		}
		last = time.Now()
		p.mutate(t.ID, func(t *Task) {
			t.Progress = pr
		})
	})
	if err != nil {
		os.Remove(outPath)
		if errors.Is(err, context.Canceled) {
			p.finish(t.ID, StateCanceled, "已取消")
			return
		}
		p.finish(t.ID, StateFailed, err.Error())
		return
	}

	p.finish(t.ID, StateDone, "")
}

func (p *Pool) mutate(id string, fn func(*Task)) {
	p.mu.Lock()
	var snap Task
	found := false
	for _, t := range p.all {
		if t.ID == id {
			fn(t)
			snap = *t
			found = true
			break
		}
	}
	p.mu.Unlock()
	if found && p.cfg.OnUpdate != nil {
		p.cfg.OnUpdate(snap)
	}
}

func (p *Pool) finish(id string, state State, msg string) {
	p.mutate(id, func(t *Task) {
		t.State = state
		t.Message = msg
		if state == StateDone {
			t.Progress = 100
		}
	})
}

func (p *Pool) statsLocked() Stats {
	s := Stats{Total: len(p.all), Running: p.running, Queued: len(p.queue)}
	for _, t := range p.all {
		switch t.State {
		case StateDone:
			s.Done++
		case StateFailed:
			s.Failed++
		case StateCanceled:
			s.Canceled++
		case StateSkipped:
			s.Skipped++
		}
	}
	return s
}

func (p *Pool) emitStats(s Stats) {
	if p.cfg.OnStats != nil {
		p.cfg.OnStats(s)
	}
}

func uniquePath(p string) string {
	if _, err := os.Stat(p); err != nil {
		return p
	}
	ext := filepath.Ext(p)
	stem := strings.TrimSuffix(p, ext)
	for i := 1; i < 10000; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", stem, i, ext)
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
	return p
}

// Close 释放资源。
func (p *Pool) Close() {
	p.CancelAll()
	p.rootCxl()
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	p.cond.Broadcast()
}
