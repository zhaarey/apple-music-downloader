package classical

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// metadataWith runs addMetadata for a one-track Recording whose only credited
// artist has the given role label.
func metadataWith(t *testing.T, language, role string) *Recording {
	t.Helper()
	key := apiPath("jp", pachelbelID) + "/tracksMetadata"
	if language != "" {
		key += "?l=" + language
	}
	body := fmt.Sprintf(`{"trackMetadataMap":{"1":{"id":"1","artists":[{"title":"Neville Marriner","role":%q}]}}}`, role)
	rp := &replayer{replies: map[string]reply{key: jsonReply([]byte(body))}}
	rec := &Recording{
		Request: Request{Storefront: "jp", Language: language, RecordingID: pachelbelID},
		Tracks:  []Track{{SongID: "1", Title: "I. Canon", Position: 1, Count: 1}},
	}
	if err := NewClient(&http.Client{Transport: rp}).addMetadata(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestConductorRoleIsRecognizedInEachLanguage(t *testing.T) {
	// Role labels the Classical API served for a conductor, 2026-09-26.
	labels := map[string]string{
		"en-US":      "Conductor",
		"ja":         "指揮者",
		"zh-Hans-CN": "指挥",
		"zh-Hant-TW": "指揮",
		"de-DE":      "Dirigent:in",
		"fr-FR":      "Direction d’orchestre",
		"ko":         "지휘자",
		"es-ES":      "Dirección",
		"it-IT":      "Direzione",
		"pt-BR":      "Regência",
	}
	for language, role := range labels {
		got := metadataWith(t, language, role).Tracks[0].Conductors
		if len(got) != 1 || got[0] != "Neville Marriner" {
			t.Errorf("%s role %q: conductors = %v", language, role, got)
		}
	}
	if got := metadataWith(t, "en-US", "Chamber Orchestra").Tracks[0].Conductors; len(got) != 0 {
		t.Errorf("an orchestra was taken as conductor: %v", got)
	}
}

func TestUnknownConductorLanguageIsReported(t *testing.T) {
	cases := []struct {
		language, role string
		warn           bool
	}{
		{"ru", "Дирижёр", true},
		{"ru", "Conductor", false},          // served in English, so found
		{"ja", "室内オーケストラ", false},           // known language, no conductor
		{"", "Chamber Orchestra", false},    // no language means English
		{"sv-SE", "Kammarorkester", true},   // unknown language, none found
		{"de-DE", "Kammerorchester", false}, // known language, no conductor
	}
	for _, tc := range cases {
		rec := metadataWith(t, tc.language, tc.role)
		if got := len(rec.Warnings) > 0; got != tc.warn {
			t.Errorf("%q role %q: warnings = %v, want a warning = %v", tc.language, tc.role, rec.Warnings, tc.warn)
		}
	}
}
