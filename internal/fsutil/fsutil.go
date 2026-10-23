// Package fsutil holds small filesystem helpers shared across packages.
package fsutil

import "os"

// OutputPerm is the mode given to files that end up in the user's library.
const OutputPerm os.FileMode = 0o644

// CreateOutputTemp is os.CreateTemp for files that will be renamed into their
// final place. os.CreateTemp creates files as 0600, which would carry over to
// the renamed output and leave it unreadable to other users (e.g. a media
// server running under its own account).
func CreateOutputTemp(dir, pattern string) (*os.File, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	if err := f.Chmod(OutputPerm); err != nil {
		name := f.Name()
		_ = f.Close()
		_ = os.Remove(name)
		return nil, err
	}
	return f, nil
}
