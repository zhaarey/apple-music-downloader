package config

// Default returns a Config initialized with sensible default values,
// structured into 5 logical domains.
func Default() Config {
	return Config{
		General: GeneralConfig{
			Storefront:         "us",
			MediaUserToken:     "",
			AuthorizationToken: "",
			Language:           "",
			Proxy:              "",
			LiteServer:         "http://127.0.0.1:12340",
			MaxMemoryLimit:     256,
			ExitOnError:        false,
		},
		Media: MediaConfig{
			GetM3u8Mode: "hires",
			AlacMax:     192000,
			AtmosMax:    2768,
			AacType:     "aac-lc",
			ALACFix:     true,
			MV: MVConfig{
				AudioType: "atmos",
				Max:       2160,
			},
		},
		Paths: PathsConfig{
			Alac:           "AM-Lossless",
			Atmos:          "AM-Atmos",
			Aac:            "AM-AAC",
			MV:             "AM-MV",
			AlbumFolder:    "{AlbumName}",
			PlaylistFolder: "{PlaylistName}",
			ArtistFolder:   "{UrlArtistName}",
			SongFile:       "{SongNumer}. {SongName}",
			LimitMax:       200,
			Explicit:       "[E]",
			Clean:          "[C]",
			AppleMaster:    "[M]",
		},
		Metadata: MetadataConfig{
			Lyrics: LyricsConfig{
				SaveFile: false,
				Embed:    true,
				Type:     "lyrics",
				Format:   "lrc",
				Extra:    "",
			},
			Artwork: ArtworkConfig{
				Embed:         true,
				Size:          "5000x5000",
				Format:        "jpg",
				SaveArtist:    false,
				SaveAnimated:  false,
				EmbyAnimated:  false,
				DlForPlaylist: false,
			},
			Tags: TagsConfig{
				SortOrder:              true,
				ItunesID:               true,
				UseSongInfoForPlaylist: false,
			},
		},
		Convert: ConvertConfig{
			AfterDownload:       false,
			Format:              "flac",
			KeepOriginal:        false,
			SkipIfSourceMatch:   true,
			FFmpegPath:          "ffmpeg",
			ExtraArgs:           "",
			WithMetadata:        true,
			WarnLossyToLossless: true,
			SkipLossyToLossless: true,
			CheckBadALAC:        false,
			DeleteBadALAC:       false,
		},
	}
}
