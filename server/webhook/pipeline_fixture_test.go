package webhook

const PipelineStart = `{
	"object_kind":"pipeline",
	"object_attributes":{
		"id":58,
		"ref":"master",
		"tag":false,
		"sha":"ec0a1bcd4580bfec3495674e412f4834ee2c2550",
		"before_sha":"c30217b62542c586fdbadc7b5ee762bfdca10663",
		"status":"pending",
		"detailed_status":"pending",
		"stages":["deploy"],
		"created_at":"2019-04-17 20:38:44 UTC",
		"finished_at":null,
		"duration":null,
		"variables":[]
	},
	"user":{
		"name":"Administrator",
		"username":"root",
		"avatar_url":"https://www.gravatar.com/avatar/e64c7d89f26bd1972efa854d13d7dd61?s=80\\u0026d=identicon"
	},
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
		"ci_config_path":null
	},
	"commit":{
		"id":"ec0a1bcd4580bfec3495674e412f4834ee2c2550",
		"message":"Start gitlab-ci",
		"timestamp":"2019-04-17T20:38:43Z",
		"url":"http://localhost:3000/manland/webhook/commit/ec0a1bcd4580bfec3495674e412f4834ee2c2550",
		"author":{
			"name":"Administrator",
			"email":"admin@example.com"
		}
	},
	"builds":[{
		"id":1126,
		"stage":"deploy",
		"name":"pages",
		"status":"pending",
		"created_at":"2019-04-17 20:38:44 UTC",
		"started_at":null,
		"finished_at":null,
		"when":"on_success",
		"manual":false,
		"user":{
			"name":"Administrator",
			"username":"root",
			"avatar_url":"https://www.gravatar.com/avatar/e64c7d89f26bd1972efa854d13d7dd61?s=80\\u0026d=identicon"},
			"runner":null,
			"artifacts_file":{
				"filename":null,
				"size":0
			}
	}]
}`
