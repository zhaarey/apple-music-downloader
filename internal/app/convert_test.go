package app

import (
	"slices"
	"strings"
	"testing"

	"amdl/internal/config"
)

// indexSeq returns the index of the first occurrence of needle inside hay,
// treating needle as a contiguous argument sequence, or -1 if absent.
func indexSeq(hay, needle []string) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		if slices.Equal(hay[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

func newConvertRunner(cfg config.ConvertConfig) *Runner {
	return &Runner{Config: config.Config{Convert: cfg}}
}

func TestBuildFFmpegArgs(t *testing.T) {
	tests := []struct {
		name    string
		convert config.ConvertConfig
		target  string
		want    [][]string
		notWant [][]string
	}{
		{
			name:    "flac keeps metadata and cover",
			convert: config.ConvertConfig{WithMetadata: true},
			target:  "flac",
			want: [][]string{
				{"-map_metadata", "0"},
				{"-c:a", "flac"},
				{"-map", "0"},
				{"-c:v", "copy"},
				{"-disposition:v", "attached_pic"},
			},
		},
		{
			name:    "mp3 defaults to LAME V2 and keeps cover as jpg",
			convert: config.ConvertConfig{WithMetadata: true},
			target:  "mp3",
			want: [][]string{
				{"-c:a", "libmp3lame"},
				{"-qscale:a", "2"},
				{"-map", "0"},
				{"-c:v", "copy"},
				{"-disposition:v", "attached_pic"},
				{"-id3v2_version", "3"},
				{"-metadata:s:v", "title=Album cover"},
				{"-metadata:s:v", "comment=Cover (front)"},
			},
		},
		{
			name:    "mp3 CBR quality switches to -b:a",
			convert: config.ConvertConfig{WithMetadata: true, Mp3Quality: "320k"},
			target:  "mp3",
			want: [][]string{
				{"-c:a", "libmp3lame"},
				{"-b:a", "320k"},
			},
			notWant: [][]string{{"-qscale:a"}},
		},
		{
			name:    "opus stays audio only and takes the configured bitrate",
			convert: config.ConvertConfig{WithMetadata: true, OpusBitrate: "160k"},
			target:  "opus",
			want: [][]string{
				{"-vn"},
				{"-c:a", "libopus"},
				{"-b:a", "160k"},
				{"-vbr", "on"},
			},
			// ogg/opus muxers reject attached pictures outright, so no cover
			// mapping may be emitted for this target.
			notWant: [][]string{{"-map"}, {"-c:v"}, {"-disposition:v"}},
		},
		{
			name:    "opus defaults to 192k",
			convert: config.ConvertConfig{},
			target:  "opus",
			want:    [][]string{{"-b:a", "192k"}},
		},
		{
			name:    "wav is audio only",
			convert: config.ConvertConfig{WithMetadata: true},
			target:  "wav",
			want:    [][]string{{"-c:a", "pcm_s16le"}},
			notWant: [][]string{
				{"-map"},
				{"-c:v"},
			},
		},
		{
			name:    "metadata can be dropped",
			convert: config.ConvertConfig{WithMetadata: false},
			target:  "flac",
			want:    [][]string{{"-map_metadata", "-1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newConvertRunner(tt.convert)
			args, err := r.buildFFmpegArgs("in.m4a", "out."+tt.target, tt.target, "")
			if err != nil {
				t.Fatalf("buildFFmpegArgs(%s) error: %v", tt.target, err)
			}
			for _, want := range tt.want {
				if indexSeq(args, want) < 0 {
					t.Errorf("args missing %v\n got: %v", want, args)
				}
			}
			for _, notWant := range tt.notWant {
				if idx := indexSeq(args, notWant); idx >= 0 {
					t.Errorf("args must not contain %v (found at %d)\n got: %v", notWant, idx, args)
				}
			}
			if last := args[len(args)-1]; last != "out."+tt.target {
				t.Errorf("output path must be last, got %q", last)
			}
		})
	}
}

func TestBuildFFmpegArgsTwiceIsStable(t *testing.T) {
	// The shared cover-art/tag argument builders must not be mutated by append.
	r := newConvertRunner(config.ConvertConfig{WithMetadata: true})
	first, err := r.buildFFmpegArgs("in.m4a", "out.mp3", "mp3", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := r.buildFFmpegArgs("in.m4a", "out.mp3", "mp3", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(first, second) {
		t.Fatalf("args differ between runs:\n %v\n %v", first, second)
	}
}

func TestBuildFFmpegArgsAppendsExtraArgs(t *testing.T) {
	r := newConvertRunner(config.ConvertConfig{})
	args, err := r.buildFFmpegArgs("in.m4a", "out.opus", "opus", "-af volume=0.9 -ar 44100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"-af", "volume=0.9", "-ar", "44100", "out.opus"}
	if indexSeq(args, want) < 0 {
		t.Fatalf("extra args not appended in order:\n got: %v", args)
	}
}

func TestBuildFFmpegArgsUnsupportedFormat(t *testing.T) {
	r := newConvertRunner(config.ConvertConfig{})
	_, err := r.buildFFmpegArgs("in.m4a", "out.aiff", "aiff", "")
	if err == nil {
		t.Fatal("expected an error for an unsupported convert-format")
	}
	if !strings.Contains(err.Error(), "aiff") {
		t.Fatalf("error should name the offending format, got %v", err)
	}
}

func TestMP3QualityArgs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty falls back to VBR V2", in: "", want: []string{"-qscale:a", "2"}},
		{name: "blank falls back to VBR V2", in: "   ", want: []string{"-qscale:a", "2"}},
		{name: "vbr level", in: "0", want: []string{"-qscale:a", "0"}},
		{name: "vbr level padded", in: " 7 ", want: []string{"-qscale:a", "7"}},
		{name: "cbr bitrate", in: "320k", want: []string{"-b:a", "320k"}},
		{name: "cbr bitrate uppercase", in: "256K", want: []string{"-b:a", "256K"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mp3QualityArgs(tt.in); !slices.Equal(got, tt.want) {
				t.Fatalf("mp3QualityArgs(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestOpusBitrateArgs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty falls back to 192k", in: "", want: []string{"-b:a", "192k", "-vbr", "on"}},
		{name: "padded", in: " 160k ", want: []string{"-b:a", "160k", "-vbr", "on"}},
		{name: "custom", in: "96k", want: []string{"-b:a", "96k", "-vbr", "on"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := opusBitrateArgs(tt.in); !slices.Equal(got, tt.want) {
				t.Fatalf("opusBitrateArgs(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
