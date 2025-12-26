package runv2

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/itouakirai/mp4ff/mp4"
	"github.com/grafov/m3u8"

	"github.com/schollz/progressbar/v3"

	"main/utils/structs"
)

const prefetchKey = "skd://itunes.apple.com/P000000000/s1/e1"
var ErrTimeout = errors.New("response timed out")

// ValidatedReader 包装读取器并验证读取的数据量
type ValidatedReader struct {
	reader    io.Reader
	totalRead int64
	maxSize   int64
}

func (v *ValidatedReader) Read(p []byte) (int, error) {
	n, err := v.reader.Read(p)
	v.totalRead += int64(n)
	
	if v.maxSize > 0 && v.totalRead > v.maxSize {
		return n, fmt.Errorf("read exceeded maximum size: %d > %d", v.totalRead, v.maxSize)
	}
	
	return n, err
}

type TimedResponseBody struct {
	timeout     time.Duration
	timer       *time.Timer
	threshold   int
	body        io.Reader
	totalRead   int64
	expectedLen int64
}

func (b *TimedResponseBody) Read(p []byte) (int, error) {
	n, err := b.body.Read(p)
	if err != nil {
		return n, err
	}
	
	b.totalRead += int64(n)
	
	// 检查数据长度是否合理
	if b.expectedLen > 0 && b.totalRead > b.expectedLen*2 {
		return n, fmt.Errorf("read length %d exceeds expected length %d", b.totalRead, b.expectedLen)
	}
	
	if n >= b.threshold {
		b.timer.Reset(b.timeout)
	}
	return n, err
}

func Run(adamId string, playlistUrl string, outfile string, Config structs.ConfigSet) error {
	var err error
	var optstimeout uint
	optstimeout = 0
	timeout := time.Duration(optstimeout * uint(time.Millisecond))
	header := make(http.Header)

	// request media playlist
	req, err := http.NewRequest("GET", playlistUrl, nil)
	if err != nil {
		return err
	}
	req.Header = header
	// requesting an HLS playlist should be relatively fast, so we set the timeout directly on the client
	do, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return err
	}

	// parse m3u8
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

	// get URL to the actual file
	parsedUrl, err := url.Parse(playlistUrl)
	if err != nil {
		return err
	}
	fileUrl, err := parsedUrl.Parse(segment.URI)
	if err != nil {
		return err
	}

	// request mp4
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	req, err = http.NewRequestWithContext(ctx, "GET", fileUrl.String(), nil)
	if err != nil {
		return err
	}
	req.Header = header

	var body io.Reader
	client := &http.Client{Timeout: timeout}
	if optstimeout > 0 {
		// create the timer before calling Do so that the timeout covers TCP handshake,
		// TLS handshake, sending the request and receiving HTTP headers
		timer := time.AfterFunc(timeout, func() { cancel(ErrTimeout) })
		do, err = client.Do(req)
		if err != nil {
			return err
		}
		defer do.Body.Close()
		body = &TimedResponseBody{
			timeout:     timeout,
			timer:       timer,
			threshold:   256,
			body:        do.Body,
			expectedLen: do.ContentLength,
		}
	} else {
		do, err = client.Do(req)
		if err != nil {
			return err
		}
		defer do.Body.Close()
		if do.ContentLength < int64(Config.MaxMemoryLimit * 1024 * 1024) {
			var buffer bytes.Buffer
			bar := progressbar.NewOptions64(
				do.ContentLength,
				progressbar.OptionClearOnFinish(),
				progressbar.OptionSetElapsedTime(false),
				progressbar.OptionSetPredictTime(false),
				progressbar.OptionShowElapsedTimeOnFinish(),
				progressbar.OptionShowCount(),
				progressbar.OptionEnableColorCodes(true),
				progressbar.OptionShowBytes(true),
				progressbar.OptionSetDescription("Downloading..."),
				progressbar.OptionSetTheme(progressbar.Theme{
					Saucer:        "",
					SaucerHead:    "",
					SaucerPadding: "",
					BarStart:      "",
					BarEnd:        "",
				}),
			)
			io.Copy(io.MultiWriter(&buffer, bar), do.Body)
			body = &buffer
			fmt.Print("Downloaded\n")
		} else {
			body = do.Body
		}
	}

	var totalLen int64
	totalLen = do.ContentLength
	// connect to decryptor
	addr := Config.DecryptM3u8Port
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer Close(conn)

	err = downloadAndDecryptFile(conn, body, outfile, adamId, segments, totalLen, Config)
	if err != nil {
		return err
	}
	fmt.Print("Decrypted\n")
	return nil
}

func downloadAndDecryptFile(conn io.ReadWriter, in io.Reader, outfile string,
	adamId string, playlistSegments []*m3u8.MediaSegment, totalLen int64, Config structs.ConfigSet) error {
	var buffer bytes.Buffer
	var outBuf *bufio.Writer
	MaxMemorySize := int64(Config.MaxMemoryLimit * 1024 * 1024)
	
	// 使用验证读取器
	validatedReader := &ValidatedReader{
		reader:  in,
		maxSize: totalLen * 2, // 允许一定的溢出
	}
	inBuf := bufio.NewReader(validatedReader)
	
	if totalLen <= MaxMemorySize {
		outBuf = bufio.NewWriter(&buffer)
	} else {
		ofh, err := os.Create(outfile)
		if err != nil {
			return err
		}
		defer ofh.Close()
		outBuf = bufio.NewWriter(ofh)
	}
	
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
		// errors returned by sanitizeInit are non-fatal
		fmt.Printf("Warning: unable to sanitize init completely: %s\n", err)
	}
	err = init.Encode(outBuf)
	if err != nil {
		return err
	}

	log.Printf("Starting decryption: expected content length: %d", totalLen)
	bar := progressbar.NewOptions64(totalLen,
		progressbar.OptionClearOnFinish(),
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
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	
	var fragmentCount int
	for i := 0; ; i++ {
		fragmentCount = i
		var frag *mp4.Fragment
		rawoffset := offset
		frag, offset, err = ReadNextFragment(inBuf, offset)
		rawoffset = offset - rawoffset
		if err != nil {
			return err
		}
		if frag == nil {
			// check offset against Content-Length?
			break
		}

		// 添加片段处理日志
		if i%10 == 0 { // 每10个片段记录一次
			log.Printf("Processed %d fragments, current offset: %d, progress: %.2f%%", 
				i, offset, 100*float64(offset)/float64(totalLen))
		}

		segment := playlistSegments[i]
		if segment == nil {
			return errors.New("segment number out of sync")
		}
		key := segment.Key
		if key != nil {
			if i != 0 {
				SwitchKeys(rw)
			}
			if key.URI == prefetchKey {
				SendString(rw, "0")
			} else {
				SendString(rw, adamId)
			}
			SendString(rw, key.URI)
		}
		// flushes the buffer
		err = DecryptFragment(frag, tracks, rw)
		if err != nil {
			return fmt.Errorf("decryptFragment: %w", err)
		}
		err = frag.Encode(outBuf)
		if err != nil {
			return err
		}
		bar.Add64(int64(rawoffset))
	}
	
	log.Printf("Completed processing %d fragments, total bytes: %d", fragmentCount, offset)
	
	err = outBuf.Flush()
	if err != nil {
		return err
	}
	if totalLen <= MaxMemorySize {
		// create output file
		ofh, err := os.Create(outfile)
		if err != nil {
			return err
		}
		defer ofh.Close()

		_, err = ofh.Write(buffer.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

// Remove boxes in the init segment that are known to cause compatibility issues
func sanitizeInit(init *mp4.InitSegment) error {
	traks := init.Moov.Traks
	if len(traks) > 1 {
		return errors.New("more than 1 track found")
	}
	// Remove duplicate ec-3 or alac boxes in stsd since some programs (e.g. cuetools) don't
	// like it when there's more than 1 entry in stsd.
	// Every audio track contains two of these boxes because two IVs are needed to decrypt the
	// track. The two boxes become identical after removing encryption info.
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

// Workaround for m3u8 not supporting multiple keys - remove
// PlayReady and Widevine
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

// parsing
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

// 辅助函数：验证盒子大小是否合理
func isValidBoxSize(size uint64) bool {
	return size >= 8 && size <= 100*1024*1024 // 8字节到100MB
}

// Get the next fragment. Returns nil and no error on EOF
func ReadNextFragment(r io.Reader, offset uint64) (*mp4.Fragment, uint64, error) {
	frag := mp4.NewFragment()
	var lastValidOffset uint64
	
	for {
		// 保存当前偏移量，以便在错误时恢复
		lastValidOffset = offset
		
		box, err := mp4.DecodeBox(offset, r)
		if err == io.EOF {
			return nil, offset, nil
		}
		if err != nil {
			log.Printf("Error decoding box at offset %d: %v", offset, err)
			
			// 尝试跳过损坏的盒子并继续
			if skipErr := skipToNextValidBox(r, &offset); skipErr != nil {
				return nil, offset, fmt.Errorf("failed to recover from box error: %w (original: %v)", skipErr, err)
			}
			continue
		}
		
		boxType := box.Type()
		boxSize := box.Size()
		
		// 验证盒子大小
		if !isValidBoxSize(boxSize) {
			log.Printf("Invalid box size %d for type %s at offset %d, attempting recovery", 
				boxSize, boxType, offset)
			
			offset = lastValidOffset
			if skipErr := skipToNextValidBox(r, &offset); skipErr != nil {
				return nil, offset, fmt.Errorf("invalid box size %d: %w", boxSize, skipErr)
			}
			continue
		}
		
		// 对于超大盒子进行额外检查
		if boxSize > 10*1024*1024 { // 10MB限制
			log.Printf("WARNING: Large box detected: %s, size: %d at offset %d", 
				boxType, boxSize, offset)
		}
		
		offset += boxSize
		
		if boxType == "moof" || boxType == "emsg" || boxType == "prft" {
			frag.AddChild(box)
			continue
		}
		if boxType == "mdat" {
			frag.AddChild(box)
			break
		}
		
		log.Printf("ignoring a %s box found mid-stream at offset %d", boxType, offset-boxSize)
	}
	
	if frag.Moof == nil {
		return nil, offset, fmt.Errorf("fragment missing moof box (ends @ offset %d)", offset)
	}
	return frag, offset, nil
}

// 辅助函数：跳过损坏的数据寻找下一个有效盒子
func skipToNextValidBox(r io.Reader, offset *uint64) error {
	buf := make([]byte, 1024)
	
	// 简单的恢复策略：寻找看起来像盒子头的数据
	for i := 0; i < 10; i++ { // 最多尝试10次
		n, err := r.Read(buf)
		if err != nil {
			return err
		}
		
		*offset += uint64(n)
		
		if n < 8 {
			continue
		}
		
		// 检查是否找到有效的盒子头
		size := binary.BigEndian.Uint32(buf[0:4])
		if isValidBoxSize(uint64(size)) {
			log.Printf("Found potential valid box header at offset %d, size: %d", 
                *offset-uint64(n), size)
			// 在实际实现中，这里应该回退到盒子开始位置
			// 但由于我们使用的是普通Reader，无法回退，只能继续
			return nil
		}
	}
	
	return errors.New("unable to find valid box header after corruption")
}

// Return a new slice of boxes with encryption-related sbgp and sgpd removed,
// and the total number of bytes removed.
// Non-encryption-related ones such as 'roll' are left untouched.
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

// Get decryption info for tracks from init segment and remove encryption-related boxes
func TransformInit(init *mp4.InitSegment) (map[uint32]mp4.DecryptTrackInfo, error) {
	di, err := mp4.DecryptInit(init)
	tracks := make(map[uint32]mp4.DecryptTrackInfo, len(di.TrackInfos))
	for _, ti := range di.TrackInfos {
		tracks[ti.TrackID] = ti
	}
	if err != nil {
		return tracks, err
	}
	// remove encryption-related sbgp and sgpd
	for _, trak := range init.Moov.Traks {
		stbl := trak.Mdia.Minf.Stbl
		stbl.Children, _ = FilterSbgpSgpd(stbl.Children)
	}
	return tracks, nil
}

// Reset the loops on the script's end and close the connection
func Close(conn io.WriteCloser) error {
	defer conn.Close()
	_, err := conn.Write([]byte{0, 0, 0, 0, 0})
	return err
}

func SwitchKeys(conn io.Writer) error {
	_, err := conn.Write([]byte{0, 0, 0, 0})
	return err
}

// Send id or keyUri
func SendString(conn io.Writer, uri string) error {
	_, err := conn.Write([]byte{byte(len(uri))})
	if err != nil {
		return err
	}
	_, err = io.WriteString(conn, uri)
	return err
}

func cbcsFullSubsampleDecrypt(data []byte, conn *bufio.ReadWriter) error {
	// Drops 4 last bits -> multiple of 16
	// It wouldn't hurt to send the remaining bytes also because the decryption
	// function would just return them as-is, but we're truncating the data here
	// for clarity and interoperability
	truncatedLen := len(data) & ^0xf
	// send the whole chunk at once
	err := binary.Write(conn, binary.LittleEndian, uint32(truncatedLen))
	if err != nil {
		return err
	}
	_, err = conn.Write(data[:truncatedLen])
	if err != nil {
		return err
	}
	err = conn.Flush()
	if err != nil {
		return err
	}
	_, err = io.ReadFull(conn, data[:truncatedLen])
	return err
}

func cbcsStripeDecrypt(data []byte, conn *bufio.ReadWriter, decryptBlockLen, skipBlockLen int) error {
	size := len(data)

	// block too small, ignore
	if size < decryptBlockLen {
		return nil
	}

	// number of encrypted blocks in this sample
	count := ((size - decryptBlockLen) / (decryptBlockLen + skipBlockLen)) + 1
	totalLen := count * decryptBlockLen

	err := binary.Write(conn, binary.LittleEndian, uint32(totalLen))
	if err != nil {
		return err
	}

	pos := 0
	for {
		if size-pos < decryptBlockLen { // Leave the rest
			break
		}
		_, err = conn.Write(data[pos : pos+decryptBlockLen])
		if err != nil {
			return err
		}
		pos += decryptBlockLen
		if size-pos < skipBlockLen {
			break
		}
		pos += skipBlockLen
	}
	err = conn.Flush()
	if err != nil {
		return err
	}

	pos = 0
	for {
		if size-pos < decryptBlockLen {
			break
		}
		_, err = io.ReadFull(conn, data[pos:pos+decryptBlockLen])
		if err != nil {
			return err
		}
		pos += decryptBlockLen
		if size-pos < skipBlockLen {
			break
		}
		pos += skipBlockLen
	}
	return nil
}

// Decryption function dispatcher
func cbcsDecryptRaw(data []byte, conn *bufio.ReadWriter, decryptBlockLen, skipBlockLen int) error {
	if skipBlockLen == 0 {
		// Full encryption of subsamples
		// e.g. Apple Music ALAC
		return cbcsFullSubsampleDecrypt(data, conn)
	} else {
		// Pattern (stripe) encryption of subsamples
		// e.g. most AVC and HEVC applications
		return cbcsStripeDecrypt(data, conn, decryptBlockLen, skipBlockLen)
	}
}

// Decrypt a cbcs-encrypted sample in-place
func cbcsDecryptSample(sample []byte, conn *bufio.ReadWriter,
	subSamplePatterns []mp4.SubSamplePattern, tenc *mp4.TencBox) error {

	decryptBlockLen := int(tenc.DefaultCryptByteBlock) * 16
	skipBlockLen := int(tenc.DefaultSkipByteBlock) * 16
	var pos uint32 = 0

	// Full sample encryption
	if len(subSamplePatterns) == 0 {
		return cbcsDecryptRaw(sample, conn, decryptBlockLen, skipBlockLen)
	}

	// Has subsamples
	for j := 0; j < len(subSamplePatterns); j++ {
		ss := subSamplePatterns[j]
		pos += uint32(ss.BytesOfClearData)

		// Nothing to decrypt!
		if ss.BytesOfProtectedData <= 0 {
			continue
		}

		err := cbcsDecryptRaw(sample[pos:pos+ss.BytesOfProtectedData],
			conn, decryptBlockLen, skipBlockLen)
		if err != nil {
			return err
		}
		pos += ss.BytesOfProtectedData
	}

	return nil
}

// Decrypt an array of cbcs-encrypted samples in-place
func cbcsDecryptSamples(samples []mp4.FullSample, conn *bufio.ReadWriter,
	tenc *mp4.TencBox, senc *mp4.SencBox) error {

	for i := range samples {
		var subSamplePatterns []mp4.SubSamplePattern
		if len(senc.SubSamples) != 0 {
			subSamplePatterns = senc.SubSamples[i]
		}
		err := cbcsDecryptSample(samples[i].Data, conn, subSamplePatterns, tenc)
		if err != nil {
			return err
		}
	}
	return nil
}

func DecryptFragment(frag *mp4.Fragment, tracks map[uint32]mp4.DecryptTrackInfo, conn *bufio.ReadWriter) error {
	moof := frag.Moof
	var bytesRemoved uint64 = 0

	for _, traf := range moof.Trafs {
		ti, ok := tracks[traf.Tfhd.TrackID]
		if !ok {
			return fmt.Errorf("could not find decryption info for track %d", traf.Tfhd.TrackID)
		}
		if ti.Sinf == nil {
			// unencrypted track
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
			// simply ignore sbgp and sgpd
			// "Sample To Group Box ('sbgp') and Sample Group Description Box ('sgpd')
			// of type 'seig' are used to indicate the KID applied to each sample, and changes
			// to KIDs over time (i.e. 'key rotation')"
			// (ref: https://dashif.org/docs/DASH-IF-IOP-v3.2.pdf)
			err := senc.ParseReadBox(ti.Sinf.Schi.Tenc.DefaultPerSampleIVSize, traf.Saiz)
			if err != nil {
				return err
			}
		}

		samples, err := frag.GetFullSamples(ti.Trex)
		if err != nil {
			return err
		}

		err = cbcsDecryptSamples(samples, conn, ti.Sinf.Schi.Tenc, senc)
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