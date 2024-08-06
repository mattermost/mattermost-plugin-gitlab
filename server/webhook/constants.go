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

	statusSuccess = "success"
	statusRunning = "running"
	statusPending = "pending"
	statusFailed  = "failed"
	statusCreated = "created"
	statusCanceled = "canceled"

	statusCreate = "create"
	statusUpdate = "update"
	statusDelete = "delete"

	PrivateVisibilityLevel = 0
	PublicVisibilityLevel  = 20
)
