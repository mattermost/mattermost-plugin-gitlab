package main

import (
	"encoding/json"

	"github.com/mattermost/mattermost-server/v6/model"
)

const (
	oauthCompleteEventID = "oauth-complete"
)

func (p *Plugin) sendOAuthCompleteEvent(event OAuthCompleteEvent) {
	p.sendMessageToCluster(oauthCompleteEventID, event)
}

func (p *Plugin) sendMessageToCluster(id string, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		p.API.LogWarn("couldn't get JSON bytes from cluster message",
			"id", id,
			"error", err,
		)
		return
	}

	event := model.PluginClusterEvent{Id: id, Data: b}
	opts := model.PluginClusterEventSendOptions{
		SendType: model.PluginClusterEventSendTypeReliable,
	}

	if err := p.API.PublishPluginClusterEvent(event, opts); err != nil {
		p.API.LogWarn("error publishing cluster event",
			"id", id,
			"error", err,
		)
	}
}

func (p *Plugin) HandleClusterEvent(ev model.PluginClusterEvent) {
	p.API.LogError("received cluster event", "id", ev.Id)

	switch ev.Id {
	case oauthCompleteEventID:
		var event OAuthCompleteEvent
		if err := json.Unmarshal(ev.Data, &event); err != nil {
			p.API.LogWarn("cannot unmarshal cluster event with OAuth complete event", "error", err)
			return
		}

		p.oauthBroker.publishOAuthComplete(event.UserID, event.Err, true)
	default:
		p.API.LogWarn("unknown cluset event", "id", ev.Id)
	}
}
