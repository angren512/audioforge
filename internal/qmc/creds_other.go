//go:build !windows

package qmc

import "errors"

func loadCredentials() (Credentials, error) {
	return Credentials{}, errors.New("QQ 音乐在线密钥仅在 Windows 下可用")
}
