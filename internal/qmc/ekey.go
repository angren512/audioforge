package qmc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"git.um-react.app/um/cli/algo/common"
)

// Credentials 是 QQ 音乐客户端的登录凭据。
type Credentials struct {
	Uin    string
	Authst string
}

// Client 负责解析 musicex 文件并在线获取 ekey。
type Client struct {
	mu        sync.Mutex
	creds     *Credentials
	cache     map[string]string
	http      *http.Client
	apiURL    string
	loadCreds func() (Credentials, error)
}

// New 创建客户端。
func New() *Client {
	return &Client{
		cache:     make(map[string]string),
		http:      &http.Client{Timeout: 20 * time.Second},
		apiURL:    "https://u.y.qq.com/cgi-bin/musicu.fcg",
		loadCreds: loadCredentials,
	}
}

// Resolve 返回该文件需要的 QMC 密钥。非 musicex 文件返回 (nil, nil)。
func (c *Client) Resolve(ctx context.Context, path string) (common.QMCKeys, error) {
	footer, err := ParseFooter(path)
	if errors.Is(err, ErrNoFooter) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取 QQ 音乐文件信息失败: %w", err)
	}

	ekey, err := c.ekey(ctx, footer)
	if err != nil {
		return nil, err
	}
	keys := common.QMCKeys{footer.FileName: ekey}
	if footer.asciiName != "" && footer.asciiName != footer.FileName {
		keys[footer.asciiName] = ekey
	}
	return common.MergeQMCKeys(keys), nil
}

func (c *Client) ekey(ctx context.Context, footer *Footer) (string, error) {
	cacheKey := footer.MediaID + "|" + footer.FileName
	c.mu.Lock()
	if k := c.cache[cacheKey]; k != "" {
		c.mu.Unlock()
		return k, nil
	}
	c.mu.Unlock()

	creds, err := c.credentials()
	if err != nil {
		return "", err
	}
	k, err := c.fetch(ctx, creds, footer)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.cache[cacheKey] = k
	c.mu.Unlock()
	return k, nil
}

func (c *Client) credentials() (Credentials, error) {
	c.mu.Lock()
	if c.creds != nil {
		creds := *c.creds
		c.mu.Unlock()
		return creds, nil
	}
	c.mu.Unlock()

	creds, err := c.loadCreds()
	if err != nil {
		return Credentials{}, err
	}
	c.mu.Lock()
	c.creds = &creds
	c.mu.Unlock()
	return creds, nil
}

type apiComm struct {
	Authst       string `json:"authst"`
	CT           string `json:"ct"`
	CV           string `json:"cv"`
	Uin          string `json:"uin"`
	TmeLoginType string `json:"tmeLoginType"`
}

type apiParam struct {
	Filename  []string `json:"filename"`
	Guid      string   `json:"guid"`
	Songmid   []string `json:"songmid"`
	Songtype  []int    `json:"songtype"`
	Uin       string   `json:"uin"`
	Loginflag int      `json:"loginflag"`
	Platform  string   `json:"platform"`
	Ctx       int      `json:"ctx"`
}

type apiRequest struct {
	Comm apiComm `json:"comm"`
	Req1 struct {
		Module string   `json:"module"`
		Method string   `json:"method"`
		Param  apiParam `json:"param"`
	} `json:"req_1"`
}

type apiResponse struct {
	Req1 struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	} `json:"req_1"`
}

type apiData struct {
	Midurlinfo []struct {
		Result int    `json:"result"`
		Ekey   string `json:"ekey"`
	} `json:"midurlinfo"`
}

func (c *Client) fetch(ctx context.Context, creds Credentials, footer *Footer) (string, error) {
	var reqBody apiRequest
	reqBody.Comm = apiComm{
		Authst:       creds.Authst,
		CT:           "19",
		CV:           "1859",
		Uin:          creds.Uin,
		TmeLoginType: "3",
	}
	reqBody.Req1.Module = "music.vkey.GetEVkey"
	reqBody.Req1.Method = "CgiGetEVkey"
	reqBody.Req1.Param = apiParam{
		Filename:  []string{footer.FileName},
		Guid:      "10000",
		Songmid:   []string{footer.MediaID},
		Songtype:  []int{1},
		Uin:       creds.Uin,
		Loginflag: 1,
		Platform:  "27",
		Ctx:       1,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://y.qq.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 QQ 音乐密钥接口失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("读取 QQ 音乐密钥响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("QQ 音乐密钥接口返回 HTTP %d", resp.StatusCode)
	}

	var r apiResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("解析 QQ 音乐密钥响应失败: %w", err)
	}
	if r.Req1.Code != 0 {
		return "", fmt.Errorf("QQ 音乐密钥接口错误（code=%d）", r.Req1.Code)
	}

	var data apiData
	if len(r.Req1.Data) == 0 || bytes.Equal(r.Req1.Data, []byte("null")) {
		return "", errors.New("QQ 音乐密钥接口未返回数据")
	}
	if err := json.Unmarshal(r.Req1.Data, &data); err != nil {
		return "", fmt.Errorf("解析 QQ 音乐密钥数据失败: %w", err)
	}
	if len(data.Midurlinfo) == 0 {
		return "", errors.New("QQ 音乐未返回该文件的密钥")
	}

	info := data.Midurlinfo[0]
	if info.Result != 0 || strings.TrimSpace(info.Ekey) == "" {
		return "", fmt.Errorf("QQ 音乐未返回有效密钥（result=%d），请确认 QQ 音乐已登录且该歌曲已购买/可下载", info.Result)
	}
	return strings.TrimSpace(info.Ekey), nil
}
