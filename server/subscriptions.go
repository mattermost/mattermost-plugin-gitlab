package main

import (
	"context"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
)

const (
	SubscriptionsKey = "subscriptions"
)

type Subscriptions struct {
	Repositories map[string][]*subscription.Subscription
}

func (p *Plugin) Subscribe(info *gitlab.UserInfo, namespace, project, channelID, features string) (*Subscriptions, error) {
	if err := p.isNamespaceAllowed(namespace); err != nil {
		return nil, err
	}

	fullPath := fullPathFromNamespaceAndProject(namespace, project)
	sub, err := subscription.New(channelID, info.UserID, features, fullPath)
	if err != nil {
		return nil, err
	}

	subs, err := p.AddSubscription(fullPath, sub)
	if err != nil {
		return nil, err
	}

	return subs, nil
}

func (p *Plugin) GetSubscriptionsByChannel(channelID string) ([]*subscription.Subscription, error) {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil, err
	}

	return filterSubscriptionsByChannel(subs, channelID), nil
}

func filterSubscriptionsByChannel(subs *Subscriptions, channelID string) []*subscription.Subscription {
	var filteredSubs []*subscription.Subscription

	for _, v := range subs.Repositories {
		for _, s := range v {
			if s.ChannelID == channelID {
				filteredSubs = append(filteredSubs, s)
			}
		}
	}

	return filteredSubs
}

func (p *Plugin) AddSubscription(fullPath string, sub *subscription.Subscription) (*Subscriptions, error) {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil, err
	}

	repoSubs := subs.Repositories[fullPath]
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

	subs.Repositories[fullPath] = repoSubs
	return subs, p.StoreSubscriptions(subs)
}

func (p *Plugin) GetSubscriptions() (*Subscriptions, error) {
	var subscriptions *Subscriptions

	err := p.client.KV.Get(SubscriptionsKey, &subscriptions)
	if err != nil {
		p.client.Log.Warn("can't get subscriptions from kvstore", "err", err.Error())
		return nil, err
	}

	if subscriptions == nil {
		subscriptions = &Subscriptions{Repositories: map[string][]*subscription.Subscription{}}
	}

	return subscriptions, nil
}

func (p *Plugin) StoreSubscriptions(s *Subscriptions) error {
	if _, err := p.client.KV.Set(SubscriptionsKey, s); err != nil {
		p.client.Log.Warn("can't set subscriptions in kvstore", "err", err.Error())
	}
	return nil
}

func (p *Plugin) GetSubscribedChannelsForProject(
	ctx context.Context,
	namespace string,
	project string,
	isPublicVisibility bool,
) []*subscription.Subscription {
	var subsForRepo []*subscription.Subscription

	subs, err := p.GetSubscriptions()
	if err != nil {
		p.client.Log.Warn("can't retrieve subscriptions", "err", err.Error())
		return nil
	}

	// Add subscriptions for the specific repo
	fullPath := fullPathFromNamespaceAndProject(namespace, project)
	if subs.Repositories[fullPath] != nil {
		subsForRepo = append(subsForRepo, subs.Repositories[fullPath]...)
	}

	// Add subscriptions for the namespace
	namespacePath := fullPathFromNamespaceAndProject(namespace, "")
	if namespacePath != fullPath && subs.Repositories[namespacePath] != nil {
		subsForRepo = append(subsForRepo, subs.Repositories[namespacePath]...)
	}

	if len(subsForRepo) == 0 {
		return nil
	}

	subsToReturn := make([]*subscription.Subscription, 0, len(subsForRepo))
	for _, sub := range subsForRepo {
		if !isPublicVisibility && !p.permissionToProject(ctx, sub.CreatorID, namespace, project) {
			continue
		}
		subsToReturn = append(subsToReturn, sub)
	}

	return subsToReturn
}

// Unsubscribe deletes the link between namespace/project and channelID.
// Returns true if subscription was found, false otherwise.
func (p *Plugin) Unsubscribe(channelID string, fullPath string) (bool, *Subscriptions, error) {
	if fullPath == "" {
		return false, nil, errors.New("invalid repository")
	}

	subs, err := p.GetSubscriptions()
	if err != nil {
		return false, nil, err
	}

	var removed bool

	// We don't know whether fullPath is a namespace or project, so we have to check both cases
	for _, path := range []string{fullPath, fullPath + "/"} {
		pathSubs := subs.Repositories[path]
		if pathSubs == nil {
			continue
		}

		pathRemoved := false
		for index, sub := range pathSubs {
			if sub.ChannelID == channelID {
				pathSubs = append(pathSubs[:index], pathSubs[index+1:]...)
				pathRemoved = true
				break
			}
		}

		if pathRemoved {
			if len(pathSubs) > 0 {
				subs.Repositories[path] = pathSubs
			} else {
				delete(subs.Repositories, path)
			}
			removed = true
		}
	}

	if !removed {
		return false, subs, nil
	}
	return true, subs, p.StoreSubscriptions(subs)
}
