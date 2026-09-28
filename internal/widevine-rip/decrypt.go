package widevinerip

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/itouakirai/mp4ff/mp4"
)

// errNoSencBox is the mp4ff error text for a fragment without a senc box.
const errNoSencBox = "no senc box in traf"

// decryptIOBufferSize is the read/write buffer used while streaming boxes.
const decryptIOBufferSize = 1 << 20

// DecryptMP4 decrypts a fragmented MP4 read from r with key and writes the
// clear MP4 to w. Supports CENC and CBCS schemes.
//
// The input is processed one top-level box at a time: only the init segment
// and the fragment currently being decrypted are held in memory, so memory
// use stays bounded by the largest fragment rather than the whole stream.
func DecryptMP4(r io.Reader, key []byte, w io.Writer) error {
	br := bufio.NewReaderSize(r, decryptIOBufferSize)
	bw := bufio.NewWriterSize(w, decryptIOBufferSize)

	var (
		pos         uint64
		ftyp        *mp4.FtypBox
		decryptInfo mp4.DecryptInfo
		initDone    bool
		frag        *mp4.Fragment
	)

	for {
		box, err := mp4.DecodeBox(pos, br)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to decode file: %w", err)
		}
		boxStart := pos
		pos += box.Size()

		if !initDone {
			switch b := box.(type) {
			case *mp4.FtypBox:
				ftyp = b
			case *mp4.MoovBox:
				if b.Mvex == nil {
					return errors.New("file is not fragmented")
				}
				init := mp4.NewMP4Init()
				if ftyp != nil {
					init.AddChild(ftyp)
				}
				init.AddChild(b)
				decryptInfo, err = mp4.DecryptInit(init)
				if err != nil {
					return fmt.Errorf("failed to decrypt init: %w", err)
				}
				if err := init.Encode(bw); err != nil {
					return fmt.Errorf("failed to write init: %w", err)
				}
				initDone = true
			case *mp4.MoofBox, *mp4.MdatBox, *mp4.StypBox, *mp4.EmsgBox:
				return errors.New("no init part of file")
			}
			continue
		}

		switch b := box.(type) {
		case *mp4.StypBox:
			if frag != nil {
				return errors.New("styp inside an incomplete fragment")
			}
			if err := b.Encode(bw); err != nil {
				return fmt.Errorf("failed to encode segment: %w", err)
			}
		case *mp4.EmsgBox, *mp4.PrftBox:
			if frag == nil {
				frag = &mp4.Fragment{StartPos: boxStart}
			}
			frag.AddChild(box)
		case *mp4.MoofBox:
			if frag == nil || frag.Moof != nil {
				frag = &mp4.Fragment{StartPos: boxStart}
			}
			b.StartPos = boxStart
			frag.AddChild(b)
		case *mp4.MdatBox:
			if frag == nil || frag.Moof == nil {
				return errors.New("mdat without preceding moof")
			}
			frag.AddChild(b)
			// A fragment without a senc box is in the clear: streams may
			// start with unencrypted segments followed by encrypted ones.
			// See https://github.com/iyear/gowidevine/pull/26#issuecomment-2385960551
			if err := mp4.DecryptFragment(frag, decryptInfo, key); err != nil && err.Error() != errNoSencBox {
				return fmt.Errorf("failed to decrypt segment: %w", err)
			}
			if err := frag.Encode(bw); err != nil {
				return fmt.Errorf("failed to encode segment: %w", err)
			}
			frag = nil
		}
		// sidx boxes are dropped (their offsets no longer hold once senc
		// data is removed); other top-level boxes are not part of the
		// decrypted output.
	}

	if !initDone {
		return errors.New("no init part of file")
	}
	if frag != nil {
		return errors.New("truncated fragment at end of file")
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}

// DecryptMP4ToFile decrypts r into outputPath. The output is written to a
// temporary file next to outputPath and renamed into place only on success,
// so a failed run never leaves a partial file behind.
func DecryptMP4ToFile(r io.Reader, key []byte, outputPath string) error {
	tempFile, err := os.CreateTemp(filepath.Dir(outputPath), filepath.Base(outputPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	tempPath := tempFile.Name()
	committed := false
	defer func() {
		if !committed {
			_ = tempFile.Close()
			_ = os.Remove(tempPath)
		}
	}()

	if err := DecryptMP4(r, key, tempFile); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}
	if err := os.Rename(tempPath, outputPath); err != nil {
		return fmt.Errorf("replace output: %w", err)
	}
	committed = true
	return nil
}
