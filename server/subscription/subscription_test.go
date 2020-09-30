package subscription

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSubscriptionSimple(t *testing.T) {
	s, err := New("", "", "issues", "")
	assert.Nil(t, err)
	assert.Equal(t, &Subscription{Features: "issues"}, s)
	assert.True(t, s.Issues())
	assert.False(t, s.Merges())
	assert.False(t, s.Pushes())
	assert.False(t, s.IssueComments())
	assert.False(t, s.MergeRequestComments())
	assert.False(t, s.Pipeline())
	assert.False(t, s.Tag())
	assert.False(t, s.PullReviews())
	assert.Equal(t, "", s.Label())
}

func TestNewSubscriptionMultiple(t *testing.T) {
	s, err := New("", "", "issues,merges", "")
	assert.Nil(t, err)
	assert.Equal(t, &Subscription{Features: "issues,merges"}, s)
	assert.True(t, s.Issues())
	assert.True(t, s.Merges())
	assert.False(t, s.Pushes())
	assert.False(t, s.IssueComments())
	assert.False(t, s.MergeRequestComments())
	assert.False(t, s.Pipeline())
	assert.False(t, s.Tag())
	assert.False(t, s.PullReviews())
	assert.Equal(t, "", s.Label())
}

func TestNewSubscriptionAll(t *testing.T) {
	s, err := New("", "", "issues,merges,pushes,issue_comments,merge_request_comments,pipeline,tag,pull_reviews", "")
	assert.Nil(t, err)
	assert.Equal(t, &Subscription{Features: "issues,merges,pushes,issue_comments,merge_request_comments,pipeline,tag,pull_reviews"}, s)
	assert.True(t, s.Issues())
	assert.True(t, s.Merges())
	assert.True(t, s.Pushes())
	assert.True(t, s.IssueComments())
	assert.True(t, s.MergeRequestComments())
	assert.True(t, s.Pipeline())
	assert.True(t, s.Tag())
	assert.True(t, s.PullReviews())
	assert.Equal(t, "", s.Label())
}

func TestNewSubscriptionUnknown(t *testing.T) {
	s, err := New("", "", "unknown,merges,missing", "")
	assert.Nil(t, s)
	assert.Equal(t, err.Error(), "unknown features unknown,missing")
}

func TestNewSubscriptionLabel(t *testing.T) {
	s, err := New("", "", `issues,label:"test",merges`, "")
	assert.Nil(t, err)
	assert.True(t, s.Merges())
	assert.True(t, s.Issues())
	assert.Equal(t, "test", s.Label())
}

func TestNewSubscriptionBadFormated(t *testing.T) {
	s, err := New("", "", `label:"`, "")
	assert.Nil(t, s)
	assert.Equal(t, err.Error(), "the label is formatted incorrectly")
}

func TestNewSubscriptionMultipleLabel(t *testing.T) {
	s, err := New("", "", `label:"1",label:"2"`, "")
	assert.Nil(t, s)
	assert.Equal(t, err.Error(), "can't add multiple labels on the same subscription")
}
