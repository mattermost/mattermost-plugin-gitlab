module github.com/manland/mattermost-plugin-gitlab

go 1.12

require github.com/manland/mattermost-plugin-gitlab/server/webhook v0.0.0

replace github.com/manland/mattermost-plugin-gitlab/server/webhook v0.0.0 => ./webhook

require github.com/manland/mattermost-plugin-gitlab/server/gitlab v0.0.0

replace github.com/manland/mattermost-plugin-gitlab/server/gitlab v0.0.0 => ./gitlab

require github.com/manland/mattermost-plugin-gitlab/server/subscription v0.0.0

replace github.com/manland/mattermost-plugin-gitlab/server/subscription v0.0.0 => ./subscription

require (
	github.com/Masterminds/squirrel v1.1.0 // indirect
	github.com/go-gorp/gorp v2.0.0+incompatible // indirect
	github.com/go-redis/redis v6.15.2+incompatible // indirect
	github.com/go-sql-driver/mysql v1.4.1 // indirect
	github.com/golang/protobuf v1.3.1 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/hashicorp/go-hclog v0.0.0-20180910232447-e45cbeb79f04 // indirect
	github.com/hashicorp/go-plugin v0.0.0-20180814222501-a4620f9913d1 // indirect
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/lib/pq v1.1.0 // indirect
	github.com/mattermost/gorp v2.0.0+incompatible // indirect
	github.com/mattermost/mattermost-server v5.10.0+incompatible
	github.com/mattermost/viper v0.0.0-20181112161711-f99c30686b86 // indirect
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.1.2 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/xanzy/go-gitlab v0.16.2-0.20190430153925-4baaa27b8479
	github.com/ziutek/mymysql v1.5.4 // indirect
	golang.org/x/net v0.0.0-20190420063019-afa5a82059c6 // indirect
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a
	golang.org/x/sys v0.0.0-20190422165155-953cdadca894 // indirect
	google.golang.org/appengine v1.5.0 // indirect
	google.golang.org/genproto v0.0.0-20190418145605-e7d98fc518a7 // indirect
	google.golang.org/grpc v1.20.1 // indirect
)
