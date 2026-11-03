// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"encoding/json"

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
	return p.modifySubscriptions(func(subs *Subscriptions) error {
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
		return nil
	})
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

// errStopModify is a sentinel returned from a modifySubscriptions mutate
// function to abort the atomic write without performing it and without treating
// it as a store failure. SetAtomicWithRetries returns immediately (no retry)
// when the callback errors, so the caller distinguishes this from a real error.
var errStopModify = errors.New("stop modifying subscriptions")

// modifySubscriptions performs an atomic read-modify-write of the whole
// subscriptions blob. The mutate function runs inside SetAtomicWithRetries'
// callback, so on every retry it receives the freshly re-read state and
// re-applies its change on top of it. This prevents concurrent mutations across
// channels from silently clobbering one another (lost updates). It returns the
// resulting subscriptions on success.
func (p *Plugin) modifySubscriptions(mutate func(*Subscriptions) error) (*Subscriptions, error) {
	var result *Subscriptions

	err := p.client.KV.SetAtomicWithRetries(SubscriptionsKey, func(oldValue []byte) (any, error) {
		subs := &Subscriptions{Repositories: map[string][]*subscription.Subscription{}}
		if len(oldValue) > 0 {
			if err := json.Unmarshal(oldValue, subs); err != nil {
				return nil, errors.Wrap(err, "can't unmarshal subscriptions from kvstore")
			}
			if subs.Repositories == nil {
				subs.Repositories = map[string][]*subscription.Subscription{}
			}
		}

		if err := mutate(subs); err != nil {
			return nil, err
		}

		result = subs
		return subs, nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (p *Plugin) GetSubscribedChannelsForProject(
	ctx context.Context,
	namespace string,
	project string,
	isPublicVisibility bool,
	isConfidential bool,
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
		if (!isPublicVisibility || isConfidential) && !p.permissionToProject(ctx, sub.CreatorID, namespace, project) {
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

	var removed bool
	var current *Subscriptions

	subs, err := p.modifySubscriptions(func(subs *Subscriptions) error {
		removed = false
		current = subs

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
			return errStopModify
		}

		return nil
	})

	if err != nil {
		// errStopModify signals that no matching subscription existed; that is a
		// no-op, not a failure. Any other error is a real store/read failure and
		// must be surfaced instead of being reported as "not subscribed".
		if errors.Cause(err) == errStopModify {
			return false, current, nil
		}
		return false, nil, err
	}

	return true, subs, nil
}
