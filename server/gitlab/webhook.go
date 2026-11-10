// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
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

// MissingTriggers returns the names of the requested event triggers that the existing webhook does not have enabled.
// SSL verification is a delivery setting, reported by SSLVerificationDisabled.
func (o *AddWebhookOptions) MissingTriggers(w *WebhookInfo) []string {
	checks := []struct {
		name      string
		requested bool
		enabled   bool
	}{
		{"Push Events", o.PushEvents, w.PushEvents},
		{"Tag Push Events", o.TagPushEvents, w.TagPushEvents},
		{"Comments", o.NoteEvents, w.NoteEvents},
		{"Confidential Comments", o.ConfidentialNoteEvents, w.ConfidentialNoteEvents},
		{"Issues Events", o.IssuesEvents, w.IssuesEvents},
		{"Confidential Issues Events", o.ConfidentialIssuesEvents, w.ConfidentialIssuesEvents},
		{"Merge Request Events", o.MergeRequestsEvents, w.MergeRequestsEvents},
		{"Job Events", o.JobEvents, w.JobEvents},
		{"Pipeline Events", o.PipelineEvents, w.PipelineEvents},
		{"Wiki Page Events", o.WikiPageEvents, w.WikiPageEvents},
		{"Release Events", o.ReleaseEvents, w.ReleaseEvents},
		{"Deployment Events", o.DeploymentEvents, w.DeploymentEvents},
	}

	var missing []string
	for _, c := range checks {
		if c.requested && !c.enabled {
			missing = append(missing, c.name)
		}
	}
	return missing
}

// SSLVerificationDisabled reports that SSL verification was requested and the existing webhook has it turned off.
func (o *AddWebhookOptions) SSLVerificationDisabled(w *WebhookInfo) bool {
	return o.EnableSSLVerification && !w.EnableSSLVerification
}
