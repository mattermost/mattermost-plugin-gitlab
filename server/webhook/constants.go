// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package webhook

const (
	actionOpen       = "open"
	actionClose      = "close"
	actionMerge      = "merge"
	actionReopen     = "reopen"
	actionUpdate     = "update"
	actionApproved   = "approved"
	actionUnapproved = "unapproved"

	stateOpened = "opened"
	stateClosed = "closed"
	stateMerged = "merged"

	statusSuccess  = "success"
	statusRunning  = "running"
	statusPending  = "pending"
	statusFailed   = "failed"
	statusCreated  = "created"
	statusCanceled = "canceled"

	statusCreate = "create"
	statusUpdate = "update"
	statusDelete = "delete"

	// eventTypeConfidentialNote is the payload's event_type value for internal
	// notes. It differs from gitlab.EventConfidentialNote, which is the
	// X-Gitlab-Event header value ("Confidential Note Hook").
	eventTypeConfidentialNote = "confidential_note"

	PrivateVisibilityLevel = 0
	PublicVisibilityLevel  = 20
)
