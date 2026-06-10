package structs

type ConfigSet struct {
	Storefront                 string `yaml:"storefront" json:"storefront"`
	MediaUserToken             string `yaml:"media-user-token" json:"media-user-token"`
	AuthorizationToken         string `yaml:"authorization-token" json:"authorization-token"`
	Language                   string `yaml:"language" json:"language"`
	SaveLrcFile                bool   `yaml:"save-lrc-file" json:"save-lrc-file"`
	LrcType                    string `yaml:"lrc-type" json:"lrc-type"`
	LrcFormat                  string `yaml:"lrc-format" json:"lrc-format"`
	SaveAnimatedArtwork        bool   `yaml:"save-animated-artwork" json:"save-animated-artwork"`
	EmbyAnimatedArtwork        bool   `yaml:"emby-animated-artwork" json:"emby-animated-artwork"`
	EmbedLrc                   bool   `yaml:"embed-lrc" json:"embed-lrc"`
	EmbedCover                 bool   `yaml:"embed-cover" json:"embed-cover"`
	SaveArtistCover            bool   `yaml:"save-artist-cover" json:"save-artist-cover"`
	CoverSize                  string `yaml:"cover-size" json:"cover-size"`
	CoverFormat                string `yaml:"cover-format" json:"cover-format"`
	TagSortOrder               bool   `yaml:"tag-sort-order" json:"tag-sort-order"`
	TagItunesID                bool   `yaml:"tag-itunes-id" json:"tag-itunes-id"`
	AlacSaveFolder             string `yaml:"alac-save-folder" json:"alac-save-folder"`
	AtmosSaveFolder            string `yaml:"atmos-save-folder" json:"atmos-save-folder"`
	AacSaveFolder              string `yaml:"aac-save-folder" json:"aac-save-folder"`
	MVSaveFolder               string `yaml:"mv-save-folder" json:"mv-save-folder"`
	AlbumFolderFormat          string `yaml:"album-folder-format" json:"album-folder-format"`
	PlaylistFolderFormat       string `yaml:"playlist-folder-format" json:"playlist-folder-format"`
	ArtistFolderFormat         string `yaml:"artist-folder-format" json:"artist-folder-format"`
	SongFileFormat             string `yaml:"song-file-format" json:"song-file-format"`
	ExplicitChoice             string `yaml:"explicit-choice" json:"explicit-choice"`
	CleanChoice                string `yaml:"clean-choice" json:"clean-choice"`
	AppleMasterChoice          string `yaml:"apple-master-choice" json:"apple-master-choice"`
	MaxMemoryLimit             int    `yaml:"max-memory-limit" json:"max-memory-limit"`
	DecryptM3u8Port            string `yaml:"decrypt-m3u8-port" json:"decrypt-m3u8-port"`
	GetM3u8Port                string `yaml:"get-m3u8-port" json:"get-m3u8-port"`
	GetM3u8Mode                string `yaml:"get-m3u8-mode" json:"get-m3u8-mode"`
	GetM3u8FromDevice          bool   `yaml:"get-m3u8-from-device" json:"get-m3u8-from-device"`
	AacType                    string `yaml:"aac-type" json:"aac-type"`
	AlacMax                    int    `yaml:"alac-max" json:"alac-max"`
	AtmosMax                   int    `yaml:"atmos-max" json:"atmos-max"`
	LimitMax                   int    `yaml:"limit-max" json:"limit-max"`
	UseSongInfoForPlaylist     bool   `yaml:"use-songinfo-for-playlist" json:"use-songinfo-for-playlist"`
	DlAlbumcoverForPlaylist    bool   `yaml:"dl-albumcover-for-playlist" json:"dl-albumcover-for-playlist"`
	MVAudioType                string `yaml:"mv-audio-type" json:"mv-audio-type"`
	MVMax                      int    `yaml:"mv-max" json:"mv-max"`
	ConvertAfterDownload       bool   `yaml:"convert-after-download" json:"convert-after-download"`
	ConvertFormat              string `yaml:"convert-format" json:"convert-format"`
	ConvertKeepOriginal        bool   `yaml:"convert-keep-original" json:"convert-keep-original"`
	ConvertSkipIfSourceMatch   bool   `yaml:"convert-skip-if-source-matches" json:"convert-skip-if-source-matches"`
	FFmpegPath                 string `yaml:"ffmpeg-path" json:"ffmpeg-path"`
	ConvertExtraArgs           string `yaml:"convert-extra-args" json:"convert-extra-args"`
	ConvertWithMetadata        bool   `yaml:"convert-with-metadata" json:"convert-with-metadata"`
	ConvertWarnLossyToLossless bool   `yaml:"convert-warn-lossy-to-lossless" json:"convert-warn-lossy-to-lossless"`
	ConvertSkipLossyToLossless bool   `yaml:"convert-skip-lossy-to-lossless" json:"convert-skip-lossy-to-lossless"`
	ConvertCheckBadALAC        bool   `yaml:"convert-check-bad-alac" json:"convert-check-bad-alac"`
	ConvertDeleteBadALAC       bool   `yaml:"convert-delete-bad-alac" json:"convert-delete-bad-alac"`
	ALACFix                    bool   `yaml:"alac-fix" json:"alac-fix"`
	ExitOnError                bool   `yaml:"exit-on-error" json:"exit-on-error"`
	YouTubeMode                bool   `yaml:"youtube-mode" json:"youtube-mode"`
	YtDlpPath                  string `yaml:"yt-dlp-path" json:"yt-dlp-path"`
	YouTubeSaveFolder          string   `yaml:"youtube-save-folder" json:"youtube-save-folder"`
	DuplicateCheckFolders      []string `yaml:"duplicate-check-folders" json:"duplicate-check-folders"`
}

type Counter struct {
	Unavailable int
	NotSong     int
	Error       int
	Success     int
	Total       int
}

// 艺术家页面
type AutoGeneratedArtist struct {
	Next string `json:"next"`
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Href       string `json:"href"`
		Attributes struct {
			Previews []struct {
				URL string `json:"url"`
			} `json:"previews"`
			Artwork struct {
				Width      int    `json:"width"`
				Height     int    `json:"height"`
				URL        string `json:"url"`
				BgColor    string `json:"bgColor"`
				TextColor1 string `json:"textColor1"`
				TextColor2 string `json:"textColor2"`
				TextColor3 string `json:"textColor3"`
				TextColor4 string `json:"textColor4"`
			} `json:"artwork"`
			ArtistName           string   `json:"artistName"`
			URL                  string   `json:"url"`
			DiscNumber           int      `json:"discNumber"`
			GenreNames           []string `json:"genreNames"`
			HasTimeSyncedLyrics  bool     `json:"hasTimeSyncedLyrics"`
			IsMasteredForItunes  bool     `json:"isMasteredForItunes"`
			IsAppleDigitalMaster bool     `json:"isAppleDigitalMaster"`
			ContentRating        string   `json:"contentRating"`
			DurationInMillis     int      `json:"durationInMillis"`
			ReleaseDate          string   `json:"releaseDate"`
			Name                 string   `json:"name"`
			Isrc                 string   `json:"isrc"`
			AudioTraits          []string `json:"audioTraits"`
			HasLyrics            bool     `json:"hasLyrics"`
			AlbumName            string   `json:"albumName"`
			PlayParams           struct {
				ID   string `json:"id"`
				Kind string `json:"kind"`
			} `json:"playParams"`
			TrackNumber  int    `json:"trackNumber"`
			AudioLocale  string `json:"audioLocale"`
			ComposerName string `json:"composerName"`
		} `json:"attributes"`
	} `json:"data"`
}
