package gitlab

// PipelineInfo is the struct describing the status of
// a running pipeline.
type PipelineInfo struct {
	PipelineID int
	Status     string
	Ref        string
	WebURL     string
	SHA        string
	User       string
}

// PipelineJobInfo is the struct describing the status of
// a running Job which is part of a pipeline
type PipelineJobInfo struct {
	JobID  int
	Status string
	Ref    string
	WebURL string
	User   string
}
