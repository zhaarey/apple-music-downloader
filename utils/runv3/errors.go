package runv3

import (
	"errors"
	"fmt"
	"strings"
)

type Stage string

const (
	StageLicense  Stage = "license"
	StageDownload Stage = "download"
	StageDecrypt  Stage = "decrypt"
	StagePlayback Stage = "playback"
)

type RunError struct {
	Stage Stage
	Err   error
}

func (e *RunError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return fmt.Sprintf("%s: %v", e.Stage, e.Err)
}

func (e *RunError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapStage(stage Stage, err error) error {
	if err == nil {
		return nil
	}
	var re *RunError
	if errors.As(err, &re) {
		return err
	}
	return &RunError{Stage: stage, Err: err}
}

// UserFacingError maps runv3 failures to actionable GUI messages.
func UserFacingError(err error) string {
	if err == nil {
		return ""
	}
	if err.Error() == "Unavailable" {
		return "Not available on Apple Music for your account or region"
	}
	var re *RunError
	if errors.As(err, &re) {
		switch re.Stage {
		case StagePlayback:
			return fmt.Sprintf("Could not start playback: %v. Check storefront and subscription.", re.Err)
		case StageLicense:
			return "Widevine license failed — refresh media-user-token in Settings (music.apple.com cookies) and confirm your subscription is active"
		case StageDownload:
			msg := re.Err.Error()
			if strings.Contains(msg, "short read") || strings.Contains(msg, "segment") || strings.Contains(msg, "too small") {
				return "Download incomplete — check your network and try again. " + msg
			}
			return "Media download failed — " + msg
		case StageDecrypt:
			if strings.Contains(re.Err.Error(), "failed to decode file") {
				return "Decrypt failed — downloaded file was incomplete or corrupt. Retry the download; see log for details."
			}
			return "Decrypt failed — " + re.Err.Error()
		}
	}
	if strings.Contains(err.Error(), "error in license response") {
		return "Widevine license failed — refresh media-user-token in Settings and confirm your subscription is active"
	}
	if strings.Contains(err.Error(), "failed to decode file") {
		return "Decrypt failed — downloaded file was incomplete or corrupt. Retry the download; see log for details."
	}
	return "AAC download failed: " + err.Error()
}
