package app

import (
	"bytes"
	"fmt"
	"amdl/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (r *Runner) isLossySource(ext string, codec string) bool {
	ext = strings.ToLower(ext)
	if ext == ".m4a" && (codec == "AAC" || strings.Contains(codec, "AAC") || strings.Contains(codec, "ATMOS")) {
		return true
	}
	if ext == ".mp3" || ext == ".opus" || ext == ".ogg" {
		return true
	}
	return false
}

// CONVERSION FEATURE: Build ffmpeg arguments for desired target.

func (r *Runner) buildFFmpegArgs(inPath, outPath, targetFmt, extraArgs string) ([]string, error) {
	args := []string{"-y", "-i", inPath, "-loglevel", "error", "-map_metadata"}
	if r.Config.Convert.WithMetadata {
		args = append(args, "0")
	} else {
		args = append(args, "-1")
	}
	switch targetFmt {
	case "flac":
		args = append(args, "-c:a", "flac")
		args = append(args, coverArtArgs()...)
	case "mp3":
		args = append(args, "-c:a", "libmp3lame")
		args = append(args, mp3QualityArgs(r.Config.Convert.Mp3Quality)...)
		args = append(args, coverArtArgs()...)
		args = append(args, mp3TagArgs()...)
	case "opus":
		// Ogg Opus cannot carry an attached picture, so drop any video stream
		// explicitly rather than relying on ffmpeg's stream auto-selection.
		args = append(args, "-vn", "-c:a", "libopus")
		args = append(args, opusBitrateArgs(r.Config.Convert.OpusBitrate)...)
	case "wav":
		// WAV has no attached-picture support either; copy audio only.
		args = append(args, "-c:a", "pcm_s16le")
	case "copy":
		// Just container copy (probably pointless for same container)
		args = append(args, "-c", "copy")
	default:
		return nil, fmt.Errorf("unsupported convert-format: %s", targetFmt)
	}
	if extraArgs != "" {
		// naive split; for complex quoting you could enhance
		args = append(args, strings.Fields(extraArgs)...)
	}
	args = append(args, outPath)
	return args, nil
}

// coverArtArgs maps every stream and copies the embedded cover (attached_pic)
// over verbatim, so album art survives the conversion instead of being
// re-encoded (ffmpeg turns the source jpg into a png on its own) or dropped.
//
// Only formats whose muxer supports attached pictures may use this. ffmpeg's
// ogg muxer, including the ogg-based opus muxer, rejects any picture stream
// with "Unsupported codec id in stream N" and then writes nothing at all, so
// opus has to stay audio only.
func coverArtArgs() []string {
	return []string{"-map", "0", "-c:v", "copy", "-disposition:v", "attached_pic"}
}

// mp3TagArgs makes the MP3 tag readable for players that are picky about ID3:
// ffmpeg's mp3 muxer writes ID3v2.4 by default, which Windows Explorer refuses
// to read a cover from, and it leaves the APIC picture type at "Other". The
// attached picture's comment maps to that picture type, its title to the APIC
// description.
func mp3TagArgs() []string {
	return []string{
		"-id3v2_version", "3",
		"-metadata:s:v", "title=Album cover",
		"-metadata:s:v", "comment=Cover (front)",
	}
}

// mp3QualityArgs translates the configured MP3 quality into ffmpeg args.
// A value ending in 'k' (e.g. "320k") selects CBR via -b:a; otherwise the
// value is treated as a LAME VBR level (0-9) via -qscale:a. Empty falls back
// to VBR V2 (~190 kbps) to preserve historical behavior.
func mp3QualityArgs(q string) []string {
	q = strings.TrimSpace(q)
	if q == "" {
		return []string{"-qscale:a", "2"}
	}
	if strings.HasSuffix(strings.ToLower(q), "k") {
		return []string{"-b:a", q}
	}
	return []string{"-qscale:a", q}
}

// opusBitrateArgs translates the configured Opus bitrate into ffmpeg args.
// Accepts values like "192k" or "160k". Empty falls back to 192k to preserve
// historical behavior.
func opusBitrateArgs(b string) []string {
	b = strings.TrimSpace(b)
	if b == "" {
		return []string{"-b:a", "192k", "-vbr", "on"}
	}
	return []string{"-b:a", b, "-vbr", "on"}
}

// CONVERSION FEATURE: Perform conversion if enabled.

func (r *Runner) convertIfNeeded(track *model.Track) {
	if !r.Config.Convert.AfterDownload {
		return
	}
	if r.Config.Convert.Format == "" {
		return
	}
	srcPath := track.SavePath
	if srcPath == "" {
		return
	}
	ext := strings.ToLower(filepath.Ext(srcPath))
	targetFmt := strings.ToLower(r.Config.Convert.Format)

	// Map extension for output
	if targetFmt == "copy" {
		fmt.Println("Convert (copy) requested; skipping because it produces no new format.")
		return
	}

	if r.Config.Convert.SkipIfSourceMatch {
		if ext == "."+targetFmt {
			fmt.Printf("Conversion skipped (already %s)\n", targetFmt)
			return
		}
	}

	outBase := strings.TrimSuffix(srcPath, ext)
	outPath := outBase + "." + targetFmt

	// Handle lossy -> lossless cases: optionally skip or warn
	if (targetFmt == "flac" || targetFmt == "wav") && r.isLossySource(ext, track.Codec) {
		if r.Config.Convert.SkipLossyToLossless {
			fmt.Println("Skipping conversion: source appears lossy and target is lossless; configured to skip.")
			return
		}
		if r.Config.Convert.WarnLossyToLossless {
			fmt.Println("Warning: Converting lossy source to lossless container will not improve quality.")
		}
	}

	if _, err := exec.LookPath(r.Config.Convert.FFmpegPath); err != nil {
		fmt.Printf("ffmpeg not found at '%s'; skipping conversion.\n", r.Config.Convert.FFmpegPath)
		return
	}

	args, err := r.buildFFmpegArgs(srcPath, outPath, targetFmt, r.Config.Convert.ExtraArgs)
	if err != nil {
		fmt.Println("Conversion config error:", err)
		return
	}

	fmt.Printf("Converting -> %s ...\n", targetFmt)
	cmd := exec.Command(r.Config.Convert.FFmpegPath, args...)
	var stderr bytes.Buffer
	if r.Config.Convert.CheckBadALAC {
		cmd.Stderr = &stderr
	} else {
		cmd.Stderr = nil
	}
	cmd.Stdout = nil
	start := time.Now()
	if err := cmd.Run(); err != nil {
		fmt.Println("Conversion failed:", err)
		// leave original
		return
	}
	if r.Config.Convert.CheckBadALAC && stderr.Len() > 0 {
		fmt.Print("Detected ALAC Error.", "\n")
		if r.Config.Convert.DeleteBadALAC {
			delPath := strings.TrimSuffix(srcPath, "m4a") + targetFmt
			logPath := strings.TrimSuffix(srcPath, "m4a") + "log"
			if err := os.Remove(delPath); err != nil {
				fmt.Println("Failed to remove convert:", err)
			} else {
				fmt.Println("Convert removed due to the bad ALAC.")
				log := stderr
				err = os.WriteFile(logPath, log.Bytes(), 0644)
				if err != nil {
					fmt.Println("Convert logs:", log)
				} else {
					fmt.Println("Convert logs are stored in:", logPath)
				}
			}
		}
	} else {
		fmt.Printf("Conversion completed in %s: %s\n", time.Since(start).Truncate(time.Millisecond), filepath.Base(outPath))

		if !r.Config.Convert.KeepOriginal {
			if err := os.Remove(srcPath); err != nil {
				fmt.Println("Failed to remove original after conversion:", err)
			} else {
				fmt.Println("Original removed.")
			}

		}
		track.SavePath = outPath
		track.SaveName = filepath.Base(outPath)
	}

}
