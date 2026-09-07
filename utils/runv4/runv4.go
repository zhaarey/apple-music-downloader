package runv4

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
	"sync"
	"golang.org/x/sync/errgroup"

	"main/utils/structs"

	"github.com/grafov/m3u8"
	"github.com/itouakirai/mp4ff/mp4"
	"github.com/schollz/progressbar/v3"
)
const prefetchKey = "skd://itunes.apple.com/P000000000/s1/e1"
var ErrTimeout = errors.New("response timed out")

type TimedResponseBody struct {
	timeout   time.Duration
	timer     *time.Timer
	threshold int
	body      io.Reader
}
type decryptJob struct {
	Seq       int           // 分片序号，用于重组
	Frag      *mp4.Fragment // 原始分片
	Tmpl      *template     // 密钥模板
	RawOffset int64
}

// 定义输出结果
type decryptResult struct {
	Seq       int
	Frag      *mp4.Fragment
	RawOffset int64
}


func (b *TimedResponseBody) Read(p []byte) (int, error) {
	n, err := b.body.Read(p)
	if err != nil {
		return n, err
	}
	// fmt.Printf("Read %d bytes, buffer size %d bytes", n, len(p))
	if n >= b.threshold {
		b.timer.Reset(b.timeout)
	}
	return n, err
}
const (
	downloadMaxAttempts = 5                 // 最多尝试次数
	downloadIdleTimeout = 30 * time.Second  // 30 秒没收到任何字节就认为卡死
)

// downloadWithResume 下载完整文件到内存，支持断点续传、空闲超时和重试。
// first 是调用方已经拿到响应头、但尚未读取正文的首个 GET 响应，作为第一次尝试直接读取，
// firstCancel 用于在它卡死时取消该请求；之后的尝试才用 Range 请求从断点续传。
// 任何时刻只保留一条到服务器的下载流：如果放着 first 不读、另开一条流下载同一文件，
// 两条流会复用同一个 HTTP/2 连接，未读取的那条会占满流量控制窗口，新开的流也会在约 4 MiB 处卡死。
// 只有拿到 first.ContentLength 字节才返回成功。
func downloadWithResume(ctx context.Context, client *http.Client, fileUrl string,
	header http.Header, first *http.Response, firstCancel context.CancelFunc,
	bar *progressbar.ProgressBar) (*bytes.Buffer, error) {

	totalLen := first.ContentLength
	buf := &bytes.Buffer{}
	var offset int64
	var lastErr error
	backoff := 2 * time.Second

	for attempt := 1; attempt <= downloadMaxAttempts; attempt++ {
		resp, attemptCancel := first, firstCancel
		if attempt > 1 {
			fmt.Printf("Download interrupted at %d/%d bytes (%v), retrying in %v\n", offset, totalLen, lastErr, backoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}

			// 每次续传用独立的子 context，卡死时只取消本次连接
			var attemptCtx context.Context
			attemptCtx, attemptCancel = context.WithCancel(ctx)
			req, err := http.NewRequestWithContext(attemptCtx, "GET", fileUrl, nil)
			if err != nil {
				attemptCancel()
				return nil, err
			}
			req.Header = header.Clone()
			if offset > 0 {
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset)) // 断点续传
			}

			resp, err = client.Do(req)
			if err != nil {
				attemptCancel()
				lastErr = err
				continue
			}
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			attemptCancel()
			return nil, fmt.Errorf("download failed: server returned %s", resp.Status)
		}
		if offset > 0 && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			attemptCancel()
			return nil, errors.New("server does not support Range requests, cannot resume")
		}

		// 空闲超时检测：没有新数据到达就取消本次请求，触发重试
		timer := time.AfterFunc(downloadIdleTimeout, attemptCancel)
		body := &TimedResponseBody{
			timeout:   downloadIdleTimeout,
			timer:     timer,
			threshold: 1, // 只要读到字节就重置计时器
			body:      resp.Body,
		}

		n, copyErr := io.Copy(io.MultiWriter(buf, bar), body)
		idleTimedOut := !timer.Stop()
		resp.Body.Close()
		attemptCancel()
		offset += n

		if copyErr == nil && offset == totalLen {
			return buf, nil // 完整拿到，才算下载成功
		}
		switch {
		case copyErr == nil:
			copyErr = fmt.Errorf("short download: got %d of %d bytes", offset, totalLen)
		case ctx.Err() != nil:
			return nil, ctx.Err() // 外层主动取消，不再重试
		case idleTimedOut:
			copyErr = fmt.Errorf("no data received for %v", downloadIdleTimeout)
		}
		lastErr = copyErr
	}
	return nil, fmt.Errorf("download failed after %d attempts (got %d/%d bytes): %w",
		downloadMaxAttempts, offset, totalLen, lastErr)
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
	// 首个请求单独一个子 context：下载卡死时只中止它，不影响之后的续传请求
	firstCtx, firstCancel := context.WithCancel(ctx)
	defer firstCancel()
	req, err = http.NewRequestWithContext(firstCtx, "GET", fileUrl.String(), nil)
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
			timeout:   timeout,
			timer:     timer,
			threshold: 256,
			body:      do.Body,
		}
	} else {
		do, err = client.Do(req)
		if err != nil {
			return err
		}
		defer do.Body.Close()
		if do.ContentLength < int64(Config.MaxMemoryLimit * 1024 * 1024) {
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
			// 直接读 do 的正文作为第一次尝试，不能放着它不读再另开一条流去下同一个文件
			buffer, err := downloadWithResume(ctx, client, fileUrl.String(), header, do, firstCancel, bar)
			if err != nil {
				return err // 下载没完成就失败退出，绝不进入解密
			}

			body = buffer
			fmt.Print("Downloaded\n")
		} else {
			body = do.Body
		}
	}

	var totalLen int64
	totalLen = do.ContentLength
	//key Server
	//keyServer := fmt.Sprintf("127.0.0.1:40020")
	keyServer := Config.KeyServer

	err = downloadAndDecryptFile(keyServer, body, outfile, adamId, segments, totalLen, Config)
	if err != nil {
		return err
	}
	fmt.Print("Decrypted\n")
	return nil
}

func downloadAndDecryptFile(keyServer string, in io.Reader, outfile string,
	adamId string, playlistSegments []*m3u8.MediaSegment, totalLen int64, Config structs.ConfigSet) error {
	var buffer bytes.Buffer
	var outBuf *bufio.Writer
	MaxMemorySize := int64(Config.MaxMemoryLimit * 1024 * 1024)
	inBuf := bufio.NewReader(in)
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

	// 'segment' in m3u8 == 'fragment' in mp4ff
	//fmt.Println("Starting decryption...")
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

	// 1. 引入 errgroup 和 context，实现任意错误瞬间熔断全局
	eg, ctx := errgroup.WithContext(context.Background())

	// 通道设计：任务分发通道 与 结果汇总通道
	// 缓冲区大小决定了最大可“乱序”的跨度，防止读取过快撑爆内存
	jobs := make(chan *decryptJob, 10)
	results := make(chan *decryptResult, 10)

	// 2. 启动 Writer (按序重组并写入)
	eg.Go(func() error {
		// 乱序重组缓冲区 (Reassembly Buffer)
		buffer := make(map[int]*decryptResult)
		expectedSeq := 0 // 期待写入的下一个序号

		for {
			select {
			case <-ctx.Done(): // 收到取消信号，立即退出
				return ctx.Err()
			case res, ok := <-results:
				if !ok {
					// results 通道已关闭，说明所有解密完成，且全部写入完毕
					return nil
				}
				
				// 将乱序到达的结果放入缓冲区
				buffer[res.Seq] = res

				// 检查当前期望的序号是否已经准备好，准备好就一直往前推进
				for {
					if readyRes, exists := buffer[expectedSeq]; exists {
						// 编码写入
						if err := readyRes.Frag.Encode(outBuf); err != nil {
							return fmt.Errorf("encode fragment seq %d failed: %w", expectedSeq, err)
						}
						bar.Add64(readyRes.RawOffset)
						
						// 清理内存并期待下一个
						delete(buffer, expectedSeq)
						expectedSeq++
					} else {
						// 还没轮到，跳出等待
						break
					}
				}
			}
		}
	})

	// 3. 启动固定 10 个解密 Worker (乱序执行)
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
						return nil // 任务分发完毕，Worker 下班
					}
					// 核心解密逻辑
					if err := DecryptFragment(job.Frag, tracks, job.Tmpl); err != nil {
						return fmt.Errorf("decryptFragment seq %d: %w", job.Seq, err)
					}
					
					// 提交解密结果
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

	// 监控所有 Worker 是否完成，完成后关闭 results 通道
	eg.Go(func() error {
		workerWg.Wait() // 等待10个工人都下班
		close(results)  // 通知 Writer 可以收尾了
		return nil
	})

	// 4. 启动 Reader (主线程负责读取)
	eg.Go(func() error {
		defer close(jobs) // 读取完毕，关闭任务通道
		seq := 0
		var tmpl *template

		for i := 0; ; i++ {
			// 检查是否发生了全局错误，如果有则放弃读取
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
				break // 读到文件末尾
			}

			segment := playlistSegments[i]
			if segment == nil {
				return errors.New("segment number out of sync")
			}
			
			key := segment.Key
			if key != nil && (i < 2) {
				if key.URI == prefetchKey {
					tmpl = prefetchTemplate()
					//tmpl, err = fetchTemplate(keyServer, "0", prefetchKey)
				} else {
					tmpl, err = fetchTemplate(keyServer, adamId, key.URI)
				}
				if err != nil {
					return err
				}
			}

			// 将任务发送给 Workers
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

	// 5. 阻塞等待：这行代码会等待 Reader、Workers、Writer 彻底完成，或者捕获第一个发生的错误
	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	// ... (后续的 Flush 和 Buffer 写盘操作保持不变) ...
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

//pasing
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

// Get the next fragment. Returns nil and no error on EOF
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
		// fmt.Printf("processing %s, box starts @ offset %d\n", boxType, offset)
		offset += box.Size()
		if boxType == "moof" || boxType == "emsg" || boxType == "prft" {
			frag.AddChild(box)
			continue
		}
		if boxType == "mdat" {
			frag.AddChild(box)
			break
		}
		fmt.Printf("ignoring a %s box found mid-stream", boxType)
	}
	// only 1 mdat box in fragment, meaning that the box doesn't have a preceding moof box
	if frag.Moof == nil {
		return nil, offset, fmt.Errorf("more than one mdat box in fragment (box ends @ offset %d)", offset)
	}
	return frag, offset, nil
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


// Decryption function dispatcher
func cbcsDecryptRaw(data []byte, decryptBlockLen, skipBlockLen int, tmpl *template) error {
	if skipBlockLen != 0 {
		return fmt.Errorf("not full encryption of subsamples")
	}
	// Drops 4 last bits -> multiple of 16
	// It wouldn't hurt to send the remaining bytes also because the decryption
	// function would just return them as-is, but we're truncating the data here
	// for clarity and interoperability
	truncatedLen := len(data) & ^0xf
	decrypted := decryptSample(tmpl, data[:truncatedLen])
	copy(data[:truncatedLen], decrypted)
	// Full encryption of subsamples
	// e.g. Apple Music ALAC
	return nil
}

// Decrypt a cbcs-encrypted sample in-place
func cbcsDecryptSample(sample []byte, subSamplePatterns []mp4.SubSamplePattern, tenc *mp4.TencBox, tmpl *template) error {

	decryptBlockLen := int(tenc.DefaultCryptByteBlock) * 16
	skipBlockLen := int(tenc.DefaultSkipByteBlock) * 16
	var pos uint32 = 0

	// Full sample encryption
	if len(subSamplePatterns) == 0 {
		return cbcsDecryptRaw(sample, decryptBlockLen, skipBlockLen, tmpl)
	}

	// Has subsamples
	for j := 0; j < len(subSamplePatterns); j++ {
		ss := subSamplePatterns[j]
		pos += uint32(ss.BytesOfClearData)

		// Nothing to decrypt!
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

// Decrypt an array of cbcs-encrypted samples in-place
func cbcsDecryptSamples(samples []mp4.FullSample, tmpl *template,
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

func DecryptFragment(frag *mp4.Fragment, tracks map[uint32]mp4.DecryptTrackInfo, tmpl *template) error {
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
