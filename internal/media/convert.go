package media

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// Convert 调用 ffmpeg 完成一次转码，onProgress 接收 0-100 的百分比（可能为 nil）。
func Convert(ctx context.Context, ffmpegPath, in, out string, opt Options, duration float64, hasCover bool, onProgress func(float64)) error {
	args, err := BuildArgs(in, out, opt, hasCover)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	hideWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr := &tailBuffer{limit: 8192}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 ffmpeg 失败: %w", err)
	}

	// 无论是否关心进度都必须把 stdout 读干净，否则管道写满会让 ffmpeg 卡死。
	go readProgress(stdout, duration, onProgress)

	err = cmd.Wait()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return fmt.Errorf("ffmpeg 执行失败: %w", err)
		}
		return fmt.Errorf("ffmpeg 执行失败: %s", lastLines(msg, 3))
	}
	if onProgress != nil {
		onProgress(100)
	}
	return nil
}

func readProgress(r io.Reader, duration float64, onProgress func(float64)) {
	total := duration * 1e6 // 微秒
	scanner := bufio.NewScanner(r)
	var last float64
	for scanner.Scan() {
		if onProgress == nil || total <= 0 {
			continue
		}
		line := scanner.Text()
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if key != "out_time_us" && key != "out_time_ms" {
			continue
		}
		us, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil || total <= 0 {
			continue
		}
		p := us / total * 100
		if p < 0 {
			p = 0
		}
		if p > 99.5 {
			p = 99.5
		}
		if p-last >= 0.4 {
			last = p
			onProgress(p)
		}
	}
}

// tailBuffer 只保留最后 limit 字节，避免长任务把内存撑爆。
type tailBuffer struct {
	mu    sync.Mutex
	limit int
	buf   bytes.Buffer
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf.Write(p)
	if t.buf.Len() > t.limit {
		b := t.buf.Bytes()
		keep := b[t.buf.Len()-t.limit:]
		nb := append([]byte(nil), keep...)
		t.buf.Reset()
		t.buf.Write(nb)
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.String()
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " | ")
}
