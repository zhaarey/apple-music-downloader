package classical

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	pachelbelID = "johann-pachelbel-1653-pp429-1452536848"
	beethovenID = "ludwig-van-beethoven-1770-pp193-1873004116"
)

// reply is one canned response keyed by request path and query.
type reply struct {
	status      int
	contentType string
	body        []byte
	location    string
}

// replayer serves canned responses and records every requested URL.
type replayer struct {
	replies  map[string]reply
	requests []string
}

func (r *replayer) RoundTrip(req *http.Request) (*http.Response, error) {
	key := req.URL.Path
	if req.URL.RawQuery != "" {
		key += "?" + req.URL.RawQuery
	}
	r.requests = append(r.requests, key)
	rep, ok := r.replies[key]
	if !ok {
		rep = reply{status: http.StatusNotFound, contentType: "text/plain", body: []byte("not found")}
	}
	header := http.Header{"Content-Type": {rep.contentType}}
	if rep.location != "" {
		header.Set("Location", rep.location)
	}
	return &http.Response{
		StatusCode: rep.status,
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader(rep.body)),
		Request:    req,
	}, nil
}

func (r *replayer) requested(prefix string) bool {
	for _, key := range r.requests {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func jsonReply(body []byte) reply {
	return reply{status: http.StatusOK, contentType: "application/json; charset=utf-8", body: body}
}

func htmlReply(body []byte) reply {
	return reply{status: http.StatusOK, contentType: "text/html; charset=utf-8", body: body}
}

func apiPath(sf, id string) string {
	return "/api/classical/v10/query/view/" + sf + "/recording/" + id
}

func pagePath(sf, id string) string {
	return "/" + sf + "/recording/" + id
}

func fetch(t *testing.T, rp *replayer, req Request) (*Recording, error) {
	t.Helper()
	return NewClient(&http.Client{Transport: rp}).Fetch(context.Background(), req)
}

func assertIDs(t *testing.T, rec *Recording, want ...string) {
	t.Helper()
	got := make([]string, len(rec.Tracks))
	for i, track := range rec.Tracks {
		got[i] = track.SongID
		if track.Position != i+1 || track.Count != len(want) || track.Title == "" {
			t.Fatalf("track %d = %#v", i, track)
		}
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("song ids = %v, want %v", got, want)
	}
}

func TestFetchAPIBeethoven(t *testing.T) {
	rp := &replayer{replies: map[string]reply{
		apiPath("us", beethovenID) + "?l=en-US":                jsonReply(fixture(t, "api/us_beethoven_en-US.json")),
		apiPath("us", beethovenID) + "/tracksMetadata?l=en-US": jsonReply(fixture(t, "metadata/us_beethoven_en-US.json")),
	}}
	rec, err := fetch(t, rp, Request{Storefront: "us", Language: "en-US", RecordingID: beethovenID})
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, rec, "1873004347", "1873004590", "1873004600")
	if rec.AlbumID != "1873004116" || rec.Source != "api" || rec.Tracks[0].Title != "I. Allegro vivace" {
		t.Fatalf("recording = %#v", rec)
	}
	if rec.WorkTitle != "Piano Sonata No. 16 in G Major, Op. 31/1" || rec.Composer != "Ludwig van Beethoven" {
		t.Fatalf("work/composer = %q / %q", rec.WorkTitle, rec.Composer)
	}
	if len(rec.Warnings) != 0 {
		t.Fatalf("unexpected warnings %v", rec.Warnings)
	}
}

func TestFetchAPIPachelbelWithEnglishConductor(t *testing.T) {
	rp := &replayer{replies: map[string]reply{
		apiPath("jp", pachelbelID) + "?l=en-US":                jsonReply(fixture(t, "api/jp_pachelbel_en-US.json")),
		apiPath("jp", pachelbelID) + "/tracksMetadata?l=en-US": jsonReply(fixture(t, "metadata/jp_pachelbel_en-US.json")),
	}}
	rec, err := fetch(t, rp, Request{Storefront: "jp", Language: "en-US", RecordingID: pachelbelID})
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, rec, "1452537828", "1452537822", "1452537956")
	for _, track := range rec.Tracks {
		if len(track.Conductors) != 1 || !strings.Contains(track.Conductors[0], "Marriner") {
			t.Fatalf("conductors for %s = %v", track.SongID, track.Conductors)
		}
	}
}

func TestFetchFallsBackToSSRWhenAPIReturnsHTML(t *testing.T) {
	rp := &replayer{replies: map[string]reply{
		apiPath("cn", pachelbelID) + "?l=zh-CN":                htmlReply(fixture(t, "ssr/cn_pachelbel_zh-CN.html")),
		pagePath("cn", pachelbelID) + "?l=zh-CN":               htmlReply(fixture(t, "ssr/cn_pachelbel_zh-CN.html")),
		apiPath("cn", pachelbelID) + "/tracksMetadata?l=zh-CN": jsonReply(fixture(t, "metadata/cn_pachelbel_zh-CN.json")),
	}}
	rec, err := fetch(t, rp, Request{Storefront: "cn", Language: "zh-CN", RecordingID: pachelbelID})
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, rec, "1452537828", "1452537822", "1452537956")
	want := []string{"I. Canon", "II. Gigue", "III. Canon da capo"}
	for i, track := range rec.Tracks {
		if track.Title != want[i] {
			t.Fatalf("title %d = %q, want %q", i, track.Title, want[i])
		}
		if len(track.Conductors) != 1 {
			t.Fatalf("localized conductor not recognized for %s: %v", track.SongID, track.Conductors)
		}
	}
	if rec.Source != "ssr" || rec.AlbumID != "1452536848" {
		t.Fatalf("recording = %#v", rec)
	}
}

func TestFetchFallsBackToSSROnScreenError(t *testing.T) {
	rp := &replayer{replies: map[string]reply{
		apiPath("cn", pachelbelID) + "?l=en-US":  {status: http.StatusBadRequest, contentType: "application/json", body: []byte(`{"type":"screen-error"}`)},
		pagePath("cn", pachelbelID) + "?l=en-US": htmlReply(fixture(t, "ssr/cn_pachelbel_en-US.html")),
	}}
	rec, err := fetch(t, rp, Request{Storefront: "cn", Language: "en-US", RecordingID: pachelbelID})
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, rec, "1452537828", "1452537822", "1452537956")
	if len(rec.Warnings) == 0 {
		t.Fatal("missing metadata should produce a warning")
	}
}

func TestFetchFailsWhenSSRRedirectsToHomepage(t *testing.T) {
	rp := &replayer{replies: map[string]reply{
		apiPath("jp", pachelbelID) + "?l=en-US":  jsonReply([]byte(`{"type":"screen-error"}`)),
		pagePath("jp", pachelbelID) + "?l=en-US": {status: http.StatusFound, contentType: "text/html", location: "/cn"},
		"/cn":                                    htmlReply(fixture(t, "redirects/cn_homepage.html")),
	}}
	if _, err := fetch(t, rp, Request{Storefront: "jp", Language: "en-US", RecordingID: pachelbelID}); err == nil {
		t.Fatal("expected failure when both channels fail")
	}
	if rp.requested("/cn") {
		t.Fatal("redirect to another storefront must not be followed")
	}
}

func TestFetchRejectsAlbumConflictWithoutSSR(t *testing.T) {
	original := fixture(t, "api/us_beethoven_en-US.json")
	body := bytes.Replace(original, []byte(`"albumId":"1873004116"`), []byte(`"albumId":"999"`), 1)
	if bytes.Equal(body, original) {
		t.Fatal("fixture does not contain the expected albumId")
	}
	rp := &replayer{replies: map[string]reply{
		apiPath("us", beethovenID) + "?l=en-US": jsonReply(body),
	}}
	if _, err := fetch(t, rp, Request{Storefront: "us", Language: "en-US", RecordingID: beethovenID}); err == nil {
		t.Fatal("expected album identity conflict")
	}
	if rp.requested(pagePath("us", beethovenID)) {
		t.Fatal("identity conflict must not fall back to SSR")
	}
}

func TestFetchMetadataFailureKeepsCoreData(t *testing.T) {
	rp := &replayer{replies: map[string]reply{
		apiPath("us", beethovenID) + "?l=en-US": jsonReply(fixture(t, "api/us_beethoven_en-US.json")),
	}}
	rec, err := fetch(t, rp, Request{Storefront: "us", Language: "en-US", RecordingID: beethovenID})
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, rec, "1873004347", "1873004590", "1873004600")
	if rec.Tracks[0].Title != "I. Allegro vivace" || len(rec.Warnings) == 0 {
		t.Fatalf("recording = %#v", rec)
	}
}
