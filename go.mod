module github.com/mattermost/mattermost-plugin-gitlab

go 1.16

require (
	github.com/golang/mock v1.6.0
	github.com/gorilla/mux v1.8.0
	github.com/mattermost/mattermost-plugin-api v0.0.26
	github.com/mattermost/mattermost-server/v6 v6.3.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/xanzy/go-gitlab v0.55.0
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)

// Until github.com/mattermost/mattermost-server/v6 v6.5.0 is releated,
// this replacement is needed to also import github.com/mattermost/mattermost-plugin-api,
// which uses a different server version.
replace github.com/mattermost/mattermost-server/v6 v6.3.0 => github.com/mattermost/mattermost-server/v6 v6.0.0-20220210052000-0d67995eb491
