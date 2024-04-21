### ！！封装杜比全景声必须先安装[MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)，并确认[MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)已正确添加到环境变量

### 添加功能
1. 调用外部MP4Box自动封装ec3为m4a
2. 更改目录结构为 歌手名\专辑名  ;Atmos下载文件则另外移动到AM-DL-Atmos downloads，并更改目录结构为 歌手名\专辑名 [Atmos]
3. 运行结束后显示总体完成情况



# Apple Music ALAC / Dolby Atmos Downloader

Original script by Sorrow. Modified by me to include some fixes and improvements.

## How to use
1. Create a virtual device on Android Studio with a image that doesn't have Google APIs.
2. Install this version of Apple Music: https://www.apkmirror.com/apk/apple/apple-music/apple-music-3-6-0-beta-release/apple-music-3-6-0-beta-4-android-apk-download/. You will also need SAI to install it: https://f-droid.org/pt_BR/packages/com.aefyr.sai.fdroid/.
3. Launch Apple Music and sign in to your account. Subscription required.
4. Port forward 10020 TCP: `adb forward tcp:10020 tcp:10020`.
5. Start frida server.
6. Start the frida agent: `frida -U -l agent.js -f com.apple.android.music`.
7. Start downloading some albums: `go run main.go https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511`.
8. Start downloading singles: `go run main_select.go https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511` input numbers separated by spaces.
9. Start downloading some playlists: `go run main.go https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b` or `go run main.go https://music.apple.com/us/playlist/hi-res-lossless-24-bit-192khz/pl.u-MDAWvpjt38370N`.
10. For dolby atmos: `go run main_atmos.go https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538`.

[中文教程-详见方法三](https://telegra.ph/Apple-Music-Alac高解析度无损音乐下载教程-04-02-2)
