package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

type dataUnsubscribeTestStruct struct {
	name         string
	channelID    string
	repoName     string
	initMock     func() *plugintest.API
	shouldDelete bool
	shouldError  bool
}

var dataUnsubscribeTest = []dataUnsubscribeTestStruct{
	{
		name:      "should delete existing subscription",
		channelID: "1",
		repoName:  "owner/project",
		initMock: func() *plugintest.API {
			m := &plugintest.API{}
			kvget := `{"Repositories":{"owner/project":[{"ChannelID":"1","CreatorID":"1","Features":"all","Repository":"owner/project"}]}}`
			kvset := `{"Repositories":{}}`
			m.On("KVGet", SubscriptionsKey).Return([]byte(kvget), nil).Once()
			m.On("KVSet", SubscriptionsKey, []byte(kvset)).Return(nil).Once()
			return m
		},
		shouldDelete: true,
		shouldError:  false,
	}, {
		name:      "should keep other channel",
		channelID: "1",
		repoName:  "owner/project",
		initMock: func() *plugintest.API {
			m := &plugintest.API{}
			kvget := `{"Repositories":{"owner/project":[{"ChannelID":"1","CreatorID":"1","Features":"all","Repository":"owner/project"},{"ChannelID":"2","CreatorID":"1","Features":"all","Repository":"owner/project"}]}}`
			kvset := `{"Repositories":{"owner/project":[{"ChannelID":"2","CreatorID":"1","Features":"all","Repository":"owner/project"}]}}`
			m.On("KVGet", SubscriptionsKey).Return([]byte(kvget), nil).Once()
			m.On("KVSet", SubscriptionsKey, []byte(kvset)).Return(nil).Once()
			return m
		},
		shouldDelete: true,
		shouldError:  false,
	}, {
		name:      "should not delete if not exist",
		channelID: "2",
		repoName:  "owner/project",
		initMock: func() *plugintest.API {
			m := &plugintest.API{}
			kvget := `{"Repositories":{"owner/project":[{"ChannelID":"1","CreatorID":"1","Features":"all","Repository":"owner/project"}]}}`
			m.On("KVGet", SubscriptionsKey).Return([]byte(kvget), nil).Once()
			return m
		},
		shouldDelete: false,
		shouldError:  false,
	}, {
		name:      "should refuse empty repo",
		channelID: "1",
		repoName:  "",
		initMock: func() *plugintest.API {
			return &plugintest.API{}
		},
		shouldDelete: false,
		shouldError:  true,
	}, {
		name:      "should delete organization",
		channelID: "1",
		repoName:  "owner",
		initMock: func() *plugintest.API {
			m := &plugintest.API{}
			kvget := `{"Repositories":{"owner/":[{"ChannelID":"1","CreatorID":"1","Features":"all","Repository":"owner/"}]}}`
			kvset := `{"Repositories":{}}`
			m.On("KVGet", SubscriptionsKey).Return([]byte(kvget), nil).Once()
			m.On("KVSet", SubscriptionsKey, []byte(kvset)).Return(nil).Once()
			return m
		},
		shouldDelete: true,
		shouldError:  false,
	},
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	for _, test := range dataUnsubscribeTest {
		t.Run(test.name, func(t *testing.T) {
			m := test.initMock()
			p := &Plugin{configuration: &configuration{}}
			p.SetAPI(m)
			res, err := p.Unsubscribe(test.channelID, test.repoName)
			assert.Equal(t, test.shouldDelete, res)
			assert.Equal(t, test.shouldError, err != nil)
			m.AssertExpectations(t)
		})
	}
}
