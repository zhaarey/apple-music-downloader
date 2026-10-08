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
	file          *os.File
	parsed        *mp4.File
	trak          *mp4.TrakBox
	trex          *mp4.TrexBox
	oldTrackID    uint32
	outputTrackID uint32
	timescale     uint32
	duration      uint64
}

type fragmentRef struct {
	input      *streamInput
	fragment   *mp4.Fragment
	traf       *mp4.TrafBox
	dataRanges []dataRange
	dataSize   uint64
	decodeTime uint64
	duration   uint64
}

type dataRange struct {
	start int64
	size  int64
}

// Mux merges separate video and audio fragmented MP4 streams and writes a
// progressive MP4 suitable for go-mp4tag. Sample payloads are copied without
// decoding or re-encoding.
func Mux(videoPath, audioPath, outputPath string) error {
	videoTracks, err := openFragmentedStreams(videoPath, "video", true)
	if err != nil {
		return fmt.Errorf("open video stream: %w", err)
	}
	defer closeStreams(videoTracks)

	audioTracks, err := openFragmentedStreams(audioPath, "audio", false)
	if err != nil {
		return fmt.Errorf("open audio stream: %w", err)
	}
	defer closeStreams(audioTracks)

	video, err := primaryStream(videoTracks, "video")
	if err != nil {
		return err
	}
	audio, err := primaryStream(audioTracks, "audio")
	if err != nil {
		return err
	}

	tracks := make([]*streamInput, 0, len(videoTracks)+len(audioTracks))
	tracks = append(tracks, videoTracks...)
	tracks = append(tracks, audioTracks...)
	assignOutputTrackIDs(video, audio, tracks)

	fragments, err := collectFragments(tracks)
	if err != nil {
		return err
	}
	sort.SliceStable(fragments, func(i, j int) bool {
		return fragmentBefore(fragments[i], fragments[j])
	})

	if err := mergeInit(videoTracks, audioTracks); err != nil {
		return fmt.Errorf("merge init segments: %w", err)
	}
	if err := setMovieDurations(video, tracks); err != nil {
		return fmt.Errorf("set movie durations: %w", err)
	}

	if err := writeProgressive(video, fragments, outputPath); err != nil {
		return fmt.Errorf("write progressive MP4: %w", err)
	}
	return nil
}

func openFragmentedStreams(path, wantType string, keepSupplementary bool) ([]*streamInput, error) {
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
	if moov.Mvex == nil {
		_ = file.Close()
		return nil, errors.New("fragmented MP4 has no mvex box")
	}

	var streams []*streamInput
	var primary *streamInput
	for _, candidate := range moov.Traks {
		if candidate == nil || candidate.Tkhd == nil || candidate.Mdia == nil || candidate.Mdia.Mdhd == nil || candidate.Mdia.Hdlr == nil {
			continue
		}
		handlerType := candidate.Mdia.Hdlr.HandlerType
		isPrimary := (wantType == "video" && handlerType == "vide") || (wantType == "audio" && handlerType == "soun")
		if !isPrimary && !keepSupplementary {
			continue
		}
		if candidate.Mdia.Mdhd.Timescale == 0 {
			_ = file.Close()
			return nil, fmt.Errorf("track %d timescale is zero", candidate.Tkhd.TrackID)
		}

		var trex *mp4.TrexBox
		for _, trexCandidate := range moov.Mvex.Trexs {
			if trexCandidate != nil && trexCandidate.TrackID == candidate.Tkhd.TrackID {
				trex = trexCandidate
				break
			}
		}
		if trex == nil {
			_ = file.Close()
			return nil, fmt.Errorf("no trex for track %d", candidate.Tkhd.TrackID)
		}

		stream := &streamInput{
			file:       file,
			parsed:     parsed,
			trak:       candidate,
			trex:       trex,
			oldTrackID: candidate.Tkhd.TrackID,
			timescale:  candidate.Mdia.Mdhd.Timescale,
		}
		streams = append(streams, stream)
		if isPrimary {
			if primary != nil {
				_ = file.Close()
				return nil, fmt.Errorf("expected exactly one %s track, got multiple", wantType)
			}
			primary = stream
		}
	}
	if primary == nil {
		_ = file.Close()
		return nil, fmt.Errorf("no %s track found", wantType)
	}
	return streams, nil
}

func primaryStream(streams []*streamInput, wantType string) (*streamInput, error) {
	for _, stream := range streams {
		if stream == nil || stream.trak == nil || stream.trak.Mdia == nil || stream.trak.Mdia.Hdlr == nil {
			continue
		}
		handlerType := stream.trak.Mdia.Hdlr.HandlerType
		if (wantType == "video" && handlerType == "vide") || (wantType == "audio" && handlerType == "soun") {
			return stream, nil
		}
	}
	return nil, fmt.Errorf("no %s track found", wantType)
}

func assignOutputTrackIDs(video, audio *streamInput, tracks []*streamInput) {
	nextSupplementaryID := uint32(3)
	for _, stream := range tracks {
		switch stream {
		case video:
			stream.outputTrackID = videoTrackID
		case audio:
			stream.outputTrackID = audioTrackID
		default:
			stream.outputTrackID = nextSupplementaryID
			nextSupplementaryID++
		}
		stream.trak.Tkhd.TrackID = stream.outputTrackID
		stream.trex.TrackID = stream.outputTrackID
	}
}

func closeStreams(streams []*streamInput) {
	for _, stream := range streams {
		if stream != nil && stream.file != nil {
			_ = stream.file.Close()
		}
	}
}

func collectFragments(inputs []*streamInput) ([]*fragmentRef, error) {
	var fragments []*fragmentRef
	for _, input := range inputs {
		streamFragments, err := collectStreamFragments(input)
		if err != nil {
			return nil, fmt.Errorf("track %d: %w", input.oldTrackID, err)
		}
		fragments = append(fragments, streamFragments...)
	}
	if len(fragments) == 0 {
		return nil, errors.New("no media fragments found")
	}
	return fragments, nil
}

func collectStreamFragments(input *streamInput) ([]*fragmentRef, error) {
	var fragments []*fragmentRef

	for _, segment := range input.parsed.Segments {
		for _, fragment := range segment.Fragments {
			if fragment.Moof == nil || fragment.Mdat == nil {
				return nil, errors.New("fragment is missing moof or mdat")
			}

			traf, err := findTrackTraf(fragment.Moof, input.oldTrackID)
			if err != nil {
				return nil, err
			}
			if traf == nil {
				continue
			}
			if traf.Tfhd == nil || traf.Tfdt == nil {
				return nil, errors.New("traf is missing tfhd or tfdt")
			}
			if len(traf.Truns) == 0 {
				return nil, errors.New("traf has no trun boxes")
			}

			var duration uint64
			for _, trun := range traf.Truns {
				duration += trun.AddSampleDefaultValues(traf.Tfhd, input.trex)
			}
			dataRanges, dataSize, err := fragmentDataRanges(fragment, traf)
			if err != nil {
				return nil, err
			}

			fragments = append(fragments, &fragmentRef{
				input:      input,
				fragment:   fragment,
				traf:       traf,
				dataRanges: dataRanges,
				dataSize:   dataSize,
				decodeTime: traf.Tfdt.BaseMediaDecodeTime(),
				duration:   duration,
			})
		}
	}

	if len(fragments) == 0 {
		return nil, errors.New("no media fragments found for track")
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

func findTrackTraf(moof *mp4.MoofBox, trackID uint32) (*mp4.TrafBox, error) {
	var found *mp4.TrafBox
	for _, traf := range moof.Trafs {
		if traf == nil || traf.Tfhd == nil || traf.Tfhd.TrackID != trackID {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("multiple trafs found for track ID %d", trackID)
		}
		found = traf
	}
	return found, nil
}

func fragmentDataRanges(fragment *mp4.Fragment, traf *mp4.TrafBox) ([]dataRange, uint64, error) {
	baseOffset := fragment.Moof.StartPos
	if traf.Tfhd.HasBaseDataOffset() {
		baseOffset = traf.Tfhd.BaseDataOffset
	}

	mdatStart := int64(fragment.Mdat.PayloadAbsoluteOffset())
	mdatEnd := mdatStart + int64(fragment.Mdat.GetLazyDataSize())
	nextOffset := baseOffset
	var ranges []dataRange
	var totalSize uint64

	for _, trun := range traf.Truns {
		var size uint64
		for _, sample := range trun.Samples {
			size += uint64(sample.Size)
		}

		start := nextOffset
		if trun.HasDataOffset() {
			start = uint64(int64(baseOffset) + int64(trun.DataOffset))
		}
		end := start + size
		if start < uint64(mdatStart) || end > uint64(mdatEnd) {
			return nil, 0, errors.New("track sample data is outside its mdat box")
		}
		if size > 0 {
			ranges = append(ranges, dataRange{start: int64(start), size: int64(size)})
			totalSize += size
		}
		nextOffset = end
	}

	return ranges, totalSize, nil
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
	return a.input.outputTrackID < b.input.outputTrackID
}

func mergeInit(videoTracks, audioTracks []*streamInput) error {
	if len(videoTracks) == 0 {
		return errors.New("video init has no tracks")
	}
	moov := videoTracks[0].parsed.Init.Moov
	if moov == nil || moov.Mvex == nil {
		return errors.New("video init has no moov/mvex")
	}

	for _, audio := range audioTracks {
		moov.AddChild(audio.trak)
		moov.Mvex.AddChild(audio.trex)
	}
	return nil
}

func setMovieDurations(video *streamInput, tracks []*streamInput) error {
	mvhd := video.parsed.Init.Moov.Mvhd
	if mvhd == nil || mvhd.Timescale == 0 {
		return errors.New("video moov has no valid mvhd")
	}

	const maxUint32 = uint64(^uint32(0))

	var movieDuration uint64
	var maxTrackID uint32
	for _, stream := range tracks {
		trackDuration := scaleDuration(stream.duration, stream.timescale, mvhd.Timescale)
		if stream.trak.Tkhd.Version == 0 && trackDuration > maxUint32 {
			stream.trak.Tkhd.Version = 1
		}
		stream.trak.Tkhd.Duration = trackDuration
		if trackDuration > movieDuration {
			movieDuration = trackDuration
		}
		if stream.outputTrackID > maxTrackID {
			maxTrackID = stream.outputTrackID
		}
	}

	if mvhd.Version == 0 && movieDuration > maxUint32 {
		mvhd.Version = 1
	}
	mvhd.Duration = movieDuration
	if mvhd.NextTrackID <= maxTrackID {
		mvhd.NextTrackID = maxTrackID + 1
	}

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
    if err := os.Chmod(tmpPath, 0644); err != nil {
       return fmt.Errorf("chmod temporary output: %w", err)
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

	moof := selectMoofTrack(fragment.Moof, ref.traf)
	fragment.Mdat.SetLazyDataSize(ref.dataSize)

	ref.traf.Tfhd.TrackID = ref.input.outputTrackID
	ref.traf.Tfdt.SetBaseMediaDecodeTime(ref.decodeTime)
	moof.Mfhd.SequenceNumber = sequenceNumber
	if err := setFragmentDataOffsets(moof, fragment.Mdat.HeaderSize()); err != nil {
		return err
	}

	if err := moof.Encode(w); err != nil {
		return fmt.Errorf("write fragment metadata: %w", err)
	}

	if err := fragment.Mdat.Encode(w); err != nil {
		return fmt.Errorf("write mdat header: %w", err)
	}

	for _, dataRange := range ref.dataRanges {
		written, err := fragment.Mdat.CopyData(dataRange.start, dataRange.size, ref.input.file, w)
		if err != nil {
			return fmt.Errorf("copy track sample data: %w", err)
		}
		if written != dataRange.size {
			return fmt.Errorf("copied %d track sample bytes, expected %d", written, dataRange.size)
		}
	}
	return nil
}

func selectMoofTrack(moof *mp4.MoofBox, traf *mp4.TrafBox) *mp4.MoofBox {
	selected := &mp4.MoofBox{
		Mfhd:     moof.Mfhd,
		Traf:     traf,
		Trafs:    []*mp4.TrafBox{traf},
		Pssh:     moof.Pssh,
		Psshs:    moof.Psshs,
		StartPos: moof.StartPos,
	}
	for _, child := range moof.Children {
		if candidate, ok := child.(*mp4.TrafBox); ok && candidate != traf {
			continue
		}
		selected.Children = append(selected.Children, child)
	}
	return selected
}

func setFragmentDataOffsets(moof *mp4.MoofBox, mdatHeaderSize uint64) error {
	const maxInt32 = int64(^uint32(0) >> 1)

	// Force the data-offset field to be present before calculating moof size.
	for _, traf := range moof.Trafs {
		for _, trun := range traf.Truns {
			trun.Flags |= mp4.TrunDataOffsetPresentFlag
		}
	}

	dataOffset := int64(moof.Size() + mdatHeaderSize)
	if dataOffset > maxInt32 {
		return errors.New("fragment data offset exceeds int32")
	}
	for _, traf := range moof.Trafs {
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
