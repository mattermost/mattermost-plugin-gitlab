// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/flow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepAfterDefaultInstanceDeclined(t *testing.T) {
	t.Run("continues to OAuth when the configured instance already is the default", func(t *testing.T) {
		fm := &FlowManager{
			getConfiguration: func() *configuration {
				return &configuration{DefaultInstanceName: "production"}
			},
		}

		next, state, err := fm.stepAfterDefaultInstanceDeclined("production")

		require.NoError(t, err)
		assert.Equal(t, stepOAuthConnect, next)
		assert.Nil(t, state)
	})

	t.Run("stops the wizard when another instance remains the default", func(t *testing.T) {
		fm := &FlowManager{
			getConfiguration: func() *configuration {
				return &configuration{DefaultInstanceName: "production"}
			},
		}

		next, state, err := fm.stepAfterDefaultInstanceDeclined("staging")

		require.NoError(t, err)
		assert.Equal(t, stepDefaultInstanceDeclined, next)
		assert.Equal(t, flow.State{keyDefaultInstanceName: "production"}, state)
	})
}
