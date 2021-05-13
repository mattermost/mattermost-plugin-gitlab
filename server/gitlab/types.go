package gitlab

import (
	"time"

	internGitlab "github.com/xanzy/go-gitlab"
)

type TriggerPipelineResponse struct {
	ID          int        `json:"id"`
	Status      string     `json:"status"`
	Ref         string     `json:"ref"`
	Tag         bool       `json:"tag"`
	Username    string     `json:"username"`
	Name        string     `json:"name"`
	UserState   string     `json:"state"`
	UpdatedAt   *time.Time `json:"updated_at"`
	CommittedAt *time.Time `json:"committed_at"`
	WebURL      string     `json:"web_url"`
	AvatarURL   string     `json:"avatar_url"`
}

func newTriggerPipelineResponse(pipeline *internGitlab.Pipeline) *TriggerPipelineResponse {
	return &TriggerPipelineResponse{
		ID:          pipeline.ID,
		Status:      pipeline.Status,
		Ref:         pipeline.Ref,
		Tag:         pipeline.Tag,
		Username:    pipeline.User.Username,
		Name:        pipeline.User.Name,
		UserState:   pipeline.User.State,
		UpdatedAt:   pipeline.UpdatedAt,
		CommittedAt: pipeline.CommittedAt,
		WebURL:      pipeline.WebURL,
		AvatarURL:   pipeline.User.AvatarURL,
	}
}

func (p *TriggerPipelineResponse) String() string {
	return internGitlab.Stringify(p)
}
