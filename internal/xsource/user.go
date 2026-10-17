// Package xsource turns the upstream's response bodies into the domain values the rest of the
// program uses. It is the only package that knows the shape of a response.
//
// It never speaks HTTP. A caller fetches through internal/xweb and hands the bytes here, which
// is what lets the parsers be driven by recorded responses instead of by the live site — and
// what keeps a shape change in one place: when the upstream moves a field, one parser and one
// fixture change, and no screen has to know.
//
// Everything in here obeys one rule, which is the phase's rule: a field that is not where it
// was expected is a typed error naming the path, never a zero value. The measured failure mode
// of this upstream is silence — the current user schema has no legacy.followers_count at all,
// so a parser that read the old field would return an empty user and an empty timeline with no
// error anywhere (measured 2026-10-14, .planning/research/UPSTREAM-PROBE.md §3).
package xsource

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LZD121/x-terminal/internal/domain"
)

// ParseError is one path a response did not have, or had in a shape this package cannot read.
// It carries the path rather than a message because the path is the actionable part: it says
// which branch of the JSON moved.
type ParseError struct {
	// Path is the dotted JSON path that failed, e.g. data.user.result.core.screen_name.
	Path string
	// Detail is why, when the failure was not simply absence: a value that is not what the
	// shape says it should be, with the offending value quoted.
	Detail string
}

func (e *ParseError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("xsource: missing %s", e.Path)
	}
	return fmt.Sprintf("xsource: %s: %s", e.Path, e.Detail)
}

// missing is the error for a path that was not there at all.
func missing(path string) error { return &ParseError{Path: path} }

// unreadable is the error for a body, or a branch of one, that is not JSON this package can
// read. The caller's raw data is never echoed: a parse error travels into a terminal and into
// whatever a user pastes into a bug report.
func unreadable(path string, err error) error {
	return &ParseError{Path: path, Detail: "is not readable JSON: " + err.Error()}
}

// userEnvelope is the part of a response that carries one user profile. Both shapes are here
// because the two operations that answer with a profile nest it differently: UserByScreenName
// answers data.user.result, and Viewer — the call that says whose session this is — answers
// data.viewer.user_results.result.
type userEnvelope struct {
	Data struct {
		User struct {
			Result *userNode `json:"result"`
		} `json:"user"`
		Viewer struct {
			UserResults struct {
				Result *userNode `json:"result"`
			} `json:"user_results"`
		} `json:"viewer"`
	} `json:"data"`
}

// userNode is the current user schema: core, avatar, banner, rest_id, id, has_graduated_access.
//
// It is deliberately incomplete — it carries the fields this project reads and no more — and
// deliberately has no legacy section. legacy is the retired shape: a body that only has one is
// a body that has no user this package can read, and it says so by name.
type userNode struct {
	RestID             string `json:"rest_id"`
	ID                 string `json:"id"`
	HasGraduatedAccess bool   `json:"has_graduated_access"`
	Core               struct {
		ScreenName string `json:"screen_name"`
		Name       string `json:"name"`
		CreatedAt  string `json:"created_at"`
	} `json:"core"`
	Avatar struct {
		ImageURL string `json:"image_url"`
	} `json:"avatar"`
	Banner struct {
		ImageURL string `json:"image_url"`
	} `json:"banner"`
	// ProfileBio, RelationshipCounts and TweetCounts are the header facts the profile
	// screen prints. They are pointers / nested optionals so an absent field stays absent
	// (nil) and an explicit zero stays a zero — which is the omit-never-zero rule.
	ProfileBio *struct {
		Description string `json:"description"`
	} `json:"profile_bio"`
	RelationshipCounts *struct {
		Followers *int `json:"followers"`
		Following *int `json:"following"`
	} `json:"relationship_counts"`
	TweetCounts *struct {
		Tweets *int `json:"tweets"`
	} `json:"tweet_counts"`
}

// ViewerAccount is the two fields a verb prints after a session check. It is a value rather than a
// domain.User because the check needs nothing else, and a caller that carried the whole user would
// be tempting other callers to read fields this read never promised.
type ViewerAccount struct {
	ScreenName string
	Name       string
}

// ParseUser reads a user profile out of a response body, from whichever of the two known
// containers the body has: data.user.result first, then data.viewer.user_results.result.
//
// rest_id and core.screen_name are required — they are the identity every other call is built
// from, and a user without them is not a user this program can act as. name, avatar, banner and
// has_graduated_access are carried when the response has them.
func ParseUser(raw []byte) (domain.User, error) {
	var envelope userEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return domain.User{}, unreadable("the response body", err)
	}

	var firstErr error
	for _, candidate := range []struct {
		path string
		node *userNode
	}{
		{"data.user.result", envelope.Data.User.Result},
		{"data.viewer.user_results.result", envelope.Data.Viewer.UserResults.Result},
	} {
		if candidate.node == nil {
			continue
		}
		user, err := userFrom(candidate.path, candidate.node)
		if err == nil {
			return user, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return domain.User{}, firstErr
	}
	return domain.User{}, missing("data.user.result (and data.viewer.user_results.result)")
}

// userFrom reads one user node, naming the container it came from in every error, so that a
// response with two containers cannot leave the caller guessing which one was tried.
func userFrom(path string, node *userNode) (domain.User, error) {
	id := node.RestID
	if id == "" {
		id = node.ID
	}
	if id == "" {
		return domain.User{}, missing(path + ".rest_id")
	}
	if node.Core.ScreenName == "" {
		return domain.User{}, missing(path + ".core.screen_name")
	}
	return domain.User{
		ID:                 id,
		ScreenName:         node.Core.ScreenName,
		Name:               node.Core.Name,
		AvatarURL:          node.Avatar.ImageURL,
		BannerURL:          node.Banner.ImageURL,
		HasGraduatedAccess: node.HasGraduatedAccess,
	}, nil
}

// Viewer is the one authenticated read that names the session's account. The
// mirror records whose list it holds from this answer, and the sync path is the
// only caller: whoami keeps its own direct GraphQL + ParseUser call.
func (a *Adapter) Viewer(ctx context.Context) (domain.User, error) {
	raw, err := a.client.GraphQL(ctx, "Viewer", nil)
	if err != nil {
		return domain.User{}, err
	}
	return ParseUser(raw)
}
