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

// PipelineInfo is the struct describing the status of
// a running Job which is part of a pieline
type PipelineJobInfo struct {
	JobID  int
	Status string
	Ref    string
	WebURL string
	User   string
}
