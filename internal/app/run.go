package app

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"

	ampapi "amdl/internal/amp-api"
	"amdl/internal/config"
	"amdl/internal/download"
	fairplayrip "amdl/internal/fairplay-rip"
	"amdl/internal/updater"
	"amdl/internal/version"
)

func Main() {
	r := NewRunner(config.ConfigSet{})

	var (
		configFile  string
		search_type string
	)
	pflag.StringVarP(&configFile, "config", "c", "", "Path to custom config file (default: config.yaml)")
	pflag.StringVar(&search_type, "search", "", "Search for 'album', 'song', or 'artist'. Provide query after flags.")
	pflag.BoolVar(&r.Flags.Atmos, "atmos", false, "Enable atmos download mode")
	pflag.BoolVar(&r.Flags.AAC, "aac", false, "Enable adm-aac download mode")
	pflag.BoolVar(&r.Flags.Select, "select", false, "Enable selective download")
	pflag.BoolVar(&r.Flags.ArtistSelect, "all-album", false, "Download all artist albums")
	pflag.BoolVar(&r.Flags.Debug, "debug", false, "Enable debug mode to show audio quality information")
	pflag.BoolVar(&r.Flags.PrintJSON, "json", false, "Output JSON summary at the end")
	pflag.BoolVar(&r.Flags.SaveM3U8, "save-m3u8-playlist", false, "Save M3U8 playlist file")
	pflag.StringVar(&r.Flags.LiteServerFlag, "lite-server", "", "wrapper-lite HTTP API endpoint for this run")
	pflag.Int("alac-max", 0, "Specify the max quality for download alac")
	pflag.Int("atmos-max", 0, "Specify the max quality for download atmos")
	pflag.String("aac-type", "", "Select AAC type, aac aac-binaural aac-downmix")
	pflag.String("mv-audio-type", "", "Select MV audio type, atmos ac3 aac")
	pflag.Int("mv-max", 0, "Specify the max quality for download MV")
	pflag.BoolVarP(&r.Flags.Version, "version", "v", false, "Print version information and exit")
	pflag.BoolVarP(&r.Flags.Update, "update", "U", false, "Perform self-update and interactive config migration")
	pflag.BoolVar(&r.Flags.CheckUpdate, "check-update", false, "Check for available updates without downloading")
	pflag.BoolVarP(&r.Flags.Yes, "yes", "y", false, "Automatically accept defaults during update")

	pflag.Usage = func() {
		prog := progName()
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [url1 url2 ...]\n", prog)
		fmt.Fprintf(os.Stderr, "Search Usage: %s --search [album|song|artist] [query]\n", prog)
		fmt.Println("\nOptions:")
		pflag.PrintDefaults()
	}

	pflag.Parse()

	if r.Flags.Version {
		fmt.Println(version.Info())
		return
	}

	if r.Flags.CheckUpdate || r.Flags.Update {
		_ = r.loadConfig(config.LoadOptions{
			ConfigFile:             configFile,
			FlagSet:                pflag.CommandLine,
			DisableMissingWarnings: true,
		})
		proxyURL := r.Config.General.Proxy

		if r.Flags.CheckUpdate {
			if err := updater.CheckUpdateOnly(proxyURL); err != nil {
				fmt.Printf("Update check error: %v\n", err)
			}
			return
		}

		if r.Flags.Update {
			if err := updater.ExecuteSelfUpdate(configFile, proxyURL, r.Flags.Yes); err != nil {
				fmt.Printf("Self-update failed: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	err := r.loadConfig(config.LoadOptions{
		ConfigFile: configFile,
		FlagSet:    pflag.CommandLine,
	})
	if err != nil {
		fmt.Printf("load Config failed: %v\n", err)
		return
	}

	updater.PrintStartupUpdateNotice(r.Config.General.Proxy)
	if r.Flags.LiteServerFlag == "" {
		r.Flags.LiteServerFlag = r.Config.General.LiteServer
	}

	if regions, err := r.getLiteRegions(); err != nil {
		fmt.Println("Warning: failed to query lite-server /status:", err)
	} else {
		fmt.Printf("lite-server regions: %s\n", regions)
	}
	if err := download.Init(r.Config.General.Proxy); err != nil {
		fmt.Printf("proxy config error: %v\n", err)
		return
	}
	if err := fairplayrip.Init(); err != nil {
		fmt.Printf("temari library error: %v\n", err)
		return
	}
	token, err := ampapi.GetToken()
	if err != nil {
		if r.Config.General.AuthorizationToken != "" && r.Config.General.AuthorizationToken != "your-authorization-token" {
			token = strings.Replace(r.Config.General.AuthorizationToken, "Bearer ", "", -1)
		} else {
			fmt.Println("Failed to get token.")
			return
		}
	}

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
		r.Config.Metadata.Format.ArtistFolder = strings.NewReplacer(
			"{UrlArtistName}", r.LimitString(urlArtistName),
			"{ArtistId}", urlArtistID,
		).Replace(r.Config.Metadata.Format.ArtistFolder)
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
				if r.Config.General.LiteServer == "" {
					fmt.Println(": lite-server is not set, skip MV dl")
					r.State.Counter.Success++
					continue
				}
				mvSaveDir := strings.NewReplacer(
					"{ArtistName}", "",
					"{UrlArtistName}", "",
					"{ArtistId}", "",
				).Replace(r.Config.Metadata.Format.ArtistFolder)
				if mvSaveDir != "" {
					mvSaveDir = filepath.Join(r.Config.Paths.MV, forbiddenNames.ReplaceAllString(mvSaveDir, "_"))
				} else {
					mvSaveDir = r.Config.Paths.MV
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
				err := r.ripSong(songId, token, storefront, r.Config.General.MediaUserToken)
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
				err := r.ripAlbum(albumId, token, storefront, r.Config.General.MediaUserToken, urlArg_i)
				if err != nil {
					fmt.Println("Failed to rip album:", err)
				}
			} else if strings.Contains(urlRaw, "/playlist/") {
				fmt.Println("Playlist")
				storefront, albumId = checkUrl(urlRaw, "playlist")
				err := r.ripPlaylist(albumId, token, storefront, r.Config.General.MediaUserToken)
				if err != nil {
					fmt.Println("Failed to rip playlist:", err)
				}
			} else if strings.Contains(urlRaw, "/station/") {
				fmt.Printf("Station")
				storefront, albumId = checkUrl(urlRaw, "station")
				if len(r.Config.General.MediaUserToken) <= 50 {
					fmt.Println(": meida-user-token is not set, skip station dl")
					continue
				}
				err := r.ripStation(albumId, token, storefront, r.Config.General.MediaUserToken)
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
		} else if r.Config.General.ExitOnError {
			fmt.Println("Error detected, exiting...")
			os.Exit(1)
		} else {
			fmt.Println("Error detected, press Enter to try again...")
			fmt.Scanln()
			fmt.Println("Start trying again...")
		}

		r.State.Counter = Counter{}
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

func getProgName(arg0 string) string {
	if arg0 == "" {
		return "amdl"
	}
	if strings.Contains(arg0, "go-build") {
		return "go run main.go"
	}
	base := filepath.Base(arg0)
	if base != "" && base != "." && base != "/" && base != "\\" {
		return base
	}
	return "amdl"
}

func progName() string {
	if len(os.Args) > 0 {
		return getProgName(os.Args[0])
	}
	return "amdl"
}

