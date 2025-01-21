// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gitlab

import (
	"time"
)

// WebhookInfo Provides information about group or project hooks.
type WebhookInfo struct {
	ID                       int
	URL                      string
	ConfidentialNoteEvents   bool
	PushEvents               bool
	IssuesEvents             bool
	ConfidentialIssuesEvents bool
	MergeRequestsEvents      bool
	TagPushEvents            bool
	NoteEvents               bool
	JobEvents                bool
	PipelineEvents           bool
	WikiPageEvents           bool
	DeploymentEvents         bool
	ReleaseEvents            bool
	EnableSSLVerification    bool
	CreatedAt                *time.Time
	Scope                    Scope
}

// AddWebhookOptions is a paramater object with options for creating a project or group hook.
type AddWebhookOptions struct {
	URL                      string
	ConfidentialNoteEvents   bool
	PushEvents               bool
	IssuesEvents             bool
	ConfidentialIssuesEvents bool
	MergeRequestsEvents      bool
	TagPushEvents            bool
	NoteEvents               bool
	JobEvents                bool
	PipelineEvents           bool
	WikiPageEvents           bool
	DeploymentEvents         bool
	ReleaseEvents            bool
	EnableSSLVerification    bool
	Token                    string
}
