package widevinerip

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/itouakirai/mp4ff/mp4"
)

// errNoSencBox is the mp4ff error text for a fragment without a senc box.
const errNoSencBox = "no senc box in traf"

// DecryptMP4 decrypts a fragmented MP4 read from r with key and writes the
// clear MP4 to w. Supports CENC and CBCS schemes.
func DecryptMP4(r io.Reader, key []byte, w io.Writer) error {
	encrypted, err := mp4.DecodeFile(r)
	if err != nil {
		return fmt.Errorf("failed to decode file: %w", err)
	}
	if !encrypted.IsFragmented() {
		return errors.New("file is not fragmented")
	}
	if encrypted.Init == nil {
		return errors.New("no init part of file")
	}

	decryptInfo, err := mp4.DecryptInit(encrypted.Init)
	if err != nil {
		return fmt.Errorf("failed to decrypt init: %w", err)
	}
	if err := encrypted.Init.Encode(w); err != nil {
		return fmt.Errorf("failed to write init: %w", err)
	}

	for _, segment := range encrypted.Segments {
		// A segment without a senc box is in the clear: streams may start
		// with unencrypted segments followed by encrypted ones. See
		// https://github.com/iyear/gowidevine/pull/26#issuecomment-2385960551
		if err := mp4.DecryptSegment(segment, decryptInfo, key); err != nil && err.Error() != errNoSencBox {
			return fmt.Errorf("failed to decrypt segment: %w", err)
		}
		if err := segment.Encode(w); err != nil {
			return fmt.Errorf("failed to encode segment: %w", err)
		}
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
