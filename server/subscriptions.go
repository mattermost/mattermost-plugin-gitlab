package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/manland/mattermost-plugin-gitlab/server/gitlab"
	"github.com/manland/mattermost-plugin-gitlab/server/subscription"
	"github.com/pkg/errors"
)

const (
	SUBSCRIPTIONS_KEY = "subscriptions"
)

type Subscriptions struct {
	Repositories map[string][]*subscription.Subscription
}

func (p *Plugin) Subscribe(info *gitlab.GitlabUserInfo, owner, repo, channelID, features string) error {
	if owner == "" {
		return fmt.Errorf("Invalid repository")
	}

	if err := p.checkGroup(owner); err != nil {
		return err
	}

	exist, err := p.GitlabClient.Exist(info, owner, repo, p.getConfiguration().EnablePrivateRepo)
	if !exist || err != nil {
		if err != nil {
			p.API.LogError(fmt.Sprintf("Unable to retreive informations for %s", fullNameFromOwnerAndRepo(owner, repo)), "err", err.Error())
		}
		return fmt.Errorf("Unable to retreive informations for %s", fullNameFromOwnerAndRepo(owner, repo))
	}

	sub, err := subscription.New(channelID, info.UserID, features, fullNameFromOwnerAndRepo(owner, repo))
	if err != nil {
		return err
	}

	if err := p.AddSubscription(fullNameFromOwnerAndRepo(owner, repo), sub); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) SubscribeGroup(info *gitlab.GitlabUserInfo, org, channelID, features string) error {
	if org == "" {
		return fmt.Errorf("Invalid group")
	}
	return p.Subscribe(info, org, "", channelID, features)
}

func (p *Plugin) GetSubscriptionsByChannel(channelID string) ([]*subscription.Subscription, error) {
	var filteredSubs []*subscription.Subscription
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil, err
	}

	for _, v := range subs.Repositories {
		for _, s := range v {
			if s.ChannelID == channelID {
				filteredSubs = append(filteredSubs, s)
			}
		}
	}

	return filteredSubs, nil
}

func (p *Plugin) AddSubscription(repo string, sub *subscription.Subscription) error {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return err
	}

	repoSubs := subs.Repositories[repo]
	if repoSubs == nil {
		repoSubs = []*subscription.Subscription{sub}
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
	return p.StoreSubscriptions(subs)
}

func (p *Plugin) GetSubscriptions() (*Subscriptions, error) {
	var subscriptions *Subscriptions

	value, err := p.API.KVGet(SUBSCRIPTIONS_KEY)
	if err != nil {
		p.API.LogError("can't get subscriptions from kvstore", "err", err.DetailedError)
		return nil, err
	}

	if value == nil {
		subscriptions = &Subscriptions{Repositories: map[string][]*subscription.Subscription{}}
	} else {
		if err := json.NewDecoder(bytes.NewReader(value)).Decode(&subscriptions); err != nil {
			return nil, err
		}
	}

	return subscriptions, nil
}

func (p *Plugin) StoreSubscriptions(s *Subscriptions) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := p.API.KVSet(SUBSCRIPTIONS_KEY, b); err != nil {
		p.API.LogError("can't set subscriptions in kvstore", "err", err.DetailedError)
	}
	return nil
}

func (p *Plugin) GetSubscribedChannelsForRepository(fullNameOwnerAndRepo string, repoPublic bool) []*subscription.Subscription {
	group := strings.Split(fullNameOwnerAndRepo, "/")[0]
	subsForRepo := []*subscription.Subscription{}

	subs, err := p.GetSubscriptions()
	if err != nil {
		p.API.LogError("can't retrieve subscriptions", "err", err.Error())
		return subsForRepo
	}

	// Add subscriptions for the specific repo
	if subs.Repositories[fullNameOwnerAndRepo] != nil {
		subsForRepo = append(subsForRepo, subs.Repositories[fullNameOwnerAndRepo]...)
	}

	// Add subscriptions for the organization
	groupKey := fullNameFromOwnerAndRepo(group, "")
	if subs.Repositories[groupKey] != nil {
		subsForRepo = append(subsForRepo, subs.Repositories[groupKey]...)
	}

	if len(subsForRepo) == 0 {
		return nil
	}

	subsToReturn := []*subscription.Subscription{}

	for _, sub := range subsForRepo {
		if !repoPublic && !p.permissionToRepo(sub.CreatorID, fullNameOwnerAndRepo) {
			continue
		}
		subsToReturn = append(subsToReturn, sub)
	}

	return subsToReturn
}

// Unsubscribe deletes the link between channelID and repo
// returns true if repo was found, else false
func (p *Plugin) Unsubscribe(channelID string, repo string) (bool, error) {
	config := p.getConfiguration()

	if repo == "" {
		return false, errors.New("Invalid repository")
	}

	_, owner, project := parseOwnerAndRepo(repo, config.GitlabURL)
	repo = fullNameFromOwnerAndRepo(owner, project)

	subs, err := p.GetSubscriptions()
	if err != nil {
		return false, err
	}

	repoSubs := subs.Repositories[repo]
	if repoSubs == nil {
		return false, nil
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
		if len(repoSubs) > 0 {
			subs.Repositories[repo] = repoSubs
		} else {
			delete(subs.Repositories, repo)
		}
		if err := p.StoreSubscriptions(subs); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}
