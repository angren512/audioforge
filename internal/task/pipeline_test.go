package task

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"audioforge/internal/media"
)

// testFFmpeg 读取真实转码所需的二进制路径，未配置则跳过。
func testFFmpeg(t *testing.T) (string, string) {
	t.Helper()
	ff := os.Getenv("AUDIOFORGE_FFMPEG")
	fp := os.Getenv("AUDIOFORGE_FFPROBE")
	if ff == "" || fp == "" {
		t.Skip("未设置 AUDIOFORGE_FFMPEG / AUDIOFORGE_FFPROBE，跳过真实转码测试")
	}
	return ff, fp
}

// makeWav 用 ffmpeg 生成一个 2 秒的 wav 样本。
func makeWav(t *testing.T, ffmpeg, dir, name string) string {
	t.Helper()
	out := filepath.Join(dir, name)
	args := []string{"-hide_banner", "-nostdin", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=2",
		"-c:a", "pcm_s16le", out}
	if b, err := exec.Command(ffmpeg, args...).CombinedOutput(); err != nil {
		t.Fatalf("生成样本失败: %v %s", err, string(b))
	}
	return out
}

// recorder 记录任务状态变化并收集终态。
type recorder struct {
	mu     sync.Mutex
	states map[string][]State
	done   chan Task
}

func newRecorder() *recorder {
	return &recorder{states: make(map[string][]State), done: make(chan Task, 64)}
}

func (r *recorder) onUpdate(t Task) {
	r.mu.Lock()
	r.states[t.ID] = append(r.states[t.ID], t.State)
	r.mu.Unlock()
	if t.State.IsFinished() {
		select {
		case r.done <- t:
		default:
		}
	}
}

func (r *recorder) wait(t *testing.T, n int) []Task {
	t.Helper()
	out := make([]Task, 0, n)
	deadline := time.After(90 * time.Second)
	for len(out) < n {
		select {
		case tk := <-r.done:
			out = append(out, tk)
		case <-deadline:
			t.Fatalf("超时：只收到 %d/%d 个终态任务", len(out), n)
		}
	}
	return out
}

func (r *recorder) sequence(id string) []State {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]State(nil), r.states[id]...)
}

func newTestPool(t *testing.T, rec *recorder, workers int) *Pool {
	t.Helper()
	ffmpeg, ffprobe := testFFmpeg(t)
	dir := t.TempDir()
	return New(Config{
		FFmpeg:    ffmpeg,
		FFprobe:   ffprobe,
		TmpDir:    dir,
		OutputDir: dir,
		Media:     media.Options{Format: "mp3", Bitrate: "192k"},
		Workers:   workers,
		OnUpdate:  rec.onUpdate,
	})
}

// TestStartProcessesQueue 覆盖「先加文件再点开始」的主流程。
func TestStartProcessesQueue(t *testing.T) {
	ffmpeg, _ := testFFmpeg(t)
	rec := newRecorder()
	p := newTestPool(t, rec, 2)
	defer p.Close()

	src := makeWav(t, ffmpeg, t.TempDir(), "song.wav")
	added := p.Add([]string{src})
	if len(added) != 1 {
		t.Fatalf("加入任务数 = %d, 期望 1", len(added))
	}

	p.Start()
	got := rec.wait(t, 1)[0]
	if got.State != StateDone {
		t.Fatalf("任务终态 = %s (%s), 期望 done", got.State, got.Message)
	}
	if got.Output == "" {
		t.Fatal("完成任务没有输出路径")
	}
	if fi, err := os.Stat(got.Output); err != nil || fi.Size() == 0 {
		t.Fatalf("输出文件无效: %v", err)
	}

	seq := rec.sequence(got.ID)
	// 同一状态可能被重复推送（进度与状态共用一条事件），因此按子序列校验。
	want := []State{StateProbing, StateConverting, StateDone}
	i := 0
	for _, s := range seq {
		if i < len(want) && s == want[i] {
			i++
		}
	}
	if i != len(want) {
		t.Fatalf("状态序列 = %v, 期望依次包含 %v", seq, want)
	}

	if s := p.Stats(); s.Done != 1 || s.Running != 0 || s.Queued != 0 {
		t.Fatalf("统计异常: %+v", s)
	}
}

// TestStartBeforeAdd 覆盖「先点开始（队列为空）再拖入文件」的场景，
// 这是 UI 上最容易出现「点了没反应」的路径。
func TestStartBeforeAdd(t *testing.T) {
	ffmpeg, _ := testFFmpeg(t)
	rec := newRecorder()
	p := newTestPool(t, rec, 2)
	defer p.Close()

	p.Start() // 队列为空，worker 应挂起等待

	src := makeWav(t, ffmpeg, t.TempDir(), "late.wav")
	p.Add([]string{src})

	got := rec.wait(t, 1)[0]
	if got.State != StateDone {
		t.Fatalf("后加入的任务终态 = %s (%s), 期望 done", got.State, got.Message)
	}
	if _, err := os.Stat(got.Output); err != nil {
		t.Fatalf("输出文件缺失: %v", err)
	}
}

// TestBatchWithLimit 验证并发上限下批量任务全部完成。
func TestBatchWithLimit(t *testing.T) {
	ffmpeg, _ := testFFmpeg(t)
	rec := newRecorder()
	p := newTestPool(t, rec, 2)
	defer p.Close()

	srcDir := t.TempDir()
	var paths []string
	for _, n := range []string{"a.wav", "b.wav", "c.wav"} {
		paths = append(paths, makeWav(t, ffmpeg, srcDir, n))
	}
	p.Add(paths)
	p.Start()

	for _, tk := range rec.wait(t, len(paths)) {
		if tk.State != StateDone {
			t.Fatalf("%s 终态 = %s (%s)", tk.Name, tk.State, tk.Message)
		}
	}
	if s := p.Stats(); s.Done != 3 || s.Failed != 0 {
		t.Fatalf("统计异常: %+v", s)
	}
}

// TestCancelAllStopsQueue 验证取消全部后队列清空且任务被标记。
func TestCancelAllStopsQueue(t *testing.T) {
	rec := newRecorder()
	p := newTestPool(t, rec, 1)
	defer p.Close()

	p.Add([]string{"/tmp/never-a.wav", "/tmp/never-b.wav"})
	p.CancelAll()

	if s := p.Stats(); s.Queued != 0 || s.Canceled != 2 {
		t.Fatalf("取消后统计异常: %+v", s)
	}
}
