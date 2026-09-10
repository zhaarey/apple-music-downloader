package fmp4unfrag

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/Eyevinn/mp4ff/mp4"
)

const (
	maxUint32 = uint64(^uint32(0))
	maxInt64  = uint64(1<<63 - 1)
)

// DefragmentMP4 converts a fragmented MP4 into a progressive MP4.
//
// The implementation is lazy:
//   - moov/moof metadata is decoded into memory
//   - mdat payloads are NOT decoded into memory
//   - sample data is copied directly from input to output
//
// The output layout is:
//
//	ftyp
//	moov
//	mdat
//
// For each track, each source trun becomes one output chunk.
// The generated stsc/stsz/stts/ctts/stss/stco tables describe
// those chunks.
//
// Media samples are never decoded or re-encoded.
func DefragmentMP4(input string) error {
	if input == "" {
		return errors.New("input path is empty")
	}
	output := input

	in, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}

	parsed, err := mp4.DecodeFile(
		in,
		mp4.WithDecodeMode(mp4.DecModeLazyMdat),
	)
	if err != nil {
		_ = in.Close()
		return fmt.Errorf("decode fragmented MP4: %w", err)
	}

	// Metadata and lazy mdat offsets are now in memory. Close the input
	// before writing so an in-place replacement is safe on Windows.
	if err := in.Close(); err != nil {
		return fmt.Errorf("close input after decoding: %w", err)
	}

	if !parsed.IsFragmented() {
		return errors.New("input is not a fragmented MP4")
	}

	if parsed.Ftyp == nil {
		return errors.New("input has no ftyp")
	}

	if parsed.Init == nil || parsed.Init.Moov == nil {
		return errors.New("fragmented MP4 has no initialization moov")
	}

	moov := parsed.Init.Moov

	if moov.Mvhd == nil {
		return errors.New("moov has no mvhd")
	}

	if len(moov.Traks) == 0 {
		return errors.New("moov has no tracks")
	}

	if moov.Mvex == nil {
		return errors.New("fragmented MP4 has no mvex")
	}

	// Edit lists describe the mapping from media time to presentation
	// time. Their semantics are independent of fragmentation and are
	// retained while the fragmented sample tables are made progressive.
	// A zero-duration placeholder common in initialization segments is
	// completed once the final track duration is known.

	// Build track metadata and source ranges.
	tracks, err := collectTrackData(parsed)
	if err != nil {
		return fmt.Errorf("collect fragmented samples: %w", err)
	}

	// Check that source ranges do not overlap.
	if err := validateSourceRanges(tracks); err != nil {
		return fmt.Errorf("validate source ranges: %w", err)
	}

	// Build progressive sample tables.
	if err := buildProgressiveMoov(
		moov,
		tracks,
	); err != nil {
		return fmt.Errorf("build progressive moov: %w", err)
	}

	// A progressive MP4 does not use mvex/trex.
	removeMvex(moov)

	// go-mp4tag only accepts a small set of ftyp brands and requires
	// moov.udta.meta.ilst when writing tags.
	compatibleFtyp, err := makeTagCompatibleFtyp(parsed.Ftyp)
	if err != nil {
		return fmt.Errorf("make ftyp tag-compatible: %w", err)
	}

	if err := ensureTagMetadata(moov); err != nil {
		return fmt.Errorf("prepare metadata for go-mp4tag: %w", err)
	}

	// stco is required by go-mp4tag.
	if err := installOutputChunkOffsets(
		compatibleFtyp.Size(),
		moov,
		tracks,
	); err != nil {
		return fmt.Errorf("calculate output chunk offsets: %w", err)
	}

	// Recalculate output mdat size.
	var mdatPayloadSize uint64

	for _, tr := range tracks {
		for _, ch := range tr.Chunks {
			mdatPayloadSize += ch.Source.Size
		}
	}

	if mdatPayloadSize == 0 {
		return errors.New("output contains no media data")
	}

	if err := writeProgressiveMP4(
		input,
		output,
		compatibleFtyp,
		moov,
		tracks,
		mdatPayloadSize,
	); err != nil {
		return fmt.Errorf("write progressive MP4: %w", err)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Internal data model
// -----------------------------------------------------------------------------

// sourceRange describes a contiguous byte range in one source mdat.
//
// Offset is an absolute file offset.
type sourceRange struct {
	Mdat   *mp4.MdatBox
	Offset uint64
	Size   uint64
}

// outputChunk describes one chunk in the progressive output.
//
// Each fragmented trun becomes one progressive chunk.
type outputChunk struct {
	Source sourceRange

	SampleCount uint32
	SampleDesc  uint32
}

// trackData contains everything needed to construct one progressive trak.
type trackData struct {
	TrackID uint32

	Samples []mp4.Sample
	Chunks  []outputChunk

	// SampleGroups preserves sample-to-group assignments in final sample
	// order. Fragment-local sbgp runs are flattened into this sequence.
	SampleGroups []sampleGroupRun

	// Total media duration in this track's mdhd timescale.
	Duration uint64
}

type sampleGroupRun struct {
	GroupingType          string
	Version               byte
	Flags                 uint32
	GroupingTypeParameter uint32
	SampleCount           uint32
	GroupDescriptionIndex uint32
}

// -----------------------------------------------------------------------------
// Fragment parsing
// -----------------------------------------------------------------------------

func collectTrackData(f *mp4.File) ([]*trackData, error) {
	moov := f.Init.Moov

	byTrackID := make(map[uint32]*trackData)
	trexByTrackID := make(map[uint32]*mp4.TrexBox)

	for _, trak := range moov.Traks {
		if trak.Tkhd == nil {
			return nil, errors.New("trak has no tkhd")
		}

		if trak.Mdia == nil ||
			trak.Mdia.Mdhd == nil ||
			trak.Mdia.Minf == nil ||
			trak.Mdia.Minf.Stbl == nil ||
			trak.Mdia.Minf.Stbl.Stsd == nil {

			return nil, fmt.Errorf(
				"track %d has incomplete media/sample table",
				trak.Tkhd.TrackID,
			)
		}

		if len(trak.Mdia.Minf.Stbl.Stsd.Children) == 0 {
			return nil, fmt.Errorf(
				"track %d has empty stsd",
				trak.Tkhd.TrackID,
			)
		}

		// Encrypted tracks cannot be converted by merely changing
		// moof/trun into stbl tables. CENC/CBCS auxiliary information
		// has to be transformed too.
		if moov.IsEncrypted(trak.Tkhd.TrackID) {
			return nil, fmt.Errorf(
				"track %d is encrypted; encrypted fMP4 is not supported",
				trak.Tkhd.TrackID,
			)
		}

		byTrackID[trak.Tkhd.TrackID] = &trackData{
			TrackID: trak.Tkhd.TrackID,
			Samples: make([]mp4.Sample, 0, 4096),
			Chunks:  make([]outputChunk, 0, 256),
		}
	}

	for _, trex := range moov.Mvex.Trexs {
		if _, exists := byTrackID[trex.TrackID]; !exists {
			return nil, fmt.Errorf(
				"trex references unknown track %d",
				trex.TrackID,
			)
		}

		trexByTrackID[trex.TrackID] = trex
	}

	for _, trak := range moov.Traks {
		if _, ok := trexByTrackID[trak.Tkhd.TrackID]; !ok {
			return nil, fmt.Errorf(
				"track %d has no trex",
				trak.Tkhd.TrackID,
			)
		}
	}

	for segIndex, seg := range f.Segments {
		for fragIndex, frag := range seg.Fragments {
			if frag.Moof == nil {
				return nil, fmt.Errorf(
					"segment %d fragment %d has no moof",
					segIndex,
					fragIndex,
				)
			}

			if frag.Mdat == nil {
				return nil, fmt.Errorf(
					"segment %d fragment %d has no mdat",
					segIndex,
					fragIndex,
				)
			}

			if !frag.Mdat.IsLazy() {
				return nil, fmt.Errorf(
					"segment %d fragment %d mdat was not decoded lazily",
					segIndex,
					fragIndex,
				)
			}

			if err := collectFragment(
				frag,
				trexByTrackID,
				byTrackID,
			); err != nil {
				return nil, fmt.Errorf(
					"segment %d fragment %d: %w",
					segIndex,
					fragIndex,
					err,
				)
			}
		}
	}

	// Preserve the track order from moov.
	result := make([]*trackData, 0, len(moov.Traks))

	for _, trak := range moov.Traks {
		td := byTrackID[trak.Tkhd.TrackID]

		if len(td.Samples) == 0 {
			return nil, fmt.Errorf(
				"track %d contains no samples",
				td.TrackID,
			)
		}

		result = append(result, td)
	}

	return result, nil
}

func collectFragment(
	frag *mp4.Fragment,
	trexByTrackID map[uint32]*mp4.TrexBox,
	byTrackID map[uint32]*trackData,
) error {
	for _, traf := range frag.Moof.Trafs {
		if traf.Tfhd == nil {
			return errors.New("traf has no tfhd")
		}

		trackID := traf.Tfhd.TrackID

		td := byTrackID[trackID]
		if td == nil {
			return fmt.Errorf(
				"traf references unknown track %d",
				trackID,
			)
		}

		trex := trexByTrackID[trackID]

		// Effective sample description index.
		descIndex := effectiveSampleDescriptionIndex(
			traf.Tfhd,
			trex,
		)

		if descIndex == 0 {
			return fmt.Errorf(
				"track %d has sample description index 0",
				trackID,
			)
		}

		fragmentSampleStart := len(td.Samples)

		// mp4ff's trun samples initially contain only fields explicitly
		// present in the trun. Fill defaults from tfhd/trex before using
		// the sample metadata.
		for trunIndex, trun := range traf.Truns {
			trun.AddSampleDefaultValues(
				traf.Tfhd,
				trex,
			)

			samples := trun.Samples

			if len(samples) == 0 {
				continue
			}

			// Calculate the absolute source offset of this trun.
			sourceOffset, err := calculateTrunDataOffset(
				frag,
				traf,
				trun,
			)
			if err != nil {
				return fmt.Errorf(
					"track %d trun %d: %w",
					trackID,
					trunIndex,
					err,
				)
			}

			var chunkSize uint64

			for i := range samples {
				s := samples[i]

				if s.Size == 0 {
					return fmt.Errorf(
						"track %d trun %d sample %d has zero size",
						trackID,
						trunIndex,
						i+1,
					)
				}

				chunkSize += uint64(s.Size)
				td.Samples = append(td.Samples, s)
				td.Duration += uint64(s.Dur)
			}

			// Validate that the complete trun lies inside its mdat.
			if err := validateMdatRange(
				frag.Mdat,
				sourceOffset,
				chunkSize,
			); err != nil {
				return fmt.Errorf(
					"track %d trun %d: %w",
					trackID,
					trunIndex,
					err,
				)
			}

			td.Chunks = append(td.Chunks, outputChunk{
				Source: sourceRange{
					Mdat:   frag.Mdat,
					Offset: sourceOffset,
					Size:   chunkSize,
				},
				SampleCount: uint32(len(samples)),
				SampleDesc:  descIndex,
			})
		}

		if traf.Sbgp != nil {
			if err := collectSampleGroupRuns(
				td,
				traf,
				fragmentSampleStart,
			); err != nil {
				return fmt.Errorf(
					"track %d: %w",
					trackID,
					err,
				)
			}
		}
	}

	return nil
}

func collectSampleGroupRuns(
	td *trackData,
	traf *mp4.TrafBox,
	fragmentSampleStart int,
) error {
	sbgp := traf.Sbgp

	if len(sbgp.GroupingType) != 4 {
		return fmt.Errorf(
			"sbgp has invalid grouping type %q",
			sbgp.GroupingType,
		)
	}

	if sbgp.Version > 1 {
		return fmt.Errorf(
			"sbgp grouping type %q has unsupported version %d",
			sbgp.GroupingType,
			sbgp.Version,
		)
	}

	if len(sbgp.SampleCounts) != len(sbgp.GroupDescriptionIndices) {
		return fmt.Errorf(
			"sbgp grouping type %q has %d sample counts but %d group indices",
			sbgp.GroupingType,
			len(sbgp.SampleCounts),
			len(sbgp.GroupDescriptionIndices),
		)
	}

	var coveredSamples uint64

	for i, sampleCount := range sbgp.SampleCounts {
		if sampleCount == 0 {
			return fmt.Errorf(
				"sbgp grouping type %q entry %d has zero sample count",
				sbgp.GroupingType,
				i+1,
			)
		}

		if coveredSamples > maxUint32-uint64(sampleCount) {
			return fmt.Errorf(
				"sbgp grouping type %q sample count overflows uint32",
				sbgp.GroupingType,
			)
		}

		coveredSamples += uint64(sampleCount)

		descriptionIndex := sbgp.GroupDescriptionIndices[i]

		if descriptionIndex >= 65536 {
			if traf.Sgpd != nil {
				return fmt.Errorf(
					"sbgp grouping type %q uses fragment-local sgpd entry %d, which is not supported",
					sbgp.GroupingType,
					descriptionIndex-65536,
				)
			}

			return fmt.Errorf(
				"sbgp grouping type %q has invalid group description index %d",
				sbgp.GroupingType,
				descriptionIndex,
			)
		}

		td.SampleGroups = append(td.SampleGroups, sampleGroupRun{
			GroupingType:          sbgp.GroupingType,
			Version:               sbgp.Version,
			Flags:                 sbgp.Flags,
			GroupingTypeParameter: sbgp.GroupingTypeParameter,
			SampleCount:           sampleCount,
			GroupDescriptionIndex: descriptionIndex,
		})
	}

	fragmentSampleCount := len(td.Samples) - fragmentSampleStart

	if coveredSamples != uint64(fragmentSampleCount) {
		return fmt.Errorf(
			"sbgp grouping type %q covers %d samples, but the fragment has %d",
			sbgp.GroupingType,
			coveredSamples,
			fragmentSampleCount,
		)
	}

	return nil
}

func effectiveSampleDescriptionIndex(
	tfhd *mp4.TfhdBox,
	trex *mp4.TrexBox,
) uint32 {
	if tfhd.HasSampleDescriptionIndex() {
		return tfhd.SampleDescriptionIndex
	}

	if trex != nil {
		return trex.DefaultSampleDescriptionIndex
	}

	return 0
}

// calculateTrunDataOffset reproduces the addressing rules used by mp4ff
// when interpreting a fragmented track.
//
// The normal CMAF case is:
//
//	tfhd(default-base-is-moof)
//	trun(data-offset)
//
// In that case:
//
//	sample-data = moof.StartPos + trun.DataOffset
//
// A base-data-offset in tfhd takes precedence.
func calculateTrunDataOffset(
	frag *mp4.Fragment,
	traf *mp4.TrafBox,
	trun *mp4.TrunBox,
) (uint64, error) {
	base := int64(frag.Moof.StartPos)

	if traf.Tfhd.HasBaseDataOffset() {
		if traf.Tfhd.BaseDataOffset > maxInt64 {
			return 0, errors.New("base-data-offset exceeds int64 range")
		}

		base = int64(traf.Tfhd.BaseDataOffset)
	} else if traf.Tfhd.DefaultBaseIfMoof() {
		base = int64(frag.Moof.StartPos)
	}

	if trun.HasDataOffset() {
		base += int64(trun.DataOffset)
	}

	if base < 0 {
		return 0, fmt.Errorf(
			"calculated negative media data offset %d",
			base,
		)
	}

	return uint64(base), nil
}

func validateMdatRange(
	mdat *mp4.MdatBox,
	offset uint64,
	size uint64,
) error {
	payloadStart := mdat.PayloadAbsoluteOffset()
	payloadSize := mdat.Size() - mdat.HeaderSize()

	if offset < payloadStart {
		return fmt.Errorf(
			"sample offset %d before mdat payload %d",
			offset,
			payloadStart,
		)
	}

	relative := offset - payloadStart

	if relative > payloadSize {
		return fmt.Errorf(
			"sample offset %d is beyond mdat payload size %d",
			offset,
			payloadSize,
		)
	}

	if size > payloadSize-relative {
		return fmt.Errorf(
			"sample range [%d,%d) exceeds mdat payload [%d,%d)",
			offset,
			offset+size,
			payloadStart,
			payloadStart+payloadSize,
		)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Source range validation
// -----------------------------------------------------------------------------

type sourceRangeForValidation struct {
	mdat  *mp4.MdatBox
	start uint64
	end   uint64
	track uint32
	chunk int
}

func validateSourceRanges(tracks []*trackData) error {
	var ranges []sourceRangeForValidation

	for _, tr := range tracks {
		for i, ch := range tr.Chunks {
			ranges = append(ranges, sourceRangeForValidation{
				mdat:  ch.Source.Mdat,
				start: ch.Source.Offset,
				end:   ch.Source.Offset + ch.Source.Size,
				track: tr.TrackID,
				chunk: i,
			})
		}
	}

	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].mdat != ranges[j].mdat {
			return fmt.Sprintf("%p", ranges[i].mdat) <
				fmt.Sprintf("%p", ranges[j].mdat)
		}

		if ranges[i].start != ranges[j].start {
			return ranges[i].start < ranges[j].start
		}

		return ranges[i].end < ranges[j].end
	})

	for i := 1; i < len(ranges); i++ {
		prev := ranges[i-1]
		curr := ranges[i]

		if prev.mdat != curr.mdat {
			continue
		}

		if curr.start < prev.end {
			return fmt.Errorf(
				"overlapping source ranges: track %d chunk %d [%d,%d) and track %d chunk %d [%d,%d)",
				prev.track,
				prev.chunk,
				prev.start,
				prev.end,
				curr.track,
				curr.chunk,
				curr.start,
				curr.end,
			)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Progressive moov construction
// -----------------------------------------------------------------------------

func buildProgressiveMoov(
	moov *mp4.MoovBox,
	tracks []*trackData,
) error {
	trackByID := make(map[uint32]*trackData)

	for _, tr := range tracks {
		trackByID[tr.TrackID] = tr
	}

	for _, trak := range moov.Traks {
		trackID := trak.Tkhd.TrackID
		td := trackByID[trackID]

		if td == nil {
			return fmt.Errorf(
				"track %d is missing collected data",
				trackID,
			)
		}

		if trak.Mdia.Mdhd.Timescale == 0 {
			return fmt.Errorf(
				"track %d has zero timescale",
				trackID,
			)
		}

		if uint64(len(td.Samples)) > uint64(^uint32(0)) {
			return fmt.Errorf(
				"track %d has too many samples",
				trackID,
			)
		}

		if err := normalizeChunkSampleDescriptions(
			trak,
			td,
		); err != nil {
			return err
		}

		if err := rebuildTrackSampleTable(
			trak,
			td,
		); err != nil {
			return err
		}

		// mdhd duration is expressed in media timescale.
		trak.Mdia.Mdhd.Duration = td.Duration

		// tkhd duration is expressed in movie timescale. If an edit list
		// is present, its segment durations define the presentation
		// timeline and take precedence over the media duration.
		movieTimescale := uint64(moov.Mvhd.Timescale)

		if movieTimescale == 0 {
			return errors.New("mvhd has zero timescale")
		}

		trackDuration, err := progressiveTrackDuration(
			trak,
			movieTimescale,
			td.Duration,
		)
		if err != nil {
			return fmt.Errorf("track %d: %w", trackID, err)
		}

		trak.Tkhd.Duration = trackDuration
	}

	// Movie duration = max(track duration).
	var maxMovieDuration uint64

	for _, trak := range moov.Traks {
		if trak.Tkhd.Duration > maxMovieDuration {
			maxMovieDuration = trak.Tkhd.Duration
		}
	}

	moov.Mvhd.Duration = maxMovieDuration

	return nil
}

// progressiveTrackDuration computes the movie-timescale duration of a
// progressive track. Fragmented initialization segments commonly carry a
// single placeholder edit with segment_duration = 0 while media_time is
// already known. Once all fragments have been collected, that placeholder can
// be completed from the actual media duration.
func progressiveTrackDuration(
	trak *mp4.TrakBox,
	movieTimescale uint64,
	mediaDuration uint64,
) (uint64, error) {
	mediaTimescale := uint64(trak.Mdia.Mdhd.Timescale)

	if mediaTimescale == 0 {
		return 0, errors.New("mdhd has zero timescale")
	}

	if trak.Edts == nil {
		return scaleDuration(mediaDuration, mediaTimescale, movieTimescale)
	}

	var (
		totalDuration uint64
		zeroEntries   []*mp4.ElstEntry
	)

	for _, elst := range trak.Edts.Elst {
		if elst == nil {
			continue
		}

		for i := range elst.Entries {
			entry := &elst.Entries[i]

			if entry.SegmentDuration == 0 {
				zeroEntries = append(zeroEntries, entry)
				continue
			}

			if totalDuration > ^uint64(0)-entry.SegmentDuration {
				return 0, errors.New("edit list duration overflows uint64")
			}

			totalDuration += entry.SegmentDuration
		}
	}

	if totalDuration != 0 {
		return totalDuration, nil
	}

	if len(zeroEntries) != 1 {
		return 0, fmt.Errorf(
			"edit list has %d zero-duration entries; cannot determine presentation duration",
			len(zeroEntries),
		)
	}

	entry := zeroEntries[0]

	if entry.MediaTime < 0 ||
		entry.MediaRateInteger != 1 ||
		entry.MediaRateFraction != 0 {
		return 0, errors.New(
			"zero-duration edit list placeholder is not a forward media edit",
		)
	}

	mediaTime := uint64(entry.MediaTime)

	if mediaTime > mediaDuration {
		return 0, fmt.Errorf(
			"edit list media time %d exceeds media duration %d",
			mediaTime,
			mediaDuration,
		)
	}

	presentationDuration := mediaDuration - mediaTime

	segmentDuration, err := scaleDuration(
		presentationDuration,
		mediaTimescale,
		movieTimescale,
	)
	if err != nil {
		return 0, fmt.Errorf("scale edit list duration: %w", err)
	}

	if segmentDuration == 0 {
		return 0, errors.New("edit list presentation duration rounds to zero")
	}

	entry.SegmentDuration = segmentDuration

	return segmentDuration, nil
}

// scaleDuration rescales a duration without overflowing the intermediate
// multiplication. The value is split into quotient and remainder so neither
func scaleDuration(value, from, to uint64) (uint64, error) {
	if from == 0 {
		return 0, errors.New("source timescale is zero")
	}
	if to == 0 {
		return 0, errors.New("destination timescale is zero")
	}

	if value == 0 || from == to {
		return value, nil
	}

	quotient := value / from
	remainder := value % from

	if to != 0 && quotient > ^uint64(0)/to {
		return 0, errors.New("duration scaling overflows uint64")
	}

	result := quotient * to

	if remainder != 0 {
		extra := remainder * to / from

		if result > ^uint64(0)-extra {
			return 0, errors.New("duration scaling overflows uint64")
		}

		result += extra
	}

	return result, nil
}

// normalizeChunkSampleDescriptions makes every stsc reference resolvable.
//
// Some fragmented sources carry a stale or alternate sample-description
// index even though their stsd contains only one sample entry. Strict
// demuxers such as VLC then try to recreate the decoder when the index
// changes and may stop playback. In that case, use the only entry that
// actually exists.
func normalizeChunkSampleDescriptions(
	trak *mp4.TrakBox,
	td *trackData,
) error {
	if trak.Mdia == nil ||
		trak.Mdia.Minf == nil ||
		trak.Mdia.Minf.Stbl == nil ||
		trak.Mdia.Minf.Stbl.Stsd == nil {

		return fmt.Errorf(
			"track %d has no stsd",
			td.TrackID,
		)
	}

	entryCount := uint32(len(trak.Mdia.Minf.Stbl.Stsd.Children))

	if entryCount == 0 {
		return fmt.Errorf(
			"track %d has empty stsd",
			td.TrackID,
		)
	}

	for i := range td.Chunks {
		descIndex := td.Chunks[i].SampleDesc

		if descIndex == 0 {
			return fmt.Errorf(
				"track %d chunk %d has sample description index 0",
				td.TrackID,
				i,
			)
		}

		if descIndex <= entryCount {
			continue
		}

		if entryCount == 1 {
			td.Chunks[i].SampleDesc = 1
			continue
		}

		return fmt.Errorf(
			"track %d chunk %d references sample description %d, but stsd has %d entries",
			td.TrackID,
			i,
			descIndex,
			entryCount,
		)
	}

	return nil
}

func rebuildTrackSampleTable(
	trak *mp4.TrakBox,
	td *trackData,
) error {
	stbl := trak.Mdia.Minf.Stbl

	if stbl.Stsd == nil {
		return fmt.Errorf(
			"track %d has no stsd",
			td.TrackID,
		)
	}

	// We intentionally rebuild the entire sample table.
	//
	// We keep stsd because codec configuration such as avcC/hvcC/esds
	// belongs there. Sample group descriptions are also retained, while
	// their fragment-level assignments are rebuilt below.
	//
	// We reject other sample-table metadata that would require
	// additional semantic conversion.
	sgpds := make([]*mp4.SgpdBox, 0, len(stbl.Sgpds))

	for _, child := range stbl.Children {
		switch child.Type() {
		case "stsd",
			"stts",
			"ctts",
			"stsc",
			"stsz",
			"stss",
			"stco",
			"co64":

			// Rebuilt below.

		case "sgpd":
			sgpd, ok := child.(*mp4.SgpdBox)
			if !ok {
				return fmt.Errorf(
					"track %d has invalid sgpd child",
					td.TrackID,
				)
			}

			sgpds = append(sgpds, sgpd)

		default:
			return fmt.Errorf(
				"track %d has unsupported stbl child %q; refusing to silently change its semantics",
				td.TrackID,
				child.Type(),
			)
		}
	}

	stsd := stbl.Stsd

	stbl.Children = nil
	stbl.Stsd = nil
	stbl.Stts = nil
	stbl.Ctts = nil
	stbl.Stsc = nil
	stbl.Stsz = nil
	stbl.Stss = nil
	stbl.Stco = nil
	stbl.Co64 = nil
	stbl.Sbgp = nil
	stbl.Sbgps = nil
	stbl.Sgpd = nil
	stbl.Sgpds = nil

	// stsd
	stbl.AddChild(stsd)

	// stts
	stts := buildStts(td.Samples)
	stbl.AddChild(stts)

	// ctts
	if ctts := buildCtts(td.Samples); ctts != nil {
		stbl.AddChild(ctts)
	}

	// stsc
	stsc := &mp4.StscBox{}

	var firstChunk uint32 = 1

	var prevSamplesPerChunk uint32
	var prevDescription uint32

	for i, chunk := range td.Chunks {
		if chunk.SampleCount == 0 {
			continue
		}

		if chunk.SampleDesc == 0 {
			return fmt.Errorf(
				"track %d chunk %d has sample description index 0",
				td.TrackID,
				i,
			)
		}

		// Collapse consecutive chunks having the same mapping.
		if i == 0 ||
			chunk.SampleCount != prevSamplesPerChunk ||
			chunk.SampleDesc != prevDescription {

			if err := stsc.AddEntry(
				firstChunk,
				chunk.SampleCount,
				chunk.SampleDesc,
			); err != nil {
				return fmt.Errorf(
					"track %d: create stsc: %w",
					td.TrackID,
					err,
				)
			}

			firstChunk += 1
			prevSamplesPerChunk = chunk.SampleCount
			prevDescription = chunk.SampleDesc

			continue
		}

		firstChunk++
	}

	// The AddEntry logic above has a subtle problem when the first
	// entry is followed by an identical chunk: firstChunk should
	// advance, but no new entry should be emitted. That's exactly
	// what the code does.
	//
	// However, AddEntry's firstChunk argument represents the first
	// chunk for the new mapping, so the implementation above works
	// because firstChunk is advanced on every chunk.
	stbl.AddChild(stsc)

	// stsz
	stsz := &mp4.StszBox{
		SampleNumber: uint32(len(td.Samples)),
	}

	allSameSize := true
	var uniformSize uint32

	if len(td.Samples) > 0 {
		uniformSize = td.Samples[0].Size

		for _, s := range td.Samples[1:] {
			if s.Size != uniformSize {
				allSameSize = false
				break
			}
		}
	}

	if allSameSize {
		stsz.SampleUniformSize = uniformSize
	} else {
		stsz.SampleSize = make([]uint32, len(td.Samples))

		for i, s := range td.Samples {
			stsz.SampleSize[i] = s.Size
		}
	}

	stbl.AddChild(stsz)

	// stss
	//
	// If every sample is sync, stss is omitted, which according to
	// ISO BMFF means every sample is a sync sample.
	allSync := true
	syncSamples := make([]uint32, 0)

	for i := range td.Samples {
		if sampleIsSync(td.Samples[i]) {
			syncSamples = append(
				syncSamples,
				uint32(i+1),
			)
		} else {
			allSync = false
		}
	}

	if !allSync {
		stss := &mp4.StssBox{
			SampleNumber: syncSamples,
		}

		stbl.AddChild(stss)
	}

	// Use stco because go-mp4tag's read/write path requires it.
	// Offsets are checked against the 32-bit stco limit when they
	// are installed below.
	stco := &mp4.StcoBox{}
	stbl.AddChild(stco)

	for _, sgpd := range sgpds {
		stbl.AddChild(sgpd)
	}

	sbgps, err := buildSbgps(sgpds, td)
	if err != nil {
		return err
	}

	for _, sbgp := range sbgps {
		stbl.AddChild(sbgp)
	}

	return nil
}

func buildSbgps(
	sgpds []*mp4.SgpdBox,
	td *trackData,
) ([]*mp4.SbgpBox, error) {
	descriptionCounts := make(map[string]int)

	for _, sgpd := range sgpds {
		if sgpd == nil {
			continue
		}

		if _, exists := descriptionCounts[sgpd.GroupingType]; exists {
			return nil, fmt.Errorf(
				"track %d has multiple sgpd boxes for grouping type %q",
				td.TrackID,
				sgpd.GroupingType,
			)
		}

		descriptionCounts[sgpd.GroupingType] = len(sgpd.SampleGroupEntries)
	}

	var sbgps []*mp4.SbgpBox

	for _, run := range td.SampleGroups {
		if run.SampleCount == 0 {
			return nil, fmt.Errorf(
				"track %d sample group %q has zero sample count",
				td.TrackID,
				run.GroupingType,
			)
		}

		if run.GroupDescriptionIndex != 0 {
			entryCount, exists := descriptionCounts[run.GroupingType]
			if !exists {
				return nil, fmt.Errorf(
					"track %d sample group %q has no sgpd description box",
					td.TrackID,
					run.GroupingType,
				)
			}

			if run.GroupDescriptionIndex > uint32(entryCount) {
				return nil, fmt.Errorf(
					"track %d sample group %q references description %d, but sgpd has %d entries",
					td.TrackID,
					run.GroupingType,
					run.GroupDescriptionIndex,
					entryCount,
				)
			}
		}

		var sbgp *mp4.SbgpBox

		for _, candidate := range sbgps {
			if candidate.GroupingType != run.GroupingType {
				continue
			}

			if candidate.Version != run.Version ||
				candidate.Flags != run.Flags ||
				candidate.GroupingTypeParameter != run.GroupingTypeParameter {
				return nil, fmt.Errorf(
					"track %d sample group %q has inconsistent sbgp versions or flags",
					td.TrackID,
					run.GroupingType,
				)
			}

			sbgp = candidate
			break
		}

		if sbgp == nil {
			sbgp = &mp4.SbgpBox{
				Version:               run.Version,
				Flags:                 run.Flags,
				GroupingType:          run.GroupingType,
				GroupingTypeParameter: run.GroupingTypeParameter,
			}
			sbgps = append(sbgps, sbgp)
		}

		entryCount := len(sbgp.SampleCounts)

		if entryCount > 0 &&
			sbgp.GroupDescriptionIndices[entryCount-1] == run.GroupDescriptionIndex {

			if sbgp.SampleCounts[entryCount-1] >
				^uint32(0)-run.SampleCount {

				return nil, fmt.Errorf(
					"track %d sample group %q sample count overflows uint32",
					td.TrackID,
					run.GroupingType,
				)
			}

			sbgp.SampleCounts[entryCount-1] += run.SampleCount
			continue
		}

		sbgp.SampleCounts = append(sbgp.SampleCounts, run.SampleCount)
		sbgp.GroupDescriptionIndices = append(
			sbgp.GroupDescriptionIndices,
			run.GroupDescriptionIndex,
		)
	}

	return sbgps, nil
}

// sampleIsSync reports whether the sample flags mark a sample as sync.
// mp4ff's Sample.IsSync also requires sample_depends_on == 2, which is not
// set by many valid fragmented files (especially audio). The normative
// sync marker is sample_is_non_sync_sample.
func sampleIsSync(sample mp4.Sample) bool {
	return !mp4.DecodeSampleFlags(sample.Flags).SampleIsNonSync
}

func buildStts(samples []mp4.Sample) *mp4.SttsBox {
	stts := &mp4.SttsBox{}

	if len(samples) == 0 {
		return stts
	}

	currentDuration := samples[0].Dur
	currentCount := uint32(1)

	flush := func() {
		stts.SampleCount = append(
			stts.SampleCount,
			currentCount,
		)

		stts.SampleTimeDelta = append(
			stts.SampleTimeDelta,
			currentDuration,
		)
	}

	for i := 1; i < len(samples); i++ {
		if samples[i].Dur == currentDuration {
			currentCount++
			continue
		}

		flush()

		currentDuration = samples[i].Dur
		currentCount = 1
	}

	flush()

	return stts
}

func buildCtts(samples []mp4.Sample) *mp4.CttsBox {
	if len(samples) == 0 {
		return nil
	}

	hasNonZero := false

	for _, s := range samples {
		if s.CompositionTimeOffset != 0 {
			hasNonZero = true
			break
		}
	}

	if !hasNonZero {
		return nil
	}

	ctts := &mp4.CttsBox{
		Version: 1,
	}

	var (
		currentOffset int32
		currentCount  uint32
	)

	flush := func() error {
		if currentCount == 0 {
			return nil
		}

		if err := ctts.AddSampleCountsAndOffset(
			[]uint32{currentCount},
			[]int32{currentOffset},
		); err != nil {
			return err
		}

		currentCount = 0
		return nil
	}

	currentOffset = samples[0].CompositionTimeOffset
	currentCount = 1

	for i := 1; i < len(samples); i++ {
		if samples[i].CompositionTimeOffset == currentOffset {
			currentCount++
			continue
		}

		if err := flush(); err != nil {
			return nil
		}

		currentOffset = samples[i].CompositionTimeOffset
		currentCount = 1
	}

	if err := flush(); err != nil {
		return nil
	}

	return ctts
}

func removeMvex(moov *mp4.MoovBox) {
	if moov.Mvex == nil {
		return
	}

	children := make([]mp4.Box, 0, len(moov.Children)-1)

	for _, child := range moov.Children {
		if child.Type() != "mvex" {
			children = append(children, child)
		}
	}

	moov.Children = children
	moov.Mvex = nil
}

// -----------------------------------------------------------------------------
// go-mp4tag compatibility
// -----------------------------------------------------------------------------

const (
	fixedFtypMajorBrand   = "M4A "
	fixedFtypMinorVersion = 1
)

var fixedFtypCompatibleBrands = []string{
	"isom",
	"iso5",
	"hlsf",
	"cmfc",
	"ccea",
	"M4A ",
	"mp42",
}

func makeTagCompatibleFtyp(
	original *mp4.FtypBox,
) (*mp4.FtypBox, error) {
	if original == nil {
		return nil, errors.New("ftyp is nil")
	}

	return mp4.NewFtyp(
		fixedFtypMajorBrand,
		fixedFtypMinorVersion,
		append([]string(nil), fixedFtypCompatibleBrands...),
	), nil
}

func ensureTagMetadata(moov *mp4.MoovBox) error {
	if moov == nil {
		return errors.New("moov is nil")
	}

	var udta *mp4.UdtaBox

	for _, child := range moov.Children {
		candidate, ok := child.(*mp4.UdtaBox)
		if ok {
			udta = candidate
			break
		}
	}

	if udta == nil {
		udta = &mp4.UdtaBox{}
		moov.AddChild(udta)
	}

	var (
		meta      *mp4.MetaBox
		metaIndex = -1
	)

	for i, child := range udta.Children {
		candidate, ok := child.(*mp4.MetaBox)
		if ok {
			meta = candidate
			metaIndex = i
			break
		}
	}

	if meta == nil {
		hdlr, err := mp4.CreateHdlr("mdir")
		if err != nil {
			return err
		}

		meta = mp4.CreateMetaBox(0, hdlr)
		udta.AddChild(meta)
	} else if meta.IsQuickTime() {
		// go-mp4tag expects an ISO meta box with a 4-byte
		// version/flags field after the box header.
		replacement := &mp4.MetaBox{
			Version: meta.Version,
			Flags:   meta.Flags,
		}

		for _, child := range meta.Children {
			replacement.AddChild(child)
		}

		if !hasBoxType(replacement.Children, "hdlr") {
			hdlr, err := mp4.CreateHdlr("mdir")
			if err != nil {
				return err
			}

			replacement.AddChild(hdlr)
		}

		udta.Children[metaIndex] = replacement
		meta = replacement
	}

	if !hasBoxType(meta.Children, "hdlr") {
		hdlr, err := mp4.CreateHdlr("mdir")
		if err != nil {
			return err
		}

		meta.AddChild(hdlr)
	}

	if hasBoxType(meta.Children, "ilst") {
		return nil
	}

	meta.AddChild(&mp4.IlstBox{})
	return nil
}

func hasBoxType(
	children []mp4.Box,
	boxType string,
) bool {
	for _, child := range children {
		if child != nil && child.Type() == boxType {
			return true
		}
	}

	return false
}

// -----------------------------------------------------------------------------
// Output chunk offsets
// -----------------------------------------------------------------------------

func installOutputChunkOffsets(
	ftypSize uint64,
	moov *mp4.MoovBox,
	tracks []*trackData,
) error {
	// We always use a normal/large mdat header depending on the final
	// payload size. The payload start therefore depends on the header
	// size, not on sample data.
	var payloadSize uint64

	for _, tr := range tracks {
		for _, ch := range tr.Chunks {
			payloadSize += ch.Source.Size
		}
	}

	mdatHeaderSize := uint64(8)

	if payloadSize+mdatHeaderSize >= 1<<32 {
		mdatHeaderSize = 16
	}

	// stco size depends on its number of entries. Populate the final
	// entry count before writing moov.
	for _, tr := range tracks {
		stbl := findStbl(moov, tr.TrackID)

		if stbl == nil {
			return fmt.Errorf(
				"track %d: stbl not found",
				tr.TrackID,
			)
		}

		if len(tr.Chunks) == 0 {
			return fmt.Errorf(
				"track %d has no output chunks",
				tr.TrackID,
			)
		}

		if stbl.Stco == nil {
			return fmt.Errorf(
				"track %d has no stco",
				tr.TrackID,
			)
		}

		stbl.Stco.ChunkOffset = make([]uint32, len(tr.Chunks))
	}

	// ftyp + moov + mdat header, because the output layout is:
	// ftyp, moov, mdat.
	payloadOffset := ftypSize + moov.Size() + mdatHeaderSize

	for _, tr := range tracks {
		stbl := findStbl(moov, tr.TrackID)
		offsets := stbl.Stco.ChunkOffset

		for i, ch := range tr.Chunks {
			if payloadOffset > maxUint32 {
				return fmt.Errorf(
					"track %d chunk %d offset %d exceeds stco's 32-bit limit required by go-mp4tag",
					tr.TrackID,
					i,
					payloadOffset,
				)
			}

			offsets[i] = uint32(payloadOffset)
			payloadOffset += ch.Source.Size
		}
	}

	return nil
}

func findStbl(
	moov *mp4.MoovBox,
	trackID uint32,
) *mp4.StblBox {
	for _, trak := range moov.Traks {
		if trak.Tkhd.TrackID == trackID {
			return trak.Mdia.Minf.Stbl
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Output
// -----------------------------------------------------------------------------

func writeProgressiveMP4(
	inputPath string,
	outputPath string,
	ftyp *mp4.FtypBox,
	moov *mp4.MoovBox,
	tracks []*trackData,
	mdatPayloadSize uint64,
) error {
	in, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open source for copying: %w", err)
	}
	defer in.Close()

	dir := filepath.Dir(outputPath)

	tmp, err := os.CreateTemp(
		dir,
		".fmp4-unfragment-*.tmp",
	)
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}

	tmpName := tmp.Name()

	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	defer cleanup()

	w := bufio.NewWriterSize(
		tmp,
		1024*1024,
	)

	// 1. ftyp
	if err := ftyp.Encode(w); err != nil {
		return fmt.Errorf("write ftyp: %w", err)
	}

	// 2. moov
	if err := moov.Encode(w); err != nil {
		return fmt.Errorf("write moov: %w", err)
	}

	// 3. mdat header
	mdatHeaderSize := uint64(8)

	if mdatPayloadSize+mdatHeaderSize >= 1<<32 {
		mdatHeaderSize = 16
	}

	mdatSize := mdatPayloadSize + mdatHeaderSize

	if err := writeMdatHeader(
		w,
		mdatSize,
		mdatHeaderSize == 16,
	); err != nil {
		return fmt.Errorf("write mdat header: %w", err)
	}

	// 4. Copy media data.
	//
	// Each output chunk corresponds to one contiguous source range.
	// Therefore we don't seek for every sample.
	for _, tr := range tracks {
		for chunkIndex, chunk := range tr.Chunks {
			if chunk.Source.Mdat == nil {
				return fmt.Errorf(
					"track %d chunk %d has nil source mdat",
					tr.TrackID,
					chunkIndex,
				)
			}

			n, err := chunk.Source.Mdat.CopyData(
				int64(chunk.Source.Offset),
				int64(chunk.Source.Size),
				in,
				w,
			)

			if err != nil {
				return fmt.Errorf(
					"copy track %d chunk %d [%d,%d): %w",
					tr.TrackID,
					chunkIndex,
					chunk.Source.Offset,
					chunk.Source.Offset+chunk.Source.Size,
					err,
				)
			}

			if uint64(n) != chunk.Source.Size {
				return fmt.Errorf(
					"copy track %d chunk %d: expected %d bytes, wrote %d",
					tr.TrackID,
					chunkIndex,
					chunk.Source.Size,
					n,
				)
			}
		}
	}

	// Close the source before replacing outputPath. In-place conversion passes
	// the same path for input and output, and Windows does not allow an open
	// file to be renamed.
	if err := in.Close(); err != nil {
		return fmt.Errorf("close source for copying: %w", err)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync output: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}

	if sameFilePath(inputPath, outputPath) {
		if err := replaceFile(tmpName, outputPath); err != nil {
			return fmt.Errorf(
				"replace %s: %w",
				outputPath,
				err,
			)
		}

		return nil
	}

	// Rename is atomic on the same filesystem when replacing a different path.
	if err := os.Rename(tmpName, outputPath); err != nil {
		return fmt.Errorf(
			"rename %s -> %s: %w",
			tmpName,
			outputPath,
			err,
		)
	}

	return nil
}

// sameFilePath reports whether inputPath and outputPath refer to the same file.
func sameFilePath(inputPath, outputPath string) bool {
	if filepath.Clean(inputPath) == filepath.Clean(outputPath) {
		return true
	}

	inputInfo, inputErr := os.Stat(inputPath)
	outputInfo, outputErr := os.Stat(outputPath)

	return inputErr == nil &&
		outputErr == nil &&
		os.SameFile(inputInfo, outputInfo)
}

// replaceFile replaces outputPath with tmpName while keeping a recoverable
// copy of the original until the new file is in place.
func replaceFile(tmpName, outputPath string) error {
	dir := filepath.Dir(outputPath)

	backup, err := os.CreateTemp(
		dir,
		".fmp4-unfragment-backup-*.tmp",
	)
	if err != nil {
		return fmt.Errorf("create backup path: %w", err)
	}

	backupName := backup.Name()

	if err := backup.Close(); err != nil {
		_ = os.Remove(backupName)
		return fmt.Errorf("close backup path: %w", err)
	}

	if err := os.Remove(backupName); err != nil {
		return fmt.Errorf("prepare backup path: %w", err)
	}

	if err := os.Rename(outputPath, backupName); err != nil {
		return fmt.Errorf("move original to backup: %w", err)
	}

	if err := os.Rename(tmpName, outputPath); err != nil {
		restoreErr := os.Rename(backupName, outputPath)
		if restoreErr != nil {
			return fmt.Errorf(
				"move replacement into place: %w (original preserved at %s; restore failed: %v)",
				err,
				backupName,
				restoreErr,
			)
		}

		return fmt.Errorf("move replacement into place: %w", err)
	}

	if err := os.Remove(backupName); err != nil {
		return fmt.Errorf(
			"converted file is in place but backup %s could not be removed: %w",
			backupName,
			err,
		)
	}

	return nil
}

func writeMdatHeader(
	w io.Writer,
	size uint64,
	large bool,
) error {
	var buf [16]byte

	if large {
		binary.BigEndian.PutUint32(buf[0:4], 1)
		copy(buf[4:8], []byte("mdat"))
		binary.BigEndian.PutUint64(buf[8:16], size)

		_, err := w.Write(buf[:16])
		return err
	}

	if size >= 1<<32 {
		return fmt.Errorf(
			"normal mdat size %d exceeds 32-bit size field",
			size,
		)
	}

	binary.BigEndian.PutUint32(buf[0:4], uint32(size))
	copy(buf[4:8], []byte("mdat"))

	_, err := w.Write(buf[:8])
	return err
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// This compile-time check makes it explicit that our uint64 -> uint32
// boundaries are intentional.
var _ = maxUint32
