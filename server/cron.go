package main

import (
	"fmt"

	"github.com/robfig/cron"
)

// Cron manage all cron jobs of this plugin
// behind the scene it's a facade to github.com/robfig/cron
type Cron struct {
	p *Plugin
	c *cron.Cron
}

// NewCron return a cron
func NewCron(p *Plugin) (*Cron, error) {
	c := cron.New()
	config := p.getConfiguration()

	if err := c.AddFunc("@every 1h", func() {
		subscriptions, err := p.GetSubscriptions()
		if err != nil {
			p.API.LogError("can't get subscription retry in 1h", "err", err.Error())
			return
		}
		for _, subs := range subscriptions.Repositories {
			for _, sub := range subs {
				gitlabUserInfo, err := p.getGitlabUserInfoByMattermostID(sub.CreatorID)
				if err != nil {
					p.API.LogError("can't get gitlab user info by mattermost id", "err", err.Message)
					continue
				}
				_, owner, repo := parseOwnerAndRepo(sub.Repository, config.GitlabURL)
				url := fmt.Sprintf("%s/plugins/%s/webhook", *p.API.GetConfig().ServiceSettings.SiteURL, manifest.ID)
				if err := p.GitlabClient.AddWebHooks(gitlabUserInfo, owner, repo, url, config.WebhookSecret); err != nil {
					p.API.LogError("can't save current analytic", "err", err.Error())
				}
			}
		}
	}); err != nil {
		return nil, err
	}

	c.Start()

	return &Cron{
		p: p,
		c: c,
	}, nil
}

// Stop the cron task and save data
func (c *Cron) Stop() {
	c.c.Stop()
}
