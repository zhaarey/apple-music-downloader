package app

import (
	"slices"
	"testing"

	"amdl/internal/config"
)

// indexSeq returns the index of the first occurrence of needle inside hay,
// treating needle as a contiguous argument sequence, or -1 if absent.
func indexSeq(hay, needle []string) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if slices.Equal(hay[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

func TestBuildFFmpegArgsCover(t *testing.T) {
	cover := [][]string{
		{"-map", "0"},
		{"-c:v", "copy"},
		{"-disposition:v", "attached_pic"},
	}
	tests := []struct {
		target  string
		want    [][]string
		notWant [][]string
	}{
		{target: "flac", want: cover},
		{
			target: "mp3",
			want: append([][]string{
				{"-id3v2_version", "3"},
				{"-metadata:s:v", "comment=Cover (front)"},
			}, cover...),
		},
		// ogg/opus and wav muxers reject attached pictures outright.
		{target: "opus", want: [][]string{{"-vn"}}, notWant: [][]string{{"-map"}, {"-c:v"}}},
		{target: "wav", want: [][]string{{"-vn"}}, notWant: [][]string{{"-map"}, {"-c:v"}}},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			r := &Runner{Config: config.Config{Convert: config.ConvertConfig{WithMetadata: true}}}
			args, err := r.buildFFmpegArgs("ffmpeg", "in.m4a", "out."+tt.target, tt.target, "")
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range tt.want {
				if indexSeq(args, w) < 0 {
					t.Errorf("args missing %v\n got: %v", w, args)
				}
			}
			for _, nw := range tt.notWant {
				if indexSeq(args, nw) >= 0 {
					t.Errorf("args must not contain %v\n got: %v", nw, args)
				}
			}
			if args[len(args)-1] != "out."+tt.target {
				t.Errorf("output path must be last, got %v", args)
			}
		})
	}
}

func TestBuildFFmpegArgsIsStable(t *testing.T) {
	r := &Runner{Config: config.Config{Convert: config.ConvertConfig{WithMetadata: true}}}
	first, _ := r.buildFFmpegArgs("ffmpeg", "in.m4a", "out.mp3", "mp3", "")
	second, _ := r.buildFFmpegArgs("ffmpeg", "in.m4a", "out.mp3", "mp3", "")
	if !slices.Equal(first, second) {
		t.Fatalf("args differ between runs:\n %v\n %v", first, second)
	}
}
