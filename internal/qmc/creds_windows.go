//go:build windows

package qmc

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	scanChunk    = 1 << 20
	maxScanBytes = 3 << 30

	memPrivate = 0x20000
	memImage   = 0x1000000
)

var qqProcessNames = []string{"qqmusic.exe", "wechatappex.exe", "qmbrowser.exe"}

func loadCredentials() (Credentials, error) {
	uin, err := readUin()
	if err != nil {
		return Credentials{}, err
	}
	authst, err := scanAuthst()
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{Uin: uin, Authst: authst}, nil
}

func readUin() (string, error) {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return "", errors.New("无法定位 QQ 音乐配置：APPDATA 未设置")
	}
	path := filepath.Join(appdata, "Tencent", "QQMusic", "QQMusicServiceConfig.ini")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("未找到 QQ 音乐配置（%s），请确认已安装 QQ 音乐", path)
	}
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if len(t) < 4 || !strings.EqualFold(t[:4], "uin=") {
			continue
		}
		if v := strings.TrimSpace(t[4:]); v != "" && v != "0" {
			return v, nil
		}
	}
	return "", errors.New("QQ 音乐配置中未找到登录账号，请先在客户端登录")
}

func scanAuthst() (string, error) {
	pids, err := findQqProcesses()
	if err != nil {
		return "", err
	}
	for _, pid := range pids {
		if authst := scanProcess(pid); authst != "" {
			return authst, nil
		}
	}
	return "", errors.New("未能在 QQ 音乐进程中找到登录凭据，请确认客户端已登录；若仍失败，请以管理员身份运行本程序")
}

func findQqProcesses() ([]uint32, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("枚举进程失败: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, fmt.Errorf("枚举进程失败: %w", err)
	}

	var pids []uint32
	for {
		name := strings.ToLower(windows.UTF16ToString(entry.ExeFile[:]))
		for _, want := range qqProcessNames {
			if name == want {
				pids = append(pids, entry.ProcessID)
				break
			}
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}
	if len(pids) == 0 {
		return nil, errors.New("未检测到正在运行的 QQ 音乐客户端，请先启动并登录 QQ 音乐后重试")
	}
	return pids, nil
}

func scanProcess(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)

	var addr uintptr
	var scanned uint64
	for scanned < maxScanBytes {
		var mbi windows.MemoryBasicInformation
		if err := windows.VirtualQueryEx(h, addr, &mbi, unsafe.Sizeof(mbi)); err != nil {
			break
		}
		size := uintptr(mbi.RegionSize)
		if size == 0 {
			break
		}
		if scannable(mbi) {
			for off := uintptr(0); off < size; off += scanChunk {
				n := size - off
				if n > scanChunk {
					n = scanChunk
				}
				buf := make([]byte, n)
				var read uintptr
				if err := windows.ReadProcessMemory(h, mbi.BaseAddress+off, &buf[0], n, &read); err == nil && read > 0 {
					if authst := findAuthst(buf[:read]); authst != "" {
						return authst
					}
				}
				scanned += uint64(n)
				if scanned >= maxScanBytes {
					break
				}
			}
		}
		next := mbi.BaseAddress + size
		if next <= addr {
			break
		}
		addr = next
	}
	return ""
}

const readableProtect = windows.PAGE_READONLY | windows.PAGE_READWRITE | windows.PAGE_WRITECOPY |
	windows.PAGE_EXECUTE_READ | windows.PAGE_EXECUTE_READWRITE | windows.PAGE_EXECUTE_WRITECOPY

func scannable(mbi windows.MemoryBasicInformation) bool {
	if mbi.State != windows.MEM_COMMIT {
		return false
	}
	if mbi.Protect&windows.PAGE_GUARD != 0 || mbi.Protect&windows.PAGE_NOACCESS != 0 {
		return false
	}
	if mbi.Protect&readableProtect == 0 {
		return false
	}
	return mbi.Type == memPrivate || mbi.Type == memImage
}

func findAuthst(data []byte) string {
	for _, marker := range [][]byte{[]byte(`"authst":"`), []byte(`"authst": "`)} {
		if v := scanNarrow(data, marker); v != "" {
			return v
		}
	}
	for _, marker := range [][]byte{utf16le(`"authst":"`), utf16le(`"authst": "`)} {
		if v := scanWide(data, marker); v != "" {
			return v
		}
	}
	return ""
}

func scanNarrow(data, marker []byte) string {
	for start := 0; start < len(data); {
		i := bytes.Index(data[start:], marker)
		if i < 0 {
			return ""
		}
		valStart := start + i + len(marker)
		end := valStart
		for end < len(data) && data[end] != '"' {
			end++
		}
		if v := string(data[valStart:end]); validAuthst(v) {
			return v
		}
		start = valStart
	}
	return ""
}

func scanWide(data, marker []byte) string {
	for start := 0; start+1 < len(data); {
		i := bytes.Index(data[start:], marker)
		if i < 0 {
			return ""
		}
		valStart := start + i + len(marker)
		var b []byte
		end := valStart
		for end+1 < len(data) {
			if data[end] == '"' && data[end+1] == 0 {
				break
			}
			b = append(b, data[end])
			end += 2
		}
		if v := string(b); validAuthst(v) {
			return v
		}
		start = valStart
	}
	return ""
}

func validAuthst(s string) bool {
	if len(s) < 10 || len(s) > 4096 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '+', r == '/', r == '=', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func utf16le(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, len(u)*2)
	for i, v := range u {
		b[i*2] = byte(v)
		b[i*2+1] = byte(v >> 8)
	}
	return b
}
