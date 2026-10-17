package xweb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/config"
)

// ErrTooLarge is an image whose body exceeded the fetcher's cap. It is a typed error rather than a
// short read, because a truncated image decodes to garbage and a decoder that fails is not the same
// news as an image that is too big to hold.
var ErrTooLarge = errors.New("xweb: image exceeds the fetch cap")

// ErrNotAnImage is a body whose content type is not an image this build decodes.
var ErrNotAnImage = errors.New("xweb: the response is not an image")

// ErrNotTwimg is a URL outside the hosts this fetcher will reach. The allow-list exists because the
// URLs it is handed come from post payloads, which is content from other accounts: a fetch that
// followed any URL an upstream named would be a request the operator never asked for.
var ErrNotTwimg = errors.New("xweb: not a twimg host")

// ImageFetcher reads image bytes for the vivid presentation's reading pane.
//
// It is deliberately not the Client: it sends no cookie, no authorization header, no csrf token and
// no transaction id, because those are the session's and this request is not made as the account. A
// twimg edge serves a public image to anyone; sending the session to a second host would widen the
// one credential the whole program exists to keep in one place.
//
// It shares the transport with every other request — the resolved proxy and the measured budgets —
// because twimg is reached through the same proxy and blocked the same way when it is missing.
type ImageFetcher struct {
	http  *http.Client
	cap   int64
	check func(*url.URL) bool
}

// NewImageFetcher builds a fetcher over the configured proxy. capBytes bounds one image; a value
// below one is refused, because a fetcher with no cap is a fetcher that can be made to hold a whole
// disk in memory by one post.
func NewImageFetcher(cfg config.Config, capBytes int64) (*ImageFetcher, error) {
	if capBytes <= 0 {
		return nil, fmt.Errorf("xweb: an image fetcher needs a positive cap, got %d", capBytes)
	}
	transport, err := NewTransport(cfg)
	if err != nil {
		return nil, err
	}
	return &ImageFetcher{
		http:  &http.Client{Transport: transport, Timeout: imageTimeout},
		cap:   capBytes,
		check: isTwimgHost,
	}, nil
}

// imageTimeout bounds one whole image fetch. It is longer than a GraphQL request's budget — an image
// is bytes rather than a small answer — and short enough that a stalled edge becomes a placeholder
// rather than a spinner the reader waits on.
const imageTimeout = 20 * time.Second

// Get fetches one image and answers with its bytes and the media type the response declared.
//
// Every failure is one of the typed errors above or the transport's own; a caller that has no
// renderer for images turns all of them into the same placeholder, because what the reader can do
// about it — open it externally — does not depend on which one it was.
func (f *ImageFetcher) Get(ctx context.Context, raw string) ([]byte, string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrNotTwimg, err)
	}
	// The scheme rule belongs to the shipped allow-list, not to the seam a test moves: a fetcher
	// whose check was widened for a test server still refuses a plain-http twimg URL.
	if !f.check(u) || (isTwimgHost(u) && u.Scheme != "https") {
		return nil, "", fmt.Errorf("%w: %s", ErrNotTwimg, u.Redacted())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	// The headers are the request's whole identity: a browser user agent and the type the build can
	// decode. There is deliberately no Cookie, no Authorization and no Referer.
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg;q=0.9,*/*;q=0.1")

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("xweb: image fetch answered %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, f.cap+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(body)) > f.cap {
		return nil, "", fmt.Errorf("%w: over %d bytes", ErrTooLarge, f.cap)
	}
	kind := sniffImage(resp.Header.Get("Content-Type"), body)
	if kind == "" {
		return nil, "", ErrNotAnImage
	}
	return body, kind, nil
}

// sniffImage is the media type this build will actually decode. The declared type is preferred and
// the bytes are the fallback, because a twimg edge has been seen to declare one thing and serve
// another, and a placeholder is a better answer than a decoder panic.
func sniffImage(declared string, body []byte) string {
	if mt, _, err := mime.ParseMediaType(declared); err == nil {
		switch mt {
		case "image/png", "image/jpeg":
			return mt
		}
	}
	switch http.DetectContentType(body) {
	case "image/png", "image/jpeg":
		return http.DetectContentType(body)
	}
	return ""
}

// isTwimgHost is the allow-list: pbs.twimg.com and its siblings. The suffix is matched on a label
// boundary, so `nottwimg.com` is not a twimg host.
func isTwimgHost(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return host == "twimg.com" || strings.HasSuffix(host, ".twimg.com")
}
