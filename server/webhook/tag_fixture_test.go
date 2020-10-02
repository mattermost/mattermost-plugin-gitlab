package webhook

const SimpleTag = `{
	"object_kind":"tag_push",
	"event_name":"tag_push",
	"before":"0000000000000000000000000000000000000000",
	"after":"48bb9a42241a84ec7c03850f0096ac671bb06641",
	"ref":"refs/tags/tag1",
	"checkout_sha":"c30217b62542c586fdbadc7b5ee762bfdca10663",
	"message":"Really beautiful tag",
	"user_id":50,
	"user_name":"manland",
	"user_username":"manland",
	"user_email":"",
	"user_avatar":"https://www.gravatar.com/avatar/c6b552a4cd47f7cf1701ea5b650cd2e3?s=80\\u0026d=identicon",
	"project_id":24,
	"project":{
		"id":24,
		"name":"webhook",
		"description":"",
		"web_url":"http://localhost:3000/manland/webhook",
		"avatar_url":null,
		"git_ssh_url":"ssh://rmaneschi@localhost:2222/manland/webhook.git",
		"git_http_url":"http://localhost:3000/manland/webhook.git",
		"namespace":"manland",
		"visibility_level":20,
		"path_with_namespace":"manland/webhook",
		"default_branch":"master",
		"ci_config_path":null,
		"homepage":"http://localhost:3000/manland/webhook",
		"url":"ssh://rmaneschi@localhost:2222/manland/webhook.git",
		"ssh_url":"ssh://rmaneschi@localhost:2222/manland/webhook.git",
		"http_url":"http://localhost:3000/manland/webhook.git"
	},
	"commits":[{
		"id":"c30217b62542c586fdbadc7b5ee762bfdca10663",
		"message":"really cool commit",
		"timestamp":"2019-04-17T20:22:03Z",
		"url":"http://localhost:3000/manland/webhook/commit/c30217b62542c586fdbadc7b5ee762bfdca10663",
		"author":{
			"name":"manland",
			"email":"rmaneschi@gmail.com"
		},
		"added":[],
		"modified":["README.md"],
		"removed":[]
	}],
	"total_commits_count":1,
	"push_options":{
		
	},
	"repository":{
		"name":"webhook",
		"url":"ssh://rmaneschi@localhost:2222/manland/webhook.git",
		"description":"",
		"homepage":"http://localhost:3000/manland/webhook",
		"git_http_url":"http://localhost:3000/manland/webhook.git",
		"git_ssh_url":"ssh://rmaneschi@localhost:2222/manland/webhook.git",
		"visibility_level":20
	}
}`
