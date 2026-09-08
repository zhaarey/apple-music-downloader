package download

import (
	"net/http"
	"net/url"
	"testing"
)

func TestInitHTTPProxy(t *testing.T) {
	original := Client
	t.Cleanup(func() {
		Client = original
	})

	proxyURL, err := url.Parse("http://127.0.0.1:7890")
	if err != nil {
		t.Fatal(err)
	}
	if err := Init("http://127.0.0.1:7890"); err != nil {
		t.Fatal(err)
	}
	transport, ok := Client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", Client.Transport)
	}
	got, err := transport.Proxy(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != proxyURL.String() {
		t.Fatalf("proxy = %q, want %q", got.String(), proxyURL.String())
	}
}
