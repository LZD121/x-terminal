package xsource

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// minimalExploreBody is one Explore page with two module-carried trends and one entry-carried one.
func minimalExploreBody(t *testing.T) []byte {
	t.Helper()
	trend := func(name, link string) map[string]any {
		return map[string]any{"__typename": "TimelineTrend", "name": name, "trend_url": link}
	}
	body := map[string]any{
		"data": map[string]any{
			"explore_page": map[string]any{
				"body": map[string]any{
					"initialTimeline": map[string]any{
						"timeline": map[string]any{
							"instructions": []any{
								map[string]any{
									"type": "TimelineAddEntries",
									"entries": []any{
										map[string]any{
											"entryId": "trends-1",
											"content": map[string]any{
												"__typename": "TimelineTimelineModule",
												"items": []any{
													map[string]any{"item": map[string]any{"itemContent": trend("Go", "/search?q=%23golang&src=trend_click")}},
													map[string]any{"item": map[string]any{"itemContent": trend("Repeat", "/search?q=%23golang&src=trend_click")}},
												},
											},
										},
										map[string]any{
											"entryId": "trend-2",
											"content": map[string]any{
												"__typename":  "TimelineTimelineItem",
												"itemContent": trend("No link trend", ""),
											},
										},
										map[string]any{
											"entryId": "tweet-1",
											"content": map[string]any{
												"__typename":  "TimelineTimelineItem",
												"itemContent": map[string]any{"__typename": "TimelineTweet"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestParseTrends: module-carried trends, a repeat dropped, a linkless trend searched by its name,
// and a tweet entry ignored.
func TestParseTrends(t *testing.T) {
	got, err := ParseTrends(minimalExploreBody(t))
	if err != nil {
		t.Fatalf("ParseTrends: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("%d trends, want 2: %+v", len(got), got)
	}
	if got[0].Name != "Go" || got[0].Query != "#golang" {
		t.Fatalf("first trend = %+v, want the link's q", got[0])
	}
	if got[1].Name != "No link trend" || got[1].Query != "No link trend" {
		t.Fatalf("second trend = %+v, want the name as its query", got[1])
	}
}

// TestParseTrendsRefusesAnEmptyPage: a body with no trends is a typed error, never an empty list
// that reads as "Explore has nothing today".
func TestParseTrendsRefusesAnEmptyPage(t *testing.T) {
	_, err := ParseTrends([]byte(`{"data":{"explore_page":{"body":{}}}}`))
	if err == nil || !strings.Contains(err.Error(), "initialTimeline") {
		t.Fatalf("err = %v", err)
	}
}

// TestAdapterTrendsAsksByItsOwnName: one request, the Explore operation, no variables.
func TestAdapterTrendsAsksByItsOwnName(t *testing.T) {
	client := &recordingClient{body: minimalExploreBody(t)}
	if _, err := New(client).Trends(context.Background()); err != nil {
		t.Fatalf("Trends: %v", err)
	}
	if client.op != opExplore {
		t.Fatalf("asked %q, want %q", client.op, opExplore)
	}
	if len(client.vars) != 0 {
		t.Fatalf("sent variables %v, want none", client.vars)
	}
}
