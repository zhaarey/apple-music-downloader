### ！！必须先安装[MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)，并确认[MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)已正确添加到环境变量

### 添加功能

1. 调用外部MP4Box添加tag
2. 更改目录结构为 歌手名\专辑名  ;Atmos下载文件则另外移动到AM-DL-Atmos downloads，并更改目录结构为 歌手名\专辑名 [Atmos]
3. 运行结束后显示总体完成情况
4. 自动内嵌封面和LRC歌词（需要media-user-token，获取方式看最后的说明）
5. 自动构建 可以到 [Actions](https://github.com/zhaarey/apple-music-alac-atmos-downloader/actions) 页面下载最新自动构建版本 可以直接`main.exe url`
6. 支持逐词与未同步歌词
7. 新增get-m3u8-from-device 改为true 且设置端口`adb forward tcp:20020 tcp:20020`即从模拟器获取m3u8
8. 文件夹和文件支持模板
9. 支持下载歌手 `go run main.go https://music.apple.com/us/artist/taylor-swift/159260351` `--all-album` 自动选择歌手的所有专辑
10. 新增[wrapper](https://github.com/zhaarey/wrapper/releases)模式 目前只能linux运行，解密速度超快，基本秒解
11. `limit-max`支持限制长度 默认200
12. 现已支持arm64解密
13. 下载解密部分更换为Sendy McSenderson的代码，实现边下载边解密

### Special thanks to `chocomint` for creating `agent-arm64.js`

本项目仅支持ALAC和Atmos

- `alac (audio-alac-stereo)`
- `ec3 (audio-atmos / audio-ec3)`

### Python项目

如需下载AAC推荐使用WorldObservationLog的[AppleMusicDecrypt](https://github.com/WorldObservationLog/AppleMusicDecrypt)

[AppleMusicDecrypt](https://github.com/WorldObservationLog/AppleMusicDecrypt)支持以下编码

- `alac (audio-alac-stereo)`
- `ec3 (audio-atmos / audio-ec3)`
- `ac3 (audio-ac3)`
- `aac (audio-stereo)`
- `aac-binaural (audio-stereo-binaural)`
- `aac-downmix (audio-stereo-downmix)`

# Apple Music ALAC / Dolby Atmos Downloader

Original script by Sorrow. Modified by me to include some fixes and improvements.

## How to use

1. Create a virtual device on Android Studio with a image that doesn't have Google APIs.
2. Install Apple Music

   for x86 install this version of [Apple Music 3.6.0 beta4](https://www.apkmirror.com/apk/apple/apple-music/apple-music-3-6-0-beta-release/apple-music-3-6-0-beta-4-android-apk-download/). You will also need [SAI](https://f-droid.org/pt_BR/packages/com.aefyr.sai.fdroid/) to install it.

   for arm64 install the last version of [Apple Music](https://www.apkmirror.com/apk/apple/apple-music/).
   
3. Launch Apple Music and sign in to your account. Subscription required.
4. Port forward 10020 TCP: `adb forward tcp:10020 tcp:10020`.
5. Start frida server.
6. Start the frida agent:

   for  x86 `frida -U -l agent.js -f com.apple.android.music`
   
   for arm64 `frida -U -l agent-arm64.js -f com.apple.android.music`
   
   
7. Start downloading some albums: `go run main.go https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511`.
8. Start downloading singles: `go run main.go --select https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511` input numbers separated by spaces.
9. Start downloading some playlists: `go run main.go https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b` or `go run main.go https://music.apple.com/us/playlist/hi-res-lossless-24-bit-192khz/pl.u-MDAWvpjt38370N`.
10. For dolby atmos: `go run main.go --atmos https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538`.

[中文教程-详见方法三](https://telegra.ph/Apple-Music-Alac高解析度无损音乐下载教程-04-02-2)

## Downloading lyrics

1. Open [Apple Music](https://music.apple.com) and log in
2. Open the Developer tools, Click `Application -> Storage -> Cookies -> https://music.apple.com`
3. Find the cookie named `media-user-token` and copy its value
4. Paste the cookie value obtained in step 3 into the config.yaml and save it
5. Start the script as usual
