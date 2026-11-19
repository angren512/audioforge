package decrypt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.um-react.app/um/cli/algo/common"

	// 通过空导入注册各平台的解码器
	_ "git.um-react.app/um/cli/algo/kgm"
	_ "git.um-react.app/um/cli/algo/kwm"
	_ "git.um-react.app/um/cli/algo/ncm"
	_ "git.um-react.app/um/cli/algo/qmc"
	_ "git.um-react.app/um/cli/algo/tm"
	_ "git.um-react.app/um/cli/algo/xiami"
	_ "git.um-react.app/um/cli/algo/ximalaya"

	"go.uber.org/zap"
)

// ErrUnsupported 表示该加密格式无法离线解密。
var ErrUnsupported = errors.New("无法离线解密")

// Params 是解密所需的密钥参数。
type Params struct {
	QmcKeys common.QMCKeys
	KggDb   string
}

// QmcKeyResolver 按源文件解析 QQ 音乐新版（musicex）密钥。
// 返回空表示该文件不需要在线密钥。
type QmcKeyResolver func(ctx context.Context, src string) (common.QMCKeys, error)

// Decode 把加密文件解密成普通音频临时文件，返回临时文件路径，调用方负责删除。
func Decode(ctx context.Context, src, tmpDir string, p Params, logger *zap.Logger) (string, error) {
	if IsSoda(src) {
		return "", fmt.Errorf("%w：汽水音乐 .mdl 为分段加密，需要联网鉴权，暂不支持", ErrUnsupported)
	}

	factories := common.GetDecoder(src, true)
	if len(factories) == 0 {
		return "", fmt.Errorf("%w：没有匹配的密钥方案", ErrUnsupported)
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	var lastErr error
	for i, factory := range factories {
		out := filepath.Join(tmpDir, fmt.Sprintf("af-%d-%d-%d.tmp", os.Getpid(), time.Now().UnixNano(), i))
		err := decodeOne(ctx, factory, src, out, p, logger)
		if err == nil {
			return out, nil
		}
		os.Remove(out)
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("未知错误")
	}
	return "", fmt.Errorf("解密失败: %w", lastErr)
}

func decodeOne(ctx context.Context, factory common.DecoderFactory, src, out string, p Params, logger *zap.Logger) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	d := factory.Create(&common.DecoderParams{
		Reader:    f,
		Extension: strings.ToLower(filepath.Ext(src)),
		FilePath:  src,
		Logger:    logger,
		CryptoParams: common.CryptoParams{
			QmcKeys:   p.QmcKeys,
			KggDbPath: p.KggDb,
		},
	})
	if err := d.Validate(); err != nil {
		return err
	}

	dst, err := os.Create(out)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(dst, d)
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n == 0 {
		return errors.New("解密结果为空，可能缺少 ekey 或文件已损坏")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return nil
}
