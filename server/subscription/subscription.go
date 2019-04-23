package subscription

import "strings"

type Subscription struct {
	ChannelID  string
	CreatorID  string
	Features   string
	Repository string
}

func New(ChannelID, CreatorID, Features, Repository string) *Subscription {
	return &Subscription{
		ChannelID:  ChannelID,
		CreatorID:  CreatorID,
		Features:   Features,
		Repository: Repository,
	}
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

	labelSplit := strings.Split(s.Features, "\"")
	if len(labelSplit) < 3 {
		return ""
	}

	return labelSplit[1]
}
