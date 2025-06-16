// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package subscription

import (
	"strings"

	"github.com/pkg/errors"
)

var allFeatures = map[string]bool{
	"merges":                 true,
	"jobs":                   true,
	"issues":                 true,
	"pushes":                 true,
	"issue_comments":         true,
	"merge_request_comments": true,
	"pipeline":               true,
	"tag":                    true,
	"pull_reviews":           true,
	"confidential_issues":    true,
	"deployments":            true,
	"releases":               true,
	// "label:":                 true,//particular case for label:XXX
}

type Subscription struct {
	ChannelID  string
	CreatorID  string
	Features   string
	Repository string
}

func New(channelID, creatorID, features, repository string) (*Subscription, error) {

	// Validate label format ― allow any number of label tokens, but each must be quoted
	if strings.Contains(features, "label:") {
		for _, t := range strings.Split(features, ",") {
			t = strings.TrimSpace(t)
			if strings.HasPrefix(t, "label:") && len(strings.Split(t, "\"")) < 3 {
				return nil, errors.New("each label must be wrapped in quotes, e.g. label:\"bug\"")
			}
		}
	}
	//if strings.Contains(features, "label:") && len(strings.Split(features, "\"")) < 3 {
	//	return nil, errors.New("the label is formatted incorrectly")
	//}
	//if strings.Contains(features, "label:") && len(strings.Split(features, "\"")) > 3 {
	//	return nil, errors.New("can't add multiple labels on the same subscription")
	//}

	badFeatures := make([]string, 0)
	for _, feature := range strings.Split(features, ",") {
		if _, ok := allFeatures[feature]; !strings.HasPrefix(feature, "label:") && !ok {
			badFeatures = append(badFeatures, feature)
		}
	}

	if len(badFeatures) > 0 {
		return nil, errors.Errorf("unknown features %s", strings.Join(badFeatures, ","))
	}
	return &Subscription{
		ChannelID:  channelID,
		CreatorID:  creatorID,
		Features:   features,
		Repository: repository,
	}, nil
}

func (s *Subscription) Merges() bool {
	return strings.Contains(s.Features, "merges")
}

func (s *Subscription) Jobs() bool {
	return strings.Contains(s.Features, "jobs")
}

func (s *Subscription) Issues() bool {
	return strings.Contains(s.Features, "issues")
}

func (s *Subscription) ConfidentialIssues() bool {
	return strings.Contains(s.Features, "confidential_issues")
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

func (s *Subscription) Labels() []string {
	labels := []string{}
	for _, f := range strings.Split(s.Features, ",") {
		f = strings.TrimSpace(f)
		if strings.HasPrefix(f, "label:") {
			parts := strings.Split(f, "\"")
			if len(parts) >= 2 {
				labels = append(labels, parts[1])
			}
		}
	}
	return labels
}

func (s *Subscription) Releases() bool {
	return strings.Contains(s.Features, "releases")
}

func (s *Subscription) Deployments() bool {
	return strings.Contains(s.Features, "deployments")
}
