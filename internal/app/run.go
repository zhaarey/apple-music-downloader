package app

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/pflag"
	"amdl/internal/amp-api"
	fairplayrip "amdl/internal/fairplay-rip"
	"amdl/internal/config"
	"amdl/internal/download"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func Main() {
	r := NewRunner(config.ConfigSet{})

	// --lite-server overrides the configured endpoint for this run. Scan the
	// raw args first so the /status check below uses the override too.
	r.Flags.LiteServerFlag = flagValueFromArgs(os.Args[1:], "lite-server")
	err := r.loadConfig()
	if err != nil {
		fmt.Printf("load Config failed: %v", err)
		return
	}
	if r.Flags.LiteServerFlag != "" {
		r.Config.LiteServer = r.Flags.LiteServerFlag
	}
	if regions, err := r.getLiteRegions(); err != nil {
		fmt.Println("Warning: failed to query lite-server /status:", err)
	} else {
		fmt.Printf("lite-server regions: %s\n", regions)
	}
	if err := download.Init(r.Config.Proxy); err != nil {
		fmt.Printf("proxy config error: %v\n", err)
		return
	}
	if err := fairplayrip.Init(); err != nil {
		fmt.Printf("temari library error: %v\n", err)
		return
	}
	token, err := ampapi.GetToken()
	if err != nil {
		if r.Config.AuthorizationToken != "" && r.Config.AuthorizationToken != "your-authorization-token" {
			token = strings.Replace(r.Config.AuthorizationToken, "Bearer ", "", -1)
		} else {
			fmt.Println("Failed to get token.")
			return
		}
	}
	var search_type string
	pflag.StringVar(&search_type, "search", "", "Search for 'album', 'song', or 'artist'. Provide query after flags.")
	pflag.BoolVar(&r.Flags.Atmos, "atmos", false, "Enable atmos download mode")
	pflag.BoolVar(&r.Flags.AAC, "aac", false, "Enable adm-aac download mode")
	pflag.BoolVar(&r.Flags.Select, "select", false, "Enable selective download")
	pflag.BoolVar(&r.Flags.ArtistSelect, "all-album", false, "Download all artist albums")
	pflag.BoolVar(&r.Flags.Debug, "debug", false, "Enable debug mode to show audio quality information")
	pflag.BoolVar(&r.Flags.PrintJSON, "json", false, "Output JSON summary at the end")
	pflag.BoolVar(&r.Flags.SaveM3U8, "save-m3u8-playlist", false, "Save M3U8 playlist file")
	pflag.StringVar(&r.Flags.LiteServerFlag, "lite-server", r.Config.LiteServer, "wrapper-lite HTTP API endpoint for this run")
	alac_max = pflag.Int("alac-max", r.Config.AlacMax, "Specify the max quality for download alac")
	atmos_max = pflag.Int("atmos-max", r.Config.AtmosMax, "Specify the max quality for download atmos")
	aac_type = pflag.String("aac-type", r.Config.AacType, "Select AAC type, aac aac-binaural aac-downmix")
	mv_audio_type = pflag.String("mv-audio-type", r.Config.MVAudioType, "Select MV audio type, atmos ac3 aac")
	mv_max = pflag.Int("mv-max", r.Config.MVMax, "Specify the max quality for download MV")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [url1 url2 ...]\n", "[main | main.exe | go run main.go]")
		fmt.Fprintf(os.Stderr, "Search Usage: %s --search [album|song|artist] [query]\n", "[main | main.exe | go run main.go]")
		fmt.Println("\nOptions:")
		pflag.PrintDefaults()
	}

	pflag.Parse()
	if r.Flags.LiteServerFlag != "" {
		r.Config.LiteServer = r.Flags.LiteServerFlag
	}
	r.Config.AlacMax = *alac_max
	r.Config.AtmosMax = *atmos_max
	r.Config.AacType = *aac_type
	r.Config.MVAudioType = *mv_audio_type
	r.Config.MVMax = *mv_max

	args := pflag.Args()

	if search_type != "" {
		if len(args) == 0 {
			fmt.Println("Error: --search flag requires a query.")
			pflag.Usage()
			return
		}
		selectedUrl, err := r.handleSearch(search_type, args, token)
		if err != nil {
			fmt.Printf("\nSearch process failed: %v\n", err)
			return
		}
		if selectedUrl == "" {
			fmt.Println("\nExiting.")
			return
		}
		os.Args = []string{selectedUrl}
	} else {
		if len(args) == 0 {
			fmt.Println("No URLs provided. Please provide at least one URL.")
			pflag.Usage()
			return
		}
		os.Args = args
	}

	if strings.Contains(os.Args[0], "/artist/") {
		urlArtistName, urlArtistID, err := r.getUrlArtistName(os.Args[0], token)
		if err != nil {
			fmt.Println("Failed to get artistname.")
			return
		}
		r.Config.ArtistFolderFormat = strings.NewReplacer(
			"{UrlArtistName}", r.LimitString(urlArtistName),
			"{ArtistId}", urlArtistID,
		).Replace(r.Config.ArtistFolderFormat)
		albumArgs, err := r.checkArtist(os.Args[0], token, "albums")
		if err != nil {
			fmt.Println("Failed to get artist albums.")
			return
		}
		mvArgs, err := r.checkArtist(os.Args[0], token, "music-videos")
		if err != nil {
			fmt.Println("Failed to get artist music-videos.")
		}
		os.Args = append(albumArgs, mvArgs...)
	}
	albumTotal := len(os.Args)
	for {
		for albumNum, urlRaw := range os.Args {
			fmt.Printf("Queue %d of %d: ", albumNum+1, albumTotal)
			var storefront, albumId string

			if strings.Contains(urlRaw, "/music-video/") {
				fmt.Println("Music Video")
				if r.Flags.Debug {
					continue
				}
				r.State.Counter.Total++
				if r.Config.LiteServer == "" {
					fmt.Println(": lite-server is not set, skip MV dl")
					r.State.Counter.Success++
					continue
				}
				mvSaveDir := strings.NewReplacer(
					"{ArtistName}", "",
					"{UrlArtistName}", "",
					"{ArtistId}", "",
				).Replace(r.Config.ArtistFolderFormat)
				if mvSaveDir != "" {
					mvSaveDir = filepath.Join(r.Config.MVSaveFolder, forbiddenNames.ReplaceAllString(mvSaveDir, "_"))
				} else {
					mvSaveDir = r.Config.MVSaveFolder
				}
				storefront, albumId = checkUrl(urlRaw, "mv")
				err := r.mvDownloader(albumId, mvSaveDir, token, storefront, nil)
				if err != nil {
					fmt.Println("\u26A0 Failed to dl MV:", err)
					r.State.Counter.Error++
					continue
				}
				r.State.Counter.Success++
				continue
			}
			if strings.Contains(urlRaw, "/song/") {
				fmt.Printf("Song->")
				storefront, songId := checkUrl(urlRaw, "song")
				if storefront == "" || songId == "" {
					fmt.Println("Invalid song URL format.")
					continue
				}
				err := r.ripSong(songId, token, storefront, r.Config.MediaUserToken)
				if err != nil {
					fmt.Println("Failed to rip song:", err)
				}
				continue
			}
			parse, err := url.Parse(urlRaw)
			if err != nil {
				fmt.Printf("Invalid URL: %v\n", err)
				r.State.Counter.Error++
				continue
			}
			var urlArg_i = parse.Query().Get("i")

			if strings.Contains(urlRaw, "/album/") {
				fmt.Println("Album")
				storefront, albumId = checkUrl(urlRaw, "album")
				err := r.ripAlbum(albumId, token, storefront, r.Config.MediaUserToken, urlArg_i)
				if err != nil {
					fmt.Println("Failed to rip album:", err)
				}
			} else if strings.Contains(urlRaw, "/playlist/") {
				fmt.Println("Playlist")
				storefront, albumId = checkUrl(urlRaw, "playlist")
				err := r.ripPlaylist(albumId, token, storefront, r.Config.MediaUserToken)
				if err != nil {
					fmt.Println("Failed to rip playlist:", err)
				}
			} else if strings.Contains(urlRaw, "/station/") {
				fmt.Printf("Station")
				storefront, albumId = checkUrl(urlRaw, "station")
				if len(r.Config.MediaUserToken) <= 50 {
					fmt.Println(": meida-user-token is not set, skip station dl")
					continue
				}
				err := r.ripStation(albumId, token, storefront, r.Config.MediaUserToken)
				if err != nil {
					fmt.Println("Failed to rip station:", err)
				}
			} else {
				fmt.Println("Invalid type")
			}
		}
		fmt.Printf("=======  [\u2714 ] Completed: %d/%d  |  [\u26A0 ] Warnings: %d  |  [\u2716 ] Errors: %d  =======\n", r.State.Counter.Success, r.State.Counter.Total, r.State.Counter.Unavailable+r.State.Counter.NotSong, r.State.Counter.Error)
		if r.State.Counter.Error == 0 {
			break
		} else if r.Config.ExitOnError {
			fmt.Println("Error detected, exiting...")
			os.Exit(1)
		} else {
			fmt.Println("Error detected, press Enter to try again...")
			fmt.Scanln()
			fmt.Println("Start trying again...")
		}

		r.State.Counter = config.Counter{}
	}

	// Print JSON output
	if r.Flags.PrintJSON {
		jsonOutput, err := json.Marshal(r.State.AddedTracks)
		if err != nil {
			fmt.Println("Error generating JSON output:", err)
		} else {
			fmt.Println(string(jsonOutput))
		}
	}
}
