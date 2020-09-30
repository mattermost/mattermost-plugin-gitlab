package webhook

const (
	actionOpen   = "open"
	actionClose  = "close"
	actionMerge  = "merge"
	actionReopen = "reopen"
	actionUpdate = "update"

	stateOpened = "opened"
	stateClosed = "closed"
	stateMerged = "merged"

	statusSuccess = "success"
	statusRunning = "running"
	statusFailed  = "failed"
)
