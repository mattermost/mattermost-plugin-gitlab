// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
)

// raceKVAPI is an in-memory plugin.API whose KVGet and KVSetWithOptions
// implement real compare-and-set semantics, so SetAtomicWithRetries behaves
// exactly as it does against the server KV store. Every other API method is
// inherited from plugintest.API and is unused by the subscription store.
type raceKVAPI struct {
	plugintest.API
	mu    sync.Mutex
	store map[string][]byte
}

func newRaceKVAPI() *raceKVAPI {
	return &raceKVAPI{store: map[string][]byte{}}
}

func (a *raceKVAPI) KVGet(key string) ([]byte, *model.AppError) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.store[key], nil
}

func (a *raceKVAPI) KVSetWithOptions(key string, value []byte, options model.PluginKVSetOptions) (bool, *model.AppError) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if options.Atomic {
		if !bytes.Equal(a.store[key], options.OldValue) {
			return false, nil
		}
	}

	if value == nil {
		delete(a.store, key)
	} else {
		a.store[key] = value
	}
	return true, nil
}

// TestSubscriptionRace drives the plugin's real AddSubscription against an
// in-memory KV store with real compare-and-set semantics. Many channels
// subscribe to the same repository concurrently; afterwards we compare how many
// AddSubscription calls reported success (nil error) against how many
// subscriptions actually persisted.
//
// Correct behaviour: persisted == reported-success (every reported success is
// durable). Before the fix, StoreSubscriptions did a plain non-atomic Set and
// swallowed the error, so concurrent writers clobbered each other and persisted
// was far lower than reported-success (silent lost updates).
func TestSubscriptionRace(t *testing.T) {
	const numChannels = 200

	api := newRaceKVAPI()
	p := &Plugin{configuration: &configuration{}}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	var wg sync.WaitGroup
	var mu sync.Mutex
	reportedSuccess := 0

	for i := 0; i < numChannels; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sub := &subscription.Subscription{
				ChannelID:  fmt.Sprintf("channel-%d", i),
				Repository: "namespace/project",
			}
			if _, err := p.AddSubscription("namespace/project", sub); err == nil {
				mu.Lock()
				reportedSuccess++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	subs, err := p.GetSubscriptions()
	if err != nil {
		t.Fatalf("GetSubscriptions failed: %v", err)
	}
	persisted := 0
	for _, chans := range subs.Repositories {
		persisted += len(chans)
	}

	silentlyLost := reportedSuccess - persisted
	t.Logf("concurrent AddSubscription calls : %d", numChannels)
	t.Logf("AddSubscription returned success : %d", reportedSuccess)
	t.Logf("subscriptions actually persisted : %d", persisted)
	t.Logf("silently lost (success but gone) : %d", silentlyLost)

	if silentlyLost > 0 {
		t.Fatalf("lost-update bug: %d subscriptions were reported as saved but silently dropped", silentlyLost)
	}
}
