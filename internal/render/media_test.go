package render

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

func TestPerEntryMarkerRequiresBothPlaceholders(t *testing.T) {
	if _, ok := PerEntryMarker(CopyAttachment); !ok {
		t.Fatalf("CopyAttachment %q must carry both placeholders", CopyAttachment)
	}
	for _, tmpl := range []string{
		CoverServiceLogAttachment,
		CoverGitFiles,
		CoverCIArtifacts,
		CoverInternalFiles,
	} {
		if _, ok := PerEntryMarker(tmpl); !ok {
			t.Fatalf("cover template %q must carry both placeholders", tmpl)
		}
	}
	if _, ok := PerEntryMarker("[{n}] only"); ok {
		t.Fatal("a template missing {noun} must fail the accessor")
	}
}

func TestAttachmentLineIsNumberedForm(t *testing.T) {
	plainEnv(t)
	post := domain.Post{
		ID: "m1", Author: domain.User{ID: "u", ScreenName: "aergon"}, CreatedAt: specNow,
		Media: []domain.MediaRef{
			{Kind: domain.Image, URL: "https://pbs.twimg.com/a.jpg"},
			{Kind: domain.Video, URL: "https://pbs.twimg.com/b.jpg", Playable: "https://video.twimg.com/v.mp4"},
		},
	}
	rows := rowsOf(Blocks([]domain.Post{post}, Plain, 80, Selection{}, specNow, nil))
	if len(rows) < 2 || strings.TrimSpace(rows[1]) != "[1] image, [2] video" {
		t.Fatalf("unsplit form rendered %q, want [1] image, [2] video", rows)
	}
}

func TestAttachmentFallbackSplitsAndWraps(t *testing.T) {
	plainEnv(t)
	url := "https://video.twimg.com/ext_tw_video/1812041/pu/vid/1280x720/9nQ4kLm3xT8aBcD.mp4"
	post := domain.Post{
		ID: "m2", Author: domain.User{ID: "u", ScreenName: "aergon"}, CreatedAt: specNow,
		Text: "two clips from the archive; one of them needs a player.",
		Media: []domain.MediaRef{
			{Kind: domain.Image, URL: "https://pbs.twimg.com/a.jpg"},
			{Kind: domain.Video, URL: "https://pbs.twimg.com/poster.jpg", Playable: url},
		},
	}
	falls := Fallbacks{"m2": {Failed: []int{2}}}

	for _, w := range []int{80, 120} {
		rows := rowsOf(Blocks([]domain.Post{post}, Plain, w, Selection{}, specNow, falls))
		joined := strings.Join(rows, "\n")
		if !strings.Contains(joined, "[1] image") {
			t.Fatalf("at %d: missing image entry:\n%s", w, joined)
		}
		if !strings.Contains(joined, "[2] "+url[:20]) && !strings.Contains(joined, url) {
			t.Fatalf("at %d: missing URL entry:\n%s", w, joined)
		}
		if strings.Contains(joined, MarkTruncation) {
			t.Fatalf("at %d: URL path used truncation mark:\n%s", w, joined)
		}
		for _, row := range rows {
			if text.Width(row) > w {
				t.Fatalf("at %d: row exceeds W: %q (%d)", w, row, text.Width(row))
			}
		}
		// Continuations start at column 7 (selection + body indent + EntCol = 1+2+4).
		measure := Plain.measure(w, 0, 0)
		urlCol := measure - EntCol
		if text.Width(url) > urlCol {
			var cont string
			for _, row := range rows {
				if strings.HasPrefix(strings.TrimLeft(row, " >"), "https://") || strings.Contains(row, "[2] ") {
					continue
				}
				trimmed := strings.TrimRight(row, " ")
				if strings.Contains(trimmed, "BcD.mp4") || (len(trimmed) > 7 && trimmed[1:7] == "      " && strings.Contains(trimmed, "mp4")) {
					cont = trimmed
					break
				}
			}
			if cont == "" {
				// Find a continuation: body indent spaces then URL fragment, no [n].
				for _, row := range rows {
					if strings.Contains(row, "[") {
						continue
					}
					if strings.Contains(row, "mp4") || strings.Contains(row, "BcD") {
						cont = strings.TrimRight(row, " ")
						break
					}
				}
			}
			if cont == "" {
				t.Fatalf("at %d: no continuation row for wrapped URL:\n%s", w, joined)
			}
			// Cont starts with selection cell + BodyIndent(2) + EntCol(4) spaces before text.
			if !strings.HasPrefix(cont, " "+strings.Repeat(" ", Plain.BodyIndent)+strings.Repeat(" ", EntCol)) {
				t.Fatalf("at %d: continuation not at url-col: %q", w, cont)
			}
		}
	}
}

func TestFourFailedAttachmentsWrapAtBothWidths(t *testing.T) {
	plainEnv(t)
	media := make([]domain.MediaRef, 4)
	for i := range media {
		media[i] = domain.MediaRef{
			Kind: domain.Image,
			URL:  "https://pbs.twimg.com/media/ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_" + strings.Repeat("x", 40) + string(rune('a'+i)) + ".jpg",
		}
	}
	post := domain.Post{ID: "m4", Author: domain.User{ID: "u", ScreenName: "lzd"}, CreatedAt: specNow, Media: media}
	falls := Fallbacks{"m4": {Failed: []int{1, 2, 3, 4}}}
	for _, w := range []int{80, 120} {
		rows := rowsOf(Blocks([]domain.Post{post}, Plain, w, Selection{}, specNow, falls))
		joined := strings.Join(rows, "\n")
		for n := 1; n <= 4; n++ {
			if !strings.Contains(joined, "["+string(rune('0'+n))+"] https://") {
				t.Fatalf("at %d: missing URL entry [%d]:\n%s", w, n, joined)
			}
		}
		for _, row := range rows {
			if text.Width(row) > w {
				t.Fatalf("at %d: row exceeds W: %q", w, row)
			}
		}
		if strings.Contains(joined, MarkTruncation) {
			t.Fatalf("at %d: truncation on URL path", w)
		}
	}
}

func TestExactMultipleURLEmitsNoTrailingEmptyRow(t *testing.T) {
	plainEnv(t)
	measure := Plain.measure(80, 0, 0)
	urlCol := measure - EntCol
	url := strings.Repeat("u", urlCol*2) // exact multiple of wrap column
	post := domain.Post{
		ID: "mx", Author: domain.User{ID: "u", ScreenName: "lzd"}, CreatedAt: specNow,
		Media: []domain.MediaRef{{Kind: domain.Image, URL: url}},
	}
	falls := Fallbacks{"mx": {Failed: []int{1}}}
	rows := rowsOf(Blocks([]domain.Post{post}, Plain, 80, Selection{}, specNow, falls))
	// Drop head; remaining should be exactly 2 URL rows, no empty trailing.
	body := rows[1:]
	if len(body) != 2 {
		t.Fatalf("exact-multiple URL produced %d body rows, want 2: %q", len(body), body)
	}
	for _, row := range body {
		if strings.TrimSpace(row) == "" {
			t.Fatalf("trailing/empty URL row: %q", body)
		}
	}
}

func TestLongURLWrapsWithoutLoss(t *testing.T) {
	plainEnv(t)
	url := "https://example.invalid/" + strings.Repeat("a", 4000)
	post := domain.Post{
		ID: "ml", Author: domain.User{ID: "u", ScreenName: "lzd"}, CreatedAt: specNow,
		Media: []domain.MediaRef{{Kind: domain.Image, URL: url}},
	}
	falls := Fallbacks{"ml": {Failed: []int{1}}}
	rows := rowsOf(Blocks([]domain.Post{post}, Plain, 80, Selection{}, specNow, falls))
	var got strings.Builder
	for _, row := range rows[1:] {
		// Strip selection + indent + prefix spaces / [1]  prefix.
		r := strings.TrimRight(row, " ")
		if i := strings.Index(r, "https://"); i >= 0 {
			got.WriteString(r[i:])
			continue
		}
		// continuation: last UrlCol-start text
		prefix := 1 + Plain.BodyIndent + EntCol
		if len(r) >= prefix {
			got.WriteString(r[prefix:])
		}
	}
	if got.String() != url {
		t.Fatalf("4000-cell URL lost characters: got %d want %d", len(got.String()), len(url))
	}
}

func TestURLFallbackOSC8AndByteIdentity(t *testing.T) {
	url := "https://video.twimg.com/ext_tw_video/1812041/pu/vid/1280x720/9nQ4kLm3xT8aBcD.mp4"
	post := domain.Post{
		ID: "mo", Author: domain.User{ID: "u", ScreenName: "aergon"}, CreatedAt: specNow,
		Media: []domain.MediaRef{
			{Kind: domain.Image, URL: "https://pbs.twimg.com/a.jpg"},
			{Kind: domain.Video, URL: "https://pbs.twimg.com/p.jpg", Playable: url},
		},
	}
	falls := Fallbacks{"mo": {Failed: []int{2}}}

	plainEnv(t)
	mono := strings.Join(rowsOf(Blocks([]domain.Post{post}, Plain, 80, Selection{}, specNow, falls)), "\n")
	if strings.Contains(mono, "\x1b") {
		t.Fatalf("mono/dumb frame carries escapes: %q", mono)
	}

	colourEnv(t)
	colour := strings.Join(rowsOf(Blocks([]domain.Post{post}, Plain, 80, Selection{}, specNow, falls)), "\n")
	if !strings.Contains(colour, "\x1b]8;;"+url+"\x1b\\") {
		t.Fatalf("colour frame missing OSC 8 open for URL")
	}
	if strings.Count(colour, "\x1b]8;;") != 2 { // open + close, possibly per wrapped segment
		// each wrap segment gets a pair; allow multiple pairs but require even count and no SGR on URL text alone
		if strings.Count(colour, "\x1b]8;;")%2 != 0 {
			t.Fatalf("OSC 8 pairs not matched: %d", strings.Count(colour, "\x1b]8;;"))
		}
	}
	stripped := ansi.Strip(colour)
	if stripped != mono {
		t.Fatalf("after stripping escapes, colour != mono\nmono:\n%s\ncolour:\n%s", mono, stripped)
	}
}

func TestZeroFallbacksMatchesNilMap(t *testing.T) {
	plainEnv(t)
	posts := framePosts()
	a := strings.Join(rowsOf(Blocks(posts, Plain, 80, Selection{ID: specSelected}, specNow, nil)), "\n")
	b := strings.Join(rowsOf(Blocks(posts, Plain, 80, Selection{ID: specSelected}, specNow, Fallbacks{})), "\n")
	if a != b {
		t.Fatal("empty Fallbacks map is not byte-identical to nil")
	}
}

func TestTargetURLPrefersPlayable(t *testing.T) {
	if got := targetURL(domain.MediaRef{URL: "poster", Playable: "play"}); got != "play" {
		t.Fatalf("got %q", got)
	}
	if got := targetURL(domain.MediaRef{URL: "poster"}); got != "poster" {
		t.Fatalf("got %q", got)
	}
}

func TestPermalinkFallbackUsesLink(t *testing.T) {
	plainEnv(t)
	link := "https://x.com/aergon/status/1812041"
	post := domain.Post{
		ID: "mp", Author: domain.User{ID: "u", ScreenName: "aergon"}, CreatedAt: specNow,
		Text: "body",
		Media: []domain.MediaRef{
			{Kind: domain.Image, URL: "https://pbs.twimg.com/a.jpg"},
		},
	}
	falls := Fallbacks{"mp": {Permalink: true, Link: link}}
	rows := rowsOf(Blocks([]domain.Post{post}, Plain, 80, Selection{}, specNow, falls))
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "url "+link) && !strings.Contains(joined, UrlPrefix+link) {
		t.Fatalf("missing permalink row with Link:\n%s", joined)
	}
}

func TestPermalinkFallbackWithoutMedia(t *testing.T) {
	plainEnv(t)
	link := "https://x.com/i/status/1812041"
	post := domain.Post{
		ID: "mn", Author: domain.User{ID: "u"}, CreatedAt: specNow,
		Text: "no attachments",
	}
	falls := Fallbacks{"mn": {Permalink: true, Link: link}}
	rows := rowsOf(Blocks([]domain.Post{post}, Plain, 80, Selection{}, specNow, falls))
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, UrlPrefix) || !strings.Contains(joined, link) {
		t.Fatalf("empty-media permalink row missing:\n%s", joined)
	}
}

func TestMediaURLRowUsesNoMarkTruncation(t *testing.T) {
	body, err := os.ReadFile("media.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "MarkTruncation") {
		t.Fatal("media.go must not use MarkTruncation on a URL path")
	}
}

// keep the clock import honest if frame helpers move.
var _ = time.Time{}
