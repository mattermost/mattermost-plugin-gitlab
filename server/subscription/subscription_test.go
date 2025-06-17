// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package subscription

import (
	"github.com/stretchr/testify/require"
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
	labels, err := s.Labels()
	require.NoError(t, err)
	assert.Equal(t, []string{}, labels)
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
	labels, err := s.Labels()
	require.NoError(t, err)
	assert.Equal(t, []string{}, labels)
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
	labels, err := s.Labels()
	require.NoError(t, err)
	assert.Equal(t, []string{}, labels)
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
	labels, err := s.Labels()
	require.NoError(t, err)
	assert.Equal(t, []string{"test"}, labels)
}

func TestNewSubscriptionBadFormated(t *testing.T) {
	s, err := New("", "", `label:"`, "")
	assert.Nil(t, s)
	assert.Equal(t, err.Error(), `each label must be wrapped in quotes, e.g. label:"bug"`)
}

func TestNewSubscriptionMultipleLabelWithIssues(t *testing.T) {
	s, err := New("", "", `issues,label:"1",label:"2"`, "")
	assert.Nil(t, err)
	labels, err := s.Labels()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "2"}, labels)
}

func TestNewSubscriptionMultipleLabelWithMerges(t *testing.T) {
	s, err := New("", "", `merges,label:"1",label:"2"`, "")
	assert.Nil(t, err)
	labels, err := s.Labels()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "2"}, labels)
}

func TestNewSubscriptionMultipleLabelWithoutIssuesMerges(t *testing.T) {
	_, err := New("", "", `label:"1",label:"2"`, "")
	assert.Equal(t, err.Error(), "label filters require 'merges' or 'issues' feature")
}

func TestNewSubscriptionMultipleLabelWithSpaces(t *testing.T) {
	s, err := New("", "", `merges,label: "1",label: "2"`, "")
	assert.Nil(t, err)
	labels, err := s.Labels()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "2"}, labels)
}
