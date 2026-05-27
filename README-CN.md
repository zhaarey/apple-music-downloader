# Apple Music ALAC / 杜比全景声下载器

[English](./README.md) | [简体中文](./README-CN.md)

> **原脚本由 Sorrow 编写。** 本仓库已作修改，包含一些修复和改进。

---

## ⚠️ 前置要求

**必须首先安装：**

- **[MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)** - 确保已正确添加到环境变量
- **[wrapper](https://github.com/WorldObservationLog/wrapper)** - 解密程序必须在使用前运行

**可选（用于 MV 下载）：**

- **[mp4decrypt](https://www.bento4.com/downloads/)**

---

## ✨ 功能特性

1. **内嵌封面和 LRC 歌词** - 需要 `media-user-token`（见下方说明）
2. **逐词与未同步歌词** 支持
3. **歌手专辑下载** - 自动下载歌手的所有专辑
   ```bash
   go run main.go https://music.apple.com/us/artist/taylor-swift/159260351 --all-album
   ```
4. **流式解密** - 使用 Sendy McSenderson 的代码实现边下载边解密，解决大文件解密时内存不足问题
5. **MV 下载** - 需要安装 mp4decrypt
6. **交互式搜索** - 支持方向键导航搜索结果
   ```bash
   go run main.go --search [song/album/artist] "search_term"
   ```

---

## 🎵 支持的音频格式

| 格式 | 描述 | 需要订阅 |
|--------|-------------|----------------------|
| `alac` | audio-alac-stereo | ✅ |
| `ec3` | audio-atmos / audio-ec3 | ✅ |
| `aac` | audio-stereo | ✅ |
| `aac-lc` | audio-stereo | ✅ |
| `aac-binaural` | audio-stereo-binaural | ✅ |
| `aac-downmix` | audio-stereo-downmix | ✅ |
| `MV` | 音乐视频 | ✅ |

> **注意：** 对于 `aac-lc`、`MV` 和 `歌词`，必须提供有效订阅的 `media-user-token`。

---

## 🚀 使用方法

### 使用 Docker 运行

1. 确保 [wrapper](https://github.com/WorldObservationLog/wrapper) 解密程序正在运行

2. 启动下载器：

```bash
# 显示帮助
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --help

# 下载专辑
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader https://music.apple.com/ru/album/children-of-forever/1443732441

# 下载单曲
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --song https://music.apple.com/ru/album/bass-folk-song/1443732441?i=1443732453

# 交互式选择
docker run -it --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --select https://music.apple.com/ru/album/children-of-forever/1443732441

# 下载播放列表
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b

# 杜比全景声
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --atmos https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538

# AAC 格式
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --aac https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538

# 调试/查看音质
docker run --network host -v ./downloads:/downloads ghcr.io/zhaarey/apple-music-downloader --debug https://music.apple.com/ru/album/miles-smiles/209407331
```

**自定义配置：**

挂载自己的 `config.yaml`：

```bash
docker run --network host -v ./downloads:/downloads -v ./config.yaml:/app/config.yaml ghcr.io/zhaarey/apple-music-downloader [参数]
```

> **注意：** 运行前请确保当前目录下存在 `config.yaml` 文件。如果不存在，Docker 会创建一个空目录而非文件，导致容器启动失败。

---

### 本地运行 (Go)

1. 确保 [wrapper](https://github.com/WorldObservationLog/wrapper) 解密程序正在运行

2. **下载专辑：**
   ```bash
   go run main.go https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511
   ```

3. **下载单曲：**
   ```bash
   go run main.go --song https://music.apple.com/us/album/never-gonna-give-you-up-2022-remaster/1624945511?i=1624945512
   # 或
   go run main.go https://music.apple.com/us/song/you-move-me-2022-remaster/1624945520
   ```

4. **交互式选择：**
   ```bash
   go run main.go --select https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511
   ```
   输入以空格分隔的曲目编号。

5. **下载播放列表：**
   ```bash
   go run main.go https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b
   # 或
   go run main.go https://music.apple.com/us/playlist/hi-res-lossless-24-bit-192khz/pl.u-MDAWvpjt38370N
   ```

6. **杜比全景声：**
   ```bash
   go run main.go --atmos https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
   ```

7. **AAC 格式：**
   ```bash
   go run main.go --aac https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
   ```

8. **查看音质信息：**
   ```bash
   go run main.go --debug https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
   ```

📖 [中文教程 - 详见方法三](https://telegra.ph/Apple-Music-Alac%E9%AB%98%E8%A7%A3%E6%9E%90%E5%BA%A6%E6%97%A0%E6%8D%9F%E9%9F%B3%E4%B9%90%E4%B8%8B%E8%BD%BD%E6%95%99%E7%A8%8B-04-02-2)

---

## 📝 获取 media-user-token（用于歌词）

1. 打开 [Apple Music](https://music.apple.com) 并登录
2. 打开开发者工具（F12）
3. 导航到 `Application → Storage → Cookies → https://music.apple.com`
4. 找到名为 `media-user-token` 的 Cookie 并复制其值
5. 将该值粘贴到 `config.yaml` 中的 `media-user-token` 设置项
6. 保存文件并启动脚本

---

## 🌐 获取翻译和发音歌词（Beta）

> **注意：** 此功能目前处于测试阶段。

1. 打开 [Apple Music Beta](https://beta.music.apple.com) 并登录
2. 打开开发者工具（F12），切换到 **Network** 标签页
3. 搜索支持翻译/发音歌词的歌曲（推荐 K-Pop 歌曲）
4. 按 **Ctrl+R** 刷新页面，让开发者工具捕获网络数据
5. 播放歌曲并点击歌词按钮 - 查找名为 `syllable-lyrics` 的请求
6. 停止录制（点击左上角红色圆圈按钮），然后选择 **Fetch/XHR** 标签
7. 点击 `syllable-lyrics` 请求查看详情
8. 找到包含以下格式的 URL：`.../syllable-lyrics?l=<language_code>&extend=ttmlLocalizations`
9. 复制语言值并粘贴到 `config.yaml` 中
10. **可选：** 如需禁用发音，在 config.yaml 中移除对应值：`...%5D=<remove_this_value>&extend...`
11. 保存并照常运行脚本

---

## 👏 特别感谢

- **chocomint** - 构建了 `agent-arm64.js`

---
