package splice

import (
	"encoding/binary"
	"io"
	"math"
	"os"
	"os/exec"
	"sync"

	appconfig "main/internal/config"
	"main/internal/media"
)

const DefaultBinCount = 3000

// PeakBin is min/max amplitude for one waveform column.
type PeakBin struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// WaveformPeaks holds downsampled waveform data for the UI.
type WaveformPeaks struct {
	Bins       []PeakBin `json:"bins"`
	DurationMs int       `json:"duration_ms"`
}

// MasterProbe is returned when loading a master file.
type MasterProbe struct {
	DurationMs int    `json:"duration_ms"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Summary    string `json:"summary"`
}

var peakCache sync.Map

type peakCacheKey struct {
	path     string
	mtime    int64
	binCount int
}

// ProbeMaster reads master file metadata via ffprobe.
func ProbeMaster(ffmpegConfigured, path string) (MasterProbe, error) {
	info, err := media.ProbeSource(ffmpegConfigured, path)
	if err != nil {
		return MasterProbe{}, err
	}
	return MasterProbe{
		DurationMs: info.DurationMs,
		SampleRate: info.SampleRate,
		Channels:   info.Channels,
		Summary:    info.Summary,
	}, nil
}

// ExtractPeaks reads waveform peaks via ffmpeg PCM decode (cached by path/mtime).
func ExtractPeaks(ffmpegConfigured, path string, binCount int) (WaveformPeaks, error) {
	if binCount <= 0 {
		binCount = DefaultBinCount
	}
	stat, err := os.Stat(path)
	if err != nil {
		return WaveformPeaks{}, err
	}
	key := peakCacheKey{path: path, mtime: stat.ModTime().UnixNano(), binCount: binCount}
	if cached, ok := peakCache.Load(key); ok {
		return cached.(WaveformPeaks), nil
	}

	probe, err := ProbeMaster(ffmpegConfigured, path)
	if err != nil {
		return WaveformPeaks{}, err
	}

	ffmpeg := appconfig.FFmpegPath(ffmpegConfigured)
	cmd := exec.Command(ffmpeg,
		"-hide_banner", "-loglevel", "error",
		"-i", path,
		"-ac", "1",
		"-ar", "8000",
		"-f", "s16le",
		"pipe:1",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return WaveformPeaks{}, err
	}
	if err := cmd.Start(); err != nil {
		return WaveformPeaks{}, err
	}

	samples, err := io.ReadAll(stdout)
	_ = cmd.Wait()
	if err != nil {
		return WaveformPeaks{}, err
	}

	sampleCount := len(samples) / 2
	bins := make([]PeakBin, binCount)
	if sampleCount == 0 {
		result := WaveformPeaks{Bins: bins, DurationMs: probe.DurationMs}
		peakCache.Store(key, result)
		return result, nil
	}

	framesPerBin := sampleCount / binCount
	if framesPerBin < 1 {
		framesPerBin = 1
	}
	for b := 0; b < binCount; b++ {
		start := b * framesPerBin
		end := start + framesPerBin
		if end > sampleCount {
			end = sampleCount
		}
		if start >= sampleCount {
			break
		}
		minV, maxV := 0.0, 0.0
		for i := start; i < end; i++ {
			v := float64(int16(binary.LittleEndian.Uint16(samples[i*2:]))) / 32768.0
			if i == start {
				minV, maxV = v, v
			} else {
				minV = math.Min(minV, v)
				maxV = math.Max(maxV, v)
			}
		}
		bins[b] = PeakBin{Min: minV, Max: maxV}
	}

	result := WaveformPeaks{Bins: bins, DurationMs: probe.DurationMs}
	peakCache.Store(key, result)
	return result, nil
}
