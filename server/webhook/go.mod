module github.com/manland/mattermost-plugin-gitlab/server/webhook

go 1.12

require github.com/manland/mattermost-plugin-gitlab/server/subscription v0.0.0

replace github.com/manland/mattermost-plugin-gitlab/server/subscription v0.0.0 => ../subscription

require (
	github.com/stretchr/testify v1.3.0
	github.com/xanzy/go-gitlab v0.16.2-0.20190430153925-4baaa27b8479
)
