# Apple Music ALAC / 杜比全景声下载器

[English](./README.md) | [简体中文](./README-CN.md) | [云服务器 / 代理配置](./PROXY-SETUP.md)

> **原脚本由 Sorrow 编写。** 本仓库已修改并包含修复与改进。

这是一个命令行工具，可从 Apple Music 下载专辑、单曲、播放列表、电台和音乐视频，支持 ALAC、AAC 和杜比全景声，并能保留或嵌入元数据与歌词。请仅用于你有权访问的内容，并遵守 Apple 条款和适用法律。

## 目录

- [功能特性](#功能特性)
- [支持的格式](#支持的格式)
- [前置要求](#前置要求)
- [配置](#配置)
- [Windows 安装](#windows-安装)
- [macOS 安装](#macos-安装)
- [Linux 安装](#linux-安装)
- [Android / Termux 安装](#android--termux-安装)
- [使用方法](#使用方法)
- [获取 media-user-token](#获取-media-user-token)
- [歌词设置](#歌词设置)
- [升级](#升级)
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

1. **Go 1.23.1 或更新版本**：[go.dev/dl](https://go.dev/dl/)。
2. **MP4Box / GPAC（可选）**：[gpac.io/downloads/gpac-nightly-builds/](https://gpac.io/downloads/gpac-nightly-builds/)。仅下载 Apple Music 电台时需要，普通歌曲和 MV 使用纯 Go mux。
3. **wrapper-lite**：[github.com/WorldObservationLog/wrapper/tree/lite](https://github.com/WorldObservationLog/wrapper/tree/lite)。必须先启动它，并在 `lite-server` 中写入其 HTTP 地址，例如 `http://127.0.0.1:12340`。
4. **ffmpeg**：仅在后下载转换、动态封面或依赖 ffmpeg 的功能中需要。见 [ffmpeg.org](https://ffmpeg.org/)。

## 配置

先把示例配置复制到项目根目录：

```bash
cp config.yaml.example config.yaml
```

Windows PowerShell 使用：

```powershell
copy config.yaml.example config.yaml
```

至少检查并设置：

```yaml
# wrapper-lite HTTP API 地址。
lite-server: "http://127.0.0.1:12340"

# 下载电台必需，见下文“获取 media-user-token”。
media-user-token: "your-media-user-token"

# 保存目录。相对路径从运行目录解析。
alac-save-folder: "AM-DL downloads"
atmos-save-folder: "AM-DL-Atmos downloads"
aac-save-folder: "AM-DL-AAC downloads"
mv-save-folder: "AM-DL-MV downloads"

# 使用 ffmpeg 转换或动态封面时需要开启。
convert-after-download: false
save-animated-artwork: false
```

如果 wrapper-lite 运行在其他机器或容器中，把 `127.0.0.1` 换成该主机的局域网地址或公网可达地址。

## Windows 安装

1. 安装 **Git**：[git-scm.com/download/win](https://git-scm.com/download/win)。
2. 安装 **Go 1.23.1 或更新版本**：[go.dev/dl](https://go.dev/dl/)。
3. 如需下载电台，再从 [GPAC 官方下载页](https://gpac.io/downloads/gpac-nightly-builds/)安装 GPAC，并确保 `MP4Box.exe` 在 `PATH` 中。
4. 如果需要转换文件或保存动态封面，安装 **ffmpeg**：[ffmpeg.org/download.html](https://ffmpeg.org/download.html)。

在 PowerShell 中：

```powershell
git clone https://github.com/zhaarey/apple-music-downloader.git
cd apple-music-downloader
copy config.yaml.example config.yaml
go build -o amdl.exe .
.\amdl.exe --help
```

示例：

```powershell
.\amdl.exe "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

## macOS 安装

1. 安装 Homebrew：[brew.sh](https://brew.sh/)。
2. 安装运行时和媒体工具：

```bash
brew install go git gpac ffmpeg
```

然后获取源码并构建：

```bash
git clone https://github.com/zhaarey/apple-music-downloader.git
cd apple-music-downloader
cp config.yaml.example config.yaml
go build -o amdl .
./amdl --help
```

示例：

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

## Linux 安装

根据发行版选择命令。其他发行版的包名可能不同。

### Debian / Ubuntu

```bash
sudo apt update
sudo apt install -y git build-essential gpac ffmpeg
```

如果发行版软件源的 Go 低于 `1.23.1`，请从 [go.dev/dl](https://go.dev/dl/) 安装官方 Go，而不是使用发行版包。

### Fedora

```bash
sudo dnf install -y git gcc make gpac ffmpeg
```

如需要，请手动安装官方 Go。

### Arch Linux

```bash
sudo pacman -S --needed git base-devel gpac ffmpeg
```

如需要，请手动安装官方 Go。

然后获取源码并构建：

```bash
git clone https://github.com/zhaarey/apple-music-downloader.git
cd apple-music-downloader
cp config.yaml.example config.yaml
go build -o amdl .
./amdl --help
```

示例：

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

## Android / Termux 安装

从 [F-Droid](https://f-droid.org/en/packages/com.termux/) 或 [官方 GitHub Releases](https://github.com/termux/termux-app/releases) 安装 Termux。不要使用 Play Store 中的过期版本。

1. 更新软件源和已安装软件：

```bash
pkg update && pkg upgrade
```

2. 安装构建工具和媒体工具：

```bash
pkg install golang git gpac ffmpeg
```

3. 可选：授权访问 Android 共享存储。同意提示后，Termux 会创建 `~/storage/shared` 目录树：

```bash
termux-setup-storage
```

4. 获取源码并构建：

```bash
git clone https://github.com/zhaarey/apple-music-downloader.git
cd apple-music-downloader
cp config.yaml.example config.yaml
go build -o amdl .
```

5. 如果要保存到 Android 共享音乐目录，把保存目录指向共享存储挂载位置：

```yaml
alac-save-folder: "/sdcard/Music/amdl"
atmos-save-folder: "/sdcard/Music/amdl-atmos"
aac-save-folder: "/sdcard/Music/amdl-aac"
mv-save-folder: "/sdcard/Music/amdl-mv"
```

正常下载：

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

长时间下载前，避免 Android 挂起 Termux：

```bash
termux-wake-lock
```

完成后释放唤醒锁：

```bash
termux-wake-unlock
```

如果 wrapper-lite 运行在其他设备上，`lite-server` 必须填写该设备的局域网地址或公网可达地址，不能使用 `127.0.0.1`。本节面向当前 Android arm64 Termux 环境；32 位 Android 不在文档保证范围内。

## 使用方法

执行任何命令前确认：

1. wrapper-lite 正在运行。
2. `config.yaml` 存在，并且 `lite-server` 正确。
3. 仅下载电台时需要确保 `MP4Box` 可在 `PATH` 中找到。

### 专辑

```bash
go run . "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
```

也可以使用编译后的程序：

```bash
./amdl "https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511"
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
5. 将值粘贴到 `config.yaml` 的 `media-user-token`。
6. 重启下载器。

## 歌词设置

`config.yaml` 中的关键配置：

```yaml
lrc-type: "lyrics"          # lyrics 或 syllable-lyrics
lrc-format: "lrc"           # lrc 或 ttml
lrc-extra: ""               # 翻译或发音
embed-lrc: true             # 将歌词嵌入媒体文件
save-lrc-file: false        # 同时保存外部 .lrc 文件
```

需要翻译或发音歌词时，根据服务支持的语言或功能代码设置 `lrc-extra`。

## 升级

拉取最新源码并重新构建：

```bash
git pull
go build -o amdl .
```

Windows：

```powershell
git pull
go build -o amdl.exe .
```

如果 `config.yaml.example` 新增了选项，先和你的 `config.yaml` 比较内容。不要直接覆盖旧文件，先保留现有凭据和保存目录。

## 致谢

- **Sorrow** 编写了原始脚本。
- **WorldObservationLog** 开发了 [wrapper / wrapper-lite](https://github.com/WorldObservationLog/wrapper)，本项目将其作为后端解密服务。
- **Sendy McSenderson** 提供了流式下载和解密实现。
- [GPAC](https://gpac.io/) 为电台下载提供 `MP4Box`。
- [FFmpeg](https://ffmpeg.org/) 支持可选的转换和动态封面功能。
