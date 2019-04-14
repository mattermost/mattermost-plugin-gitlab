package webhook

import "fmt"

type fakeWebhook struct{}

func (fakeWebhook) GetUserURL(username string) string {
	return fmt.Sprintf("http://my.gitlab.com/%s", username)
}

func (fakeWebhook) GetUsernameByID(id int) string {
	if id == 1 {
		return "root"
	} else if id == 50 {
		return "manland"
	} else {
		return ""
	}
}

func (fakeWebhook) ParseGitlabUsernamesFromText(body string) []string {
	return []string{}
}
