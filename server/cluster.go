// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"

	"github.com/mattermost/mattermost/server/public/model"
)

const (
	oauthCompleteEventID   = "oauth-complete"
	instanceChangedEventID = "instance-changed"
)

func (p *Plugin) sendOAuthCompleteEvent(event OAuthCompleteEvent) {
	p.sendMessageToCluster(oauthCompleteEventID, event)
}

// sendInstanceChangedEvent tells the other nodes to drop their cached effective instance.
// Installing and uninstalling an instance only touches the KV store, so without this the other
// nodes would keep serving a stale instance until the next plugin configuration change.
func (p *Plugin) sendInstanceChangedEvent() {
	p.sendMessageToCluster(instanceChangedEventID, struct{}{})
}

func (p *Plugin) sendMessageToCluster(id string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		p.client.Log.Warn("couldn't get JSON bytes from cluster message",
			"id", id,
			"error", err,
		)
		return
	}

	event := model.PluginClusterEvent{Id: id, Data: b}
	opts := model.PluginClusterEventSendOptions{
		SendType: model.PluginClusterEventSendTypeReliable,
	}

	if err := p.client.Cluster.PublishPluginEvent(event, opts); err != nil {
		p.client.Log.Warn("error publishing cluster event",
			"id", id,
			"error", err,
		)
	}
}

func (p *Plugin) HandleClusterEvent(ev model.PluginClusterEvent) {
	switch ev.Id {
	case oauthCompleteEventID:
		var event OAuthCompleteEvent
		if err := json.Unmarshal(ev.Data, &event); err != nil {
			p.client.Log.Warn("cannot unmarshal cluster event with OAuth complete event", "error", err)
			return
		}

		p.oauthBroker.publishOAuthComplete(event.UserID, event.Err, true)
	case instanceChangedEventID:
		p.refreshLocalEffectiveInstance()
	default:
		p.client.Log.Warn("unknown cluster event", "id", ev.Id)
	}
}
