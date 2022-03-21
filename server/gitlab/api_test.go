package gitlab

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestTriggerNewBuildPipeline(t *testing.T) {
	userInfo := &UserInfo{
		Token: &oauth2.Token{
			AccessToken: "glpat-KSxw-zYxNstVGy-ZjSK7",
		},
	}

	var (
		g         gitlab
		repo      = "20012149"
		commitRef = "master"
	)

	resp, err := g.TriggerNewBuildPipeline(userInfo, repo, commitRef)
	require.NoError(t, err)
	require.NotNil(t, resp)

	t.Log(resp)
}