package subscription

import (
	"errors"
	"fmt"
	"strings"
)

var allFeatures = map[string]bool{
	"merges":                 true,
	"issues":                 true,
	"pushes":                 true,
	"issue_comments":         true,
	"merge_request_comments": true,
	"pipeline":               true,
	"tag":                    true,
	"pull_reviews":           true,
	// "label:":                 true,//particular case for label:XXX
}

type Subscription struct {
	ChannelID  string
	CreatorID  string
	Features   string
	Repository string
}

func New(ChannelID, CreatorID, Features, Repository string) (*Subscription, error) {
	if strings.Contains(Features, "label:") && len(strings.Split(Features, "\"")) < 3 {
		return nil, errors.New("The label is formatted incorrectly")
	}
	if strings.Contains(Features, "label:") && len(strings.Split(Features, "\"")) > 3 {
		return nil, errors.New("Can't add multiple labels on same subscription")
	}

	features := strings.Split(Features, ",")
	badFeatures := make([]string, 0)
	for _, f := range features {
		if _, ok := allFeatures[f]; !strings.HasPrefix(f, "label:") && !ok {
			badFeatures = append(badFeatures, f)
		}
	}
	if len(badFeatures) > 0 {
		return nil, fmt.Errorf("Unknown features %s", strings.Join(badFeatures, ","))
	}
	return &Subscription{
		ChannelID:  ChannelID,
		CreatorID:  CreatorID,
		Features:   Features,
		Repository: Repository,
	}, nil
}

func (s *Subscription) Merges() bool {
	return strings.Contains(s.Features, "merges")
}

func (s *Subscription) Issues() bool {
	return strings.Contains(s.Features, "issues")
}

func (s *Subscription) Pushes() bool {
	return strings.Contains(s.Features, "pushes")
}

func (s *Subscription) IssueComments() bool {
	return strings.Contains(s.Features, "issue_comments")
}

func (s *Subscription) MergeRequestComments() bool {
	return strings.Contains(s.Features, "merge_request_comments")
}

func (s *Subscription) Pipeline() bool {
	return strings.Contains(s.Features, "pipeline")
}

func (s *Subscription) Tag() bool {
	return strings.Contains(s.Features, "tag")
}

func (s *Subscription) PullReviews() bool {
	return strings.Contains(s.Features, "pull_reviews")
}

func (s *Subscription) Label() string {
	if !strings.Contains(s.Features, "label:") {
		return ""
	}
	return strings.Split(s.Features, "\"")[1]
}
