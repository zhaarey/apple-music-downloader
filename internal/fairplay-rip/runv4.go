package fairplayrip

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	temarimod "github.com/WorldObservationLog/Temari/bindings/go"

	"amdl/internal/config"
	"amdl/internal/download"

	"github.com/grafov/m3u8"
	"github.com/itouakirai/mp4ff/mp4"
	"github.com/schollz/progressbar/v3"
)

const prefetchKey = "skd://itunes.apple.com/P000000000/s1/e1"

var ErrTimeout = errors.New("response timed out")

// lib is the loaded Temari cdylib handle, set by Init.
var lib *temarimod.Library

// Init loads the Temari decryption library bundled with the module.
func Init() error {
	var err error
	lib, err = temarimod.LoadDefault()
	if err != nil {
		return fmt.Errorf("runv4: load temari library: %w", err)
	}
	return nil
}

type templateResponse struct {
	Ctx   string `json:"ctx"`
	State string `json:"state"`
	RCX   string `json:"rcx"`
	RAX   string `json:"rax"`
	RDX   string `json:"rdx"`
	R9    string `json:"r9"`
	RBP   string `json:"rbp"`
}

type liteResponse struct {
	Code int              `json:"code"`
	Msg  string           `json:"msg"`
	Data templateResponse `json:"data"`
}

// fetchTemplate obtains the decryption template for adamId/uri from lite-server
// and hands the 40020-style JSON body to Temari.
func fetchTemplate(baseURL, adam, uri string) (*temarimod.Temari, error) {
	if lib == nil {
		return nil, errors.New("runv4: temari library not initialized (call runv4.Init)")
	}
	if adam == "0" && uri == prefetchKey {
		return lib.FromJSON([]byte(prefetchTemplateJSON))
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/key?adamId=" + url.QueryEscape(adam) + "&uri=" + url.QueryEscape(uri)
	resp, err := download.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lite-server /key returned %s", resp.Status)
	}
	var envelope liteResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("lite-server /key returned code=%d msg=%s", envelope.Code, envelope.Msg)
	}
	body, err := json.Marshal(envelope.Data)
	if err != nil {
		return nil, err
	}
	return lib.FromJSON(body)
}

// streamBody resets an idle timer on every read; when no bytes arrive within
// timeout the request context is cancelled (download stall detection).
type streamBody struct {
	timeout   time.Duration
	timer     *time.Timer
	threshold int
	body      io.Reader
}

func (b *streamBody) Read(p []byte) (int, error) {
	n, err := b.body.Read(p)
	if err != nil {
		return n, err
	}
	if n >= b.threshold {
		b.timer.Reset(b.timeout)
	}
	return n, err
}

const downloadIdleTimeout = 30 * time.Second

type decryptJob struct {
	Seq       int               // fragment sequence number, used for ordering
	Frag      *mp4.Fragment     // original fragment
	Tmpl      *temarimod.Temari // key template
	RawOffset int64
}

type decryptResult struct {
	Seq       int
	Frag      *mp4.Fragment
	RawOffset int64
}

// Run streams fragmented MP4 and decrypts on the fly: HTTP body is fed directly
// through a fragment-reader -> decrypt-workers -> in-order-writer pipeline.
func Run(adamId string, playlistUrl string, outfile string, Config config.ConfigSet) error {
	if lib == nil {
		return errors.New("runv4: temari library not initialized (call runv4.Init)")
	}
	if Config.LiteServer == "" {
		return errors.New("lite-server is not configured in config.yaml")
	}
	header := make(http.Header)

	req, err := http.NewRequest("GET", playlistUrl, nil)
	if err != nil {
		return err
	}
	req.Header = header
	do, err := download.Client.Do(req)
	if err != nil {
		return err
	}

	segments, err := parseMediaPlaylist(do.Body)
	if err != nil {
		return err
	}
	segment := segments[0]
	if segment == nil {
		return errors.New("no segments extracted from playlist")
	}
	if segment.Limit <= 0 {
		return errors.New("non-byterange playlists are currently unsupported")
	}

	parsedUrl, err := url.Parse(playlistUrl)
	if err != nil {
		return err
	}
	fileUrl, err := parsedUrl.Parse(segment.URI)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	req, err = http.NewRequestWithContext(ctx, "GET", fileUrl.String(), nil)
	if err != nil {
		return err
	}
	req.Header = header

	do, err = download.Client.Do(req)
	if err != nil {
		return err
	}
	defer do.Body.Close()
	totalLen := do.ContentLength

	timer := time.AfterFunc(downloadIdleTimeout, func() { cancel(ErrTimeout) })
	body := &streamBody{
		timeout:   downloadIdleTimeout,
		timer:     timer,
		threshold: 1,
		body:      do.Body,
	}

	err = downloadAndDecryptFile(Config.LiteServer, body, outfile, adamId, segments, totalLen, Config)
	timer.Stop()
	if err != nil {
		return err
	}
	fmt.Print("Decrypted\n")
	return nil
}

func downloadAndDecryptFile(liteServer string, in io.Reader, outfile string,
	adamId string, playlistSegments []*m3u8.MediaSegment, totalLen int64, Config config.ConfigSet) error {
	var buffer bytes.Buffer
	var outBuf *bufio.Writer
	var outFile *os.File
	var tmpPath string
	var err error
	MaxMemorySize := int64(Config.MaxMemoryLimit * 1024 * 1024)
	inBuf := bufio.NewReader(in)

	var fetchedTemplates []*temarimod.Temari
	defer func() {
		for _, t := range fetchedTemplates {
			t.Close()
		}
	}()

	if totalLen <= MaxMemorySize {
		outBuf = bufio.NewWriter(&buffer)
	} else {
		// Stream to a temp file first and rename into place on success, so a
		// failed download never leaves a truncated track at the final path.
		tmpPath = outfile + ".part"
		outFile, err = os.Create(tmpPath)
		if err != nil {
			return err
		}
		outBuf = bufio.NewWriter(outFile)
	}
	defer func() {
		if outFile != nil {
			outFile.Close()
			os.Remove(tmpPath)
		}
	}()
	init, offset, err := ReadInitSegment(inBuf)
	if err != nil {
		return err
	}
	if init == nil {
		return errors.New("no init segment found")
	}

	tracks, err := TransformInit(init)
	if err != nil {
		return err
	}
	err = sanitizeInit(init)
	if err != nil {
		fmt.Printf("Warning: unable to sanitize init completely: %s\n", err)
	}
	err = init.Encode(outBuf)
	if err != nil {
		return err
	}

	bar := progressbar.NewOptions64(totalLen,
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionShowCount(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetDescription("Decrypting..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "",
			SaucerHead:    "",
			SaucerPadding: "",
			BarStart:      "",
			BarEnd:        "",
		}),
	)
	bar.Add64(int64(offset))

	eg, ctx := errgroup.WithContext(context.Background())

	jobs := make(chan *decryptJob, 10)
	results := make(chan *decryptResult, 10)

	eg.Go(func() error {
		buffer := make(map[int]*decryptResult)
		expectedSeq := 0
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case res, ok := <-results:
				if !ok {
					return nil
				}
				buffer[res.Seq] = res
				for {
					if readyRes, exists := buffer[expectedSeq]; exists {
						if err := readyRes.Frag.Encode(outBuf); err != nil {
							return fmt.Errorf("encode fragment seq %d failed: %w", expectedSeq, err)
						}
						bar.Add64(readyRes.RawOffset)
						delete(buffer, expectedSeq)
						expectedSeq++
					} else {
						break
					}
				}
			}
		}
	})

	var workerWg sync.WaitGroup
	for i := 0; i < 10; i++ {
		workerWg.Add(1)
		eg.Go(func() error {
			defer workerWg.Done()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case job, ok := <-jobs:
					if !ok {
						return nil
					}
					if err := DecryptFragment(job.Frag, tracks, job.Tmpl); err != nil {
						return fmt.Errorf("decryptFragment seq %d: %w", job.Seq, err)
					}
					select {
					case results <- &decryptResult{
						Seq:       job.Seq,
						Frag:      job.Frag,
						RawOffset: job.RawOffset,
					}:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
		})
	}

	eg.Go(func() error {
		workerWg.Wait()
		close(results)
		return nil
	})

	eg.Go(func() error {
		defer close(jobs)
		seq := 0
		var tmpl *temarimod.Temari

		for i := 0; ; i++ {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			var frag *mp4.Fragment
			rawoffset := offset
			frag, offset, err = ReadNextFragment(inBuf, offset)
			rawoffset = offset - rawoffset
			if err != nil {
				return fmt.Errorf("read fragment: %w", err)
			}
			if frag == nil {
				break
			}

			segment := playlistSegments[i]
			if segment == nil {
				return errors.New("segment number out of sync")
			}

			key := segment.Key
			if key != nil && (i < 2) {
				if key.URI == prefetchKey {
					tmpl, err = fetchTemplate(liteServer, "0", key.URI)
				} else {
					tmpl, err = fetchTemplate(liteServer, adamId, key.URI)
				}
				if err != nil {
					return err
				}
				fetchedTemplates = append(fetchedTemplates, tmpl)
			}

			job := &decryptJob{
				Seq:       seq,
				Frag:      frag,
				Tmpl:      tmpl,
				RawOffset: int64(rawoffset),
			}

			select {
			case jobs <- job:
			case <-ctx.Done():
				return ctx.Err()
			}
			seq++
		}
		return nil
	})

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	err = outBuf.Flush()
	if err != nil {
		return err
	}
	if totalLen <= MaxMemorySize {
		ofh, err := os.Create(outfile)
		if err != nil {
			return err
		}
		defer ofh.Close()

		_, err = ofh.Write(buffer.Bytes())
		if err != nil {
			return err
		}
	} else {
		// Commit: close the temp file and atomically move it to the final name.
		if err := outFile.Close(); err != nil {
			return err
		}
		if err := os.Rename(tmpPath, outfile); err != nil {
			os.Remove(tmpPath)
			return err
		}
		outFile = nil
		tmpPath = ""
	}
	return nil
}

// --- Sanitize / helpers -------------------------------------------------------

func sanitizeInit(init *mp4.InitSegment) error {
	traks := init.Moov.Traks
	if len(traks) > 1 {
		return errors.New("more than 1 track found")
	}
	stsd := traks[0].Mdia.Minf.Stbl.Stsd
	if stsd.SampleCount == 1 {
		return nil
	}
	if stsd.SampleCount > 2 {
		return fmt.Errorf("expected only 1 or 2 entries in stsd, got %d", stsd.SampleCount)
	}
	children := stsd.Children
	if children[0].Type() != children[1].Type() {
		return errors.New("children in stsd are not of the same type")
	}
	stsd.Children = children[:1]
	stsd.SampleCount = 1
	return nil
}

func filterResponse(f io.Reader) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	scanner := bufio.NewScanner(f)
	prefix := []byte("#EXT-X-KEY:")
	keyFormat := []byte("streamingkeydelivery")
	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		if bytes.HasPrefix(lineBytes, prefix) && !bytes.Contains(lineBytes, keyFormat) {
			continue
		}
		_, err := buf.Write(lineBytes)
		if err != nil {
			return nil, err
		}
		_, err = buf.WriteString("\n")
		if err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return buf, nil
}

func parseMediaPlaylist(r io.ReadCloser) ([]*m3u8.MediaSegment, error) {
	defer r.Close()
	playlistBuf, err := filterResponse(r)
	if err != nil {
		return nil, err
	}
	playlist, listType, err := m3u8.Decode(*playlistBuf, true)
	if err != nil {
		return nil, err
	}
	if listType != m3u8.MEDIA {
		return nil, errors.New("m3u8 not of media type")
	}
	mediaPlaylist := playlist.(*m3u8.MediaPlaylist)
	return mediaPlaylist.Segments, nil
}

func ReadInitSegment(r io.Reader) (*mp4.InitSegment, uint64, error) {
	var offset uint64 = 0
	init := mp4.NewMP4Init()
	for i := 0; i < 2; i++ {
		box, err := mp4.DecodeBox(offset, r)
		if err != nil {
			return nil, offset, err
		}
		boxType := box.Type()
		if boxType != "ftyp" && boxType != "moov" {
			return nil, offset, fmt.Errorf("unexpected box type %s, should be ftyp or moov", boxType)
		}
		init.AddChild(box)
		offset += box.Size()
	}
	return init, offset, nil
}

func ReadNextFragment(r io.Reader, offset uint64) (*mp4.Fragment, uint64, error) {
	frag := mp4.NewFragment()
	for {
		box, err := mp4.DecodeBox(offset, r)
		if err == io.EOF {
			return nil, offset, nil
		}
		if err != nil {
			return nil, offset, err
		}
		boxType := box.Type()
		offset += box.Size()
		if boxType == "moof" || boxType == "emsg" || boxType == "prft" {
			frag.AddChild(box)
			continue
		}
		if boxType == "mdat" {
			frag.AddChild(box)
			break
		}
	}
	if frag.Moof == nil {
		return nil, offset, fmt.Errorf("more than one mdat box in fragment (box ends @ offset %d)", offset)
	}
	return frag, offset, nil
}

func FilterSbgpSgpd(children []mp4.Box) ([]mp4.Box, uint64) {
	var bytesRemoved uint64 = 0
	remainingChildren := make([]mp4.Box, 0, len(children))
	for _, child := range children {
		switch box := child.(type) {
		case *mp4.SbgpBox:
			if box.GroupingType == "seam" || box.GroupingType == "seig" {
				bytesRemoved += child.Size()
				continue
			}
		case *mp4.SgpdBox:
			if box.GroupingType == "seam" || box.GroupingType == "seig" {
				bytesRemoved += child.Size()
				continue
			}
		}
		remainingChildren = append(remainingChildren, child)
	}
	return remainingChildren, bytesRemoved
}

func TransformInit(init *mp4.InitSegment) (map[uint32]mp4.DecryptTrackInfo, error) {
	di, err := mp4.DecryptInit(init)
	tracks := make(map[uint32]mp4.DecryptTrackInfo, len(di.TrackInfos))
	for _, ti := range di.TrackInfos {
		tracks[ti.TrackID] = ti
	}
	if err != nil {
		return tracks, err
	}
	for _, trak := range init.Moov.Traks {
		stbl := trak.Mdia.Minf.Stbl
		stbl.Children, _ = FilterSbgpSgpd(stbl.Children)
	}
	return tracks, nil
}

// --- Temari-based decryption -------------------------------------------------

func cbcsDecryptRaw(data []byte, decryptBlockLen, skipBlockLen int, tmpl *temarimod.Temari) error {
	if skipBlockLen != 0 {
		return fmt.Errorf("not full encryption of subsamples")
	}
	truncatedLen := len(data) & ^0xf
	decrypted, err := tmpl.Decrypt(data[:truncatedLen])
	if err != nil {
		return err
	}
	copy(data[:truncatedLen], decrypted)
	return nil
}

func cbcsDecryptSample(sample []byte, subSamplePatterns []mp4.SubSamplePattern, tenc *mp4.TencBox, tmpl *temarimod.Temari) error {
	decryptBlockLen := int(tenc.DefaultCryptByteBlock) * 16
	skipBlockLen := int(tenc.DefaultSkipByteBlock) * 16
	var pos uint32 = 0

	if len(subSamplePatterns) == 0 {
		return cbcsDecryptRaw(sample, decryptBlockLen, skipBlockLen, tmpl)
	}

	for j := 0; j < len(subSamplePatterns); j++ {
		ss := subSamplePatterns[j]
		pos += uint32(ss.BytesOfClearData)

		if ss.BytesOfProtectedData <= 0 {
			continue
		}

		err := cbcsDecryptRaw(sample[pos:pos+ss.BytesOfProtectedData], decryptBlockLen, skipBlockLen, tmpl)
		if err != nil {
			return err
		}
		pos += ss.BytesOfProtectedData
	}

	return nil
}

func cbcsDecryptSamples(samples []mp4.FullSample, tmpl *temarimod.Temari,
	tenc *mp4.TencBox, senc *mp4.SencBox) error {

	for i := range samples {
		var subSamplePatterns []mp4.SubSamplePattern
		if len(senc.SubSamples) != 0 {
			subSamplePatterns = senc.SubSamples[i]
		}
		err := cbcsDecryptSample(samples[i].Data, subSamplePatterns, tenc, tmpl)
		if err != nil {
			return err
		}
	}
	return nil
}

func DecryptFragment(frag *mp4.Fragment, tracks map[uint32]mp4.DecryptTrackInfo, tmpl *temarimod.Temari) error {
	moof := frag.Moof
	var bytesRemoved uint64 = 0

	for _, traf := range moof.Trafs {
		ti, ok := tracks[traf.Tfhd.TrackID]
		if !ok {
			return fmt.Errorf("could not find decryption info for track %d", traf.Tfhd.TrackID)
		}
		if ti.Sinf == nil {
			continue
		}

		schemeType := ti.Sinf.Schm.SchemeType
		if schemeType != "cbcs" {
			return fmt.Errorf("scheme type %s not supported", schemeType)
		}
		hasSenc, isParsed := traf.ContainsSencBox()
		if !hasSenc {
			return fmt.Errorf("no senc box in traf")
		}

		var senc *mp4.SencBox
		if traf.Senc != nil {
			senc = traf.Senc
		} else {
			senc = traf.UUIDSenc.Senc
		}

		if !isParsed {
			err := senc.ParseReadBox(ti.Sinf.Schi.Tenc.DefaultPerSampleIVSize, traf.Saiz)
			if err != nil {
				return err
			}
		}

		samples, err := frag.GetFullSamples(ti.Trex)
		if err != nil {
			return err
		}

		err = cbcsDecryptSamples(samples, tmpl, ti.Sinf.Schi.Tenc, senc)
		if err != nil {
			return err
		}

		bytesRemoved += traf.RemoveEncryptionBoxes()
	}
	_, psshBytesRemoved := moof.RemovePsshs()
	bytesRemoved += psshBytesRemoved
	for _, traf := range moof.Trafs {
		for _, trun := range traf.Truns {
			trun.DataOffset -= int32(bytesRemoved)
		}
	}

	return nil
}
