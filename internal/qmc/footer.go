package qmc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"unicode/utf16"
)

// ErrNoFooter 表示文件尾部没有 musicex 标签（不属于需要在线密钥的新版格式）。
var ErrNoFooter = errors.New("no musicex footer")

const (
	footerMagic       = "musicex\x00"
	footerMinSize     = 0xC0
	mediaIDOffset     = 0x0C
	mediaIDBytes      = 60 // 30 个 UTF-16 字符
	mediaFileNameOff  = 0x48
	mediaFileNameSize = 100 // 50 个 UTF-16 字符
)

// Footer 是 QQ 音乐新版文件的尾部元数据。
type Footer struct {
	SongID   uint32
	MediaID  string // 歌曲 media mid，用于在线取密钥
	FileName string // 原始文件名（UTF-16 解码）
	// ASCII 形式与解密引擎的解析保持一致，用于密钥查找。
	asciiName string
}

// ParseFooter 读取文件尾部的 musicex 标签；不存在时返回 ErrNoFooter。
func ParseFooter(path string) (*Footer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size < 16+footerMinSize {
		return nil, ErrNoFooter
	}

	var tail [16]byte
	if _, err := f.ReadAt(tail[:], size-16); err != nil {
		return nil, err
	}
	tagSize := binary.LittleEndian.Uint32(tail[0:4])
	version := binary.LittleEndian.Uint32(tail[4:8])
	if !bytes.Equal(tail[8:], []byte(footerMagic)) || version != 1 ||
		tagSize < footerMinSize || int64(tagSize) > size {
		return nil, ErrNoFooter
	}

	buf := make([]byte, tagSize)
	if _, err := f.ReadAt(buf, size-int64(tagSize)); err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, err
		}
	}

	ft := &Footer{
		SongID:    binary.LittleEndian.Uint32(buf[0:4]),
		MediaID:   decodeUTF16(buf[mediaIDOffset:], mediaIDBytes),
		FileName:  decodeUTF16(buf[mediaFileNameOff:], mediaFileNameSize),
		asciiName: asciiLowBytes(buf[mediaFileNameOff:], mediaFileNameSize),
	}
	if ft.MediaID == "" || ft.FileName == "" {
		return nil, ErrNoFooter
	}
	if ft.asciiName == "" {
		ft.asciiName = ft.FileName
	}
	return ft, nil
}

// decodeUTF16 以 UTF-16LE 解码到第一个 NUL 为止。
func decodeUTF16(buf []byte, maxLen int) string {
	if maxLen > len(buf) {
		maxLen = len(buf)
	}
	units := make([]uint16, 0, maxLen/2)
	for i := 0; i+1 < maxLen; i += 2 {
		u := binary.LittleEndian.Uint16(buf[i:])
		if u == 0 {
			break
		}
		units = append(units, u)
	}
	return string(utf16.Decode(units))
}

// asciiLowBytes 复刻解密引擎的解析方式（仅取每个 UTF-16 字符的低字节）。
func asciiLowBytes(buf []byte, maxLen int) string {
	if maxLen > len(buf) {
		maxLen = len(buf)
	}
	out := make([]byte, 0, maxLen/2)
	for i := 0; i < maxLen; i += 2 {
		b := buf[i]
		if b == 0 {
			break
		}
		out = append(out, b)
	}
	return string(out)
}
