package decrypt

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// qmcTestDataDir 定位 unlock-music 模块自带的 QMC 测试样本。
// 找不到时返回空串，测试会跳过——这属于可选的真实样本校验。
func qmcTestDataDir(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "git.um-react.app/um/cli").Output()
	if err != nil {
		return ""
	}
	dir := filepath.Join(strings.TrimSpace(string(out)), "algo", "qmc", "testdata")
	if _, err := os.Stat(dir); err != nil {
		return ""
	}
	return dir
}

func TestDecodeQMCStatic(t *testing.T) {
	dataDir := qmcTestDataDir(t)
	if dataDir == "" {
		t.Skip("未找到 unlock-music 测试样本，跳过真实解密测试")
	}

	body, err := os.ReadFile(filepath.Join(dataDir, "qmc0_static_raw.bin"))
	if err != nil {
		t.Skipf("读取样本失败: %v", err)
	}
	suffix, err := os.ReadFile(filepath.Join(dataDir, "qmc0_static_suffix.bin"))
	if err != nil {
		t.Skipf("读取样本失败: %v", err)
	}
	target, err := os.ReadFile(filepath.Join(dataDir, "qmc0_static_target.bin"))
	if err != nil {
		t.Skipf("读取样本失败: %v", err)
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "晴天.qmc0")
	if err := os.WriteFile(src, bytes.Join([][]byte{body, suffix}, nil), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := Decode(context.Background(), src, dir, Params{}, zap.NewNop())
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	defer os.Remove(out)

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, target) {
		t.Fatalf("解密结果与期望不一致: 得到 %d 字节，期望 %d 字节", len(got), len(target))
	}
}

func TestDecodeSodaRejected(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.mdl")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Decode(context.Background(), src, dir, Params{}, zap.NewNop())
	if err == nil {
		t.Fatal("汽水音乐 .mdl 应返回不支持错误")
	}
	if !strings.Contains(err.Error(), "汽水音乐") {
		t.Errorf("错误信息应说明原因: %v", err)
	}
}

func TestDecodeUnknownFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.qmc0")
	if err := os.WriteFile(src, []byte("garbage content here"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(context.Background(), src, dir, Params{}, zap.NewNop()); err == nil {
		t.Fatal("损坏的加密文件应返回错误")
	}
}
