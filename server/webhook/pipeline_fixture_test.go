package webhook

const PipelinePending = `{
	"object_kind":"pipeline",
	"object_attributes":{
		"id":58,
		"ref":"master",
		"tag":false,
		"sha":"ec0a1bcd4580bfec3495674e412f4834ee2c2550",
		"before_sha":"c30217b62542c586fdbadc7b5ee762bfdca10663",
		"source": "merge_request_event",
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
		"message":"Start gitlab-ci\n",
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

const PipelineRun = `{
	"object_kind":"pipeline",
	"object_attributes":{
		"id":62,
		"ref":"master",
		"tag":false,
		"sha":"ec0a1bcd4580bfec3495674e412f4834ee2c2550",
		"before_sha":"0000000000000000000000000000000000000000",
		"source": "merge_request_event",
		"status":"running",
		"detailed_status":"running",
		"stages":["deploy"],
		"created_at":"2019-05-01 12:32:47 UTC",
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
		"message":"Start gitlab-ci\n",
		"timestamp":"2019-04-17T20:38:43Z",
		"url":"http://localhost:3000/manland/webhook/commit/ec0a1bcd4580bfec3495674e412f4834ee2c2550",
		"author":{
			"name":"Administrator",
			"email":"admin@example.com"
		}
	},
	"builds": [
	{
		"id":1136,
		"stage":"deploy",
		"name":"pages",
		"status":"running",
		"created_at":"2019-05-01 12:32:47 UTC",
		"started_at":"2019-05-01 12:32:49 UTC",
		"finished_at":null,
		"when":"on_success",
		"manual":false,
		"user":{
			"name":"Administrator",
			"username":"root",
			"avatar_url":"https://www.gravatar.com/avatar/e64c7d89f26bd1972efa854d13d7dd61?s=80\\u0026d=identicon"
		},
		"runner":{
			"id":1,
			"description":"localhost",
			"active":true,
			"is_shared":true
		},
		"artifacts_file":{
			"filename":null,
			"size":0
		}
	}]
}`

const PipelineFail = `{
	"object_kind":"pipeline",
	"object_attributes":{
		"id":62,
		"ref":"master",
		"tag":false,
		"sha":"ec0a1bcd4580bfec3495674e412f4834ee2c2550",
		"before_sha":"0000000000000000000000000000000000000000",
		"source": "merge_request_event",
		"status":"failed",
		"detailed_status":"failed",
		"stages":["deploy"],
		"created_at":"2019-05-01 12:32:47 UTC",
		"finished_at":"2019-05-01 12:33:04 UTC",
		"duration":14,
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
		"message":"Start gitlab-ci\n",
		"timestamp":"2019-04-17T20:38:43Z",
		"url":"http://localhost:3000/manland/webhook/commit/ec0a1bcd4580bfec3495674e412f4834ee2c2550",
		"author":{
			"name":"Administrator",
			"email":"admin@example.com"
		}
	},
	"builds":[
	{
		"id":1136,
		"stage":"deploy",
		"name":"pages",
		"status":"failed",
		"created_at":"2019-05-01 12:32:47 UTC",
		"started_at":"2019-05-01 12:32:49 UTC",
		"finished_at":"2019-05-01 12:33:04 UTC",
		"when":"on_success",
		"manual":false,
		"user":{
			"name":"Administrator",
			"username":"root",
			"avatar_url":"https://www.gravatar.com/avatar/e64c7d89f26bd1972efa854d13d7dd61?s=80\\u0026d=identicon"
		},
		"runner":{
			"id":1,
			"description":"localhost",
			"active":true,
			"is_shared":true
		},
		"artifacts_file":{
			"filename":null,
			"size":0
		}
	}]
}`

const PipelineSuccess = `{
	"object_kind":"pipeline",
	"object_attributes":{
		"id":62,
		"ref":"master",
		"tag":false,
		"sha":"ec0a1bcd4580bfec3495674e412f4834ee2c2550",
		"before_sha":"0000000000000000000000000000000000000000",
		"source": "merge_request_event",
		"status":"success",
		"detailed_status":"passed",
		"stages":["deploy", "deploy"],
		"created_at":"2019-05-01 12:32:47 UTC",
		"finished_at":"2019-05-01 13:55:51 UTC",
		"duration":19,
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
		"message":"Start gitlab-ci\n",
		"timestamp":"2019-04-17T20:38:43Z",
		"url":"http://localhost:3000/manland/webhook/commit/ec0a1bcd4580bfec3495674e412f4834ee2c2550",
		"author":{
			"name":"Administrator",
			"email":"admin@example.com"
		}
	},
	"builds":[
	{
		"id":1142,
		"stage":"deploy",
		"name":"pages",
		"status":"success",
		"created_at":"2019-05-01 13:55:29 UTC",
		"started_at":"2019-05-01 13:55:31 UTC",
		"finished_at":"2019-05-01 13:55:51 UTC",
		"when":"on_success",
		"manual":false,
		"user":{
			"name":"Administrator",
			"username":"root",
			"avatar_url":"https://www.gravatar.com/avatar/e64c7d89f26bd1972efa854d13d7dd61?s=80\\u0026d=identicon"
		},
		"runner":{
			"id":3,
			"description":"ip",
			"active":true,
			"is_shared":true
		},
		"artifacts_file":{
			"filename":"artifacts.zip",
			"size":2520
		}
	}
	]
}`
