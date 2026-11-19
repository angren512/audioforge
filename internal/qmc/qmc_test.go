package qmc

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func putUTF16(dst []byte, s string, maxBytes int) {
	units := utf16.Encode([]rune(s))
	for i, u := range units {
		if i*2+1 >= maxBytes {
			return
		}
		binary.LittleEndian.PutUint16(dst[i*2:], u)
	}
}

func writeMusicExFile(t *testing.T, mediaID, fileName string, tagSize uint32, version uint32) string {
	t.Helper()
	tag := make([]byte, tagSize)
	binary.LittleEndian.PutUint32(tag[0:], 42)
	putUTF16(tag[mediaIDOffset:], mediaID, mediaIDBytes)
	putUTF16(tag[mediaFileNameOff:], fileName, mediaFileNameSize)

	// 尾部 16 字节属于 tagSize 的一部分。
	tail := tag[tagSize-16:]
	binary.LittleEndian.PutUint32(tail[0:], tagSize)
	binary.LittleEndian.PutUint32(tail[4:], version)
	copy(tail[8:], footerMagic)

	content := append([]byte("fLaC-encrypted-audio-payload"), tag...)

	path := filepath.Join(t.TempDir(), "song.mflac")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseFooter(t *testing.T) {
	path := writeMusicExFile(t, "0039MnYb0qxYhV", "我们的爱.mflac", footerMinSize, 1)
	ft, err := ParseFooter(path)
	if err != nil {
		t.Fatalf("ParseFooter: %v", err)
	}
	if ft.MediaID != "0039MnYb0qxYhV" {
		t.Errorf("MediaID = %q", ft.MediaID)
	}
	if ft.FileName != "我们的爱.mflac" {
		t.Errorf("FileName = %q", ft.FileName)
	}
	if ft.asciiName == "" || ft.asciiName == ft.FileName {
		t.Errorf("asciiName = %q", ft.asciiName)
	}
	if ft.SongID != 42 {
		t.Errorf("SongID = %d", ft.SongID)
	}
}

func TestParseFooterMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain.flac")
	if err := os.WriteFile(path, []byte("just some audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFooter(path); err != ErrNoFooter {
		t.Fatalf("err = %v, want ErrNoFooter", err)
	}

	bad := writeMusicExFile(t, "mid", "x.mflac", footerMinSize, 2)
	if _, err := ParseFooter(bad); err != ErrNoFooter {
		t.Fatalf("version 2 err = %v, want ErrNoFooter", err)
	}
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New()
	c.apiURL = srv.URL
	c.loadCreds = func() (Credentials, error) {
		return Credentials{Uin: "123456", Authst: "FAKEAUTHST+/="}, nil
	}
	return c
}

func TestResolveFetchesEkey(t *testing.T) {
	var got apiRequest
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"code":0,"req_1":{"code":0,"data":{"midurlinfo":[{"result":0,"ekey":"FAKE-EKEY-123"}]}}}`))
	})

	path := writeMusicExFile(t, "0039MnYb0qxYhV", "我们的爱.mflac", footerMinSize, 1)
	keys, err := c.Resolve(context.Background(), path)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if v, ok := keys.Get("我们的爱.mflac"); !ok || v != "FAKE-EKEY-123" {
		t.Errorf("utf16 key = %q, ok=%v", v, ok)
	}

	ft, _ := ParseFooter(path)
	if v, ok := keys.Get(ft.asciiName); !ok || v != "FAKE-EKEY-123" {
		t.Errorf("ascii key = %q, ok=%v", v, ok)
	}

	if got.Req1.Module != "music.vkey.GetEVkey" || got.Req1.Method != "CgiGetEVkey" {
		t.Errorf("module/method = %s/%s", got.Req1.Module, got.Req1.Method)
	}
	if got.Comm.Uin != "123456" || got.Comm.Authst != "FAKEAUTHST+/=" {
		t.Errorf("comm = %+v", got.Comm)
	}
	if len(got.Req1.Param.Songmid) != 1 || got.Req1.Param.Songmid[0] != "0039MnYb0qxYhV" {
		t.Errorf("songmid = %v", got.Req1.Param.Songmid)
	}
	if len(got.Req1.Param.Filename) != 1 || got.Req1.Param.Filename[0] != "我们的爱.mflac" {
		t.Errorf("filename = %v", got.Req1.Param.Filename)
	}
}

func TestResolveEmptyEkey(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"req_1":{"code":0,"data":{"midurlinfo":[{"result":104003,"ekey":""}]}}}`))
	})
	path := writeMusicExFile(t, "mid123", "a.mflac", footerMinSize, 1)
	_, err := c.Resolve(context.Background(), path)
	if err == nil || !strings.Contains(err.Error(), "104003") {
		t.Fatalf("err = %v, want result 104003", err)
	}
}

func TestResolveSkipsPlainFile(t *testing.T) {
	c := New()
	path := filepath.Join(t.TempDir(), "plain.mp3")
	if err := os.WriteFile(path, []byte("ID3 fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	keys, err := c.Resolve(context.Background(), path)
	if err != nil || keys != nil {
		t.Fatalf("keys=%v err=%v, want nil,nil", keys, err)
	}
}
