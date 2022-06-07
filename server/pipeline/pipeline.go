package pipeline

// Subscription struct describing the subscription for a pipeline
type Subscription struct {
	PipelineID int
	ChannelID  string
	ProjectID  string
	Repository string
}
