# AudioForge 音频工坊

多平台加密音乐批量解密与转换工具。把 QQ 音乐、酷狗、酷我、网易云、虾米等平台下载的加密文件还原为普通音频，并批量输出为 MP3 / FLAC / WAV / ALAC / AAC / OGG / Opus / AIFF / WMA。

单 Windows `.exe`，双击即用，全程离线，无需网络。

## ✨ 支持的输入格式

| 平台 | 加密扩展名 |
|---|---|
| QQ 音乐 | `.qmc0` `.qmc3` `.qmcflac` `.qmcogg` `.mflac` `.mflac0` `.mgg` `.mgg1` |
| 酷狗音乐 | `.kgm` `.kgma` `.kgg` `.vpr` |
| 酷我音乐 | `.kwm` `.tm0` `.tm2` `.tm3` |
| 网易云音乐 | `.ncm` |
| 虾米音乐 | `.xm` `.x2m` `.x3m` |
| 喜马拉雅 | `.xm` |
| 咪咕音乐 | `.tkm` |

另支持全部普通音频：`.mp3` `.flac` `.wav` `.ape` `.m4a` `.aac` `.ogg` `.opus` `.wma` 等。

> 汽水音乐 `.mdl` 为分段加密 + 联网鉴权，暂不支持离线解密。

## 🎧 支持的输出格式

- **MP3** — 兼容性最好，可调 CBR/VBR 码率
- **FLAC** — 无损压缩，体积约减半
- **WAV** — 无损未压缩
- **ALAC** — 苹果无损
- **AAC** — 同码率优于 MP3
- **OGG / Opus** — 开源有损，Opus 低码率音质最佳
- **AIFF / WMA** — 苹果 / Windows 生态

## 🔧 技术栈

- **Wails v2.17** (Go + Vue3) 原生桌面窗口
- **unlock-music** 解密引擎（MIT）
- **FFmpeg 6.1.1** 静态二进制 `go:embed` 打进 exe
- **WebView2** 渲染（Win11 内置、Win10 随 Edge 预装）

## 🚀 构建

```bash
# 交叉编译 Windows exe（从 macOS/Linux 均可）
wails build -platform windows/amd64 -webview2 browser -s -o AudioForge.exe

# 自检：生成样本 → 转 9 种格式 → ffprobe 校验
AudioForge.exe --selftest
```

> `internal/assets/bin/ffmpeg.exe` 与 `ffprobe.exe`（各 ~83MB）通过 `//go:embed` 嵌入，Windows 构建必需。

## 🧪 测试

```bash
# 后端单测 + 集成测试（需本机有 ffmpeg/ffprobe）
AUDIOFORGE_FFMPEG=/path/to/ffmpeg AUDIOFORGE_FFPROBE=/path/to/ffprobe go test ./...

# 前端类型检查
cd frontend && npm run build
```

## 📄 License

MIT. 本工具仅供对已购买/已授权的音乐做本地备份之用，请勿用于传播或售卖。
