### ！！必须先安装[MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)，并确认[MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)已正确添加到环境变量

### 添加功能

1. 支持内嵌封面和LRC歌词（需要`media-user-token`，获取方式看最后的说明）
2. 支持获取逐词与未同步歌词
3. 支持下载歌手 `go run main.go https://music.apple.com/us/artist/taylor-swift/159260351` `--all-album` 自动选择歌手的所有专辑
4. 下载解密部分更换为Sendy McSenderson的代码，实现边下载边解密,解决大文件解密时内存不足
5. MV下载，需要安装[mp4decrypt](https://www.bento4.com/downloads/)

### Special thanks to `chocomint` for creating `agent-arm64.js`

对于获取`aac-lc` `MV` `歌词` 必须填入有订阅的`media-user-token`

- `alac (audio-alac-stereo)`
- `ec3 (audio-atmos / audio-ec3)`
- `aac (audio-stereo)`
- `aac-lc (audio-stereo)`
- `aac-binaural (audio-stereo-binaural)`
- `aac-downmix (audio-stereo-downmix)`
- `MV`

# Apple Music ALAC / Dolby Atmos Downloader

Original script by Sorrow. Modified by me to include some fixes and improvements.

## How to use
1. Make sure the decryption program [wrapper](https://github.com/zhaarey/wrapper) is running
2. Start downloading some albums: `go run main.go https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511`.
3. Start downloading single song: `go run main.go --song https://music.apple.com/us/album/never-gonna-give-you-up-2022-remaster/1624945511?i=1624945512` or `go run main.go https://music.apple.com/us/song/you-move-me-2022-remaster/1624945520`.
4. Start downloading select: `go run main.go --select https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511` input numbers separated by spaces.
5. Start downloading some playlists: `go run main.go https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b` or `go run main.go https://music.apple.com/us/playlist/hi-res-lossless-24-bit-192khz/pl.u-MDAWvpjt38370N`.
6. For dolby atmos: `go run main.go --atmos https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538`.
7. For aac: `go run main.go --aac https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538`.
8. For see quality: `go run main.go --debug https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538`.

[中文教程-详见方法三](https://telegra.ph/Apple-Music-Alac高解析度无损音乐下载教程-04-02-2)

## Downloading lyrics

1. Open [Apple Music](https://music.apple.com) and log in
2. Open the Developer tools, Click `Application -> Storage -> Cookies -> https://music.apple.com`
3. Find the cookie named `media-user-token` and copy its value
4. Paste the cookie value obtained in step 3 into the config.yaml and save it
5. Start the script as usual
