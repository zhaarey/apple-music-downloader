# Apple Music ALAC / 杜比全景声下载器

[English](./README.md) | [简体中文](./README-CN.md) | [云服务器 / 代理配置](./PROXY-SETUP.md)

> **原脚本由 Sorrow 编写。** 本仓库已修改并包含修复与改进。

这是一个命令行工具，可从 Apple Music 下载专辑、单曲、播放列表、电台和音乐视频，支持 ALAC、AAC 和杜比全景声，并能保留或嵌入元数据与歌词。请仅用于你有权访问的内容，并遵守 Apple 条款和适用法律。

## 目录

- [功能特性](#功能特性)
- [支持的格式](#支持的格式)
- [前置要求](#前置要求)
- [快速开始](#快速开始)
- [配置](#配置)
- [使用方法](#使用方法)
- [升级](#升级)
- [开发者指南](#开发者指南)
- [获取 media-user-token](#获取-media-user-token)
- [歌词设置](#歌词设置)
- [致谢](#致谢)

## 功能特性

1. 内嵌封面和 LRC 歌词。
2. 逐词歌词和未同步歌词。
3. 歌手全部专辑下载。
4. 大文件流式下载和解密。
5. 音乐视频下载，使用进程内 mp4ff 解密。
6. 交互式搜索和曲目选择。

## 支持的格式

| 格式 | 描述 | 需要订阅 |
|---|---|---|
| `alac` | `audio-alac-stereo` | 是 |
| `ec3` | `audio-atmos` / `audio-ec3` | 是 |
| `aac` | `audio-stereo` | 是 |
| `aac-lc` | `audio-stereo` | 是 |
| `aac-binaural` | `audio-stereo-binaural` | 是 |
| `aac-downmix` | `audio-stereo-downmix` | 是 |
| `MV` | 音乐视频 | 是 |

下载电台需要来自有效订阅的 `media-user-token`。

## 前置要求

运行前必须准备：

1. **wrapper-lite**：[github.com/WorldObservationLog/wrapper/tree/lite](https://github.com/WorldObservationLog/wrapper/tree/lite)。必需的后端解密服务。必须先启动它，并在 `lite-server` 中写入其 HTTP 地址，例如 `http://127.0.0.1:12340`。
2. **ffmpeg**（可选）：仅在后下载转换、动态封面或依赖 ffmpeg 的功能中需要。见 [ffmpeg.org](https://ffmpeg.org/)。

> **提示**：直接使用 Release 预编译二进制文件**不需要安装 Go**。只有自行从源码编译时才需要 Go 1.23.1+（见 [开发者指南](#开发者指南)）。

## 快速开始

从 [最新 GitHub Releases](https://github.com/itouakirai/apple-music-downloader/releases/latest) 下载适合你系统和架构的预编译二进制文件：

| 系统平台 | 硬件架构 | 预编译二进制文件名 |
|---|---|---|
| **Windows** | x86_64 (64位) | `amdl_windows_amd64.exe` |
| **Windows** | ARM64 | `amdl_windows_arm64.exe` |
| **macOS** | Apple Silicon (M1/M2/M3/M4) | `amdl_darwin_arm64` |
| **macOS** | Intel (x86_64) | `amdl_darwin_amd64` |
| **Linux** | x86_64 (amd64) | `amdl_linux_amd64` |
| **Linux** | ARM64 (aarch64) | `amdl_linux_arm64` |
| **Android (Termux)** | ARM64 (aarch64) | `amdl_android_arm64` |

---

### Windows (PowerShell)

1. 打开 PowerShell 终端，执行以下命令下载程序：

```powershell
# 下载预编译二进制（以 64 位 Windows 为例）
Invoke-WebRequest -Uri "https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_windows_amd64.exe" -OutFile "amdl.exe"
```

2. 首次运行 `amdl.exe`，程序将自动在当前目录下生成默认的 `config.yaml`：

```powershell
.\amdl.exe
```

3. 用文本编辑器打开生成的 `config.yaml`，按需配置 `lite-server` 地址与保存路径。
4. 运行测试：

```powershell
.\amdl.exe --help
```

下载示例：

```powershell
.\amdl.exe "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

---

### macOS

1. 下载预编译二进制：

```bash
# Apple Silicon 芯片（M 系列）：
curl -L -o amdl https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_darwin_arm64

# Intel 芯片：
# curl -L -o amdl https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_darwin_amd64

# 赋予执行权限
chmod +x amdl
```

> **macOS Gatekeeper 提示**：如果系统提示“无法打开，因为无法验证开发者”，请在终端执行以下命令移除隔离属性：
> ```bash
> xattr -d com.apple.quarantine amdl
> ```

2. 首次运行 `amdl`，程序将自动生成默认的 `config.yaml`：

```bash
./amdl
```

3. 编辑 `config.yaml` 填写你的配置。
4. 运行测试：

```bash
./amdl --help
```

下载示例：

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

---

### Linux

1. 下载预编译二进制：

```bash
# x86_64 / amd64 架构：
curl -L -o amdl https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_linux_amd64

# ARM64 架构：
# curl -L -o amdl https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_linux_arm64

# 赋予执行权限
chmod +x amdl
```

2. 首次运行 `amdl`，程序将自动生成默认的 `config.yaml`：

```bash
./amdl
```

3. 编辑 `config.yaml` 填写你的配置。
4. 运行测试：

```bash
./amdl --help
```

下载示例：

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

---

### Android / Termux

请从 [F-Droid](https://f-droid.org/en/packages/com.termux/) 或 [官方 GitHub Releases](https://github.com/termux/termux-app/releases) 安装 Termux（切勿使用 Google Play 商店的旧版）。

1. 更新软件包并安装 curl（ffmpeg 可选，用于格式转换或动态封面）：

```bash
pkg update && pkg upgrade
pkg install -y curl ffmpeg
```

2. 授权访问 Android 存储空间（同意权限后将创建 `~/storage/shared`）：

```bash
termux-setup-storage
```

3. 下载预编译二进制：

```bash
curl -L -o amdl https://github.com/itouakirai/apple-music-downloader/releases/latest/download/amdl_android_arm64
chmod +x amdl
```

4. 首次运行 `amdl`，程序将自动生成默认的 `config.yaml`：

```bash
./amdl
```

5. 编辑 `config.yaml`。若希望将音乐直接下载到系统音乐目录，可将配置中的路径设置为共享存储路径：

```yaml
paths:
  alac: "/sdcard/Music/amdl"
  atmos: "/sdcard/Music/amdl-atmos"
  aac: "/sdcard/Music/amdl-aac"
  mv: "/sdcard/Music/amdl-mv"
```

6. 正常下载：

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

> **Termux 使用提示**：
> - 长时间或后台批量下载前，建议使用 `termux-wake-lock` 防止系统休眠，下载完成后执行 `termux-wake-unlock` 释放。
> - 如果 wrapper-lite 运行在局域网内其他设备上，`lite-server` 需填写该设备的局域网 IP，不能填写 `127.0.0.1`。
> - 本说明面向 Android arm64 环境；32 位 Android 不在文档支持范围内。
> - 旧版本自更新若报错 `lookup api.github.com on [::1]:53 ... connection refused`，请按第 3 步重新下载一次，此后即可正常自更新。

## 配置

首次运行 `amdl` 时，如果当前目录下不存在配置文件，程序将自动从内置默认模板生成 `config.yaml`。

至少检查并设置：

```yaml
general:
  # wrapper-lite HTTP API 地址。
  lite-server: "http://127.0.0.1:12340"

  # 下载电台必需，见下文“获取 media-user-token”。
  media-user-token: "your-media-user-token"

# 保存目录。相对路径从运行目录解析。
paths:
  alac: "AM-Lossless"
  atmos: "AM-Atmos"
  aac: "AM-AAC"
  mv: "AM-MV"
```

如果 wrapper-lite 运行在其他机器或容器中，把 `127.0.0.1` 换成该主机的局域网地址或公网可达地址。

## 使用方法

执行任何命令前确认：

1. wrapper-lite 正在运行。
2. `config.yaml` 存在，并且 `lite-server` 正确。

### 专辑

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

Windows PowerShell：

```powershell
.\amdl.exe "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

### 单曲

```bash
./amdl "https://music.apple.com/us/album/never-gonna-give-you-up-2022-remaster/1624945511?i=1624945512"
./amdl "https://music.apple.com/us/song/you-move-me-2022-remaster/1624945520"
```

### 歌手全部专辑

```bash
./amdl --all-album "https://music.apple.com/us/artist/taylor-swift/159260351"
```

### 播放列表

```bash
./amdl "https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b"
```

### 交互式选择

```bash
./amdl --select "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

输入以空格分隔的曲目编号。

### 交互式搜索

```bash
./amdl --search album "never gonna give you up"
./amdl --search song "you move me"
./amdl --search artist "taylor swift"
```

### 杜比全景声

```bash
./amdl --atmos "https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538"
```

### AAC

```bash
./amdl --aac "https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538"
```

### 查看音质信息

```bash
./amdl --debug "https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538"
```

### 常用参数

```text
--alac-max <sample-rate>
--atmos-max <bitrate>
--aac-type <aac|aac-lc|aac-binaural|aac-downmix>
--mv-max <resolution>
--mv-audio-type <atmos|ac3|aac>
--lite-server <wrapper-lite-url>
```

## 获取 media-user-token

`media-user-token` 是下载电台必需的。

1. 打开 [Apple Music](https://music.apple.com) 并登录。
2. 按 `F12` 打开开发者工具。
3. 进入 `Application` > `Storage` > `Cookies` > `https://music.apple.com`。
4. 找到名为 `media-user-token` 的 Cookie，复制它的值。
5. 将值粘贴到 `config.yaml` 的 `general.media-user-token`。
6. 重启下载器。

## 歌词设置

在 `config.yaml` 的 `metadata.lyrics` 节点下配置歌词选项：

```yaml
metadata:
  lyrics:
    save-file: false                      # 是否保存外部歌词文件（.lrc / .ttml）到磁盘
    embed: true                           # 是否将歌词直接嵌入音频文件标签
    type: "lyrics"                        # 歌词类型："lyrics"（标准逐行歌词）或 "syllable-lyrics"（逐词/逐字动态歌词）
    format: "lrc"                         # 歌词格式："lrc" 或 "ttml"
    extra: ""                             # 附加歌词选项：""（默认原语种）、"translation"（翻译歌词）或 "pronunciation"（发音/罗马音歌词）
```

> **提示**：
> - 可在 `config.yaml` 的 `general.language` 中配置目标语言代码（如 `"zh-Hans-CN"`、`"en-US"`、`"ja"` 等），用以控制歌词翻译的目标语言。
> - `extra` 设置为 `"translation"` 获取翻译歌词，设置为 `"pronunciation"` 获取发音/罗马音歌词（需 Apple Music 官方提供对应数据）。

## 升级

### 内置自升级（推荐）

通过内置的自升级命令一键升级至最新官方 Release，自动下载校验对应平台二进制，并提供配置增量迁移向导：

```bash
# 检查并执行自升级与配置迁移
./amdl --update

# 也可以使用简写
./amdl -U

# 仅检查是否有新版本，不执行下载
./amdl --check-update

# 脚本或自动化模式（自动接受新增配置默认值）
./amdl -U -y
```

Windows PowerShell：

```powershell
# 执行自升级
.\amdl.exe --update

# 简写
.\amdl.exe -U
```

> **提示**：自升级会自动创建 `config.yaml.bak_...` 备份文件，并安全引导合并新配置项，原有注释与凭据不会丢失。如果配置了 `proxy`，自升级将自动走代理连接官方 GitHub。

### 手动更新二进制

直接前往 [GitHub Releases](https://github.com/itouakirai/apple-music-downloader/releases/latest) 下载对应平台的最新预编译二进制文件，替换原有的 `amdl` 或 `amdl.exe` 即可。

> 如果是通过源码构建的用户，请参阅 [开发者指南](#开发者指南)。

## 开发者指南

如果你希望参与代码贡献或自行从源码构建：

### 环境要求

1. **Go 1.23.1 或更新版本**：[go.dev/dl](https://go.dev/dl/)。
2. **Git**：[git-scm.com](https://git-scm.com/)。
3. **ffmpeg**：[ffmpeg.org](https://ffmpeg.org/)（可选，用于后处理转换或动态封面）。

### 克隆与构建

**macOS / Linux**：

```bash
git clone https://github.com/itouakirai/apple-music-downloader.git
cd apple-music-downloader
cp config.yaml.example config.yaml
go build -o amdl .
./amdl --help
```

**Windows (PowerShell)**：

```powershell
git clone https://github.com/itouakirai/apple-music-downloader.git
cd apple-music-downloader
copy config.yaml.example config.yaml
go build -o amdl.exe .
.\amdl.exe --help
```

### 开发环境运行

```bash
go run . "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

### 源码更新

```bash
git pull
go build -o amdl .
```

Windows (PowerShell)：

```powershell
git pull
go build -o amdl.exe .
```

## 致谢

- **Sorrow** 编写了原始脚本。
- **WorldObservationLog** 开发了 [wrapper / wrapper-lite](https://github.com/WorldObservationLog/wrapper)，本项目将其作为后端解密服务。
- **Sendy McSenderson** 提供了流式下载和解密实现。
- [go-mp4tag](https://github.com/itouakirai/go-mp4tag) 用于写入电台 MP4 元数据。
- [FFmpeg](https://ffmpeg.org/) 支持可选的转换和动态封面功能。
