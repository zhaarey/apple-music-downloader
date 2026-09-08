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

func (r *Runner) buildFFmpegArgs(ffmpegPath, inPath, outPath, targetFmt, extraArgs string) ([]string, error) {
	args := []string{"-y", "-i", inPath, "-loglevel", "error", "-map_metadata"}
	if r.Config.ConvertWithMetadata {
		args = append(args, "0")
	} else {
		args = append(args, "-1")
	}
	switch targetFmt {
	case "flac":
		// Map all streams and copy the embedded cover (attached_pic) so album
		// art survives the ALAC(.m4a) -> FLAC transcode. Without -map 0 / -c:v copy
		// ffmpeg only keeps the audio stream and the artwork is silently dropped.
		args = append(args, "-map", "0", "-c:a", "flac", "-c:v", "copy", "-disposition:v", "attached_pic")
	case "mp3":
		// VBR quality 2 ~ high quality
		args = append(args, "-c:a", "libmp3lame", "-qscale:a", "2")
	case "opus":
		// Medium/high quality
		args = append(args, "-c:a", "libopus", "-b:a", "192k", "-vbr", "on")
	case "wav":
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

// CONVERSION FEATURE: Perform conversion if enabled.

func (r *Runner) convertIfNeeded(track *model.Track) {
	if !r.Config.ConvertAfterDownload {
		return
	}
	if r.Config.ConvertFormat == "" {
		return
	}
	srcPath := track.SavePath
	if srcPath == "" {
		return
	}
	ext := strings.ToLower(filepath.Ext(srcPath))
	targetFmt := strings.ToLower(r.Config.ConvertFormat)

	// Map extension for output
	if targetFmt == "copy" {
		fmt.Println("Convert (copy) requested; skipping because it produces no new format.")
		return
	}

	if r.Config.ConvertSkipIfSourceMatch {
		if ext == "."+targetFmt {
			fmt.Printf("Conversion skipped (already %s)\n", targetFmt)
			return
		}
	}

	outBase := strings.TrimSuffix(srcPath, ext)
	outPath := outBase + "." + targetFmt

	// Handle lossy -> lossless cases: optionally skip or warn
	if (targetFmt == "flac" || targetFmt == "wav") && r.isLossySource(ext, track.Codec) {
		if r.Config.ConvertSkipLossyToLossless {
			fmt.Println("Skipping conversion: source appears lossy and target is lossless; configured to skip.")
			return
		}
		if r.Config.ConvertWarnLossyToLossless {
			fmt.Println("Warning: Converting lossy source to lossless container will not improve quality.")
		}
	}

	if _, err := exec.LookPath(r.Config.FFmpegPath); err != nil {
		fmt.Printf("ffmpeg not found at '%s'; skipping conversion.\n", r.Config.FFmpegPath)
		return
	}

	args, err := r.buildFFmpegArgs(r.Config.FFmpegPath, srcPath, outPath, targetFmt, r.Config.ConvertExtraArgs)
	if err != nil {
		fmt.Println("Conversion config error:", err)
		return
	}

	fmt.Printf("Converting -> %s ...\n", targetFmt)
	cmd := exec.Command(r.Config.FFmpegPath, args...)
	var stderr bytes.Buffer
	if r.Config.ConvertCheckBadALAC {
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
	if r.Config.ConvertCheckBadALAC && stderr.Len() > 0 {
		fmt.Print("Detected ALAC Error.", "\n")
		if r.Config.ConvertDeleteBadALAC {
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

		if !r.Config.ConvertKeepOriginal {
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
