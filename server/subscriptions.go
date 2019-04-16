package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/manland/go-gitlab"
	"github.com/pkg/errors"
)

const (
	SUBSCRIPTIONS_KEY = "subscriptions"
)

type Subscription struct {
	ChannelID  string
	CreatorID  string
	Features   string
	Repository string
}

type Subscriptions struct {
	Repositories map[string][]*Subscription
}

func (s *Subscription) Pulls() bool {
	return strings.Contains(s.Features, "pulls")
}

func (s *Subscription) Issues() bool {
	return strings.Contains(s.Features, "issues")
}

func (s *Subscription) Pushes() bool {
	return strings.Contains(s.Features, "pushes")
}

func (s *Subscription) Creates() bool {
	return strings.Contains(s.Features, "creates")
}

func (s *Subscription) Deletes() bool {
	return strings.Contains(s.Features, "deletes")
}

func (s *Subscription) IssueComments() bool {
	return strings.Contains(s.Features, "issue_comments")
}

func (s *Subscription) PullReviews() bool {
	return strings.Contains(s.Features, "pull_reviews")
}

func (s *Subscription) Label() string {
	if !strings.Contains(s.Features, "label:") {
		return ""
	}

	labelSplit := strings.Split(s.Features, "\"")
	if len(labelSplit) < 3 {
		return ""
	}

	return labelSplit[1]
}

func (p *Plugin) Subscribe(client *gitlab.Client, userId, owner, repo, channelID, features string) error {
	if owner == "" {
		return fmt.Errorf("Invalid repository")
	}

	if err := p.checkGroup(owner); err != nil {
		return err
	}

	if result, _, err := client.Projects.GetProject(owner+"/"+repo, &gitlab.GetProjectOptions{}); result == nil || err != nil {
		if err != nil {
			p.API.LogError("can't get project", "err", err.Error(), "project", owner+"/"+repo)
		}
		return fmt.Errorf("Unknown repository %s/%s", owner, repo)
	}

	sub := &Subscription{
		ChannelID:  channelID,
		CreatorID:  userId,
		Features:   features,
		Repository: fmt.Sprintf("%s/%s", owner, repo),
	}

	if err := p.AddSubscription(fmt.Sprintf("%s/%s", owner, repo), sub); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) SubscribeGroup(client *gitlab.Client, userId, org, channelID, features string) error {
	if org == "" {
		return fmt.Errorf("Invalid group")
	}
	if err := p.checkGroup(org); err != nil {
		return err
	}

	var allRepos []*gitlab.Project
	requestOptions := &gitlab.ListGroupProjectsOptions{ListOptions: gitlab.ListOptions{PerPage: 50}}
	for {
		repos, response, err := client.Groups.ListGroupProjects(org, requestOptions)
		if err != nil {
			p.API.LogError("can't list group project", "err", err.Error())
			return errors.Wrap(err, "can't list group project")
		}
		allRepos = append(allRepos, repos...)
		if response.NextPage == 0 {
			break
		}
		requestOptions.Page = response.NextPage
	}

	for _, repo := range allRepos {
		sub := &Subscription{
			ChannelID:  channelID,
			CreatorID:  userId,
			Features:   features,
			Repository: repo.Name,
		}

		if err := p.AddSubscription(sub.Repository, sub); err != nil {
			p.API.LogError("can't add subscipriotn", "err", err.Error(), "repository", sub.Repository)
			continue
		}
	}

	return nil
}

func (p *Plugin) GetSubscriptionsByChannel(channelID string) ([]*Subscription, error) {
	var filteredSubs []*Subscription
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil, err
	}

	for repo, v := range subs.Repositories {
		for _, s := range v {
			if s.ChannelID == channelID {
				// this is needed to be backwards compatible
				if len(s.Repository) == 0 {
					s.Repository = repo
				}
				filteredSubs = append(filteredSubs, s)
			}
		}
	}

	return filteredSubs, nil
}

func (p *Plugin) AddSubscription(repo string, sub *Subscription) error {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return err
	}

	repoSubs := subs.Repositories[repo]
	if repoSubs == nil {
		repoSubs = []*Subscription{sub}
	} else {
		exists := false
		for index, s := range repoSubs {
			if s.ChannelID == sub.ChannelID {
				repoSubs[index] = sub
				exists = true
				break
			}
		}

		if !exists {
			repoSubs = append(repoSubs, sub)
		}
	}

	subs.Repositories[repo] = repoSubs

	err = p.StoreSubscriptions(subs)
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) GetSubscriptions() (*Subscriptions, error) {
	var subscriptions *Subscriptions

	value, err := p.API.KVGet(SUBSCRIPTIONS_KEY)
	if err != nil {
		return nil, err
	}

	if value == nil {
		subscriptions = &Subscriptions{Repositories: map[string][]*Subscription{}}
	} else {
		json.NewDecoder(bytes.NewReader(value)).Decode(&subscriptions)
	}

	return subscriptions, nil
}

func (p *Plugin) StoreSubscriptions(s *Subscriptions) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	p.API.KVSet(SUBSCRIPTIONS_KEY, b)
	return nil
}

func (p *Plugin) GetSubscribedChannelsForRepository(repoName string, repoPublic bool) []*Subscription {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil
	}

	subsForRepo := subs.Repositories[repoName]
	if subsForRepo == nil {
		return nil
	}

	subsToReturn := []*Subscription{}

	for _, sub := range subsForRepo {
		if !repoPublic && !p.permissionToRepo(sub.CreatorID, repoName) {
			continue
		}
		subsToReturn = append(subsToReturn, sub)
	}

	return subsToReturn
}

func (p *Plugin) Unsubscribe(channelID string, repo string) error {
	config := p.getConfiguration()

	repo, _, _ = parseOwnerAndRepo(repo, config.EnterpriseBaseURL)

	if repo == "" {
		return fmt.Errorf("Invalid repository")
	}

	subs, err := p.GetSubscriptions()
	if err != nil {
		return err
	}

	repoSubs := subs.Repositories[repo]
	if repoSubs == nil {
		return nil
	}

	removed := false
	for index, sub := range repoSubs {
		if sub.ChannelID == channelID {
			repoSubs = append(repoSubs[:index], repoSubs[index+1:]...)
			removed = true
			break
		}
	}

	if removed {
		subs.Repositories[repo] = repoSubs
		if err := p.StoreSubscriptions(subs); err != nil {
			return err
		}
	}

	return nil
}
