package task

import (
	"fmt"
	"os"
	"testing"
)

func TestAddUnbounded(t *testing.T) {
	p := New(Config{})
	const n = 5000
	paths := make([]string, n)
	for i := range paths {
		paths[i] = fmt.Sprintf("/tmp/song-%d.mflac", i)
	}
	added := p.Add(paths)
	if len(added) != n {
		t.Fatalf("期望加入 %d 个任务，实际 %d", n, len(added))
	}
	if got := len(p.Snapshot()); got != n {
		t.Fatalf("快照长度 = %d, 期望 %d", got, n)
	}
	s := p.Stats()
	if s.Total != n || s.Queued != n {
		t.Fatalf("统计异常: %+v", s)
	}
}

func TestIDsUnique(t *testing.T) {
	p := New(Config{})
	a := p.Add([]string{"/tmp/a.ncm", "/tmp/a.ncm"})
	if a[0].ID == a[1].ID {
		t.Error("任务 ID 应唯一")
	}
}

func TestEncryptedFlag(t *testing.T) {
	p := New(Config{})
	added := p.Add([]string{"/tmp/a.mflac", "/tmp/b.mp3"})
	if !added[0].Encrypted {
		t.Error(".mflac 应标记为加密")
	}
	if added[1].Encrypted {
		t.Error(".mp3 不应标记为加密")
	}
}

func TestCancelQueued(t *testing.T) {
	p := New(Config{})
	added := p.Add([]string{"/tmp/x.ncm", "/tmp/y.ncm"})
	p.CancelTask(added[0].ID)

	snap := p.Snapshot()
	if snap[0].State != StateCanceled {
		t.Errorf("任务状态 = %s, 期望 canceled", snap[0].State)
	}
	s := p.Stats()
	if s.Canceled != 1 || s.Queued != 1 {
		t.Errorf("统计异常: %+v", s)
	}
}

func TestClearFinishedOnly(t *testing.T) {
	p := New(Config{})
	added := p.Add([]string{"/tmp/x.ncm", "/tmp/y.ncm"})
	p.mutate(added[0].ID, func(t *Task) { t.State = StateDone })

	p.ClearFinished()
	snap := p.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("期望保留 1 个未完成任务，实际 %d", len(snap))
	}
	if snap[0].ID != added[1].ID {
		t.Error("保留的任务不正确")
	}
}

func TestUniquePath(t *testing.T) {
	dir := t.TempDir()
	first := dir + "/a.mp3"
	if got := uniquePath(first); got != first {
		t.Errorf("文件不存在时应返回原路径，实际 %s", got)
	}
	if err := os.WriteFile(first, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := uniquePath(first); got != dir+"/a (1).mp3" {
		t.Errorf("重名时应追加序号，实际 %s", got)
	}
}
