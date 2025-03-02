package structs

type ConfigSet struct {
	MediaUserToken          string `yaml:"media-user-token"`
	AuthorizationToken      string `yaml:"authorization-token"`
	Language                string `yaml:"language"`
	SaveLrcFile             bool   `yaml:"save-lrc-file"`
	LrcType                 string `yaml:"lrc-type"`
	LrcFormat               string `yaml:"lrc-format"`
	SaveAnimatedArtwork     bool   `yaml:"save-animated-artwork"`
	EmbyAnimatedArtwork     bool   `yaml:"emby-animated-artwork"`
	EmbedLrc                bool   `yaml:"embed-lrc"`
	EmbedCover              bool   `yaml:"embed-cover"`
	SaveArtistCover         bool   `yaml:"save-artist-cover"`
	CoverSize               string `yaml:"cover-size"`
	CoverFormat             string `yaml:"cover-format"`
	AlacSaveFolder          string `yaml:"alac-save-folder"`
	AtmosSaveFolder         string `yaml:"atmos-save-folder"`
	AlbumFolderFormat       string `yaml:"album-folder-format"`
	PlaylistFolderFormat    string `yaml:"playlist-folder-format"`
	ArtistFolderFormat      string `yaml:"artist-folder-format"`
	SongFileFormat          string `yaml:"song-file-format"`
	ExplicitChoice          string `yaml:"explicit-choice"`
	CleanChoice             string `yaml:"clean-choice"`
	AppleMasterChoice       string `yaml:"apple-master-choice"`
	MaxMemoryLimit          int    `yaml:"max-memory-limit"`
	DecryptM3u8Port         string `yaml:"decrypt-m3u8-port"`
	GetM3u8Port             string `yaml:"get-m3u8-port"`
	GetM3u8Mode             string `yaml:"get-m3u8-mode"`
	GetM3u8FromDevice       bool   `yaml:"get-m3u8-from-device"`
	AacType                 string `yaml:"aac-type"`
	AlacMax                 int    `yaml:"alac-max"`
	AtmosMax                int    `yaml:"atmos-max"`
	LimitMax                int    `yaml:"limit-max"`
	UseSongInfoForPlaylist  bool   `yaml:"use-songinfo-for-playlist"`
	DlAlbumcoverForPlaylist bool   `yaml:"dl-albumcover-for-playlist"`
	MVAudioType             string `yaml:"mv-audio-type"`
	MVMax                   int    `yaml:"mv-max"`
}

type Counter struct {
	Unavailable int
	NotSong     int
	Error       int
	Success     int
	Total       int
}

type ApiResult struct {
	Data []SongData `json:"data"`
}

type SongAttributes struct {
	ArtistName        string   `json:"artistName"`
	DiscNumber        int      `json:"discNumber"`
	GenreNames        []string `json:"genreNames"`
	ExtendedAssetUrls struct {
		EnhancedHls string `json:"enhancedHls"`
	} `json:"extendedAssetUrls"`
	IsMasteredForItunes  bool   `json:"isMasteredForItunes"`
	IsAppleDigitalMaster bool   `json:"isAppleDigitalMaster"`
	ContentRating        string `json:"contentRating"`
	ReleaseDate          string `json:"releaseDate"`
	Name                 string `json:"name"`
	Isrc                 string `json:"isrc"`
	AlbumName            string `json:"albumName"`
	TrackNumber          int    `json:"trackNumber"`
	ComposerName         string `json:"composerName"`
}

type AlbumAttributes struct {
	ArtistName           string   `json:"artistName"`
	IsSingle             bool     `json:"isSingle"`
	IsComplete           bool     `json:"isComplete"`
	GenreNames           []string `json:"genreNames"`
	TrackCount           int      `json:"trackCount"`
	IsMasteredForItunes  bool     `json:"isMasteredForItunes"`
	IsAppleDigitalMaster bool     `json:"isAppleDigitalMaster"`
	ContentRating        string   `json:"contentRating"`
	ReleaseDate          string   `json:"releaseDate"`
	Name                 string   `json:"name"`
	RecordLabel          string   `json:"recordLabel"`
	Upc                  string   `json:"upc"`
	Copyright            string   `json:"copyright"`
	IsCompilation        bool     `json:"isCompilation"`
}

type SongData struct {
	ID            string         `json:"id"`
	Attributes    SongAttributes `json:"attributes"`
	Relationships struct {
		Albums struct {
			Data []struct {
				ID         string          `json:"id"`
				Type       string          `json:"type"`
				Href       string          `json:"href"`
				Attributes AlbumAttributes `json:"attributes"`
			} `json:"data"`
		} `json:"albums"`
		Artists struct {
			Href string `json:"href"`
			Data []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
				Href string `json:"href"`
			} `json:"data"`
		} `json:"artists"`
	} `json:"relationships"`
}

type SongResult struct {
	Artwork struct {
		Width                int    `json:"width"`
		URL                  string `json:"url"`
		Height               int    `json:"height"`
		TextColor3           string `json:"textColor3"`
		TextColor2           string `json:"textColor2"`
		TextColor4           string `json:"textColor4"`
		HasAlpha             bool   `json:"hasAlpha"`
		TextColor1           string `json:"textColor1"`
		BgColor              string `json:"bgColor"`
		HasP3                bool   `json:"hasP3"`
		SupportsLayeredImage bool   `json:"supportsLayeredImage"`
	} `json:"artwork"`
	ArtistName             string   `json:"artistName"`
	CollectionID           string   `json:"collectionId"`
	DiscNumber             int      `json:"discNumber"`
	GenreNames             []string `json:"genreNames"`
	ID                     string   `json:"id"`
	DurationInMillis       int      `json:"durationInMillis"`
	ReleaseDate            string   `json:"releaseDate"`
	ContentRatingsBySystem struct {
	} `json:"contentRatingsBySystem"`
	Name     string `json:"name"`
	Composer struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"composer"`
	EditorialArtwork struct {
	} `json:"editorialArtwork"`
	CollectionName string `json:"collectionName"`
	AssetUrls      struct {
		Plus             string `json:"plus"`
		Lightweight      string `json:"lightweight"`
		SuperLightweight string `json:"superLightweight"`
		LightweightPlus  string `json:"lightweightPlus"`
		EnhancedHls      string `json:"enhancedHls"`
	} `json:"assetUrls"`
	AudioTraits []string `json:"audioTraits"`
	Kind        string   `json:"kind"`
	Copyright   string   `json:"copyright"`
	ArtistID    string   `json:"artistId"`
	Genres      []struct {
		GenreID   string `json:"genreId"`
		Name      string `json:"name"`
		URL       string `json:"url"`
		MediaType string `json:"mediaType"`
	} `json:"genres"`
	TrackNumber int    `json:"trackNumber"`
	AudioLocale string `json:"audioLocale"`
	Offers      []struct {
		ActionText struct {
			Short       string `json:"short"`
			Medium      string `json:"medium"`
			Long        string `json:"long"`
			Downloaded  string `json:"downloaded"`
			Downloading string `json:"downloading"`
		} `json:"actionText"`
		Type           string  `json:"type"`
		PriceFormatted string  `json:"priceFormatted"`
		Price          float64 `json:"price"`
		BuyParams      string  `json:"buyParams"`
		Variant        string  `json:"variant,omitempty"`
		Assets         []struct {
			Flavor  string `json:"flavor"`
			Preview struct {
				Duration int    `json:"duration"`
				URL      string `json:"url"`
			} `json:"preview"`
			Size     int `json:"size"`
			Duration int `json:"duration"`
		} `json:"assets"`
	} `json:"offers"`
}

type TrackData struct {
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
	Relationships struct {
		Artists struct {
			Href string `json:"href"`
			Data []struct {
				ID         string `json:"id"`
				Type       string `json:"type"`
				Href       string `json:"href"`
				Attributes struct {
					Name string `json:"name"`
				} `json:"attributes"`
			} `json:"data"`
		} `json:"artists"`
		Albums struct {
			Href string      `json:"href"`
			Data []AlbumData `json:"data"`
		}
	} `json:"relationships"`
}

type AlbumData struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Href       string `json:"href"`
	Attributes struct {
		ArtistName string `json:"artistName"`
		Artwork    struct {
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			URL        string `json:"url"`
			BgColor    string `json:"bgColor"`
			TextColor1 string `json:"textColor1"`
			TextColor2 string `json:"textColor2"`
			TextColor3 string `json:"textColor3"`
			TextColor4 string `json:"textColor4"`
		} `json:"artwork"`
		GenreNames          []string `json:"genreNames"`
		IsCompilation       bool     `json:"isCompilation"`
		IsComplete          bool     `json:"isComplete"`
		IsMasteredForItunes bool     `json:"isMasteredForItunes"`
		IsPrerelease        bool     `json:"isPrerelease"`
		IsSingle            bool     `json:"isSingle"`
		Name                string   `json:"name"`
		PlayParams          struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"playParams"`
		ReleaseDate string `json:"releaseDate"`
		TrackCount  int    `json:"trackCount"`
		Upc         string `json:"upc"`
		URL         string `json:"url"`
	}
}

type AutoGenerated struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Href       string `json:"href"`
		Attributes struct {
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
			IsSingle             bool     `json:"isSingle"`
			URL                  string   `json:"url"`
			IsComplete           bool     `json:"isComplete"`
			GenreNames           []string `json:"genreNames"`
			TrackCount           int      `json:"trackCount"`
			IsMasteredForItunes  bool     `json:"isMasteredForItunes"`
			IsAppleDigitalMaster bool     `json:"isAppleDigitalMaster"`
			ContentRating        string   `json:"contentRating"`
			ReleaseDate          string   `json:"releaseDate"`
			Name                 string   `json:"name"`
			RecordLabel          string   `json:"recordLabel"`
			Upc                  string   `json:"upc"`
			AudioTraits          []string `json:"audioTraits"`
			Copyright            string   `json:"copyright"`
			PlayParams           struct {
				ID   string `json:"id"`
				Kind string `json:"kind"`
			} `json:"playParams"`
			IsCompilation  bool `json:"isCompilation"`
			EditorialVideo struct {
				MotionTall struct {
					Video string `json:"video"`
				} `json:"motionTallVideo3x4"`
				MotionSquare struct {
					Video string `json:"video"`
				} `json:"motionSquareVideo1x1"`
				MotionDetailTall struct {
					Video string `json:"video"`
				} `json:"motionDetailTall"`
				MotionDetailSquare struct {
					Video string `json:"video"`
				} `json:"motionDetailSquare"`
			} `json:"editorialVideo"`
		} `json:"attributes"`
		Relationships struct {
			RecordLabels struct {
				Href string        `json:"href"`
				Data []interface{} `json:"data"`
			} `json:"record-labels"`
			Artists struct {
				Href string `json:"href"`
				Data []struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Href       string `json:"href"`
					Attributes struct {
						Name    string `json:"name"`
						Artwork struct {
							Url string `json:"url"`
						} `json:"artwork"`
					} `json:"attributes"`
				} `json:"data"`
			} `json:"artists"`
			Tracks struct {
				Href string      `json:"href"`
				Next string      `json:"next"`
				Data []TrackData `json:"data"`
			} `json:"tracks"`
		} `json:"relationships"`
	} `json:"data"`
}

type AutoGeneratedTrack struct {
	Href string `json:"href"`
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
		Relationships struct {
			Artists struct {
				Href string `json:"href"`
				Data []struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Href       string `json:"href"`
					Attributes struct {
						Name string `json:"name"`
					} `json:"attributes"`
				} `json:"data"`
			} `json:"artists"`
			Albums struct {
				Href string      `json:"href"`
				Data []AlbumData `json:"data"`
			}
		} `json:"relationships"`
	} `json:"data"`
}

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

type AutoGeneratedMusicVideo struct {
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
			AlbumName        string   `json:"albumName"`
			ArtistName       string   `json:"artistName"`
			URL              string   `json:"url"`
			GenreNames       []string `json:"genreNames"`
			DurationInMillis int      `json:"durationInMillis"`
			Isrc             string   `json:"isrc"`
			TrackNumber      int      `json:"trackNumber"`
			DiscNumber       int      `json:"discNumber"`
			ContentRating    string   `json:"contentRating"`
			ReleaseDate      string   `json:"releaseDate"`
			Name             string   `json:"name"`
			Has4K            bool     `json:"has4K"`
			HasHDR           bool     `json:"hasHDR"`
			PlayParams       struct {
				ID   string `json:"id"`
				Kind string `json:"kind"`
			} `json:"playParams"`
		} `json:"attributes"`
	} `json:"data"`
}

type SongLyrics struct {
	Data []struct {
		Id         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Ttml       string `json:"ttml"`
			PlayParams struct {
				Id          string `json:"id"`
				Kind        string `json:"kind"`
				CatalogId   string `json:"catalogId"`
				DisplayType int    `json:"displayType"`
			} `json:"playParams"`
		} `json:"attributes"`
	} `json:"data"`
}
