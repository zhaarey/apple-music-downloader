package mv

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	stdbits "math/bits"
	"os"
	"path/filepath"
	"sort"

	defrag "amdl/internal/media/defrag"

	"github.com/Eyevinn/mp4ff/mp4"
)

const (
	videoTrackID = uint32(1)
	audioTrackID = uint32(2)

	baseMediaMajorBrand   = "isom"
	baseMediaMinorVersion = uint32(0x200)
)

var baseMediaCompatibleBrands = []string{"isom", "iso4"}

type streamInput struct {
	file       *os.File
	parsed     *mp4.File
	trak       *mp4.TrakBox
	trex       *mp4.TrexBox
	oldTrackID uint32
	timescale  uint32
	duration   uint64
}

type fragmentRef struct {
	input      *streamInput
	fragment   *mp4.Fragment
	traf       *mp4.TrafBox
	decodeTime uint64
	duration   uint64
	kind       int
}

// Mux merges separate video and audio fragmented MP4 streams and writes a
// progressive MP4 suitable for go-mp4tag. Sample payloads are copied without
// decoding or re-encoding.
func Mux(videoPath, audioPath, outputPath string) error {
	video, err := openFragmentedStream(videoPath, "video")
	if err != nil {
		return fmt.Errorf("open video stream: %w", err)
	}
	defer video.close()

	audio, err := openFragmentedStream(audioPath, "audio")
	if err != nil {
		return fmt.Errorf("open audio stream: %w", err)
	}
	defer audio.close()

	videoFragments, err := collectFragments(video, 0)
	if err != nil {
		return fmt.Errorf("collect video fragments: %w", err)
	}
	audioFragments, err := collectFragments(audio, 1)
	if err != nil {
		return fmt.Errorf("collect audio fragments: %w", err)
	}

	fragments := make([]*fragmentRef, 0, len(videoFragments)+len(audioFragments))
	fragments = append(fragments, videoFragments...)
	fragments = append(fragments, audioFragments...)
	sort.SliceStable(fragments, func(i, j int) bool {
		return fragmentBefore(fragments[i], fragments[j])
	})

	if err := mergeInit(video, audio); err != nil {
		return fmt.Errorf("merge init segments: %w", err)
	}
	if err := setMovieDurations(video, audio); err != nil {
		return fmt.Errorf("set movie durations: %w", err)
	}

	if err := writeProgressive(video, fragments, outputPath); err != nil {
		return fmt.Errorf("write progressive MP4: %w", err)
	}
	return nil
}

func openFragmentedStream(path, wantType string) (*streamInput, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	parsed, err := mp4.DecodeFile(file, mp4.WithDecodeMode(mp4.DecModeLazyMdat))
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("decode fragmented MP4: %w", err)
	}
	if !parsed.IsFragmented() {
		_ = file.Close()
		return nil, errors.New("input is not a fragmented MP4")
	}
	if parsed.Ftyp == nil {
		_ = file.Close()
		return nil, errors.New("input has no ftyp box")
	}
	if parsed.Init == nil || parsed.Init.Moov == nil {
		_ = file.Close()
		return nil, errors.New("input has no initialization moov")
	}

	moov := parsed.Init.Moov
	if len(moov.Traks) != 1 {
		_ = file.Close()
		return nil, fmt.Errorf("expected exactly one track, got %d", len(moov.Traks))
	}
	if moov.Mvex == nil {
		_ = file.Close()
		return nil, errors.New("fragmented MP4 has no mvex box")
	}

	trak := moov.Traks[0]
	if trak.Tkhd == nil || trak.Mdia == nil || trak.Mdia.Mdhd == nil || trak.Mdia.Hdlr == nil {
		_ = file.Close()
		return nil, errors.New("track is missing required header boxes")
	}
	if trak.Mdia.Mdhd.Timescale == 0 {
		_ = file.Close()
		return nil, errors.New("track timescale is zero")
	}

	handlerType := trak.Mdia.Hdlr.HandlerType
	gotType := ""
	switch handlerType {
	case "vide":
		gotType = "video"
	case "soun":
		gotType = "audio"
	default:
		_ = file.Close()
		return nil, fmt.Errorf("unsupported track handler %q", handlerType)
	}
	if gotType != wantType {
		_ = file.Close()
		return nil, fmt.Errorf("expected %s track, got %s", wantType, gotType)
	}

	var trex *mp4.TrexBox
	for _, candidate := range moov.Mvex.Trexs {
		if candidate.TrackID == trak.Tkhd.TrackID {
			trex = candidate
			break
		}
	}
	if trex == nil {
		_ = file.Close()
		return nil, fmt.Errorf("no trex for track %d", trak.Tkhd.TrackID)
	}

	return &streamInput{
		file:       file,
		parsed:     parsed,
		trak:       trak,
		trex:       trex,
		oldTrackID: trak.Tkhd.TrackID,
		timescale:  trak.Mdia.Mdhd.Timescale,
	}, nil
}

func (s *streamInput) close() {
	if s != nil && s.file != nil {
		_ = s.file.Close()
	}
}

func collectFragments(input *streamInput, kind int) ([]*fragmentRef, error) {
	var fragments []*fragmentRef

	for _, segment := range input.parsed.Segments {
		for _, fragment := range segment.Fragments {
			if fragment.Moof == nil || fragment.Mdat == nil {
				return nil, errors.New("fragment is missing moof or mdat")
			}
			if len(fragment.Moof.Trafs) != 1 {
				return nil, fmt.Errorf("expected one traf per fragment, got %d", len(fragment.Moof.Trafs))
			}

			traf := fragment.Moof.Trafs[0]
			if traf.Tfhd == nil || traf.Tfdt == nil {
				return nil, errors.New("traf is missing tfhd or tfdt")
			}
			if traf.Tfhd.TrackID != input.oldTrackID {
				return nil, fmt.Errorf("fragment track ID %d does not match init track ID %d", traf.Tfhd.TrackID, input.oldTrackID)
			}
			if len(traf.Truns) == 0 {
				return nil, errors.New("traf has no trun boxes")
			}

			var duration uint64
			for _, trun := range traf.Truns {
				duration += trun.AddSampleDefaultValues(traf.Tfhd, input.trex)
			}
			fragments = append(fragments, &fragmentRef{
				input:      input,
				fragment:   fragment,
				traf:       traf,
				decodeTime: traf.Tfdt.BaseMediaDecodeTime(),
				duration:   duration,
				kind:       kind,
			})
		}
	}

	if len(fragments) == 0 {
		return nil, errors.New("no media fragments found")
	}

	minDecodeTime := fragments[0].decodeTime
	for _, fragment := range fragments[1:] {
		if fragment.decodeTime < minDecodeTime {
			minDecodeTime = fragment.decodeTime
		}
	}

	for _, fragment := range fragments {
		fragment.decodeTime -= minDecodeTime
		endTime := fragment.decodeTime + fragment.duration
		if endTime > input.duration {
			input.duration = endTime
		}
	}
	return fragments, nil
}

func fragmentBefore(a, b *fragmentRef) bool {
	// Compare decodeTime/timescale exactly without floating point.
	hiA, loA := stdbits.Mul64(a.decodeTime, uint64(b.input.timescale))
	hiB, loB := stdbits.Mul64(b.decodeTime, uint64(a.input.timescale))
	if hiA != hiB {
		return hiA < hiB
	}
	if loA != loB {
		return loA < loB
	}
	return a.kind < b.kind
}

func mergeInit(video, audio *streamInput) error {
	moov := video.parsed.Init.Moov
	if moov == nil || moov.Mvex == nil {
		return errors.New("video init has no moov/mvex")
	}

	video.trak.Tkhd.TrackID = videoTrackID
	video.trex.TrackID = videoTrackID
	audio.trak.Tkhd.TrackID = audioTrackID
	audio.trex.TrackID = audioTrackID

	moov.AddChild(audio.trak)
	moov.Mvex.AddChild(audio.trex)
	return nil
}

func setMovieDurations(video, audio *streamInput) error {
	mvhd := video.parsed.Init.Moov.Mvhd
	if mvhd == nil || mvhd.Timescale == 0 {
		return errors.New("video moov has no valid mvhd")
	}

	videoDuration := scaleDuration(video.duration, video.timescale, mvhd.Timescale)
	audioDuration := scaleDuration(audio.duration, audio.timescale, mvhd.Timescale)
	movieDuration := videoDuration
	if audioDuration > movieDuration {
		movieDuration = audioDuration
	}

	const maxUint32 = uint64(^uint32(0))
	if mvhd.Version == 0 && movieDuration > maxUint32 {
		mvhd.Version = 1
	}
	mvhd.Duration = movieDuration
	if mvhd.NextTrackID <= audioTrackID {
		mvhd.NextTrackID = audioTrackID + 1
	}

	if video.trak.Tkhd.Version == 0 && videoDuration > maxUint32 {
		video.trak.Tkhd.Version = 1
	}
	if audio.trak.Tkhd.Version == 0 && audioDuration > maxUint32 {
		audio.trak.Tkhd.Version = 1
	}
	video.trak.Tkhd.Duration = videoDuration
	audio.trak.Tkhd.Duration = audioDuration

	if mehd := video.parsed.Init.Moov.Mvex.Mehd; mehd != nil {
		const maxInt64 = uint64(^uint64(0) >> 1)
		if movieDuration > maxInt64 {
			return errors.New("movie duration exceeds int64")
		}
		if mehd.Version == 0 && movieDuration > maxUint32 {
			mehd.Version = 1
		}
		mehd.FragmentDuration = int64(movieDuration)
	}
	return nil
}

func scaleDuration(duration uint64, fromTimescale, toTimescale uint32) uint64 {
	if duration == 0 || fromTimescale == toTimescale {
		return duration
	}
	divisor := uint64(fromTimescale)
	return (duration*uint64(toTimescale) + divisor - 1) / divisor
}

func writeProgressive(video *streamInput, fragments []*fragmentRef, outputPath string) error {
	dir := filepath.Dir(outputPath)
	base := filepath.Base(outputPath)
	tmp, err := os.CreateTemp(dir, "."+base+".mux-*.mp4")
	if err != nil {
		return fmt.Errorf("create temp output: %w", err)
	}
	tmpPath := tmp.Name()
	closed := false
	defer func() {
		if !closed {
			_ = tmp.Close()
		}
		_ = os.Remove(tmpPath)
	}()

	writer := bufio.NewWriterSize(tmp, 4*1024*1024)
	if err := video.parsed.Ftyp.Encode(writer); err != nil {
		return fmt.Errorf("write ftyp: %w", err)
	}
	if err := video.parsed.Init.Moov.Encode(writer); err != nil {
		return fmt.Errorf("write merged moov: %w", err)
	}

	for i, fragment := range fragments {
		sequenceNumber := uint32(i + 1)
		if err := writeFragment(writer, fragment, sequenceNumber); err != nil {
			return fmt.Errorf("write fragment %d: %w", i+1, err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush merged MP4: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync merged MP4: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close merged MP4: %w", err)
	}
	closed = true

	if err := defrag.DefragmentMP4WithFtyp(tmpPath, baseMediaMajorBrand, baseMediaMinorVersion, baseMediaCompatibleBrands); err != nil {
		return fmt.Errorf("defragment merged MP4: %w", err)
	}
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("move output into place: %w", err)
	}
	return nil
}

func writeFragment(w io.Writer, ref *fragmentRef, sequenceNumber uint32) error {
	fragment := ref.fragment
	if fragment.Moof == nil || fragment.Moof.Mfhd == nil || fragment.Mdat == nil {
		return errors.New("fragment is missing moof/mfhd/mdat")
	}

	ref.traf.Tfhd.TrackID = trackIDForKind(ref.kind)
	ref.traf.Tfdt.SetBaseMediaDecodeTime(ref.decodeTime)
	fragment.Moof.Mfhd.SequenceNumber = sequenceNumber
	if err := setFragmentDataOffsets(fragment); err != nil {
		return err
	}

	for _, child := range fragment.Children {
		if _, isMdat := child.(*mp4.MdatBox); isMdat {
			continue
		}
		if err := child.Encode(w); err != nil {
			return fmt.Errorf("write fragment metadata: %w", err)
		}
	}

	if err := fragment.Mdat.Encode(w); err != nil {
		return fmt.Errorf("write mdat header: %w", err)
	}

	payloadStart := int64(fragment.Mdat.StartPos + fragment.Mdat.HeaderSize())
	payloadSize := int64(fragment.Mdat.GetLazyDataSize())
	if payloadSize < 0 {
		return errors.New("mdat payload exceeds int64")
	}
	written, err := fragment.Mdat.CopyData(payloadStart, payloadSize, ref.input.file, w)
	if err != nil {
		return fmt.Errorf("copy mdat payload: %w", err)
	}
	if written != payloadSize {
		return fmt.Errorf("copied %d mdat bytes, expected %d", written, payloadSize)
	}
	return nil
}

func setFragmentDataOffsets(fragment *mp4.Fragment) error {
	const maxInt32 = int64(^uint32(0) >> 1)

	// Force the data-offset field to be present before calculating moof size.
	for _, traf := range fragment.Moof.Trafs {
		for _, trun := range traf.Truns {
			trun.Flags |= mp4.TrunDataOffsetPresentFlag
		}
	}

	dataOffset := int64(fragment.Moof.Size() + fragment.Mdat.HeaderSize())
	if dataOffset > maxInt32 {
		return errors.New("fragment data offset exceeds int32")
	}
	for _, traf := range fragment.Moof.Trafs {
		for _, trun := range traf.Truns {
			trun.DataOffset = int32(dataOffset)
			dataOffset += int64(trun.SizeOfData())
			if dataOffset > maxInt32 {
				return errors.New("fragment data offset exceeds int32")
			}
		}
	}
	return nil
}

func trackIDForKind(kind int) uint32 {
	if kind == 1 {
		return audioTrackID
	}
	return videoTrackID
}
