// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {GlobalState as ReduxGlobalState} from '@mattermost/types/store';

import type combinedReducers from '../reducers';

export type GlobalState = ReduxGlobalState & {
    'plugins-com.github.manland.mattermost-plugin-gitlab': PluginState
};

export type PluginState = ReturnType<typeof combinedReducers>

export const pluginStateKey = 'plugins-com.github.manland.mattermost-plugin-gitlab' as const;
